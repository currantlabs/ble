package ble

import "strings"

// Addr represents a network end point address.
// It's MAC address on Linux or Device UUID on OS X.
type Addr interface {
	String() string
}

// NewAddr creates an Addr from string
func NewAddr(s string) Addr {
	return addr(strings.ToLower(s))
}

type addr string

func (a addr) String() string {
	return string(a)
}
