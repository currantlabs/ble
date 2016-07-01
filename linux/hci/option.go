package hci

import "time"

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
