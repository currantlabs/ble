package gatt

import (
	"golang.org/x/net/context"

	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/l2cap"
)

// Central is the interface that represent a remote central device.
type Central struct {
	l2c    l2cap.Conn
	server *att.Server
}

func newCentral(a *att.Range, l2c l2cap.Conn) *Central {
	c := &Central{
		l2c: l2c,
	}
	ctx := context.WithValue(context.Background(), keyCentral, c)
	c.server = att.NewServer(ctx, a, l2c, 1024)
	return c
}

// ID returns platform specific ID of the remote central device.
func (c *Central) ID() string {
	return c.l2c.RemoteAddr().String()
}

// Close disconnects the connection.
func (c *Central) Close() error {
	return c.l2c.Close()
}

// MTU returns the current connection mtu.
func (c *Central) MTU() int {
	return c.l2c.TxMTU()
}
