package gatt

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/l2cap"
	"github.com/currantlabs/bt/uuid"
)

// NewClient ...
func NewClient(l2c l2cap.Conn) Client {
	h := newNHandler()
	p := &client{
		c:       att.NewClient(l2c, h),
		handler: h,
	}
	go p.c.Loop()
	return p
}

type client struct {
	// NameChanged is called whenever the client GAP device name has changed.
	NameChanged func(*client)

	// ServicedModified is called when one or more service of a client have changed.
	// A list of invalid service is provided in the parameter.
	ServicesModified func(*client, []Service)

	svcs []*Service

	name string
	addr net.HardwareAddr

	handler *nHandlers
	c       *att.Client
}

func (p *client) Address() net.HardwareAddr { return p.addr }
func (p *client) Name() string              { return p.name }
func (p *client) Services() []*Service      { return p.svcs }

func (p *client) DiscoverServices(filter []uuid.UUID) ([]*Service, error) {
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
				p.svcs = append(p.svcs, &Service{uuid: u, h: h, endh: endh})
			}
			if endh == 0xFFFF {
				return p.svcs, nil
			}
			start = endh + 1
			b = b[length:]
		}
	}
}

func (p *client) DiscoverIncludedServices(ss []uuid.UUID, s *Service) ([]*Service, error) {
	return nil, nil
}

func (p *client) DiscoverCharacteristics(filter []uuid.UUID, s *Service) ([]*Characteristic, error) {
	start := s.h
	var lastChar *Characteristic
	for start <= s.endh {
		length, b, err := p.c.ReadByType(start, s.endh, attrCharacteristicUUID)
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
			c := &Characteristic{
				uuid:  u,
				props: props,
				value: attValue{},
				h:     h,
				vh:    vh,
				endh:  s.endh,
			}
			if filter == nil || uuid.Contains(filter, u) {
				s.chars = append(s.chars, c)
			}
			if lastChar != nil {
				lastChar.endh = c.h - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.chars, nil
}

func (p *client) DiscoverDescriptors(filter []uuid.UUID, c *Characteristic) ([]*Descriptor, error) {
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
			d := &Descriptor{uuid: u, h: h}
			if filter == nil || uuid.Contains(filter, u) {
				c.descs = append(c.descs, d)
			}
			if u.Equal(attrClientCharacteristicConfigUUID) {
				c.cccd = d
			}
			start = h + 1
			b = b[length:]
		}
	}
	return c.descs, nil
}

func (p *client) ReadCharacteristic(c *Characteristic) ([]byte, error) { return p.c.Read(c.vh) }

func (p *client) ReadLongCharacteristic(c *Characteristic) ([]byte, error) { return nil, nil }

func (p *client) WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error {
	if noRsp {
		p.c.WriteCommand(c.vh, value)
		return nil
	}
	return p.c.Write(c.vh, value)
}

func (p *client) ReadDescriptor(d *Descriptor) ([]byte, error) { return p.c.Read(d.h) }

func (p *client) WriteDescriptor(d *Descriptor, v []byte) error { return p.c.Write(d.h, v) }

func (p *client) ReadRSSI() int { return -1 }

func (p *client) SetMTU(mtu int) error {
	_, err := p.c.ExchangeMTU(mtu)
	return err
}

func (p *client) SetNotificationHandler(c *Characteristic, h NotificationHandler) error {
	return p.setHandlers(c.cccd.h, c.vh, flagCCCNotify, h)
}

func (p *client) SetIndicationHandler(c *Characteristic, h NotificationHandler) error {
	return p.setHandlers(c.cccd.h, c.vh, flagCCCIndicate, h)
}

func (p *client) setHandlers(cccdh, vh, flag uint16, h NotificationHandler) error {
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

func (p *client) ClearHandlers() error {
	for _, s := range p.handler.stubs {
		if err := p.c.Write(s.cccdh, make([]byte, 2)); err != nil {
			return err
		}
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
