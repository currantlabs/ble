package gatt

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

// NewClient ...
func NewClient(l2c bt.Conn) *Client {
	h := newNHandler()
	p := &Client{
		handler: h,
		c:       att.NewClient(l2c, h),
		l2c:     l2c,
	}
	go p.c.Loop()
	return p
}

// Client ...
type Client struct {
	svcs []bt.Service

	name string

	handler *nHandlers
	c       *att.Client
	l2c     bt.Conn
}

// Address ...
func (p *Client) Address() bt.Addr { return p.l2c.RemoteAddr().(net.HardwareAddr) }

// Name ...
func (p *Client) Name() string { return p.name }

// Services ...
func (p *Client) Services() []bt.Service { return p.svcs }

// DiscoverServices ...
func (p *Client) DiscoverServices(filter []uuid.UUID) ([]bt.Service, error) {
	start := uint16(0x0001)
	for {
		length, b, err := p.c.ReadByGroupType(start, 0xFFFF, uuid.UUID(attrPrimaryServiceUUID))
		if err == bt.ErrAttrNotFound {
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
				p.svcs = append(p.svcs, &svc{
					uuid: u,
					attr: attr{h: h, endh: endh},
				})
			}
			if endh == 0xFFFF {
				return p.svcs, nil
			}
			start = endh + 1
			b = b[length:]
		}
	}
}

// DiscoverIncludedServices ...
func (p *Client) DiscoverIncludedServices(ss []uuid.UUID, s bt.Service) ([]bt.Service, error) {
	return nil, nil
}

// DiscoverCharacteristics ...
func (p *Client) DiscoverCharacteristics(filter []uuid.UUID, s bt.Service) ([]bt.Characteristic, error) {
	svc := s.(*svc)
	start := svc.attr.h
	var lastChar *char
	for start <= svc.attr.endh {
		length, b, err := p.c.ReadByType(start, svc.attr.endh, attrCharacteristicUUID)
		if err == bt.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			props := bt.Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u := uuid.UUID(b[5:length])
			c := &char{
				uuid:  u,
				props: props,
				attr: attr{
					h:    h,
					vh:   vh,
					endh: svc.attr.endh,
				},
			}
			if filter == nil || uuid.Contains(filter, u) {
				svc.chars = append(svc.chars, c)
			}
			if lastChar != nil {
				lastChar.attr.endh = c.attr.h - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.Characteristics(), nil
}

// DiscoverDescriptors ...
func (p *Client) DiscoverDescriptors(filter []uuid.UUID, c bt.Characteristic) ([]bt.Descriptor, error) {
	char := c.(*char)
	start := char.attr.vh + 1
	for start <= char.attr.endh {
		fmt, b, err := p.c.FindInformation(start, char.attr.endh)
		if err == bt.ErrAttrNotFound {
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
			d := &desc{
				uuid: u,
				attr: attr{h: h},
			}
			if filter == nil || uuid.Contains(filter, u) {
				char.descs = append(char.descs, d)
			}
			if u.Equal(attrClientCharacteristicConfigUUID) {
				char.cccd = d
			}
			start = h + 1
			b = b[length:]
		}
	}
	return c.Descriptors(), nil
}

// ReadCharacteristic ...
func (p *Client) ReadCharacteristic(c bt.Characteristic) ([]byte, error) {
	char := c.(*char)
	return p.c.Read(char.attr.vh)
}

// ReadLongCharacteristic ...
func (p *Client) ReadLongCharacteristic(c bt.Characteristic) ([]byte, error) {
	return nil, nil
}

// WriteCharacteristic ...
func (p *Client) WriteCharacteristic(c bt.Characteristic, value []byte, noRsp bool) error {
	char := c.(*char)
	if noRsp {
		p.c.WriteCommand(char.attr.vh, value)
		return nil
	}
	return p.c.Write(char.attr.vh, value)
}

// ReadDescriptor ...
func (p *Client) ReadDescriptor(d bt.Descriptor) ([]byte, error) {
	desc := d.(*desc)
	return p.c.Read(desc.attr.h)
}

// WriteDescriptor ...
func (p *Client) WriteDescriptor(d bt.Descriptor, v []byte) error {
	desc := d.(*desc)
	return p.c.Write(desc.attr.h, v)
}

// ReadRSSI ...
func (p *Client) ReadRSSI() int { return -1 }

// ExchangeMTU informs the server of the client’s maximum receive MTU size and
// request the server to respond with its maximum receive MTU size. [Vol 3, Part F, 3.4.2.1]
func (p *Client) ExchangeMTU(mtu int) (int, error) {
	txMTU, err := p.c.ExchangeMTU(mtu + 3)
	if err != nil {
		return 0, err
	}
	return txMTU - 3, nil
}

// Subscribe ...
func (p *Client) Subscribe(c bt.Characteristic, ind bool, h bt.NotificationHandler) error {
	char := c.(*char)
	if ind {
		return p.setHandlers(char.cccd.attr.h, char.attr.vh, cccIndicate, h)
	}
	return p.setHandlers(char.cccd.attr.h, char.attr.vh, cccNotify, h)
}

func (p *Client) setHandlers(cccdh, vh, flag uint16, h bt.NotificationHandler) error {
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
	if flag == cccNotify {
		s.nHandler = h
	} else {
		s.iHandler = h
	}
	return p.c.Write(s.cccdh, v)
}

// ClearSubscriptions ...
func (p *Client) ClearSubscriptions() error {
	for _, s := range p.handler.stubs {
		if err := p.c.Write(s.cccdh, make([]byte, 2)); err != nil {
			return err
		}
	}
	return nil
}

// CancelConnection disconnects the connection.
func (p *Client) CancelConnection() error {
	return p.l2c.Close()
}

type nHandlers struct {
	*sync.RWMutex
	stubs map[uint16]*stub
}

type stub struct {
	cccdh    uint16
	ccc      uint16
	nHandler bt.NotificationHandler
	iHandler bt.NotificationHandler
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
