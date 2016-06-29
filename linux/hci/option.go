package hci

import (
	"time"

	"github.com/currantlabs/ble/linux/hci/cmd"
)

// An Option is a configuration function, which configures the device.
type Option func(*HCI) error

// OptDeviceID sets HCI device ID.
func OptDeviceID(id int) Option {
	return func(h *HCI) error {
		h.id = id
		return nil
	}
}

// OptDialerTimeout sets dialing timeout for Dialer.
func OptDialerTimeout(d time.Duration) Option {
	return func(h *HCI) error {
		h.dialerTmo = d
		return nil
	}
}

// OptListenerTimeout sets dialing timeout for Listener.
func OptListenerTimeout(d time.Duration) Option {
	return func(h *HCI) error {
		h.listenerTmo = d
		return nil
	}
}

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

// OptConnParams sets connection parameters.
func OptConnParams(p ConnParams) Option {
	return func(h *HCI) error {
		h.states.connParams = cmd.LECreateConnection(p)
		return nil
	}
}
