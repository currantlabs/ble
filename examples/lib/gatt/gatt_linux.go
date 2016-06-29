package gatt

import (
	"log"

	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci"
	"github.com/currantlabs/ble"
)

type manager struct {
	server ble.Server
	dev    ble.Device
}

var m manager

func server() ble.Server {
	if m.server == nil {
		s, err := gatt.NewServer(dev())
		if err != nil {
			log.Fatalf("create server failed: %s", err)
		}
		m.server = s
		s.Start()
	}
	return m.server
}

func dev() ble.Device {
	if m.dev != nil {
		return m.dev
	}
	dev, err := hci.NewHCI()
	if err != nil {
		log.Fatalf("create hci failed: %s", err)
	}
	if err = dev.Init(); err != nil {
		log.Fatalf("init hci failed: %s", err)
	}
	m.dev = dev
	return dev
}

// AddService adds a service to database.
func AddService(svc *ble.Service) error {
	return server().AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func RemoveAllServices() error {
	return server().RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func SetServices(svcs []*ble.Service) error {
	return server().SetServices(svcs)
}

// Stop detatch the GATT server from a peripheral device.
func Stop() error {
	return server().Stop()
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func AdvertiseNameAndServices(name string, uuids ...ble.UUID) error {
	return dev().AdvertiseNameAndServices(name, uuids...)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func AdvertiseIBeaconData(b []byte) error {
	return dev().AdvertiseIBeaconData(b)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
	return dev().AdvertiseIBeacon(u, major, minor, pwr)
}

// StopAdvertising stops advertising.
func StopAdvertising() error {
	return dev().StopAdvertising()
}

// SetAdvHandler sets filter, handler.
func SetAdvHandler(h ble.AdvHandler) error {
	return dev().SetAdvHandler(h)
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func Scan(allowDup bool) error {
	return dev().Scan(allowDup)
}

// StopScanning stops scanning.
func StopScanning() error {
	return dev().StopScanning()
}

// Close closes the listner.
// Any blocked Accept operations will be unblocked and return errors.
func Close() error {
	return dev().Close()
}

// Addr returns the listener's device address.
func Addr() ble.Addr {
	return dev().Addr()
}

// Dial ...
func Dial(a ble.Addr) (ble.Client, error) {
	return dev().Dial(a)
}
