package l2cap

import (
	"io"
	"net"

	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/hci/evt"
)

// TODO: hadle error / timeout
// Rework API to allow user to convey params, or not?

// A Listener is a le for L2CAP protocol.
type Listener interface {
	// Accept waits for and returns the next connection to the le.
	Accept() (Conn, error)

	// Close closes the le.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the le's network address.
	Addr() net.HardwareAddr
}

// Dialer ...
type Dialer interface {
	Dial(net.HardwareAddr) (Conn, error)
}

// Conn implements a L2CAP connection.
// Currently, it only supports LE-U logical transport, and not ACL-U.
type Conn interface {
	io.ReadWriteCloser

	// LocalAddr returns local device's MAC address.
	LocalAddr() net.HardwareAddr

	// RemoteAddr returns remote device's MAC address.
	RemoteAddr() net.HardwareAddr

	// RxMTU returns the MTU which the upper layer is capable of accepting.
	RxMTU() int

	// SetRxMTU sets the MTU which the upper layer of remote device is capable of accepting.
	SetRxMTU(mtu int)

	// TxMTU returns the MTU which the upper layer of remote device is capable of accepting.
	TxMTU() int

	// SetTxMTU sets the MTU which the upper layer is capable of accepting.
	SetTxMTU(mtu int)

	// Parameters ...
	Parameters() evt.LEConnectionComplete
}

// Dial ...
func Dial(d dev.Device) (Dialer, error) {
	mu.Lock()
	dl, ok := l2devs[d]
	if !ok {
		dl = newLE(d)
		l2devs[d] = dl
	}
	defer mu.Unlock()
	return dl, nil
}

// Listen ...
func Listen(d dev.Device) (Listener, error) {
	mu.Lock()
	dl, ok := l2devs[d]
	if !ok {
		dl = newLE(d)
		l2devs[d] = dl
	}
	defer mu.Unlock()
	return dl, nil
}
