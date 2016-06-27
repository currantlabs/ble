package gatt

import (
	"github.com/currantlabs/ble/darwin"

	"github.com/currantlabs/x/io/bt"
	"github.com/pkg/errors"
)

// NewBroadcaster ...
func NewBroadcaster() (bt.Broadcaster, error) {
	return newDev("broadcaster", darwin.OptPeripheralRole())
}

// NewObserver ...
func NewObserver() (bt.Observer, error) {
	return newDev("observer", darwin.OptCentralRole())
}

// NewPeripheral ...
func NewPeripheral() (bt.Peripheral, error) {
	return newDev("peripheral", darwin.OptPeripheralRole())
}

// NewCentral ...
func NewCentral() (bt.Central, error) {
	return newDev("central", darwin.OptCentralRole())
}

func newDev(role string, opts ...darwin.Option) (*darwin.Device, error) {
	dev, err := darwin.NewDevice(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "create %s failed", role)
	}
	if err := dev.Init(nil); err != nil {
		return nil, errors.Wrapf(err, "init %s failed", role)
	}
	return dev, nil
}

// NewServer ...
func NewServer() (bt.Server, error) {
	return darwin.NewServer()
}

// NewClient ...
func NewClient(c bt.Conn) (bt.Client, error) {
	return darwin.NewClient(c)
}
