package gatt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/currantlabs/bt"
)

type security int

const (
	securityLow = iota
	securityMed
	securityHigh
)

type central struct {
	attrs       *attrRange
	txMTU       uint16
	rxMTU       uint16
	security    security
	l2conn      bt.Conn
	notifiers   map[uint16]*notifier
	notifiersmu *sync.Mutex
}

func newCentral(a *attrRange, l2conn bt.Conn) *central {
	return &central{
		attrs:       a,
		txMTU:       23,
		rxMTU:       512,
		security:    securityLow,
		l2conn:      l2conn,
		notifiers:   make(map[uint16]*notifier),
		notifiersmu: &sync.Mutex{},
	}
}

func (c *central) ID() string {
	return c.l2conn.RemoteAddr().String()
}

func (c *central) Close() error {
	c.notifiersmu.Lock()
	defer c.notifiersmu.Unlock()
	for _, n := range c.notifiers {
		n.stop()
	}
	return c.l2conn.Close()
}

func (c *central) MTU() int {
	return c.l2conn.TxMTU()
}

func (c *central) loop() {
	for {
		b := make([]byte, c.l2conn.RxMTU())
		n, err := c.l2conn.Read(b)
		if n == 0 || err != nil {
			c.Close()
			break
		}
		if rsp := c.handleReq(b[:n]); rsp != nil {
			c.l2conn.Write(rsp)
		}
	}
}

func (c *central) handleReq(b []byte) []byte {
	var resp []byte
	switch reqType := b[0]; reqType {
	case ExchangeMTURequestCode:
		resp = c.handleMTU(b)
	case FindInformationRequestCode:
		resp = c.handleFindInfo(b)
	case FindByTypeValueRequestCode:
		resp = c.handleFindByTypeValue(b)
	case ReadByTypeRequestCode:
		resp = c.handleReadByType(b)
	case ReadRequestCode:
		resp = c.handleRead(b)
	case ReadBlobRequestCode:
		resp = c.handleReadBlob(b)
	case ReadByGroupTypeRequestCode:
		resp = c.handleReadByGroup(b)
	case WriteRequestCode, WriteCommandCode:
		resp = c.handleWrite(reqType, b)
	case ReadMultipleRequestCode,
		PrepareWriteRequestCode,
		ExecuteWriteRequestCode,
		SignedWriteCommandCode:
		fallthrough
	default:
		resp = NewErrorResponse(reqType, 0x0000, ErrReqNotSupp)
	}
	return resp
}

func (c *central) handleMTU(r ExchangeMTURequest) []byte {
	c.txMTU = r.ClientRxMTU()
	if c.txMTU < 23 {
		c.txMTU = 23
	}
	if c.txMTU > 512 {
		c.txMTU = 512
	}
	c.l2conn.SetTxMTU(int(c.txMTU))
	c.l2conn.SetRxMTU(int(c.rxMTU))
	rsp := ExchangeMTUResponse(make([]byte, 3))
	rsp.SetAttributeOpcode()
	rsp.SetServerRxMTU(c.txMTU)
	return rsp
}

func (c *central) handleFindInfo(r FindInformationRequest) []byte {
	rsp := FindInformationResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.InformationData())
	buf.Reset()

	for _, a := range c.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if rsp.Format() == 0 {
			rsp.SetFormat(0x01)
			if a.typ.Len() == 16 {
				rsp.SetFormat(0x02)
			}
		}
		if rsp.Format() == 0x01 && a.typ.Len() != 2 {
			break
		}
		if rsp.Format() == 0x02 && a.typ.Len() != 16 {
			break
		}
		if buf.Len()+2+a.typ.Len() > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.h)
		binary.Write(buf, binary.LittleEndian, a.typ)
	}

	if rsp.Format() == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

func (c *central) handleFindByTypeValue(r FindByTypeValueRequest) []byte {
	if !UUID16(r.AttributeType()).Equal(attrPrimaryServiceUUID) {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}

	rsp := FindByTypeValueResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.HandleInformationList())
	buf.Reset()

	for _, a := range c.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if !a.typ.Equal(attrPrimaryServiceUUID) {
			continue
		}
		if !(UUID(a.value).Equal(UUID(r.AttributeValue()))) {
			continue
		}
		s := a.pvt.(*Service)
		if buf.Len()+4 > buf.Cap() {
			break
		}
		binary.Write(buf, binary.LittleEndian, s.h)
		binary.Write(buf, binary.LittleEndian, s.endh)
	}
	if buf.Len() == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}

	return rsp[:1+buf.Len()]
}

func (c *central) handleReadByType(r ReadByTypeRequest) []byte {
	rsp := ReadByTypeResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()
	dlen := 0
	for _, a := range c.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if !a.typ.Equal(UUID(r.AttributeType())) {
			continue
		}
		if (a.secure&CharRead) != 0 && c.security > securityLow {
			return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAuthentication)
		}
		v := a.value
		if v == nil {
			rsp := newResponseWriter(int(c.txMTU - 1))
			req := &Request{
				Central: c,
				Cap:     int(c.txMTU - 1),
				Offset:  0,
			}
			if c, ok := a.pvt.(*Characteristic); ok {
				c.rhandler.ServeRead(rsp, req)
			} else if d, ok := a.pvt.(*Descriptor); ok {
				d.rhandler.ServeRead(rsp, req)
			}
			v = rsp.bytes()
		}
		if dlen == 0 {
			dlen = 2 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 2+len(v) != dlen {
			break
		}
		binary.Write(buf, binary.LittleEndian, a.h)
		binary.Write(buf, binary.LittleEndian, v[:dlen-2])
	}
	if dlen == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

func (c *central) handleRead(r ReadRequest) []byte {
	a, ok := c.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidHandle)
	}
	if a.props&CharRead == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrReadNotPerm)
	}
	if a.secure&CharRead != 0 && c.security > securityLow {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrAuthentication)
	}

	rsp := ReadResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	v := a.value
	if v == nil {
		req := &Request{
			Central: c,
			Cap:     buf.Cap(),
			Offset:  0,
		}
		rsp := newResponseWriter(buf.Cap())
		if c, ok := a.pvt.(*Characteristic); ok {
			c.rhandler.ServeRead(rsp, req)
		} else if d, ok := a.pvt.(*Descriptor); ok {
			d.rhandler.ServeRead(rsp, req)
		}
		v = rsp.bytes()
	}

	if len(v) > buf.Cap() {
		v = v[:buf.Cap()]
	}
	binary.Write(buf, binary.LittleEndian, v)
	return rsp[:1+buf.Len()]
}

func (c *central) handleReadBlob(r ReadBlobRequest) []byte {
	a, ok := c.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidHandle)
	}
	if a.props&CharRead == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrReadNotPerm)
	}
	if a.secure&CharRead != 0 && c.security > securityLow {
		return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrAuthentication)
	}

	rsp := ReadBlobResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.PartAttributeValue())
	buf.Reset()

	v := a.value
	if v == nil {
		req := &Request{
			Central: c,
			Cap:     buf.Cap(),
			Offset:  int(r.ValueOffset()),
		}
		rsp := newResponseWriter(buf.Cap())
		if c, ok := a.pvt.(*Characteristic); ok {
			c.rhandler.ServeRead(rsp, req)
		} else if d, ok := a.pvt.(*Descriptor); ok {
			d.rhandler.ServeRead(rsp, req)
		}
		v = rsp.bytes()
	} else {
		if len(a.value) < int(r.ValueOffset()) {
			return NewErrorResponse(r.AttributeOpcode(), r.AttributeHandle(), ErrInvalidOffset)
		}
	}

	if len(v) > buf.Cap() {
		v = v[:buf.Cap()]
	}
	binary.Write(buf, binary.LittleEndian, v)
	return rsp[:1+buf.Len()]
}

func (c *central) handleReadByGroup(r ReadByGroupTypeRequest) []byte {
	if !attrPrimaryServiceUUID.Equal(UUID(r.AttributeGroupType())) {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrUnsuppGrpType)
	}

	rsp := ReadByGroupTypeResponse(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	buf := bytes.NewBuffer(rsp.AttributeDataList())
	buf.Reset()

	dlen := 0
	for _, a := range c.attrs.Subrange(r.StartingHandle(), r.EndingHandle()) {
		if !a.typ.Equal(attrPrimaryServiceUUID) {
			continue
		}
		s := a.pvt.(*Service)
		v := a.value
		if dlen == 0 {
			dlen = 4 + len(v)
			if dlen > 255 {
				dlen = 255
			}
			if dlen > buf.Cap() {
				dlen = buf.Cap()
			}
			rsp.SetLength(uint8(dlen))
		} else if 4+len(v) != dlen {
			break
		}
		binary.Write(buf, binary.LittleEndian, s.h)
		binary.Write(buf, binary.LittleEndian, s.endh)
		binary.Write(buf, binary.LittleEndian, v[:dlen-4])
	}
	if dlen == 0 {
		return NewErrorResponse(r.AttributeOpcode(), r.StartingHandle(), ErrAttrNotFound)
	}
	return rsp[:2+buf.Len()]
}

func (c *central) handleWrite(reqType byte, r WriteRequest) []byte {
	value := r.AttributeValue()

	a, ok := c.attrs.At(r.AttributeHandle())
	if !ok {
		return NewErrorResponse(reqType, r.AttributeHandle(), ErrInvalidHandle)
	}

	noRsp := reqType == WriteCommandCode
	charFlag := CharWrite
	if noRsp {
		charFlag = CharWriteNR
	}
	if a.props&charFlag == 0 {
		return NewErrorResponse(reqType, r.AttributeHandle(), ErrWriteNotPerm)
	}
	if a.secure&charFlag == 0 && c.security > securityLow {
		return NewErrorResponse(reqType, r.AttributeHandle(), ErrAuthentication)
	}

	// Props of Service and Characteristic declration are read only.
	// So we only need deal with writable descriptors here.
	// (Characteristic's value is implemented with descriptor)
	if !a.typ.Equal(attrClientCharacteristicConfigUUID) {
		// Regular write, not CCC
		r := Request{Central: c}
		if c, ok := a.pvt.(*Characteristic); ok {
			c.whandler.ServeWrite(r, value)
		} else if d, ok := a.pvt.(*Characteristic); ok {
			d.whandler.ServeWrite(r, value)
		}
		if noRsp {
			return nil
		}
		return []byte{WriteResponseCode}
	}

	// CCC/descriptor write
	if len(value) != 2 {
		return NewErrorResponse(reqType, r.AttributeHandle(), ErrInvalAttrValueLen)
	}
	ccc := binary.LittleEndian.Uint16(value)
	// char := a.pvt.(*Descriptor).char
	if ccc&(gattCCCNotifyFlag|gattCCCIndicateFlag) != 0 {
		c.startNotify(&a, int(c.txMTU-3))
	} else {
		c.stopNotify(&a)
	}
	if noRsp {
		return nil
	}
	return []byte{WriteResponseCode}
}

func (c *central) sendNotification(a *attr, data []byte) (int, error) {
	rsp := HandleValueNotification(make([]byte, c.txMTU))
	rsp.SetAttributeOpcode()
	rsp.SetAttributeHandle(a.pvt.(*Descriptor).char.vh)
	buf := bytes.NewBuffer(rsp.AttributeValue())
	buf.Reset()
	if len(data) > buf.Cap() {
		data = data[:buf.Cap()]
	}
	buf.Write(data)
	return c.l2conn.Write(rsp[:3+buf.Len()])
}

func (c *central) startNotify(a *attr, maxlen int) {
	c.notifiersmu.Lock()
	defer c.notifiersmu.Unlock()
	if _, found := c.notifiers[a.h]; found {
		return
	}
	char := a.pvt.(*Descriptor).char
	n := newNotifier(c, a, maxlen)
	c.notifiers[a.h] = n
	go char.nhandler.ServeNotify(Request{Central: c}, n)
}

func (c *central) stopNotify(a *attr) {
	c.notifiersmu.Lock()
	defer c.notifiersmu.Unlock()
	// char := a.pvt.(*Characteristic)
	if n, found := c.notifiers[a.h]; found {
		n.stop()
		delete(c.notifiers, a.h)
	}
}

type notifier struct {
	central *central
	a       *attr
	maxlen  int
	donemu  sync.RWMutex
	done    bool
}

func newNotifier(c *central, a *attr, maxlen int) *notifier {
	return &notifier{central: c, a: a, maxlen: maxlen}
}

func (n *notifier) Write(b []byte) (int, error) {
	n.donemu.RLock()
	defer n.donemu.RUnlock()
	if n.done {
		return 0, errors.New("central stopped notifications")
	}
	return n.central.sendNotification(n.a, b)
}

func (n *notifier) Cap() int {
	return n.maxlen
}

func (n *notifier) Done() bool {
	n.donemu.RLock()
	defer n.donemu.RUnlock()
	return n.done
}

func (n *notifier) stop() {
	n.donemu.Lock()
	n.done = true
	n.donemu.Unlock()
}

// responseWriter is the default implementation of ResponseWriter.
type responseWriter struct {
	capacity int
	buf      *bytes.Buffer
	status   byte
}

func newResponseWriter(c int) *responseWriter {
	return &responseWriter{
		capacity: c,
		buf:      new(bytes.Buffer),
		status:   StatusSuccess,
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if avail := w.capacity - w.buf.Len(); avail < len(b) {
		return 0, fmt.Errorf("requested write %d bytes, %d available", len(b), avail)
	}
	return w.buf.Write(b)
}

func (w *responseWriter) SetStatus(status byte) { w.status = status }
func (w *responseWriter) bytes() []byte         { return w.buf.Bytes() }
