package gatt

// Central is the interface that represent a remote central device.
type Central interface {
	ID() string   // ID returns platform specific ID of the remote central device.
	Close() error // Close disconnects the connection.
	MTU() int     // MTU returns the current connection mtu.
}

// ResponseWriter ...
type ResponseWriter interface {
	// Write writes data to return as the characteristic value.
	Write([]byte) (int, error)

	// SetStatus reports the result of the read operation. See the Status* constants.
	SetStatus(byte)
}

// A ReadHandler handles GATT read requests.
type ReadHandler interface {
	ServeRead(resp ResponseWriter, req *Request)
}

// ReadHandlerFunc is an adapter to allow the use of
// ordinary functions as ReadHandlers. If f is a function
// with the appropriate signature, ReadHandlerFunc(f) is a
// ReadHandler that calls f.
type ReadHandlerFunc func(resp ResponseWriter, req *Request)

// ServeRead returns f(r, maxlen, offset).
func (f ReadHandlerFunc) ServeRead(resp ResponseWriter, req *Request) {
	f(resp, req)
}

// A WriteHandler handles GATT write requests.
// Write and WriteNR requests are presented identically;
// the server will ensure that a response is sent if appropriate.
type WriteHandler interface {
	ServeWrite(r Request, data []byte) (status byte)
}

// WriteHandlerFunc is an adapter to allow the use of
// ordinary functions as WriteHandlers. If f is a function
// with the appropriate signature, WriteHandlerFunc(f) is a
// WriteHandler that calls f.
type WriteHandlerFunc func(r Request, data []byte) byte

// ServeWrite returns f(r, data).
func (f WriteHandlerFunc) ServeWrite(r Request, data []byte) byte {
	return f(r, data)
}

// A NotifyHandler handles GATT notification requests.
// Notifications can be sent using the provided notifier.
type NotifyHandler interface {
	ServeNotify(r Request, n Notifier)
}

// NotifyHandlerFunc is an adapter to allow the use of
// ordinary functions as NotifyHandlers. If f is a function
// with the appropriate signature, NotifyHandlerFunc(f) is a
// NotifyHandler that calls f.
type NotifyHandlerFunc func(r Request, n Notifier)

// ServeNotify calls f(r, n).
func (f NotifyHandlerFunc) ServeNotify(r Request, n Notifier) {
	f(r, n)
}

// A Notifier provides a means for a GATT server to send
// notifications about value changes to a connected device.
// Notifiers are provided by NotifyHandlers.
type Notifier interface {
	// Write sends data to the central.
	Write(data []byte) (int, error)

	// Done reports whether the central has requested not to
	// receive any more notifications with this notifier.
	Done() bool

	// Cap returns the maximum number of bytes that may be sent
	// in a single notification.
	Cap() int
}
