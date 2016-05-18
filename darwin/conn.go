package darwin

import (
	"context"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/darwin/xpc"
)

func newConn(d *Device, a bt.Addr) *conn {
	return &conn{
		dev:   d,
		rxMTU: 23,
		txMTU: 23,
		addr:  a,

		notifiers: make(map[uint16]bt.Notifier),
		subs:      make(map[uint16]bt.NotificationHandler),

		rspc: make(chan msg),
	}
}

type conn struct {
	sync.RWMutex

	dev   *Device
	role  int
	ctx   context.Context
	rxMTU int
	txMTU int
	addr  bt.Addr

	rspc chan msg

	connInterval       int
	connLatency        int
	supervisionTimeout int

	notifiers map[uint16]bt.Notifier // central connection only

	subs map[uint16]bt.NotificationHandler
}

func (c *conn) Context() context.Context {
	return c.ctx
}

func (c *conn) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *conn) LocalAddr() bt.Addr {
	return c.dev.Addr()
}

func (c *conn) RemoteAddr() bt.Addr {
	return c.addr
}

func (c *conn) RxMTU() int {
	return c.rxMTU
}

func (c *conn) SetRxMTU(mtu int) {
	c.rxMTU = mtu
}

func (c *conn) TxMTU() int {
	return c.txMTU
}

func (c *conn) SetTxMTU(mtu int) {
	c.txMTU = mtu
}

func (c *conn) Read(b []byte) (int, error) {
	return 0, nil
}

func (c *conn) Write(b []byte) (int, error) {
	return 0, nil
}

func (c *conn) Close() error {
	return nil
}

// server (peripheral)
func (c *conn) subscribed(char *bt.Characteristic) {
	h := char.Handle
	if _, found := c.notifiers[h]; found {
		return
	}
	send := func(b []byte) (int, error) {
		c.dev.sendCmd(15, xpc.Dict{
			"kCBMsgArgUUIDs":       [][]byte{},
			"kCBMsgArgAttributeID": h,
			"kCBMsgArgData":        b,
		})
		return len(b), nil
	}
	n := bt.NewNotifier(send)
	c.notifiers[h] = n
	go char.NotifyHandler.ServeNotify(&request{}, n)
}

// server (peripheral)
func (c *conn) unsubscribed(char *bt.Characteristic) {
	if n, found := c.notifiers[char.Handle]; found {
		n.Close()
		delete(c.notifiers, char.Handle)
	}
}

func (c *conn) sendReq(id int, args xpc.Dict) msg {
	c.dev.sendCBMsg(id, args)
	m := <-c.rspc
	return msg(m.args())
}

func (c *conn) sendCmd(id int, args xpc.Dict) {
	c.dev.sendCBMsg(id, args)
}
