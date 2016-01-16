package gatt

import (
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
)

// LnxDeviceID specifies which HCI device to use.
// If n is set to -1, all the available HCI devices will be probed.
// This option can only be used with NewDevice on Linux implementation.
func LnxDeviceID(n int) Option {
	return func(d *Device) error {
		d.devID = n
		return nil
	}
}

// LnxMaxConnections is an optional parameter.
// If set, it overrides the default max connections supported.
// This option can only be used with NewDevice on Linux implementation.
func LnxMaxConnections(n int) Option {
	return func(d *Device) error {
		d.maxConn = n
		return nil
	}
}

// LnxSetAdvertisingData sets the advertising data to the HCI device.
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSetAdvertisingData(c *cmd.LESetAdvertisingData) Option {
	return func(d *Device) error {
		d.advData = c
		return nil
	}
}

// LnxSetScanResponseData sets the scan response data to the HXI device.
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSetScanResponseData(c *cmd.LESetScanResponseData) Option {
	return func(d *Device) error {
		d.scanResp = c
		return nil
	}
}

// LnxSetScanParameters sets the LE scan parameters to the HXI device.
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSetScanParameters(c *cmd.LESetScanParameters) Option {
	return func(d *Device) error {
		d.scanParam = c
		return nil
	}
}

// LnxSetAdvertisingParameters sets the advertising parameters to the HCI device.
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSetAdvertisingParameters(c *cmd.LESetAdvertisingParameters) Option {
	return func(d *Device) error {
		d.advParam = c
		return nil
	}
}

// LnxSetConnectionParameters sets the LE create connection parameters to the HCI device.
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSetConnectionParameters(c *cmd.LECreateConnection) Option {
	return func(d *Device) error {
		d.connParam = c
		return nil
	}
}

// LnxSendHCIRawCommand sends a raw command to the HCI device
// This option can be used with NewDevice or Option on Linux implementation.
func LnxSendHCIRawCommand(c hci.Command, r hci.CommandRP) Option {
	return func(d *Device) error {
		return d.hci.Send(c, r)
	}
}
