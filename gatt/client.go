//go:generate sh -c "go run ../tools/codegen/codegen.go -tmpl att -in ../tools/codegen/att.json -out att_gen.go && goimports -w att_gen.go"

package gatt

import (
	"encoding/binary"
	"errors"
	"log"
	"sync"

	"github.com/currantlabs/bt"
)

var (
	// ErrInvalidArgument means one or more of the arguments specified are invalid.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrInvalidResponse means one or more of the response fields are invalid.
	ErrInvalidResponse = errors.New("invalid response")
)

// client ...
type client struct {
	conn  bt.Conn
	rspc  chan []byte
	txBuf []byte

	handlers   map[uint16]attHandler
	muHandlers *sync.Mutex
}

// newClient ...
func newClient(l2c bt.Conn) *client {
	a := &client{
		conn:  l2c,
		rspc:  make(chan []byte),
		txBuf: make([]byte, l2c.TxMTU()),

		handlers:   make(map[uint16]attHandler),
		muHandlers: &sync.Mutex{},
	}
	return a
}

// ExchangeMTU informs the server of the client’s maximum receive MTU size and
// request the server to respond with its maximum receive MTU size. [Vol 3, PartF, 3.4.2.1]
func (a *client) ExchangeMTU(clientRxMTU int) (serverRxMTU int, err error) {
	if clientRxMTU < 23 || clientRxMTU > 65535 {
		return 0, ErrInvalidArgument
	}
	a.conn.SetRxMTU(clientRxMTU)
	req := ExchangeMTURequest(a.txBuf[:3])
	req.SetAttributeOpcode()
	req.SetClientRxMTU(uint16(clientRxMTU))

	rsp := ExchangeMTUResponse(a.sendReq(req))
	txMTU := int(rsp.ServerRxMTU())
	a.conn.SetTxMTU(txMTU)
	return 0, nil
}

// FindInformation obtains the mapping of attribute handles with their associated types.
// This allows a client to discover the list of attributes and their types on a server.
// [Vol 3, PartF, 3.4.3.1 & 3.4.3.2]
func (a *client) FindInformation(starth, endh uint16) (fmt int, data []byte, err error) {
	req := FindInformationRequest(a.txBuf[:5])
	req.SetAttributeOpcode()
	req.SetStartingHandle(starth)
	req.SetEndingHandle(endh)

	rsp := FindInformationResponse(a.sendReq(req))
	if rsp[0] == ErrorResponseCode {
		return 0, nil, Error(rsp[4])
	}
	return int(rsp.Format()), rsp.InformationData(), nil
}

// // HandleInformationList ...
// type HandleInformationList []byte
//
// // FoundAttributeHandle ...
// func (l HandleInformationList) FoundAttributeHandle() []byte { return l[:2] }
//
// // GroupEndHandle ...
// func (l HandleInformationList) GroupEndHandle() []byte { return l[2:4] }
//
// // FindByTypeValue ...
// func (a *client) FindByTypeValue(starth, endh, attrType uint16, value []byte) ([]HandleInformationList, error) {
// 	return nil, nil
// }

// ReadByType obtains the values of attributes where the attribute type is known
// but the handle is not known. [Vol 3, PartF, 3.4.4.1 & 3.4.4.2]
func (a *client) ReadByType(starth, endh uint16, uuid UUID) (int, []byte, error) {
	if starth > endh || (len(uuid) != 2 && len(uuid) != 16) {
		return 0, nil, ErrInvalidArgument
	}
	req := ReadByTypeRequest(a.txBuf[:5+len(uuid)])
	req.SetAttributeOpcode()
	req.SetStartingHandle(starth)
	req.SetEndingHandle(endh)
	req.SetAttributeType(uuid)

	rsp := ReadByTypeResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return 0, nil, Error(rsp[4])
	case len(rsp) < 4:
		return 0, nil, ErrInvalidResponse
	case len(rsp.AttributeDataList())%int(rsp.Length()) != 0:
		return 0, nil, ErrInvalidResponse
	}

	return int(rsp.Length()), rsp.AttributeDataList(), nil
}

// Read requests the server to read the value of an attribute and return its
// value in a Read Response. [Vol 3, PartF, 3.4.4.3 & 3.4.4.4]
func (a *client) Read(handle uint16) ([]byte, error) {
	req := ReadRequest(a.txBuf[:3])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)

	rsp := ReadResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return nil, Error(rsp[4])
	case len(rsp) < 1:
		return nil, ErrInvalidResponse
	}
	return rsp.AttributeValue(), nil
}

// ReadBlob requests the server to read part of the value of an attribute at a
// given offset and return a specific part of the value in a Read Blob Response.
// [Vol 3, PartF, 3.4.4.5 & 3.4.4.6]
func (a *client) ReadBlob(handle, offset uint16) ([]byte, error) {
	req := ReadBlobRequest(a.txBuf[:5])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)
	req.SetValueOffset(offset)

	rsp := ReadBlobResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return nil, Error(rsp[4])
	case len(rsp) < 1:
		return nil, ErrInvalidResponse
	}
	return rsp.PartAttributeValue(), nil
}

// ReadMultiple requests the server to read two or more values of a set of
// attributes and return their values in a Read Multiple Response.
// Only values that have a known fixed size can be read, with the exception of
// the last value that can have a variable length. The knowledge of whether
// attributes have a known fixed size is defined in a higher layer specification.
// [Vol 3, PartF, 3.4.4.7 & 3.4.4.8]
func (a *client) ReadMultiple(handles []uint16) ([]byte, error) {
	// Should request to read two or more values.
	if len(handles) < 2 || len(handles)*2 > a.conn.TxMTU()-1 {
		return nil, ErrInvalidArgument
	}
	req := ReadMultipleRequest(a.txBuf[:1+len(handles)*2])
	req.SetAttributeOpcode()
	p := req.SetOfHandles()
	for _, h := range handles {
		binary.LittleEndian.PutUint16(p, h)
		p = p[2:]
	}

	rsp := ReadMultipleResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return nil, Error(rsp[4])
	case len(rsp) < 1:
		return nil, ErrInvalidResponse
	}
	return rsp.SetOfValues(), nil
}

// ReadByGroupType obtains the values of attributes where the attribute type is known,
// the type of a grouping attribute as defined by a higher layer specification, but
// the handle is not known. [Vol 3, PartF, 3.4.4.9 & 3.4.4.10]
func (a *client) ReadByGroupType(starth, endh uint16, uuid UUID) (int, []byte, error) {
	if starth > endh || (len(uuid) != 2 && len(uuid) != 16) {
		return 0, nil, ErrInvalidArgument
	}
	req := ReadByGroupTypeRequest(a.txBuf[:5+len(uuid)])
	req.SetAttributeOpcode()
	req.SetStartingHandle(starth)
	req.SetEndingHandle(endh)
	req.SetAttributeGroupType(uuid)

	rsp := ReadByGroupTypeResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return 0, nil, Error(rsp[4])
	case len(rsp) < 4:
		return 0, nil, ErrInvalidResponse
	case len(rsp.AttributeDataList())%int(rsp.Length()) != 0:
		return 0, nil, ErrInvalidResponse
	}

	return int(rsp.Length()), rsp.AttributeDataList(), nil
}

// Write requests the server to write the value of an attribute and acknowledge that
// this has been achieved in a Write Response. [Vol 3, PartF, 3.4.5.1 & 3.4.5.2]
func (a *client) Write(handle uint16, value []byte) error {
	if len(value) > a.conn.TxMTU()-3 {
		return ErrInvalidArgument
	}
	req := WriteRequest(a.txBuf[:3+len(value)])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)
	req.SetAttributeValue(value)

	rsp := WriteResponse(a.sendReq(req))
	if rsp[0] == ErrorResponseCode {
		return Error(rsp[4])
	}
	return nil
}

// WriteCommand requests the server to write the value of an attribute, typically
// into a control-point attribute. [Vol 3, PartF, 3.4.5.3]
func (a *client) WriteCommand(handle uint16, value []byte) {
	if len(value) > a.conn.TxMTU()-3 {
		return
	}
	req := WriteCommand(a.txBuf[:3+len(value)])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)
	req.SetAttributeValue(value)
	a.sendReq(req)
}

// SignedWrite requests the server to write the value of an attribute with an authentication
// signature, typically into a control-point attribute. [Vol 3, PartF, 3.4.5.4]
func (a *client) SignedWrite(handle uint16, value []byte, signature [12]byte) {
	if len(value) > a.conn.TxMTU()-15 {
		return
	}
	req := SignedWriteCommand(a.txBuf[:15+len(value)])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)
	req.SetAttributeValue(value)
	req.SetAuthenticationSignature(signature)
	a.sendReq(req)
}

// PrepareWrite requests the server to prepare to write the value of an attribute.
// The server will respond to this request with a Prepare Write Response, so that
// the client can verify that the value was received correctly.
// [Vol 3, PartF, 3.4.6.1 & 3.4.6.2]
func (a *client) PrepareWrite(handle uint16, offset uint16, value []byte) (uint16, uint16, []byte, error) {
	if len(value) > a.conn.TxMTU()-5 {
		return 0, 0, nil, ErrInvalidArgument
	}
	req := PrepareWriteRequest(a.txBuf[:5+len(value)])
	req.SetAttributeOpcode()
	req.SetAttributeHandle(handle)
	req.SetValueOffset(offset)

	rsp := PrepareWriteResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return 0, 0, nil, Error(rsp[4])
	case len(rsp) < 5:
		return 0, 0, nil, ErrInvalidResponse
	}
	return rsp.AttributeHandle(), rsp.ValueOffset(), rsp.PartAttributeValue(), nil
}

// ExecuteWrite requests the server to write or cancel the write of all the prepared
// values currently held in the prepare queue from this client. This request shall be
// handled by the server as an atomic operation. [Vol 3, PartF, 3.4.6.3 & 3.4.6.4]
func (a *client) ExecuteWrite(flags uint8) error {
	req := ExecuteWriteRequest(a.txBuf[:1])
	req.SetAttributeOpcode()
	req.SetFlags(flags)

	rsp := ExecuteWriteResponse(a.sendReq(req))
	switch {
	case rsp[0] == ErrorResponseCode && len(rsp) == 5:
		return Error(rsp[4])
	}
	return nil
}

// HandleValueNotification ...
func (a *client) HandleValueNotification(handle uint, value []byte) {}

// HandleValueIndication ...
func (a *client) HandleValueIndication(handle uint, value []byte) {}

func (a *client) sendCmd(b []byte) {
	a.conn.Write(b)
}

func (a *client) sendReq(b []byte) []byte {
	a.conn.Write(b)
	return <-a.rspc
}

// Loop ...
func (a *client) Loop() {
	buf := make([]byte, 512) // TODO: MTU
	for {
		n, err := a.conn.Read(buf)
		if n == 0 || err != nil {
			return
		}

		b := make([]byte, n)
		copy(b, buf)

		if (b[0] != HandleValueNotificationCode) && (b[0] != HandleValueIndicationCode) {
			a.rspc <- b
			continue
		}

		h := binary.LittleEndian.Uint16(b[1:3])
		fn := a.handlers[h]
		if fn == nil {
			log.Printf("notified by unsubscribed handle")
		} else {
			go fn(b[3:], nil)
		}

		if b[0] == HandleValueIndicationCode {
			// write aknowledgement for indication
			a.conn.Write([]byte{HandleValueConfirmationCode})
		}
	}
}

// attHandler ...
type attHandler func([]byte, error)

// SetNotifyValue ...
func (a *client) SetNotifyValue(cccdh, valueh, flag uint16, fn attHandler) error {
	// log.Printf("SetNotiFy, handle: 0x%04X, flag: 0x%04x", valueh, flag)
	a.muHandlers.Lock()
	defer a.muHandlers.Unlock()
	ccc := make([]byte, 2)
	if fn != nil {
		binary.LittleEndian.PutUint16(ccc, flag)
		a.handlers[valueh] = fn
	}
	if err := a.Write(cccdh, ccc); err != nil {
		return err
	}
	if fn == nil {
		delete(a.handlers, valueh)
	}
	return nil
}
