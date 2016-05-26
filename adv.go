package bt

import "net"

// AdvFilter ...
type AdvFilter interface {
	Filter(a Advertisement) bool
}

// AdvFilterFunc ...
type AdvFilterFunc func(a Advertisement) bool

// Filter ...
func (f AdvFilterFunc) Filter(a Advertisement) bool {
	return f(a)
}

// AdvHandler ...
type AdvHandler interface {
	Handle(a Advertisement)
}

// The AdvHandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type AdvHandlerFunc func(a Advertisement)

// Handle handles an advertisement.
func (f AdvHandlerFunc) Handle(a Advertisement) {
	f(a)
}

// Advertisement ...
type Advertisement interface {
	// EventType ...
	EventType() uint8

	// AddressType ...
	AddressType() uint8

	// RSSI ...
	RSSI() int8

	// Address ...
	Address() net.HardwareAddr

	// Data ...
	Data() []byte

	// ScanResponse ...
	ScanResponse() []byte
}
