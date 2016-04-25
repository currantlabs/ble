package gatt

import (
	"encoding/binary"

	"github.com/currantlabs/bt/att"

	"golang.org/x/net/context"
)

// A ReadHandler handles GATT requests.
type ReadHandler interface {
	ServeRead(req Request, rsp ResponseWriter)
}

// ReadHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type ReadHandlerFunc func(req Request, rsp ResponseWriter)

// ServeRead returns f(r, maxlen, offset).
func (f ReadHandlerFunc) ServeRead(req Request, rsp ResponseWriter) {
	f(req, rsp)
}

// A WriteHandler handles GATT requests.
type WriteHandler interface {
	ServeWrite(req Request, rsp ResponseWriter)
}

// WriteHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type WriteHandlerFunc func(req Request, rsp ResponseWriter)

// ServeWrite returns f(r, maxlen, offset).
func (f WriteHandlerFunc) ServeWrite(req Request, rsp ResponseWriter) {
	f(req, rsp)
}

// A NotifyHandler handles GATT requests.
type NotifyHandler interface {
	ServeNotify(req Request, n Notifier)
}

// NotifyHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type NotifyHandlerFunc func(req Request, n Notifier)

// ServeNotify returns f(r, maxlen, offset).
func (f NotifyHandlerFunc) ServeNotify(req Request, n Notifier) {
	f(req, n)
}

// A IndicateHandler handles GATT requests.
type IndicateHandler interface {
	ServeIndicate(req Request, n Notifier)
}

// IndicateHandlerFunc is an adapter to allow the use of ordinary functions as Handlers.
type IndicateHandlerFunc func(req Request, n Notifier)

// ServeIndicate returns f(r, maxlen, offset).
func (f IndicateHandlerFunc) ServeIndicate(req Request, n Notifier) {
	f(req, n)
}

// Request ...
type Request interface {
	Data() []byte
	Offset() int
}

// ResponseWriter ...
type ResponseWriter interface {
	// Write writes data to return as the characteristic value.
	Write(b []byte) (int, error)

	// Status reports the result of the request.
	Status() att.Error

	// SetStatus reports the result of the request.
	SetStatus(status att.Error)

	// Server ...
	Server() *att.Server

	// Len ...
	Len() int

	// Cap ...
	Cap() int
}

// Notifier ...
type Notifier interface {
	// Context sends data to the central.
	Context() context.Context

	// Write sends data to the central.
	Write(b []byte) (int, error)

	// Cap returns the maximum number of bytes that may be sent in a single notification.
	Cap() int
}

type request struct {
	data   []byte
	offset int
}

func (r *request) Data() []byte { return r.data }
func (r *request) Offset() int  { return r.offset }

type notifier struct {
	ctx    context.Context
	maxlen int
	cancel func()
	send   func([]byte) (int, error)
}

func (n *notifier) Context() context.Context    { return n.ctx }
func (n *notifier) Write(b []byte) (int, error) { return n.send(b) }
func (n *notifier) Cap() int                    { return n.maxlen }

func config(c *Characteristic, p Property, nh NotifyHandler, ih IndicateHandler) {
	if c.cccd == nil {
		cd := &Descriptor{uuid: attrClientCharacteristicConfigUUID}
		c.cccd = cd
		var ccc uint16
		cd.HandleRead(ReadHandlerFunc(func(req Request, rsp ResponseWriter) {
			binary.Write(rsp, binary.LittleEndian, ccc)
		}))
		cd.HandleWrite(WriteHandlerFunc(func(req Request, rsp ResponseWriter) {
			value := binary.LittleEndian.Uint16(req.Data())
			if value&flagCCCNotify != 0 && c.nn == nil {
				n := &notifier{}
				n.ctx, n.cancel = context.WithCancel(context.Background())
				n.send = func(b []byte) (int, error) { return rsp.Server().Notify(c.attr.vh, b) }
				c.nn = n
				go c.nh.ServeNotify(req, n)
			}
			if value&flagCCCNotify == 0 && c.nn != nil {
				c.nn.cancel()
				c.nn = nil
			}
			if value&flagCCCIndicate != 0 && c.in == nil {
				n := &notifier{}
				n.ctx, n.cancel = context.WithCancel(context.Background())
				n.send = func(b []byte) (int, error) { return rsp.Server().Indicate(c.attr.vh, b) }
				c.in = n
				go c.ih.ServeIndicate(req, n)
			}
			if value&flagCCCIndicate == 0 && c.in != nil {
				c.in.cancel()
				c.in = nil
			}
			ccc = value
		}))
		c.descs = append(c.descs, c.cccd)
	}
	switch {
	case p == CharNotify && nh != nil:
		c.props |= p
		c.nh = nh
	case p == CharIndicate && ih != nil:
		c.props |= p
		c.ih = ih
	case p == CharNotify && nh == nil:
		c.nh = nh
		c.props ^= p
	case p == CharIndicate && ih == nil:
		c.ih = ih
		c.props ^= p
	}
}
