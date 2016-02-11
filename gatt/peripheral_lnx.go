package gatt

import (
	"encoding/binary"
	"errors"
	"net"

	"github.com/currantlabs/bt"
)

type peripheral struct {
	// NameChanged is called whenever the peripheral GAP device name has changed.
	NameChanged func(*peripheral)

	// ServicedModified is called when one or more service of a peripheral have changed.
	// A list of invalid service is provided in the parameter.
	ServicesModified func(*peripheral, []*Service)

	d    *device
	svcs []*Service

	name      string
	adv       *Advertisement
	advReport *advertisingReport
	addr      net.HardwareAddr

	c *client
}

func (p *peripheral) Device() Device       { return p.d }
func (p *peripheral) ID() string           { return p.addr.String() }
func (p *peripheral) Name() string         { return p.name }
func (p *peripheral) Services() []*Service { return p.svcs }

func newPeripheral(d *device, l2c bt.Conn) *peripheral {
	p := &peripheral{
		d: d,
		c: newClient(l2c),
	}
	return p
}

// DiscoverServices discovers all the primary service on a server. [Vol 3, Parg G, 4.4.1]
func (p *peripheral) DiscoverServices(filter []UUID) ([]*Service, error) {
	start := uint16(0x0001)
	for {
		length, b, err := p.c.ReadByGroupType(start, 0xFFFF, attrPrimaryServiceUUID)
		if err == ErrAttrNotFound {
			return p.svcs, nil
		}
		if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u := UUID(b[4:length])
			if filter == nil || UUIDContains(filter, u) {
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

func (p *peripheral) DiscoverIncludedServices(ss []UUID, s *Service) ([]*Service, error) {
	return nil, nil
}

func (p *peripheral) DiscoverCharacteristics(filter []UUID, s *Service) ([]*Characteristic, error) {
	done := false
	start := s.h
	var lastChar *Characteristic
	for !done {
		length, b, err := p.c.ReadByType(start, s.endh, attrCharacteristicUUID)
		if err == ErrAttrNotFound {
			return s.chars, nil
		} else if err != nil {
			return nil, err
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			props := Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u := UUID(b[5:length])
			c := &Characteristic{uuid: u, svc: s, props: props, h: h, vh: vh}
			if filter == nil || UUIDContains(filter, u) {
				s.chars = append(s.chars, c)
			}
			if lastChar != nil {
				lastChar.endh = c.h - 1
			}
			lastChar = c
			done = vh == s.endh
			start = vh + 1
			b = b[length:]
		}
	}
	if len(s.chars) > 1 {
		s.chars[len(s.chars)-1].endh = s.endh
	}
	return s.chars, nil
}

func (p *peripheral) DiscoverDescriptors(filter []UUID, c *Characteristic) ([]*Descriptor, error) {
	done := false
	start := c.vh + 1
	for !done {
		if c.endh == 0 {
			c.endh = c.svc.endh
		}

		fmt, b, err := p.c.FindInformation(start, c.endh)
		if err == ErrAttrNotFound {
			return c.descs, nil
		} else if err != nil {
			return nil, err
		}
		length := 4
		if fmt == 2 {
			length = 18
		}
		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			u := UUID(b[2:length])
			d := &Descriptor{uuid: u, h: h, char: c}
			if filter == nil || UUIDContains(filter, u) {
				c.descs = append(c.descs, d)
			}
			if u.Equal(attrClientCharacteristicConfigUUID) {
				c.cccd = d
			}
			done = h == c.endh
			start = h + 1
			b = b[length:]
		}
	}
	return c.descs, nil
}

func (p *peripheral) ReadCharacteristic(c *Characteristic) ([]byte, error) {
	return p.c.Read(c.vh)
}

func (p *peripheral) ReadLongCharacteristic(c *Characteristic) ([]byte, error) {
	return nil, nil
}

func (p *peripheral) WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error {
	if noRsp {
		p.c.WriteCommand(c.vh, value)
		return nil
	}
	return p.c.Write(c.vh, value)
}

func (p *peripheral) ReadDescriptor(d *Descriptor) ([]byte, error) {
	return p.c.Read(d.h)
}

func (p *peripheral) WriteDescriptor(d *Descriptor, value []byte) error {
	return p.c.Write(d.h, value)
}

func (p *peripheral) SetNotifyValue(c *Characteristic, f func(*Characteristic, []byte, error)) error {
	if c.cccd == nil {
		return errors.New("no cccd") // FIXME
	}
	fn := func(b []byte, err error) { f(c, b, err) }
	return p.c.SetNotifyValue(c.cccd.h, c.vh, gattCCCNotifyFlag, fn)
}

func (p *peripheral) SetIndicateValue(c *Characteristic, f func(*Characteristic, []byte, error)) error {
	if c.cccd == nil {
		return errors.New("no cccd") // FIXME
	}
	fn := func(b []byte, err error) { f(c, b, err) }
	return p.c.SetNotifyValue(c.cccd.h, c.vh, gattCCCIndicateFlag, fn)
}

func (p *peripheral) ReadRSSI() int {
	return -1
}

func (p *peripheral) SetMTU(mtu int) error {
	_, err := p.c.ExchangeMTU(mtu)
	return err
}
