package dev

import "github.com/currantlabs/ble"

// NewDevice ...
func NewDevice(impl string) (d ble.Device, err error) {
	return DefaultDevice()
}
