package gatt

import (
	"encoding/binary"
	"log"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/linux/att"
)

const (
	cccNotify   = 0x0001
	cccIndicate = 0x0002
)

// NewClient ...
func NewClient(conn bt.Conn) *Client {
	p := &Client{
		subs: make(map[uint16]*sub),
		conn: conn,
	}
	p.ac = att.NewClient(conn, p)
	go p.ac.Loop()
	return p
}

// Client ...
type Client struct {
	sync.RWMutex

	svcs []*bt.Service
	name string
	subs map[uint16]*sub

	ac   *att.Client
	conn bt.Conn
}

// Address ...
func (p *Client) Address() bt.Addr {
	p.RLock()
	defer p.RUnlock()
	return p.conn.RemoteAddr()
}

// Name ...
func (p *Client) Name() string {
	p.RLock()
	defer p.RUnlock()
	return p.name
}

// Services ...
func (p *Client) Services() []*bt.Service {
	p.RLock()
	defer p.RUnlock()
	return p.svcs
}

// DiscoverServices ...
func (p *Client) DiscoverServices(filter []bt.UUID) ([]*bt.Service, error) {
	p.Lock()
	defer p.Unlock()
	start := uint16(0x0001)
	for {
		length, b, err := p.ac.ReadByGroupType(start, 0xFFFF, bt.PrimaryServiceUUID)
		if err == bt.ErrAttrNotFound {
			return p.svcs, nil
		}
		if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u := bt.UUID(b[4:length])
			if filter == nil || bt.Contains(filter, u) {
				s := &bt.Service{
					UUID:      u,
					Handle:    h,
					EndHandle: endh,
				}
				p.svcs = append(p.svcs, s)
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
func (p *Client) DiscoverIncludedServices(ss []bt.UUID, s *bt.Service) ([]*bt.Service, error) {
	p.Lock()
	defer p.Unlock()
	return nil, nil
}

// DiscoverCharacteristics ...
func (p *Client) DiscoverCharacteristics(filter []bt.UUID, s *bt.Service) ([]*bt.Characteristic, error) {
	p.Lock()
	defer p.Unlock()
	start := s.Handle
	var lastChar *bt.Characteristic
	for start <= s.EndHandle {
		length, b, err := p.ac.ReadByType(start, s.EndHandle, bt.CharacteristicUUID)
		if err == bt.ErrAttrNotFound {
			break
		} else if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			p := bt.Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u := bt.UUID(b[5:length])
			c := &bt.Characteristic{
				UUID:        u,
				Property:    p,
				Handle:      h,
				ValueHandle: vh,
				EndHandle:   s.EndHandle,
			}
			if filter == nil || bt.Contains(filter, u) {
				s.Characteristics = append(s.Characteristics, c)
			}
			if lastChar != nil {
				lastChar.EndHandle = c.Handle - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.Characteristics, nil
}

// DiscoverDescriptors ...
func (p *Client) DiscoverDescriptors(filter []bt.UUID, c *bt.Characteristic) ([]*bt.Descriptor, error) {
	p.Lock()
	defer p.Unlock()
	start := c.ValueHandle + 1
	for start < c.EndHandle {
		fmt, b, err := p.ac.FindInformation(start, c.EndHandle)
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
			u := bt.UUID(b[2:length])
			d := &bt.Descriptor{UUID: u, Handle: h}
			if filter == nil || bt.Contains(filter, u) {
				c.Descriptors = append(c.Descriptors, d)
			}
			if u.Equal(bt.ClientCharacteristicConfigUUID) {
				c.CCCD = d
			}
			start = h + 1
			b = b[length:]
		}
	}
	return c.Descriptors, nil
}

// ReadCharacteristic ...
func (p *Client) ReadCharacteristic(c *bt.Characteristic) ([]byte, error) {
	p.Lock()
	defer p.Unlock()
	return p.ac.Read(c.ValueHandle)
}

// ReadLongCharacteristic ...
func (p *Client) ReadLongCharacteristic(c *bt.Characteristic) ([]byte, error) {
	p.Lock()
	defer p.Unlock()
	return nil, nil
}

// WriteCharacteristic ...
func (p *Client) WriteCharacteristic(c *bt.Characteristic, v []byte, noRsp bool) error {
	p.Lock()
	defer p.Unlock()
	if noRsp {
		p.ac.WriteCommand(c.ValueHandle, v)
		return nil
	}
	return p.ac.Write(c.ValueHandle, v)
}

// ReadDescriptor ...
func (p *Client) ReadDescriptor(d *bt.Descriptor) ([]byte, error) {
	p.Lock()
	defer p.Unlock()
	return p.ac.Read(d.Handle)
}

// WriteDescriptor ...
func (p *Client) WriteDescriptor(d *bt.Descriptor, v []byte) error {
	p.Lock()
	defer p.Unlock()
	return p.ac.Write(d.Handle, v)
}

// ReadRSSI ...
func (p *Client) ReadRSSI() int {
	p.Lock()
	defer p.Unlock()
	// TODO:
	return 0
}

// ExchangeMTU informs the server of the client’s maximum receive MTU size and
// request the server to respond with its maximum receive MTU size. [Vol 3, Part F, 3.4.2.1]
func (p *Client) ExchangeMTU(mtu int) (int, error) {
	p.Lock()
	defer p.Unlock()
	return p.ac.ExchangeMTU(mtu)
}

// Subscribe ...
func (p *Client) Subscribe(c *bt.Characteristic, ind bool, h bt.NotificationHandler) error {
	p.Lock()
	defer p.Unlock()
	if ind {
		return p.setHandlers(c.CCCD.Handle, c.ValueHandle, cccIndicate, h)
	}
	return p.setHandlers(c.CCCD.Handle, c.ValueHandle, cccNotify, h)
}

// Unsubscribe ...
func (p *Client) Unsubscribe(c *bt.Characteristic, ind bool) error {
	p.Lock()
	defer p.Unlock()
	if ind {
		return p.setHandlers(c.CCCD.Handle, c.ValueHandle, cccIndicate, nil)
	}
	return p.setHandlers(c.CCCD.Handle, c.ValueHandle, cccNotify, nil)
}

func (p *Client) setHandlers(cccdh, vh, flag uint16, h bt.NotificationHandler) error {
	s, ok := p.subs[vh]
	if !ok {
		s = &sub{cccdh, 0x0000, nil, nil}
		p.subs[vh] = s
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
	return p.ac.Write(s.cccdh, v)
}

// ClearSubscriptions ...
func (p *Client) ClearSubscriptions() error {
	p.Lock()
	defer p.Unlock()
	zero := make([]byte, 2)
	for vh, s := range p.subs {
		if err := p.ac.Write(s.cccdh, zero); err != nil {
			return err
		}
		delete(p.subs, vh)
	}
	return nil
}

// CancelConnection disconnects the connection.
func (p *Client) CancelConnection() error {
	p.Lock()
	defer p.Unlock()
	return p.conn.Close()
}

// HandleNotification ...
func (p *Client) HandleNotification(req []byte) {
	p.Lock()
	defer p.Unlock()
	vh := att.HandleValueIndication(req).AttributeHandle()
	sub, ok := p.subs[vh]
	if !ok {
		// FIXME: disconnects and propagate an error to the user.
		log.Printf("Got an unregistered notification")
		return
	}
	fn := sub.nHandler
	if req[0] == att.HandleValueIndicationCode {
		fn = sub.iHandler
	}
	if fn != nil {
		fn(req[3:])
	}
}

type sub struct {
	cccdh    uint16
	ccc      uint16
	nHandler bt.NotificationHandler
	iHandler bt.NotificationHandler
}
