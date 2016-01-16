//go:generate sh -c "go run ../tools/codegen/codegen.go -tmpl cmd -in ../tools/codegen/cmd.json -out cmd_gen.go && goimports -w cmd_gen.go"

package cmd

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Command ...
type Command interface {
	OpCode() int
	Len() int
	Marshal([]byte) error
}

// CommandRP ...
type CommandRP interface {
	Unmarshal(b []byte) error
}

// Sender ...
type Sender interface {
	// Send sends a HCI Command and returns unserialized return parameter.
	Send(Command, CommandRP) error
}

// Send ...
func Send(s Sender, c Command, r CommandRP) error {
	return s.Send(c, r)
}

func marshal(c Command, b []byte) error {
	buf := bytes.NewBuffer(b)
	buf.Reset()
	if buf.Cap() < c.Len() {
		return io.ErrShortBuffer
	}
	return binary.Write(buf, binary.LittleEndian, c)
}

func unmarshal(c CommandRP, b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, binary.LittleEndian, c)
}
