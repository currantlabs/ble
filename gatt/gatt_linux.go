package gatt

import (
	"github.com/pkg/errors"

	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci"
	"github.com/currantlabs/x/io/bt"
)

// NewPeripheral ...
func NewPeripheral(opts ...hci.Option) (bt.Peripheral, error) {
	return newDev("peripheral", opts...)
}

func NewCentral(opts ...hci.Option) (bt.Central, error) {
	return newDev("central", opts...)
}

func NewBroadcaster(opts ...hci.Option) (bt.Broadcaster, error) {
	return newDev("broadcaster", opts...)
}

func NewObserver(opts ...hci.Option) (bt.Observer, error) {
	return newDev("observer", opts...)
}

func newDev(role string, opts ...hci.Option) (*hci.HCI, error) {
	dev, err := hci.NewHCI(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "create hci failed")
	}
	if err = dev.Init(); err != nil {
		return nil, errors.Wrap(err, "init hci failed")
	}
	return dev, nil
}

func NewServer() (bt.Server, error) {
	return gatt.NewServer()
}

func NewClient(conn bt.Conn) (bt.Client, error) {
	return gatt.NewClient(conn)
}
