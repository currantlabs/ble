package ble

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"golang.org/x/net/context"
)

// ErrDefaultDevice ...
var ErrDefaultDevice = errors.New("default device is not set")

var defaultDevice Device

// SetDefaultDevice returns the default HCI device.
func SetDefaultDevice(d Device) {
	defaultDevice = d
}

// AddService adds a service to database.
func AddService(svc *Service) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	return defaultDevice.AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func RemoveAllServices() error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	return defaultDevice.RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func SetServices(svcs []*Service) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	return defaultDevice.SetServices(svcs)
}

// Stop detatch the GATT server from a peripheral device.
func Stop() error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	return nil
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possi
// If name doesn't fit in the advertising packet, it will be put in scan response.
func AdvertiseNameAndServices(ctx context.Context, name string, uuids ...UUID) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	defer untrap(trap(ctx))
	return defaultDevice.AdvertiseNameAndServices(ctx, name, uuids...)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func AdvertiseIBeaconData(ctx context.Context, b []byte) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	defer untrap(trap(ctx))
	return defaultDevice.AdvertiseIBeaconData(ctx, b)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func AdvertiseIBeacon(ctx context.Context, u UUID, major, minor uint16, pwr int8) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	defer untrap(trap(ctx))
	return defaultDevice.AdvertiseIBeacon(ctx, u, major, minor, pwr)
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func Scan(ctx context.Context, allowDup bool, h AdvHandler, f AdvFilter) error {
	if defaultDevice == nil {
		return ErrDefaultDevice
	}
	defer untrap(trap(ctx))

	if f == nil {
		return defaultDevice.Scan(ctx, allowDup, h)
	}

	h2 := func(a Advertisement) {
		if f(a) {
			h(a)
		}
	}
	return defaultDevice.Scan(ctx, allowDup, h2)
}

// Find ...
func Find(ctx context.Context, allowDup bool, f AdvFilter) ([]Advertisement, error) {
	if defaultDevice == nil {
		return nil, ErrDefaultDevice
	}
	var advs []Advertisement
	h := func(a Advertisement) {
		advs = append(advs, a)
	}
	defer untrap(trap(ctx))
	return advs, Scan(ctx, allowDup, h, f)
}

// Dial ...
func Dial(ctx context.Context, a Addr) (Client, error) {
	if defaultDevice == nil {
		return nil, ErrDefaultDevice
	}
	defer untrap(trap(ctx))
	return defaultDevice.Dial(ctx, a)
}

// Connect searches for and connects to a Peripheral which matches specified condition.
func Connect(ctx context.Context, f AdvFilter) (Client, error) {
	ctx2, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-ctx2.Done():
		}
	}()

	ch := make(chan Advertisement)
	fn := func(a Advertisement) {
		cancel()
		ch <- a
	}
	if err := Scan(ctx2, false, fn, f); err != nil {
		if err != context.Canceled {
			return nil, errors.Wrap(err, "can't scan")
		}
	}

	cln, err := Dial(ctx, (<-ch).Address())
	return cln, errors.Wrap(err, "can't dial")
}

// A Client is a GATT client.
type Client interface {
	// Address returns platform specific unique ID of the remote peripheral, e.g. MAC on Linux, Client UUID on OS X.
	Address() Addr

	// Name returns the name of the remote peripheral.
	// This can be the advertised name, if exists, or the GAP device name, which takes priority.
	Name() string

	// Profile returns discovered profile.
	Profile() *Profile

	// DiscoverProfile discovers the whole hierachy of a server.
	DiscoverProfile(force bool) (*Profile, error)
	// DiscoverServices finds all the primary services on a server. [Vol 3, Part G, 4.4.1]
	// If filter is specified, only filtered services are returned.
	DiscoverServices(filter []UUID) ([]*Service, error)

	// DiscoverIncludedServices finds the included services of a service. [Vol 3, Part G, 4.5.1]
	// If filter is specified, only filtered services are returned.
	DiscoverIncludedServices(filter []UUID, s *Service) ([]*Service, error)

	// DiscoverCharacteristics finds all the characteristics within a service. [Vol 3, Part G, 4.6.1]
	// If filter is specified, only filtered characteristics are returned.
	DiscoverCharacteristics(filter []UUID, s *Service) ([]*Characteristic, error)

	// DiscoverDescriptors finds all the descriptors within a characteristic. [Vol 3, Part G, 4.7.1]
	// If filter is specified, only filtered descriptors are returned.
	DiscoverDescriptors(filter []UUID, c *Characteristic) ([]*Descriptor, error)

	// ReadCharacteristic reads a characteristic value from a server. [Vol 3, Part G, 4.8.1]
	ReadCharacteristic(c *Characteristic) ([]byte, error)

	// ReadLongCharacteristic reads a characteristic value which is longer than the MTU. [Vol 3, Part G, 4.8.3]
	ReadLongCharacteristic(c *Characteristic) ([]byte, error)

	// WriteCharacteristic writes a characteristic value to a server. [Vol 3, Part G, 4.9.3]
	WriteCharacteristic(c *Characteristic, value []byte, noRsp bool) error

	// ReadDescriptor reads a characteristic descriptor from a server. [Vol 3, Part G, 4.12.1]
	ReadDescriptor(d *Descriptor) ([]byte, error)

	// WriteDescriptor writes a characteristic descriptor to a server. [Vol 3, Part G, 4.12.3]
	WriteDescriptor(d *Descriptor, v []byte) error

	// ReadRSSI retrieves the current RSSI value of remote peripheral. [Vol 2, Part E, 7.5.4]
	ReadRSSI() int

	// ExchangeMTU set the ATT_MTU to the maximum possible value that can be supported by both devices [Vol 3, Part G, 4.3.1]
	ExchangeMTU(rxMTU int) (txMTU int, err error)

	// Subscribe subscribes to indication (if ind is set true), or notification of a characteristic value. [Vol 3, Part G, 4.10 & 4.11]
	Subscribe(c *Characteristic, ind bool, h NotificationHandler) error

	// Unsubscribe unsubscribes to indication (if ind is set true), or notification of a specified characteristic value. [Vol 3, Part G, 4.10 & 4.11]
	Unsubscribe(c *Characteristic, ind bool) error

	// ClearSubscriptions clears all subscriptions to notifications and indications.
	ClearSubscriptions() error

	// CancelConnection disconnects the connection.
	CancelConnection() error
}

// A NotificationHandler handles notification or indication from a server.
type NotificationHandler func(req []byte)

// WithSigHandler ...
func WithSigHandler(ctx context.Context, cancel func()) context.Context {
	return context.WithValue(ctx, "sig", cancel)
}

// Cleanup for the interrupted case.
func trap(ctx context.Context) chan<- os.Signal {
	v := ctx.Value("sig")
	if v == nil {
		return nil
	}
	cancel, ok := v.(func())
	if cancel == nil || !ok {
		return nil
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
	}()
	return sigs
}

func untrap(sigs chan<- os.Signal) {
	if sigs == nil {
		return
	}
	signal.Stop(sigs)
}
