package bt

import (
	"io"
	"net"
)

// HCI ...
type HCI interface {
	CommandSender
	EventHub
	ACLHandler

	// LocalAddr returns the MAC address of local skt.
	LocalAddr() net.HardwareAddr

	// Stop closes the HCI socket.
	Stop() error
}

// Command ...
type Command interface {
	OpCode() int
	Len() int
	Marshal([]byte) error
}

// CommandRP ...
type CommandRP interface {
	Unmarshal(b []byte) error
}

// CommandSender ...
type CommandSender interface {
	// Send sends a HCI Command and returns unserialized return parameter.
	Send(Command, CommandRP) error
}

// A Handler handles an HCI event packets.
type Handler interface {
	Handle([]byte) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(b []byte) error

// Handle handles an event packet.
func (f HandlerFunc) Handle(b []byte) error {
	return f(b)
}

// EventHub ...
type EventHub interface {
	// SetEventHandler registers the handler to handle the HCI event, and returns current handler.
	SetEventHandler(c int, h Handler) Handler

	// SetSubeventHandler registers the handler to handle the HCI subevent, and returns current handler.
	SetSubeventHandler(c int, h Handler) Handler
}

// ACLHandler ...
type ACLHandler interface {
	SetACLHandler(Handler) (w io.Writer, size int, cnt int)
}
