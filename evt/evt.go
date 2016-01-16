//go:generate sh -c "go run ../tools/codegen/codegen.go -tmpl evt -in ../tools/codegen/evt.json -out evt_gen.go && goimports -w evt_gen.go"

package evt

import (
	"bytes"
	"encoding/binary"
)

type event interface {
	Unmarshal(b []byte) error
}

func unmarshal(e event, b []byte) error {
	return binary.Read(bytes.NewBuffer(b), binary.LittleEndian, e)
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *CommandCompleteEvent) Unmarshal(b []byte) error {
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, binary.LittleEndian, &e.NumHCICommandPackets); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.LittleEndian, &e.CommandOpcode); err != nil {
		return err
	}
	e.ReturnParameters = buf.Bytes()
	return nil
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *NumberOfCompletedPacketsEvent) Unmarshal(b []byte) error {
	e.NumberOfHandles, b = b[0], b[1:]
	n := int(e.NumberOfHandles)
	e.ConnectionHandle = make([]uint16, n)
	e.HCNumOfCompletedPackets = make([]uint16, n)

	for i := 0; i < n; i++ {
		e.ConnectionHandle[i] = binary.LittleEndian.Uint16(b)
		e.HCNumOfCompletedPackets[i] = binary.LittleEndian.Uint16(b[2:])
	}

	return nil
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
// This implementation only serves as a working reference.
// To maintain the consistency with those generated code, we stick with the
// stride style of data fields close to the packet we receive.
// Serious applications should register their own advertising report event
// handlers which more efficiently parse interested fields in their own representation.
func (e *LEAdvertisingReportEvent) Unmarshal(b []byte) error {

	e.SubeventCode, b = b[0], b[1:]
	e.NumReports, b = b[0], b[1:]
	n := int(e.NumReports)

	e.EventType = make([]uint8, n)
	e.AddressType = make([]uint8, n)
	e.Address = make([][6]uint8, n)
	e.Data = make([][]byte, n)
	e.Length = make([]uint8, n)
	e.RSSI = make([]int8, n)

	for i := 0; i < n; i++ {
		e.EventType[i] = b[0]
		e.AddressType[i] = b[1]
		copy(e.Address[i][:6], b[2:])
		e.Length[i] = b[8]
		dlen := int(e.Length[i])
		e.Data[i] = make([]byte, dlen)
		copy(e.Data[i], b[9:])
		e.RSSI[i] = int8(b[9+dlen])
		b = b[1+1+6+1+dlen+1:]
	}

	return nil
}
