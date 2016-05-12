package gatt

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/uuid"
)

type attr struct {
	h    uint16
	vh   uint16
	endh uint16
	typ  uuid.UUID

	rh bt.ReadHandler
	wh bt.WriteHandler
	v  []byte
}

func (v *attr) SetValue(b []byte) {
	if v.v != nil && v.rh != nil {
		panic("static attr and read handler can't be configured at the same time")
	}
	v.v = make([]byte, len(b))
	copy(v.v, b)
}

func (v *attr) HandleRead(h bt.ReadHandler) {
	if v.v != nil && v.rh != nil {
		panic("static attr and read handler can't be configured at the same time")
	}
	v.rh = h
}

func (v *attr) HandleWrite(h bt.WriteHandler) { v.wh = h }

func (v *attr) Value() []byte        { return v.v }
func (v *attr) Handle() uint16       { return v.h }
func (v *attr) EndingHandle() uint16 { return v.endh }
func (v *attr) Type() uuid.UUID      { return v.typ }
func (v *attr) HandleATT(conn bt.Conn, req []byte, rsp bt.ResponseWriter) bt.AttError {
	r := &request{conn: conn}
	rsp.SetStatus(bt.ErrSuccess)
	switch req[0] {
	case att.ReadByTypeRequestCode:
		if v.rh == nil {
			return bt.ErrReadNotPerm
		}
		v.rh.ServeRead(r, rsp)
	case att.ReadRequestCode:
		if v.rh == nil {
			return bt.ErrReadNotPerm
		}
		v.rh.ServeRead(r, rsp)
	case att.ReadBlobRequestCode:
		if v.rh == nil {
			return bt.ErrReadNotPerm
		}
		r.offset = int(att.ReadBlobRequest(req).ValueOffset())
		v.rh.ServeRead(r, rsp)
	case att.WriteRequestCode:
		if v.wh == nil {
			return bt.ErrWriteNotPerm
		}
		r.data = att.WriteRequest(req).AttributeValue()
		v.wh.ServeWrite(r, rsp)
	case att.WriteCommandCode:
		if v.wh == nil {
			return bt.ErrWriteNotPerm
		}
		r.data = att.WriteRequest(req).AttributeValue()
		v.wh.ServeWrite(r, rsp)
	// case att.PrepareWriteRequestCode:
	// case att.ExecuteWriteRequestCode:
	// case att.SignedWriteCommandCode:
	// case att.ReadByGroupTypeRequestCode:
	// case att.ReadMultipleRequestCode:
	default:
		return bt.ErrReqNotSupp
	}

	return rsp.Status()
}
func genAttr(ss []bt.Service, base uint16) *att.Range {
	var svrAttr []*attr
	h := base
	for i, s := range ss {
		var svcAttrs []*attr
		h, svcAttrs = genSvcAttr(s.(*svc), h)
		if i == len(ss)-1 {
			svcAttrs[0].endh = 0xFFFF
		}
		svrAttr = append(svrAttr, svcAttrs...)
	}
	var svrRange []att.Attribute
	for _, a := range svrAttr {
		svrRange = append(svrRange, a)
	}
	att.DumpAttributes(svrRange)
	return &att.Range{Attributes: svrRange, Base: base}
}

func genSvcAttr(s *svc, h uint16) (uint16, []*attr) {
	s.attr.h = h
	s.attr.typ = uuid.UUID(attrPrimaryServiceUUID)
	s.attr.v = s.UUID()
	h++
	svcAttrs := []*attr{&s.attr}

	for _, c := range s.Characteristics() {
		var charRange []*attr
		h, charRange = genCharAttr(c.(*char), h)
		svcAttrs = append(svcAttrs, charRange...)
	}

	s.attr.endh = h - 1
	return h, svcAttrs
}

func genCharAttr(c *char, h uint16) (uint16, []*attr) {
	vh := h + 1

	c.attr.h = h
	c.attr.vh = vh
	c.attr.typ = uuid.UUID(attrCharacteristicUUID)
	c.attr.v = append([]byte{byte(c.Properties()), byte(vh), byte((vh) >> 8)}, c.UUID()...)

	c.vattr.h = vh
	c.vattr.typ = c.UUID()

	h += 2

	charRange := []*attr{&c.attr, &c.vattr}
	for _, d := range c.Descriptors() {
		charRange = append(charRange, genDescAttr(d.(*desc), h))
		h++
	}

	c.attr.endh = h - 1
	return h, charRange
}

func genDescAttr(d *desc, h uint16) *attr {
	d.attr.h = h
	d.attr.endh = h
	d.attr.typ = d.UUID()
	return &d.attr
}
