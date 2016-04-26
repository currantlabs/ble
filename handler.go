package bt

import "golang.org/x/net/context"

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
	Status() AttError

	// SetStatus reports the result of the request.
	SetStatus(status AttError)

	// // Server ...
	Server() AttServer

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
