package hci

import (
	"fmt"
	"io"
)

type aclHandler struct {
	skt     io.Writer
	bufSize int
	bufCnt  int
	handler Handler
}

func newACLHandler(skt io.Writer) *aclHandler {
	return &aclHandler{skt: skt}
}

func (a *aclHandler) setACLHandler(h Handler) (w io.Writer, size int, cnt int) {
	a.handler = h
	return a.skt, a.bufSize, a.bufCnt
}

func (a *aclHandler) handle(b []byte) error {
	if a.handler == nil {
		return fmt.Errorf("hci: unhandled ACL packet: % X", b)
	}
	return a.handler.Handle(b)
}
