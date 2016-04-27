package gatt

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/l2cap"
	"github.com/currantlabs/bt/uuid"
)

// NotificationHandler ...
type NotificationHandler func(req []byte)

// Client ...
type Client struct {
	// NameChanged is called whenever the Client GAP device name has changed.
	// NameChanged func(*Client)

	// ServicedModified is called when one or more service of a Client have changed.
	// A list of invalid service is provided in the parameter.
	// ServicesModified func(*Client, []Service)

	svcs []*Service

	name string
	addr net.HardwareAddr

	handler *nHandlers
	c       *att.Client
}

// Init ...
func (gc *Client) Init(l2c l2cap.Conn) error {
	h := newNHandler()
	gc.c = att.NewClient(l2c, h)
	gc.handler = h
	go gc.c.Loop()
	return nil
}

// Address is the platform specific unique ID of the remote peripheral, e.g. MAC for Linux, Client UUID for MacOS.
func (gc *Client) Address() net.HardwareAddr { return gc.addr }

// Name returns the name of the remote peripheral.
// This can be the advertised name, if exists, or the GAP device name, which takes priority.
func (gc *Client) Name() string { return gc.name }

// Services returnns the services of the remote peripheral which has been discovered.
func (gc *Client) Services() []*Service { return gc.svcs }

// DiscoverServices discovers all the primary service on a server. [Vol 3, Parg G, 4.4.1]
// DiscoverServices discover the specified services of the remote peripheral.
// If the specified services is set to nil, all the available services of the remote peripheral are returned.
func (gc *Client) DiscoverServices(filter []uuid.UUID) ([]*Service, error) {
	start := uint16(0x0001)
	for {
		length, b, err := gc.c.ReadByGroupType(start, 0xFFFF, uuid.UUID(attrPrimaryServiceUUID))
		if err == att.ErrAttrNotFound {
			return gc.svcs, nil
		}
		if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u := uuid.UUID(b[4:length])
			if filter == nil || uuid.Contains(filter, u) {
				gc.svcs = append(gc.svcs, &Service{
					uuid: u,
					attr: attr{h: h, endh: endh},
				})
			}
			if endh == 0xFFFF {
				return gc.svcs, nil
			}
			start = endh + 1
			b = b[length:]
		}
	}
}

// DiscoverIncludedServices discovers the specified included services of a service.
// If the specified services is set to nil, all the included services of the service are returned.
func (gc *Client) DiscoverIncludedServices(ss []uuid.UUID, s *Service) ([]*Service, error) {
	return nil, nil
}

// DiscoverCharacteristics discovers the specified characteristics of a service.
// If the specified characterstics is set to nil, all the characteristic of the service are returned.
func (gc *Client) DiscoverCharacteristics(filter []uuid.UUID, s *Service) ([]*Characteristic, error) {
	start := s.attr.h
	var lastChar *Characteristic
	for start <= s.attr.endh {
		length, b, err := gc.c.ReadByType(start, s.attr.endh, attrCharacteristicUUID)
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
				attr: attr{
					h:    h,
					vh:   vh,
					endh: s.attr.endh,
				},
			}
			if filter == nil || uuid.Contains(filter, u) {
				s.chars = append(s.chars, c)
			}
			if lastChar != nil {
				lastChar.attr.endh = c.attr.h - 1
			}
			lastChar = c
			start = vh + 1
			b = b[length:]
		}
	}
	return s.chars, nil
}

// DiscoverDescriptors discovers the descriptors of a characteristic.
// If the specified descriptors is set to nil, all the descriptors of the characteristic are returned.
func (gc *Client) DiscoverDescriptors(filter []uuid.UUID, c *Characteristic) ([]*Descriptor, error) {
	start := c.attr.vh + 1
	for start <= c.attr.endh {
		fmt, b, err := gc.c.FindInformation(start, c.attr.endh)
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
			d := &Descriptor{
				uuid: u,
				attr: attr{h: h},
			}
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

// ReadCharacteristic retrieves the value of a specified characteristic.
func (gc *Client) ReadCharacteristic(c *Characteristic) ([]byte, error) { return gc.c.Read(c.attr.vh) }

// ReadLongCharacteristic retrieves the value of a specified characteristic that is longer than the MTU.
func (gc *Client) ReadLongCharacteristic(c *Characteristic) ([]byte, error) { return nil, nil }

// WriteCharacteristic writes the value of a characteristic.
func (gc *Client) WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error {
	if noRsp {
		gc.c.WriteCommand(c.attr.vh, value)
		return nil
	}
	return gc.c.Write(c.attr.vh, value)
}

// ReadDescriptor retrieves the value of a specified characteristic descriptor.
func (gc *Client) ReadDescriptor(d *Descriptor) ([]byte, error) { return gc.c.Read(d.attr.h) }

// WriteDescriptor writes the value of a characteristic descriptor.
func (gc *Client) WriteDescriptor(d *Descriptor, v []byte) error { return gc.c.Write(d.attr.h, v) }

// ReadRSSI retrieves the current RSSI value for the remote peripheral.
func (gc *Client) ReadRSSI() int { return -1 }

// SetMTU sets the mtu for the remote peripheral.
func (gc *Client) SetMTU(mtu int) error {
	_, err := gc.c.ExchangeMTU(mtu)
	return err
}

// SetNotificationHandler sets notifications for the value of a specified characteristic.
func (gc *Client) SetNotificationHandler(c *Characteristic, h NotificationHandler) error {
	return gc.setHandlers(c.cccd.attr.h, c.attr.vh, flagCCCNotify, h)
}

// SetIndicationHandler sets indications for the value of a specified characteristic.
func (gc *Client) SetIndicationHandler(c *Characteristic, h NotificationHandler) error {
	return gc.setHandlers(c.cccd.attr.h, c.attr.vh, flagCCCIndicate, h)
}

// ClearHandlers ...
func (gc *Client) ClearHandlers() error {
	for _, s := range gc.handler.stubs {
		if err := gc.c.Write(s.cccdh, make([]byte, 2)); err != nil {
			return err
		}
	}
	return nil
}

func (gc *Client) setHandlers(cccdh, vh, flag uint16, h NotificationHandler) error {
	s, ok := gc.handler.stubs[vh]
	if !ok {
		s = &stub{cccdh, 0x0000, nil, nil}
		gc.handler.stubs[vh] = s
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
	return gc.c.Write(s.cccdh, v)
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
