package dev

import (
	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/bled"
)

// NewDevice ...
func NewDevice(impl string) (d ble.Device, err error) {
	if impl == "bled" {
		return bled.NewDevice()
	}
	return DefaultDevice()
}
