package ble

import (
	"io"

	"golang.org/x/net/context"
)

// Addr represents a network end point address.
type Addr interface {
	String() string
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

// Device ...
type Device interface {
	// AddService adds a service to database.
	AddService(svc *Service) error

	// RemoveAllServices removes all services that are currently in the database.
	RemoveAllServices() error

	// SetServices set the specified service to the database.
	// It removes all currently added services, if any.
	SetServices(svcs []*Service) error

	// Stop detatch the GATT server from a peripheral device.
	Stop() error

	// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
	// It tres to fit the UUIDs in the advertising packet as much as possi
	// If name doesn't fit in the advertising packet, it will be put in scan response.
	AdvertiseNameAndServices(ctx context.Context, name string, uuids ...UUID) error

	// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
	AdvertiseIBeaconData(ctx context.Context, b []byte) error

	// AdvertiseIBeacon advertises iBeacon with specified parameters.
	AdvertiseIBeacon(ctx context.Context, u UUID, major, minor uint16, pwr int8) error

	// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
	Scan(ctx context.Context, allowDup bool, h AdvHandler) error

	// Dial ...
	Dial(ctx context.Context, a Addr) (Client, error)
}
