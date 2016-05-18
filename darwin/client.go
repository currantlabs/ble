package darwin

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/darwin/xpc"
)

// Client ...
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

// Address ...
func (p *Client) Address() bt.Addr {
	return p.conn.RemoteAddr()
}

// Name ...
func (p *Client) Name() string {
	return p.name
}

// Services ...
func (p *Client) Services() []*bt.Service {
	return p.svcs
}

// DiscoverServices ...
func (p *Client) DiscoverServices(ss []bt.UUID) ([]*bt.Service, error) {
	rsp := p.conn.sendReq(45, xpc.Dict{
		"kCBMsgArgDeviceUUID": p.id,
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
	p.svcs = svcs
	return svcs, nil
}

// DiscoverIncludedServices ...
func (p *Client) DiscoverIncludedServices(ss []bt.UUID, s *bt.Service) ([]*bt.Service, error) {
	rsp := p.conn.sendReq(60, xpc.Dict{
		"kCBMsgArgDeviceUUID":         p.id,
		"kCBMsgArgServiceStartHandle": s.Handle,
		"kCBMsgArgServiceEndHandle":   s.EndHandle,
		"kCBMsgArgUUIDs":              uuidSlice(ss),
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return nil, bt.ErrNotImplemented
}

// DiscoverCharacteristics ...
func (p *Client) DiscoverCharacteristics(cs []bt.UUID, s *bt.Service) ([]*bt.Characteristic, error) {
	rsp := p.conn.sendReq(62, xpc.Dict{
		"kCBMsgArgDeviceUUID":         p.id,
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

// DiscoverDescriptors ...
func (p *Client) DiscoverDescriptors(ds []bt.UUID, c *bt.Characteristic) ([]*bt.Descriptor, error) {
	rsp := p.conn.sendReq(70, xpc.Dict{
		"kCBMsgArgDeviceUUID":                p.id,
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

// ReadCharacteristic ...
func (p *Client) ReadCharacteristic(c *bt.Characteristic) ([]byte, error) {
	rsp := p.conn.sendReq(65, xpc.Dict{
		"kCBMsgArgDeviceUUID":                p.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return rsp.data(), nil
}

// ReadLongCharacteristic ...
func (p *Client) ReadLongCharacteristic(c *bt.Characteristic) ([]byte, error) {
	return nil, bt.ErrNotImplemented
}

// WriteCharacteristic ...
func (p *Client) WriteCharacteristic(c *bt.Characteristic, b []byte, noRsp bool) error {
	args := xpc.Dict{
		"kCBMsgArgDeviceUUID":                p.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgData":                      b,
		"kCBMsgArgType":                      map[bool]int{false: 0, true: 1}[noRsp],
	}
	if noRsp {
		p.conn.sendCmd(66, args)
		return nil
	}
	rsp := p.conn.sendReq(65, args)
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}

// ReadDescriptor ...
func (p *Client) ReadDescriptor(d *bt.Descriptor) ([]byte, error) {
	rsp := p.conn.sendReq(77, xpc.Dict{
		"kCBMsgArgDeviceUUID":       p.id,
		"kCBMsgArgDescriptorHandle": d.Handle,
	})
	if res := rsp.result(); res != 0 {
		return nil, bt.ATTError(res)
	}
	return rsp.data(), nil
}

// WriteDescriptor ...
func (p *Client) WriteDescriptor(d *bt.Descriptor, b []byte) error {
	rsp := p.conn.sendReq(78, xpc.Dict{
		"kCBMsgArgDeviceUUID":       p.id,
		"kCBMsgArgDescriptorHandle": d.Handle,
		"kCBMsgArgData":             b,
	})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}

// ReadRSSI ...
func (p *Client) ReadRSSI() int {
	rsp := p.conn.sendReq(44, xpc.Dict{"kCBMsgArgDeviceUUID": p.id})
	if res := rsp.result(); res != 0 {
		return 0
	}
	return rsp.rssi()
}

// ExchangeMTU ...
func (p *Client) ExchangeMTU(mtu int) (int, error) {
	return 23, bt.ErrNotImplemented
}

// Subscribe ...
func (p *Client) Subscribe(c *bt.Characteristic, ind bool, fn bt.NotificationHandler) error {
	p.conn.Lock()
	defer p.conn.Unlock()
	p.conn.subs[c.Handle] = fn
	rsp := p.conn.sendReq(68, xpc.Dict{
		"kCBMsgArgDeviceUUID":                p.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgState":                     1,
	})
	if res := rsp.result(); res != 0 {
		delete(p.conn.subs, c.Handle)
		return bt.ATTError(res)
	}
	return nil
}

// Unsubscribe ...
func (p *Client) Unsubscribe(c *bt.Characteristic, ind bool) error {
	rsp := p.conn.sendReq(68, xpc.Dict{
		"kCBMsgArgDeviceUUID":                p.id,
		"kCBMsgArgCharacteristicHandle":      c.Handle,
		"kCBMsgArgCharacteristicValueHandle": c.ValueHandle,
		"kCBMsgArgState":                     0,
	})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	p.conn.Lock()
	defer p.conn.Unlock()
	delete(p.conn.subs, c.Handle)
	return nil
}

// ClearSubscriptions ...
func (p *Client) ClearSubscriptions() error {
	return nil
}

// CancelConnection ...
func (p *Client) CancelConnection() error {
	rsp := p.conn.sendReq(32, xpc.Dict{"kCBMsgArgDeviceUUID": p.id})
	if res := rsp.result(); res != 0 {
		return bt.ATTError(res)
	}
	return nil
}
