package gatt

import (
	"encoding/binary"

	"github.com/currantlabs/bt"

	"golang.org/x/net/context"
)

type request struct {
	data   []byte
	offset int
}

func (r *request) Data() []byte { return r.data }
func (r *request) Offset() int  { return r.offset }

type notifier struct {
	ctx    context.Context
	maxlen int
	cancel func()
	send   func([]byte) (int, error)
}

func (n *notifier) Context() context.Context    { return n.ctx }
func (n *notifier) Write(b []byte) (int, error) { return n.send(b) }
func (n *notifier) Cap() int                    { return n.maxlen }

func config(c *char, p bt.Property, nh bt.NotifyHandler, ih bt.IndicateHandler) {
	if c.cccd == nil {
		cd := &desc{uuid: attrClientCharacteristicConfigUUID}
		c.cccd = cd
		var ccc uint16
		cd.HandleRead(bt.ReadHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
			binary.Write(rsp, binary.LittleEndian, ccc)
		}))
		cd.HandleWrite(bt.WriteHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
			value := binary.LittleEndian.Uint16(req.Data())
			if value&flagCCCNotify != 0 && c.nn == nil {
				n := &notifier{}
				n.ctx, n.cancel = context.WithCancel(context.Background())
				n.send = func(b []byte) (int, error) { return rsp.Server().Notify(c.attr.vh, b) }
				c.nn = n
				go c.nh.ServeNotify(req, n)
			}
			if value&flagCCCNotify == 0 && c.nn != nil {
				c.nn.cancel()
				c.nn = nil
			}
			if value&flagCCCIndicate != 0 && c.in == nil {
				n := &notifier{}
				n.ctx, n.cancel = context.WithCancel(context.Background())
				n.send = func(b []byte) (int, error) { return rsp.Server().Indicate(c.attr.vh, b) }
				c.in = n
				go c.ih.ServeIndicate(req, n)
			}
			if value&flagCCCIndicate == 0 && c.in != nil {
				c.in.cancel()
				c.in = nil
			}
			ccc = value
		}))
		c.descs = append(c.descs, c.cccd)
	}
	switch {
	case p == bt.CharNotify && nh != nil:
		c.props |= p
		c.nh = nh
	case p == bt.CharIndicate && ih != nil:
		c.props |= p
		c.ih = ih
	case p == bt.CharNotify && nh == nil:
		c.nh = nh
		c.props ^= p
	case p == bt.CharIndicate && ih == nil:
		c.ih = ih
		c.props ^= p
	}
}
