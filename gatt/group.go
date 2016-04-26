package gatt

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/uuid"
)

// NewService returns a GATT service.
func NewService(u uuid.UUID) bt.Service { return &svc{uuid: u} }

type svc struct {
	uuid  uuid.UUID
	chars []bt.Characteristic

	attr attr
}

func (s *svc) UUID() uuid.UUID                      { return s.uuid }
func (s *svc) Characteristics() []bt.Characteristic { return s.chars }

func (s *svc) AddCharacteristic(u uuid.UUID) bt.Characteristic {
	for _, c := range s.chars {
		if c.UUID().Equal(u) {
			panic("Service already contains a characteristic with uuid " + u.String())
		}
	}
	c := &char{uuid: u}
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
	ih bt.IndicateHandler
	nn *notifier
	in *notifier
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

func (c *char) HandleNotify(h bt.NotifyHandler) bt.Characteristic {
	config(c, bt.CharNotify, h, nil)
	return c
}

func (c *char) HandleIndicate(h bt.IndicateHandler) bt.Characteristic {
	config(c, bt.CharIndicate, nil, h)
	return c
}

func (c *char) AddDescriptor(u uuid.UUID) bt.Descriptor {
	for _, d := range c.descs {
		if d.UUID().Equal(u) {
			panic("Service already contains a characteristic with uuid " + u.String())
		}
	}
	d := &desc{uuid: u}
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
