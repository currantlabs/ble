package gatt

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/l2cap"
	"github.com/currantlabs/bt/uuid"
)

// Client represent a remote peripheral device.
type Client struct {
	// NameChanged is called whenever the Client GAP device name has changed.
	NameChanged func(*Client)

	// ServicedModified is called when one or more service of a Client have changed.
	// A list of invalid service is provided in the parameter.
	ServicesModified func(*Client, []*Service)

	svcs []*Service

	name string
	addr net.HardwareAddr

	handler *nHandlers
	c       *att.Client
}

// Address is the platform specific unique ID of the remote peripheral, e.g. MAC for Linux, Client UUID for MacOS.
func (p *Client) Address() net.HardwareAddr {
	return p.addr
}

// Name returns the name of the remote peripheral.
// This can be the advertised name, if exists, or the GAP device name, which takes priority
func (p *Client) Name() string {
	return p.name
}

// Services returnns the services of the remote peripheral which has been discovered.
func (p *Client) Services() []*Service {
	return p.svcs
}

// NewClient ...
func NewClient(l2c l2cap.Conn) *Client {
	h := newNHandler()
	p := &Client{
		c:       att.NewClient(l2c, h),
		handler: h,
	}
	go p.c.Loop()
	return p
}

// DiscoverServices discovers all the primary service on a server. [Vol 3, Parg G, 4.4.1]
// DiscoverServices discover the specified services of the remote peripheral.
// If the specified services is set to nil, all the available services of the remote peripheral are returned.
func (p *Client) DiscoverServices(filter []uuid.UUID) ([]*Service, error) {
	start := uint16(0x0001)
	for {
		length, b, err := p.c.ReadByGroupType(start, 0xFFFF, uuid.UUID(attrPrimaryServiceUUID))
		if err == att.ErrAttrNotFound {
			return p.svcs, nil
		}
		if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u := uuid.UUID(b[4:length])
			if filter == nil || uuid.Contains(filter, u) {
				p.svcs = append(p.svcs, &Service{UUID: u, h: h, endh: endh})
			}
			if endh == 0xFFFF {
				return p.svcs, nil
			}
			start = endh + 1
			b = b[length:]
		}
	}
}

// DiscoverIncludedServices discovers the specified included services of a service.
// If the specified services is set to nil, all the included services of the service are returned.
func (p *Client) DiscoverIncludedServices(ss []uuid.UUID, s *Service) ([]*Service, error) {
	return nil, nil
}

// DiscoverCharacteristics discovers the specified characteristics of a service.
// If the specified characterstics is set to nil, all the characteristic of the service are returned.
func (p *Client) DiscoverCharacteristics(filter []uuid.UUID, s *Service) ([]*Characteristic, error) {
	start := s.h
	var lastChar *Characteristic
	for start <= s.endh {
		length, b, err := p.c.ReadByType(start, s.endh, uuid.UUID(attrCharacteristicUUID))
		if err == att.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			props := Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u := uuid.UUID(b[5:length])
			c := &Characteristic{UUID: u, Property: props, h: h, vh: vh, endh: s.endh}
			if filter == nil || uuid.Contains(filter, u) {
				s.Characteristics = append(s.Characteristics, c)
			}
			if lastChar != nil {
				lastChar.endh = c.h - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.Characteristics, nil
}

// DiscoverDescriptors discovers the descriptors of a characteristic.
// If the specified descriptors is set to nil, all the descriptors of the characteristic are returned.
func (p *Client) DiscoverDescriptors(filter []uuid.UUID, c *Characteristic) ([]*Descriptor, error) {
	start := c.vh + 1
	for start <= c.endh {
		fmt, b, err := p.c.FindInformation(start, c.endh)
		if err == att.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		length := 2 + 2
		if fmt == 0x02 {
			length = 2 + 16
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			u := uuid.UUID(b[2:length])
			d := &Descriptor{UUID: u, h: h}
			if filter == nil || uuid.Contains(filter, u) {
				c.Descriptors = append(c.Descriptors, d)
			}
			if u.Equal(attrClientCharacteristicConfigUUID) {
				c.cccd = d
			}
			start = h + 1
			b = b[length:]
		}
	}
	return c.Descriptors, nil
}

// ReadCharacteristic retrieves the value of a specified characteristic.
func (p *Client) ReadCharacteristic(c *Characteristic) ([]byte, error) { return p.c.Read(c.vh) }

// ReadLongCharacteristic retrieves the value of a specified characteristic that is longer than the MTU.
func (p *Client) ReadLongCharacteristic(c *Characteristic) ([]byte, error) {
	return nil, nil
}

// WriteCharacteristic writes the value of a characteristic.
func (p *Client) WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error {
	if noRsp {
		p.c.WriteCommand(c.vh, value)
		return nil
	}
	return p.c.Write(c.vh, value)
}

// ReadDescriptor retrieves the value of a specified characteristic descriptor.
func (p *Client) ReadDescriptor(d *Descriptor) ([]byte, error) {
	return p.c.Read(d.h)
}

// WriteDescriptor writes the value of a characteristic descriptor.
func (p *Client) WriteDescriptor(d *Descriptor, v []byte) error {
	return p.c.Write(d.h, v)
}

// ReadRSSI retrieves the current RSSI value for the remote peripheral.
func (p *Client) ReadRSSI() int {
	return -1
}

// SetMTU sets the mtu for the remote peripheral.
func (p *Client) SetMTU(mtu int) error {
	_, err := p.c.ExchangeMTU(mtu)
	return err
}

// NotificationHandler ...
type NotificationHandler func(req []byte)

// SetNotificationHandler sets notifications for the value of a specified characteristic.
func (p *Client) SetNotificationHandler(c *Characteristic, h NotificationHandler) error {
	return p.setHandlers(c.cccd.h, c.vh, flagCCCNotify, h)
}

// SetIndicationHandler sets indications for the value of a specified characteristic.
func (p *Client) SetIndicationHandler(c *Characteristic, h NotificationHandler) error {
	return p.setHandlers(c.cccd.h, c.vh, flagCCCIndicate, h)
}

func (p *Client) setHandlers(cccdh, vh, flag uint16, h NotificationHandler) error {
	s, ok := p.handler.stubs[vh]
	if !ok {
		s = &stub{cccdh, 0x0000, nil, nil}
		p.handler.stubs[vh] = s
	}
	switch {
	case h == nil && (s.ccc&flag) == 0:
		return nil
	case h != nil && (s.ccc&flag) != 0:
		return nil
	case h == nil && (s.ccc&flag) != 0:
		s.ccc &= ^uint16(flag)
	case h != nil && (s.ccc&flag) == 0:
		s.ccc |= flag
	}

	v := make([]byte, 2)
	binary.LittleEndian.PutUint16(v, s.ccc)
	if flag == flagCCCNotify {
		s.nHandler = h
	} else {
		s.iHandler = h
	}
	return p.c.Write(s.cccdh, v)
}

// ClearHandlers ...
func (p *Client) ClearHandlers() error {
	for _, s := range p.handler.stubs {
		v := make([]byte, 2)
		binary.LittleEndian.PutUint16(v, 0)
		return p.c.Write(s.cccdh, v)
	}
	return nil
}

type nHandlers struct {
	*sync.RWMutex
	stubs map[uint16]*stub
}

type stub struct {
	cccdh    uint16
	ccc      uint16
	nHandler NotificationHandler
	iHandler NotificationHandler
}

// newNHandler ...
func newNHandler() *nHandlers {
	h := &nHandlers{
		RWMutex: &sync.RWMutex{},
		stubs:   make(map[uint16]*stub),
	}
	return h
}

func (n *nHandlers) HandleNotification(req []byte) {
	n.RLock()
	valueh := att.HandleValueIndication(req).AttributeHandle()
	stub := n.stubs[valueh]
	h := stub.nHandler
	if req[0] == att.HandleValueIndicationCode {
		h = stub.iHandler
	}
	n.RUnlock()
	h(req[3:])
}
