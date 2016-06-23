package darwin

import (
	"github.com/currantlabs/ble/darwin/xpc"
	"github.com/currantlabs/x/io/bt"
)

// A Client is a GATT client.
type Client struct {
	svcs []*bt.Service
	name string

	id   xpc.UUID
	conn *conn
}

// NewClient ...
func NewClient(c bt.Conn) *Client {
	return &Client{
		conn: c.(*conn),
		id:   xpc.MakeUUID(c.RemoteAddr().String()),
	}
}

// Address returns UUID of the remote peripheral.
func (cln *Client) Address() bt.Addr {
	return cln.conn.RemoteAddr()
}

// Name returns the name of the remote peripheral.
// This can be the advertised name, if exists, or the GAP device name, which takes priority.
func (cln *Client) Name() string {
	return cln.name
}

// Services returns discovered services.
func (cln *Client) Services() []*bt.Service {
	return cln.svcs
}

// DiscoverServices finds all the primary services on a server. [Vol 3, Part G, 4.4.1]
// If filter is specified, only filtered services are returned.
func (cln *Client) DiscoverServices(ss []bt.UUID) ([]*bt.Service, error) {
	rsp := cln.conn.sendReq(45, xpc.Dict{
		"kCBMsgArgDeviceUUID": cln.id,
		"kCBMsgArgUUIDs":      uuidSlice(ss),
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	svcs := []*bt.Service{}
	for _, xss := range rsp.services() {
		xs := msg(xss.(xpc.Dict))
		svcs = append(svcs, &bt.Service{
			UUID:      bt.MustParse(xs.uuid()),
			Handle:    uint16(xs.serviceStartHandle()),
			EndHandle: uint16(xs.serviceEndHandle()),
		})
	}
	cln.svcs = svcs
	return svcs, nil
}

// DiscoverIncludedServices finds the included services of a service. [Vol 3, Part G, 4.5.1]
// If filter is specified, only filtered services are returned.
func (cln *Client) DiscoverIncludedServices(ss []bt.UUID, s *bt.Service) ([]*bt.Service, error) {
	rsp := cln.conn.sendReq(60, xpc.Dict{
		"kCBMsgArgDeviceUUID":         cln.id,
		"kCBMsgArgServiceStartHandle": s.Handle,
		"kCBMsgArgServiceEndHandle":   s.EndHandle,
		"kCBMsgArgUUIDs":              uuidSlice(ss),
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return nil, bt.ErrNotImplemented
}

// DiscoverCharacteristics finds all the characteristics within a service. [Vol 3, Part G, 4.6.1]
// If filter is specified, only filtered characteristics are returned.
func (cln *Client) DiscoverCharacteristics(cs []bt.UUID, s *bt.Service) ([]*bt.Characteristic, error) {
	rsp := cln.conn.sendReq(62, xpc.Dict{
		"kCBMsgArgDeviceUUID":         cln.id,
		"kCBMsgArgServiceStartHandle": s.Handle,
		"kCBMsgArgServiceEndHandle":   s.EndHandle,
		"kCBMsgArgUUIDs":              uuidSlice(cs),
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	for _, xcs := range rsp.characteristics() {
		xc := msg(xcs.(xpc.Dict))
		s.Characteristics = append(s.Characteristics, &bt.Characteristic{
			UUID:        bt.MustParse(xc.uuid()),
			Property:    bt.Property(xc.characteristicProperties()),
			Handle:      uint16(xc.characteristicHandle()),
			ValueHandle: uint16(xc.characteristicValueHandle()),
		})
	}
	return s.Characteristics, nil
}

// DiscoverDescriptors finds all the descriptors within a characteristic. [Vol 3, Part G, 4.7.1]
// If filter is specified, only filtered descriptors are returned.
func (cln *Client) DiscoverDescriptors(ds []bt.UUID, c *bt.Characteristic) ([]*bt.Descriptor, error) {
	rsp := cln.conn.sendReq(70, xpc.Dict{
		"kCBMsgArgDeviceUUID":                cln.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgUUIDs":                     uuidSlice(ds),
	})
	for _, xds := range rsp.descriptors() {
		xd := msg(xds.(xpc.Dict))
		c.Descriptors = append(c.Descriptors, &bt.Descriptor{
			UUID:   bt.MustParse(xd.uuid()),
			Handle: uint16(xd.descriptorHandle()),
		})
	}
	return c.Descriptors, nil
}

// ReadCharacteristic reads a characteristic value from a server. [Vol 3, Part G, 4.8.1]
func (cln *Client) ReadCharacteristic(c *bt.Characteristic) ([]byte, error) {
	rsp := cln.conn.sendReq(65, xpc.Dict{
		"kCBMsgArgDeviceUUID":                cln.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return rsp.data(), nil
}

// ReadLongCharacteristic reads a characteristic value which is longer than the MTU. [Vol 3, Part G, 4.8.3]
func (cln *Client) ReadLongCharacteristic(c *bt.Characteristic) ([]byte, error) {
	return nil, bt.ErrNotImplemented
}

// WriteCharacteristic writes a characteristic value to a server. [Vol 3, Part G, 4.9.3]
func (cln *Client) WriteCharacteristic(c *bt.Characteristic, b []byte, noRsp bool) error {
	args := xpc.Dict{
		"kCBMsgArgDeviceUUID":                cln.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgData":                      b,
		"kCBMsgArgType":                      map[bool]int{false: 0, true: 1}[noRsp],
	}
	if noRsp {
		cln.conn.sendCmd(66, args)
		return nil
	}
	rsp := cln.conn.sendReq(65, args)
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}

// ReadDescriptor reads a characteristic descriptor from a server. [Vol 3, Part G, 4.12.1]
func (cln *Client) ReadDescriptor(d *bt.Descriptor) ([]byte, error) {
	rsp := cln.conn.sendReq(77, xpc.Dict{
		"kCBMsgArgDeviceUUID":       cln.id,
		"kCBMsgArgDescriptorHandle": d.Handle,
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return rsp.data(), nil
}

// WriteDescriptor writes a characteristic descriptor to a server. [Vol 3, Part G, 4.12.3]
func (cln *Client) WriteDescriptor(d *bt.Descriptor, b []byte) error {
	rsp := cln.conn.sendReq(78, xpc.Dict{
		"kCBMsgArgDeviceUUID":       cln.id,
		"kCBMsgArgDescriptorHandle": d.Handle,
		"kCBMsgArgData":             b,
	})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}

// ReadRSSI retrieves the current RSSI value of remote peripheral. [Vol 2, Part E, 7.5.4]
func (cln *Client) ReadRSSI() int {
	rsp := cln.conn.sendReq(44, xpc.Dict{"kCBMsgArgDeviceUUID": cln.id})
	if res := rsp.result(); res != 0 {
		return 0
	}
	return rsp.rssi()
}

// ExchangeMTU set the ATT_MTU to the maximum possible value that can be
// supported by both devices [Vol 3, Part G, 4.3.1]
func (cln *Client) ExchangeMTU(mtu int) (int, error) {
	// TODO: find the xpc command to tell OS X the rxMTU we can handle.
	return cln.conn.TxMTU(), nil
}

// Subscribe subscribes to indication (if ind is set true), or notification of a
// characteristic value. [Vol 3, Part G, 4.10 & 4.11]
func (cln *Client) Subscribe(c *bt.Characteristic, ind bool, fn bt.NotificationHandler) error {
	cln.conn.Lock()
	defer cln.conn.Unlock()
	cln.conn.subs[c.Handle] = fn
	rsp := cln.conn.sendReq(68, xpc.Dict{
		"kCBMsgArgDeviceUUID":                cln.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgState":                     1,
	})
	if res := rsp.result(); res != 0 {
		delete(cln.conn.subs, c.Handle)
		return bt.ATTError(res)
	}
	return nil
}

// Unsubscribe unsubscribes to indication (if ind is set true), or notification
// of a specified characteristic value. [Vol 3, Part G, 4.10 & 4.11]
func (cln *Client) Unsubscribe(c *bt.Characteristic, ind bool) error {
	rsp := cln.conn.sendReq(68, xpc.Dict{
		"kCBMsgArgDeviceUUID":                cln.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgState":                     0,
	})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	cln.conn.Lock()
	defer cln.conn.Unlock()
	delete(cln.conn.subs, c.Handle)
	return nil
}

// ClearSubscriptions clears all subscriptions to notifications and indications.
func (cln *Client) ClearSubscriptions() error {
	return nil
}

// CancelConnection disconnects the connection.
func (cln *Client) CancelConnection() error {
	rsp := cln.conn.sendReq(32, xpc.Dict{"kCBMsgArgDeviceUUID": cln.id})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}
