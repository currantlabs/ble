package gatt

import (
	"encoding/binary"

	"golang.org/x/net/context"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

const (
	keyData = iota
	keyOffset
	keyServer
	keyNotifier
)

// Data ...
func Data(ctx context.Context) []byte { return ctx.Value(keyData).([]byte) }

// WithData ...
func WithData(ctx context.Context, d []byte) context.Context {
	return context.WithValue(ctx, keyData, d)
}

// Offset ...
func Offset(ctx context.Context) int { return ctx.Value(keyOffset).(int) }

// WithOffset ...
func WithOffset(ctx context.Context, o int) context.Context {
	return context.WithValue(ctx, keyOffset, o)
}

// server ...
func server(ctx context.Context) *Server { return ctx.Value(keyServer).(*Server) }

// withServer ...
func withserver(ctx context.Context, s *Server) context.Context {
	return context.WithValue(ctx, keyServer, s)
}

// Notifier ...
func Notifier(ctx context.Context) *notifier { return ctx.Value(keyNotifier).(*notifier) }

// WithNotifier ...
func WithNotifier(ctx context.Context, n *notifier) context.Context {
	return context.WithValue(ctx, keyNotifier, n)
}

// Property ...
type Property int

// Characteristic property flags (spec 3.3.3.1)
const (
	CharBroadcast   Property = 0x01 // may be brocasted
	CharRead        Property = 0x02 // may be read
	CharWriteNR     Property = 0x04 // may be written to, with no reply
	CharWrite       Property = 0x08 // may be written to, with a reply
	CharNotify      Property = 0x10 // supports notifications
	CharIndicate    Property = 0x20 // supports Indications
	CharSignedWrite Property = 0x40 // supports signed write
	CharExtended    Property = 0x80 // supports extended properties
)

// A Service is a BLE service.
type Service struct {
	UUID            uuid.UUID
	Characteristics []*Characteristic

	h    uint16
	endh uint16
}

// NewService ...
func NewService(u uuid.UUID) *Service {
	return &Service{UUID: u}
}

// AddCharacteristic adds a characteristic to a service.
func (s *Service) AddCharacteristic(u uuid.UUID) *Characteristic {
	c := &Characteristic{UUID: u, value: make(map[Property]Handler)}
	s.Characteristics = append(s.Characteristics, c)
	return c
}

// A Characteristic is a BLE characteristic.
type Characteristic struct {
	UUID        uuid.UUID
	Property    Property // enabled properties
	Descriptors []*Descriptor

	cccd *Descriptor

	value attValue

	h    uint16
	vh   uint16
	endh uint16
}

func setupCCCD(c *Characteristic, h Handler) *Descriptor {
	d := c.cccd
	if d == nil {
		d = c.AddDescriptor(attrClientCharacteristicConfigUUID)
		c.cccd = d
	}

	var ccc uint16
	n := &notifier{indicate: false}
	i := &notifier{indicate: true}

	d.Handle(
		CharRead,
		HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
			binary.Write(resp, binary.LittleEndian, ccc)
		}))

	d.Handle(
		CharWrite|CharWriteNR,
		HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
			data := Data(ctx)
			if len(data) != 2 {
				resp.SetStatus(att.ErrInvalAttrValueLen)
				return
			}
			ccc := binary.LittleEndian.Uint16(data)
			// Ignore bits that are not defined in spec.
			ccc &= flagCCCNotify | flagCCCIndicate
			n.config(c, ccc&flagCCCNotify != 0, ctx, h, resp)
			i.config(c, ccc&flagCCCIndicate != 0, ctx, h, resp)
		}))
	return d
}

// Handle ...
func (c *Characteristic) Handle(p Property, h Handler) *Characteristic {
	c.value[p&CharRead] = h
	c.value[p&CharWriteNR] = h
	c.value[p&CharWrite] = h
	c.value[p&CharSignedWrite] = h
	c.value[p&CharExtended] = h

	if p&(CharNotify|CharIndicate) != 0 {
		setupCCCD(c, h)
	}

	c.Property |= p
	return c
}

// SetValue ...
func (c *Characteristic) SetValue(value []byte) {
	c.value.setvalue(value)
}

// AddDescriptor adds a descriptor to a characteristic.
func (c *Characteristic) AddDescriptor(u uuid.UUID) *Descriptor {
	d := &Descriptor{UUID: u, value: make(map[Property]Handler)}
	c.Descriptors = append(c.Descriptors, d)
	return d
}

// Descriptor is a BLE descriptor
type Descriptor struct {
	UUID     uuid.UUID
	Property Property // enabled properties

	h uint16

	value attValue
}

// Handle ...
func (d *Descriptor) Handle(p Property, h Handler) *Descriptor {
	if p&^(CharRead|CharWrite|CharWriteNR) != 0 {
		panic("Invalid Property")
	}
	d.value[p&CharRead] = h
	d.value[p&CharWrite] = h
	d.value[p&CharWriteNR] = h
	d.Property |= p
	return d
}

// SetValue ...
func (d *Descriptor) SetValue(value []byte) {
	d.value.setvalue(value)
}

type attValue map[Property]Handler

// Handle ...
func (v attValue) Handle(ctx context.Context, req []byte, resp *att.ResponseWriter) att.Error {
	gattResp := &ResponseWriter{resp: resp, status: att.ErrSuccess}
	var h Handler
	switch req[0] {
	case att.ReadByTypeRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
	case att.ReadRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
	case att.ReadBlobRequestCode:
		if h = v[CharRead]; h == nil {
			return att.ErrReadNotPerm
		}
		ctx = WithOffset(ctx, int(att.ReadBlobRequest(req).ValueOffset()))
	case att.WriteRequestCode:
		if h = v[CharWrite]; h == nil {
			return att.ErrWriteNotPerm
		}
		ctx = WithData(ctx, att.WriteRequest(req).AttributeValue())
	case att.WriteCommandCode:
		if h = v[CharWriteNR]; h == nil {
			return att.ErrWriteNotPerm
		}
		ctx = WithData(ctx, att.WriteRequest(req).AttributeValue())
	// case att.PrepareWriteRequestCode:
	// case att.ExecuteWriteRequestCode:
	// case att.SignedWriteCommandCode:
	// case att.ReadByGroupTypeRequestCode:
	// case att.ReadMultipleRequestCode:
	default:
		return att.ErrReqNotSupp
	}

	h.Serve(ctx, gattResp)
	return gattResp.status
}

func (v attValue) setvalue(value []byte) {
	v[CharRead] = HandlerFunc(func(ctx context.Context, resp *ResponseWriter) {
		resp.Write(value)
	})
}

// A Notifier provides a means for a GATT server to send notifications about value changes to a connected device.
type notifier struct {
	indicate bool
	maxlen   int
	enabled  bool
	cancel   func()
	send     func([]byte) (int, error)
}

// Write sends data to the central.
func (n *notifier) Write(b []byte) (int, error) {
	return n.send(b)
}

// Cap returns the maximum number of bytes that may be sent in a single notification.
func (n *notifier) Cap() int { return n.maxlen }

func (n *notifier) config(c *Characteristic, en bool, ctx context.Context, h Handler, resp *ResponseWriter) {
	if en == n.enabled {
		return
	}
	if n.enabled = en; !en {
		n.cancel()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	s := server(ctx)
	n.send = func(b []byte) (int, error) { return s.as.Notify(c.vh, b) }
	if n.indicate {
		n.send = func(b []byte) (int, error) { return s.as.Indicate(c.vh, b) }
	}
	n.cancel = cancel
	go h.Serve(WithNotifier(ctx, n), resp)
}

// ResponseWriter ...
type ResponseWriter struct {
	resp   *att.ResponseWriter
	status att.Error
}

// Write writes data to return as the characteristic value.
func (r *ResponseWriter) Write(b []byte) (int, error) { return r.resp.Write(b) }

// SetStatus reports the result of the request.
func (r *ResponseWriter) SetStatus(status att.Error) { r.status = status }

// A Handler handles GATT requests.
type Handler interface {
	Serve(ctx context.Context, resp *ResponseWriter)
}

// HandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type HandlerFunc func(ctx context.Context, resp *ResponseWriter)

// Serve returns f(r, maxlen, offset).
func (f HandlerFunc) Serve(ctx context.Context, resp *ResponseWriter) {
	f(ctx, resp)
}
