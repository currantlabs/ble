package bt

import (
	"io"

	"golang.org/x/net/context"
)

// A Broadcaster is a device that sends advertising events.
type Broadcaster interface {
	// SetAdvertisement ...
	SetAdvertisement(ad []byte, sr []byte) error

	// Advertise ...
	Advertise() error

	// StopAdvertising ...
	StopAdvertising() error
}

// A Peripheral is a device that accepts the establishment of an LE physical link.
type Peripheral interface {
	Broadcaster
	Listener
}

// Observer a device that receives advertising events.
type Observer interface {
	// SetAdvHandler ...
	SetAdvHandler(af AdvFilter, ah AdvHandler) error

	// Scan starts scanning.
	Scan() error

	// StopScanning stops scanning.
	StopScanning() error
}

// A Central is a device that initiates the establishment of a physical connection.
type Central interface {
	Observer
	Dialer
}

// Addr represents a network end point address.
type Addr interface {
	String() string
}

// A Listener is a le for L2CAP protocol.
type Listener interface {
	// Accept starts advertising and accepts connection.
	Accept() (Conn, error)

	// Close closes the listner.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the listener's network address.
	Addr() Addr
}

// A Dialer contains options for connecting to an address.
type Dialer interface {
	Dial(Addr) (Conn, error)
}

// Conn implements a L2CAP connection.
type Conn interface {
	io.ReadWriteCloser

	// Context returns the context that is used by this Conn.
	Context() context.Context

	// SetContext sets the context that is used by this Conn.
	SetContext(ctx context.Context)

	// LocalAddr returns local device's MAC address.
	LocalAddr() Addr

	// RemoteAddr returns remote device's MAC address.
	RemoteAddr() Addr

	// RxMTU returns the ATT_MTU which the local device is capable of accepting.
	RxMTU() int

	// SetRxMTU sets the ATT_MTU which the local device is capable of accepting.
	SetRxMTU(mtu int)

	// TxMTU returns the ATT_MTU which the remote device is capable of accepting.
	TxMTU() int

	// SetTxMTU sets the ATT_MTU which the remote device is capable of accepting.
	SetTxMTU(mtu int)
}
