package dev

import (
	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux"
)

// DefaultDevice ...
func DefaultDevice() (d ble.Device, err error) {
	return linux.NewDevice()
}
