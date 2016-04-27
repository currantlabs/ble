package gap

import (
	"net"

	"github.com/currantlabs/bt/hci/evt"
)

// AdvFilter ...
type AdvFilter interface {
	AdvFilter(a Advertisement) bool
}

// AdvFilterFunc ...
type AdvFilterFunc func(a Advertisement) bool

// AdvFilter ...
func (f AdvFilterFunc) AdvFilter(a Advertisement) bool {
	return f(a)
}

// AdvHandler ...
type AdvHandler interface {
	Handle(a Advertisement)
}

// AdvHandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a AdvHandler object that calls f.
type AdvHandlerFunc func(a Advertisement)

// Handle handles an advertisement.
func (f AdvHandlerFunc) Handle(a Advertisement) {
	f(a)
}

// Advertisement ...
type Advertisement struct {
	e evt.LEAdvertisingReport
	i int
}

// EventType ...
func (a Advertisement) EventType() uint8 {
	return a.e.EventType(a.i)
}

// AddressType ...
func (a Advertisement) AddressType() uint8 {
	return a.e.AddressType(a.i)
}

// RSSI ...
func (a Advertisement) RSSI() int8 {
	return a.e.RSSI(a.i)
}

// Address ...
func (a Advertisement) Address() net.HardwareAddr {
	b := a.e.Address(a.i)
	return []byte{b[5], b[4], b[3], b[2], b[1], b[0]}
}

// Data ...
func (a Advertisement) Data() []byte {
	return a.e.Data(a.i)
}
