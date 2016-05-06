package gatt

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/uuid"
)

// NewService returns a GATT service.
func NewService(u uuid.UUID) bt.Service { return &svc{uuid: u} }

// NewCharacteristic returns a GATT characteristic.
func NewCharacteristic(u uuid.UUID) bt.Characteristic { return &char{uuid: u} }

type svc struct {
	uuid  uuid.UUID
	chars []bt.Characteristic

	attr attr
}

func (s *svc) UUID() uuid.UUID                      { return s.uuid }
func (s *svc) Characteristics() []bt.Characteristic { return s.chars }

func (s *svc) NewCharacteristic(u uuid.UUID) bt.Characteristic {
	return s.AddCharacteristic(&char{uuid: u})
}

func (s *svc) AddCharacteristic(c bt.Characteristic) bt.Characteristic {
	u := c.UUID()
	for _, c := range s.chars {
		if c.UUID().Equal(u) {
			panic("Service already contains a characteristic with uuid " + u.String())
		}
	}
	s.chars = append(s.chars, c)
	return c
}

type char struct {
	uuid  uuid.UUID
	props bt.Property
	descs []bt.Descriptor
	cccd  *desc

	attr  attr
	vattr attr

	nh bt.NotifyHandler
	ih bt.NotifyHandler
}

func (c *char) UUID() uuid.UUID              { return c.uuid }
func (c *char) Properties() bt.Property      { return c.props }
func (c *char) Descriptors() []bt.Descriptor { return c.descs }

func (c *char) SetValue(b []byte) bt.Characteristic {
	c.props |= bt.CharRead
	c.vattr.SetValue(b)
	return c
}

func (c *char) HandleRead(h bt.ReadHandler) bt.Characteristic {
	c.props |= bt.CharRead
	c.vattr.HandleRead(h)
	return c
}

func (c *char) HandleWrite(h bt.WriteHandler) bt.Characteristic {
	c.props |= bt.CharWrite | bt.CharWriteNR
	c.vattr.HandleWrite(h)
	return c
}

func (c *char) HandleNotify(ind bool, h bt.NotifyHandler) bt.Characteristic {
	if c.cccd == nil {
		c.cccd = newCCCD(c)
		c.descs = append(c.descs, c.cccd)
	}
	config(c, ind, h)
	return c
}

func (c *char) NewDescriptor(u uuid.UUID) bt.Descriptor {
	return c.AddDescriptor(&desc{uuid: u})
}

func (c *char) AddDescriptor(d bt.Descriptor) bt.Descriptor {
	u := d.UUID()
	for _, d := range c.descs {
		if d.UUID().Equal(u) {
			panic("Characteristic already contains a descriptor with uuid " + u.String())
		}
	}
	c.descs = append(c.descs, d)
	return d
}

type desc struct {
	uuid  uuid.UUID
	props bt.Property

	attr attr
}

func (d *desc) UUID() uuid.UUID { return d.uuid }

func (d *desc) SetValue(b []byte) bt.Descriptor {
	d.props |= bt.CharRead
	d.attr.SetValue(b)
	return d
}

func (d *desc) HandleRead(h bt.ReadHandler) bt.Descriptor {
	d.props |= bt.CharRead
	d.attr.HandleRead(h)
	return d
}

func (d *desc) HandleWrite(h bt.WriteHandler) bt.Descriptor {
	d.props |= bt.CharWrite | bt.CharWriteNR
	d.attr.HandleWrite(h)
	return d
}
