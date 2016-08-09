package gatt

import (
	"log"

	"github.com/currantlabs/ble/darwin"

	"github.com/currantlabs/ble"
)

// DefaultDevice returns the default device.
func DefaultDevice() Device {
	return nil
}

type manager struct {
	central    *darwin.Device
	peripheral *darwin.Device
}

var m manager

func central() *darwin.Device {
	if m.central == nil {
		m.central = newDev(darwin.OptCentralRole())
	}
	return m.central
}

func peripheral() *darwin.Device {
	if m.peripheral == nil {
		m.peripheral = newDev(darwin.OptPeripheralRole())
	}
	return m.peripheral
}

func newDev(opts ...darwin.Option) *darwin.Device {
	dev, err := darwin.NewDevice(opts...)
	if err != nil {
		log.Fatalf("create device failed: %s", err)
	}
	if err := dev.Init(); err != nil {
		log.Fatalf("init device failed: %s", err)
	}
	return dev
}

// AddService adds a service to database.
func AddService(svc *ble.Service) error {
	return peripheral().AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func RemoveAllServices() error {
	return peripheral().RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func SetServices(svcs []*ble.Service) error {
	return peripheral().SetServices(svcs)
}

// Stop detatch the GATT peripheral from a peripheral device.
func Stop() error {
	return peripheral().Stop()
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func AdvertiseNameAndServices(name string, uuids ...ble.UUID) error {
	return peripheral().AdvertiseNameAndServices(name, uuids...)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func AdvertiseIBeaconData(b []byte) error {
	return peripheral().AdvertiseIBeaconData(b)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
	return peripheral().AdvertiseIBeacon(u, major, minor, pwr)
}

// StopAdvertising stops advertising.
func StopAdvertising() error {
	return peripheral().StopAdvertising()
}

// SetAdvHandler sets filter, handler.
func SetAdvHandler(h ble.AdvHandler) error {
	return central().SetAdvHandler(h)
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func Scan(allowDup bool) error {
	return central().Scan(allowDup)
}

// StopScanning stops scanning.
func StopScanning() error {
	return central().StopScanning()
}

// Addr returns the listener's device address.
func Addr() ble.Addr {
	return central().Addr()
}

// Dial ...
func Dial(a ble.Addr) (ble.Client, error) {
	return central().Dial(a)
}
