package bt

import "io"

// Addr represents a network end point address.
type Addr interface {
	String() string
}

// A Listener is a le for L2CAP protocol.
type Listener interface {
	// Accept waits for and returns the next connection to the le.
	Accept() (Conn, error)

	// Close closes the le.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the le's network address.
	Addr() Addr
}

// Dialer ...
type Dialer interface {
	Dial(Addr) (Conn, error)
}

// Conn implements a L2CAP connection.
// Currently, it only supports LE-U logical transport, and not ACL-U.
type Conn interface {
	io.ReadWriteCloser

	// LocalAddr returns local device's MAC address.
	LocalAddr() Addr

	// RemoteAddr returns remote device's MAC address.
	RemoteAddr() Addr

	// RxMTU returns the MTU which the upper layer is capable of accepting.
	RxMTU() int

	// SetRxMTU sets the MTU which the upper layer of remote device is capable of accepting.
	SetRxMTU(mtu int)

	// TxMTU returns the MTU which the upper layer of remote device is capable of accepting.
	TxMTU() int

	// SetTxMTU sets the MTU which the upper layer is capable of accepting.
	SetTxMTU(mtu int)

	// // Parameters ...
	// Parameters() evt.LEConnectionComplete
}
