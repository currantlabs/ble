package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

// NewService ...
func NewService(u uuid.UUID) *Service {
	return &Service{uuid: u}
}

// A Service is a BLE service.
type Service struct {
	uuid  uuid.UUID
	chars []*Characteristic

	h    uint16
	endh uint16
}

// UUID returns the UUID of the service.
func (s *Service) UUID() uuid.UUID { return s.uuid }

// Characteristics returns the contained characteristic of this service.
func (s *Service) Characteristics() []*Characteristic { return s.chars }

// AddCharacteristic adds a characteristic to a service.
// AddCharacteristic panics if the service already contains another
// characteristic with the same UUID.
func (s *Service) AddCharacteristic(u uuid.UUID) *Characteristic {
	for _, c := range s.chars {
		if c.UUID().Equal(u) {
			panic("Service already contains a characteristic with uuid " + u.String())
		}
	}
	c := &Characteristic{uuid: u}
	s.chars = append(s.chars, c)
	return c
}

// A Characteristic is a BLE characteristic.
type Characteristic struct {
	uuid  uuid.UUID
	props Property
	descs []*Descriptor
	cccd  *Descriptor

	h    uint16
	vh   uint16
	endh uint16

	value attValue

	nh NotifyHandler
	ih IndicateHandler
	nn *notifier
	in *notifier
}

// UUID returns the UUID of the characteristic.
func (c *Characteristic) UUID() uuid.UUID { return c.uuid }

// Properties returns the properties of this characteristic.
func (c *Characteristic) Properties() Property { return c.props }

// Descriptors returns the contained descriptors of this characteristic.
func (c *Characteristic) Descriptors() []*Descriptor { return c.descs }

// SetValue panics if the characteristic has been configured with a ReadHandler.
// SetValue makes the characteristic support read requests, and returns a static value.
// SetValue must be called before the containing service is added to a server.
func (c *Characteristic) SetValue(b []byte) *Characteristic {
	c.props |= CharRead
	c.value.setValue(b)
	return c
}

// HandleRead makes the characteristic support read requests, and routes read requests to h.
// HandleRead must be called before the containing service is added to a server.
// HandleRead panics if the characteristic has been configured with a static value.
func (c *Characteristic) HandleRead(h ReadHandler) *Characteristic {
	c.props |= CharRead
	c.value.handleRead(h)
	return c
}

// HandleWrite makes the characteristic support write and write-no-response requests, and routes write requests to h.
// The WriteHandler does not differentiate between write and write-no-response requests; it is handled automatically.
// HandleWrite must be called before the containing service is added to a server.
func (c *Characteristic) HandleWrite(h WriteHandler) *Characteristic {
	c.props |= CharWrite | CharWriteNR
	c.value.handleWrite(h)
	return c
}

// HandleNotify makes the characteristic support notify requests, and routes notification requests to h.
// HandleNotify must be called before the containing service is added to a server.
func (c *Characteristic) HandleNotify(h NotifyHandler) *Characteristic {
	config(c, CharNotify, h, nil)
	return c
}

// HandleIndicate makes the characteristic support notify requests, and routes notification requests to h.
// HandleIndicate must be called before the containing service is added to a server.
func (c *Characteristic) HandleIndicate(h IndicateHandler) *Characteristic {
	config(c, CharIndicate, nil, h)
	return c
}

// AddDescriptor adds a descriptor to a characteristic.
// AddDescriptor panics if the characteristic already contains another descriptor with the same UUID.
func (c *Characteristic) AddDescriptor(u uuid.UUID) *Descriptor {
	for _, d := range c.descs {
		if d.UUID().Equal(u) {
			panic("Service already contains a characteristic with uuid " + u.String())
		}
	}
	d := &Descriptor{uuid: u}
	c.descs = append(c.descs, d)
	return d
}

// Descriptor is a BLE descriptor
type Descriptor struct {
	uuid  uuid.UUID
	props Property

	h     uint16
	value attValue
}

// UUID returns the UUID of the descriptor.
func (d *Descriptor) UUID() uuid.UUID { return d.uuid }

// SetValue makes the descriptor support read requests, and returns a static value.
// SetValue must be called before the containing service is added to a server.
// SetValue panics if the descriptor has already configured with a ReadHandler.
func (d *Descriptor) SetValue(b []byte) *Descriptor {
	d.props |= CharRead
	d.value.setValue(b)
	return d
}

// HandleRead makes the descriptor support read requests, and routes read requests to h.
// HandleRead must be called before the containing service is added to a server.
// HandleRead panics if the descriptor has been configured with a static value.
func (d *Descriptor) HandleRead(h ReadHandler) *Descriptor {
	d.props |= CharRead
	d.value.handleRead(h)
	return d
}

// HandleWrite makes the descriptor support write and write-no-response requests, and routes write requests to h.
// The WriteHandler does not differentiate between write and write-no-response requests; it is handled automatically.
// HandleWrite must be called before the containing service is added to a server.
func (d *Descriptor) HandleWrite(h WriteHandler) *Descriptor {
	d.props |= CharWrite | CharWriteNR
	d.value.handleWrite(h)
	return d
}

type attValue struct {
	rh ReadHandler
	wh WriteHandler
	v  []byte
}

func (v *attValue) setValue(b []byte) {
	if v.v != nil && v.rh != nil {
		panic("static value and read handler can't be configured at the same time")
	}
	v.v = make([]byte, len(b))
	copy(v.v, b)
}

func (v *attValue) handleRead(h ReadHandler) {
	if v.v != nil && v.rh != nil {
		panic("static value and read handler can't be configured at the same time")
	}
	v.rh = h
}

func (v *attValue) handleWrite(h WriteHandler) { v.wh = h }

func (v attValue) Value() []byte { return v.v }

func (v attValue) HandleATT(req []byte, rsp *att.ResponseWriter) att.Error {
	r := &request{}
	rsp.SetStatus(att.ErrSuccess)
	switch req[0] {
	case att.ReadByTypeRequestCode:
		if v.rh == nil {
			return att.ErrReadNotPerm
		}
		v.rh.ServeRead(r, rsp)
	case att.ReadRequestCode:
		if v.rh == nil {
			return att.ErrReadNotPerm
		}
		v.rh.ServeRead(r, rsp)
	case att.ReadBlobRequestCode:
		if v.rh == nil {
			return att.ErrReadNotPerm
		}
		r.offset = int(att.ReadBlobRequest(req).ValueOffset())
		v.rh.ServeRead(r, rsp)
	case att.WriteRequestCode:
		if v.wh == nil {
			return att.ErrWriteNotPerm
		}
		r.data = att.WriteRequest(req).AttributeValue()
		v.wh.ServeWrite(r, rsp)
	case att.WriteCommandCode:
		if v.wh == nil {
			return att.ErrWriteNotPerm
		}
		r.data = att.WriteRequest(req).AttributeValue()
		v.wh.ServeWrite(r, rsp)
	// case att.PrepareWriteRequestCode:
	// case att.ExecuteWriteRequestCode:
	// case att.SignedWriteCommandCode:
	// case att.ReadByGroupTypeRequestCode:
	// case att.ReadMultipleRequestCode:
	default:
		return att.ErrReqNotSupp
	}

	return rsp.Status()
}
