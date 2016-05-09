package gatt

import (
	"encoding/binary"

	"github.com/currantlabs/bt"

	"golang.org/x/net/context"
)

type request struct {
	conn   bt.Conn
	data   []byte
	offset int
}

func (r *request) Conn() bt.Conn { return r.conn }
func (r *request) Data() []byte  { return r.data }
func (r *request) Offset() int   { return r.offset }

type notifier struct {
	ctx    context.Context
	maxlen int
	cancel func()
	send   func([]byte) (int, error)
}

func newNotifier(send func([]byte) (int, error)) *notifier {
	n := &notifier{}
	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.send = send
	// n.maxlen = cap
	return n
}

func (n *notifier) Context() context.Context    { return n.ctx }
func (n *notifier) Write(b []byte) (int, error) { return n.send(b) }
func (n *notifier) Cap() int                    { return n.maxlen }

func config(c *char, ind bool, h bt.NotifyHandler) {
	switch {
	case ind && h != nil:
		c.props |= bt.CharIndicate
		c.ih = h
	case ind && h == nil:
		c.props &= ^bt.CharIndicate
		c.ih = h
	case !ind && h != nil:
		c.props |= bt.CharNotify
		c.nh = h
	case !ind && h == nil:
		c.props &= ^bt.CharNotify
		c.nh = h
	}
}

func newCCCD(c *char) *desc {
	var nn *notifier
	var in *notifier

	d := &desc{uuid: attrClientCharacteristicConfigUUID}

	d.HandleRead(bt.ReadHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
		cccs := req.Conn().Context().Value("ccc").(map[uint16]uint16)
		ccc := cccs[c.attr.Handle()]
		binary.Write(rsp, binary.LittleEndian, ccc)
	}))

	d.HandleWrite(bt.WriteHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
		cccs := req.Conn().Context().Value("ccc").(map[uint16]uint16)
		ccc := cccs[c.attr.Handle()]

		newCCC := binary.LittleEndian.Uint16(req.Data())
		if newCCC&cccNotify != 0 && ccc&cccNotify == 0 {
			send := func(b []byte) (int, error) { return rsp.Notify(false, c.attr.vh, b) }
			nn = newNotifier(send)
			go c.nh.ServeNotify(req, nn)
		}
		if newCCC&cccNotify == 0 && ccc&cccNotify != 0 {
			nn.cancel()
		}
		if newCCC&cccIndicate != 0 && ccc&cccIndicate == 0 {
			send := func(b []byte) (int, error) { return rsp.Notify(true, c.attr.vh, b) }
			in = newNotifier(send)
			go c.ih.ServeNotify(req, in)
		}
		if newCCC&cccIndicate == 0 && ccc&cccIndicate != 0 {
			in.cancel()
		}
		cccs[c.attr.Handle()] = newCCC
	}))
	return d
}
