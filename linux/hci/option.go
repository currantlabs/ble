package hci

import (
	"github.com/currantlabs/ble/linux/hci/cmd"
)

// An Option is a configuration function, which configures the device.
type Option func(*HCI) error

// OptScanParams sets scanning parameters.
func OptScanParams(p ScanParams) Option {
	return func(h *HCI) error {
		h.states.scanParams = cmd.LESetScanParameters(p)
		return nil
	}
}

// OptAdvParams sets advertising parameters.
func OptAdvParams(p AdvParams) Option {
	return func(h *HCI) error {
		h.states.advParams = cmd.LESetAdvertisingParameters(p)
		return nil
	}
}

// OptConnParams ...
func OptConnParams(p ConnParams) Option {
	return func(h *HCI) error {
		h.states.connParams = cmd.LECreateConnection(p)
		return nil
	}
}
