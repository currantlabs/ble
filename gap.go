package bt

import (
	"context"
	"io"
)

// A Broadcaster is a device that sends advertising events.
type Broadcaster interface {
	// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
	// It tres to fit the UUIDs in the advertising packet as much as possible.
	// If name doesn't fit in the advertising packet, it will be put in scan response.
	AdvertiseNameAndServices(name string, uuids ...UUID) error

	// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
	AdvertiseIBeaconData(b []byte) error

	// AdvertisingIbeacon advertises iBeacon with specified parameters.
	AdvertiseIBeacon(u UUID, major, minor uint16, pwr int8) error

	// StopAdvertising stops advertising.
	StopAdvertising() error
}

// A Peripheral is a device that accepts the establishment of an LE physical link.
type Peripheral interface {
	Broadcaster
	Listener
}

// An Observer is a device that receives advertising events.
type Observer interface {
	// SetHandler sets filter, handler.
	SetAdvHandler(h AdvHandler) error

	// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
	Scan(allowDup bool) error

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

// A Listener is a listener for L2CAP protocol.
type Listener interface {
	// Accept starts advertising and accepts connection.
	Accept() (Conn, error)

	// Close closes the listner.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the listener's device address.
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

	// LocalAddr returns local device's address.
	LocalAddr() Addr

	// RemoteAddr returns remote device's address.
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
