package linux

import (
	"log"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux/att"
	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci"
)

// NewDevice returns the default HCI device.
func NewDevice() (*Device, error) {
	dev, err := hci.NewHCI()
	if err != nil {
		return nil, errors.Wrap(err, "can't create hci")
	}
	if err = dev.Init(); err != nil {
		return nil, errors.Wrap(err, "can't init hci")
	}

	s, err := gatt.NewServer()
	if err != nil {
		return nil, errors.Wrap(err, "can't create server")
	}

	mtu := ble.DefaultMTU
	mtu = ble.MaxMTU // TODO: get this from user using Option.
	if mtu > ble.MaxMTU {
		return nil, errors.Wrapf(err, "maximum ATT_MTU is %d", ble.MaxMTU)
	}

	go func() {
		for {
			l2c, err := dev.Accept()
			if err != nil {
				log.Printf("can't accept: %s", err)
				return
			}

			// Initialize the per-connection cccd values.
			l2c.SetContext(context.WithValue(l2c.Context(), "ccc", make(map[uint16]uint16)))
			l2c.SetRxMTU(mtu)

			s.Lock()
			as, err := att.NewServer(s.DB(), l2c)
			s.Unlock()
			if err != nil {
				log.Printf("can't create ATT server: %s", err)
				continue

			}
			go as.Loop()
		}
	}()
	return &Device{HCI: dev, Server: s}, nil
}

// Device ...
type Device struct {
	HCI    *hci.HCI
	Server *gatt.Server
}

// AddService adds a service to database.
func (d *Device) AddService(svc *ble.Service) error {
	return d.Server.AddService(svc)
}

// RemoveAllServices removes all services that are currently in the database.
func (d *Device) RemoveAllServices() error {
	return d.Server.RemoveAllServices()
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func (d *Device) SetServices(svcs []*ble.Service) error {
	return d.Server.SetServices(svcs)
}

// Stop stops gatt server.
func (d *Device) Stop() error {
	return d.HCI.Close()
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func (d *Device) AdvertiseNameAndServices(ctx context.Context, name string, uuids ...ble.UUID) error {
	if err := d.HCI.AdvertiseNameAndServices(name, uuids...); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopAdvertising()
	return ctx.Err()
}

// AdvertiseMfgData avertises the given manufacturer data.
func (d *Device) AdvertiseMfgData(ctx context.Context, id uint16, b []byte) error {
	if err := d.HCI.AdvertiseMfgData(id, b); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopAdvertising()
	return ctx.Err()
}

// AdvertiseServiceData16 advertises data associated with a 16bit service uuid
func (d *Device) AdvertiseServiceData16(ctx context.Context, id uint16, b []byte) error {
	if err := d.HCI.AdvertiseServiceData16(id, b); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopAdvertising()
	return ctx.Err()
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func (d *Device) AdvertiseIBeaconData(ctx context.Context, b []byte) error {
	if err := d.HCI.AdvertiseIBeaconData(b); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopAdvertising()
	return ctx.Err()
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func (d *Device) AdvertiseIBeacon(ctx context.Context, u ble.UUID, major, minor uint16, pwr int8) error {
	if err := d.HCI.AdvertiseIBeacon(u, major, minor, pwr); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopAdvertising()
	return ctx.Err()
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func (d *Device) Scan(ctx context.Context, allowDup bool, h ble.AdvHandler) error {
	if err := d.HCI.SetAdvHandler(h); err != nil {
		return err
	}
	if err := d.HCI.Scan(allowDup); err != nil {
		return err
	}
	<-ctx.Done()
	d.HCI.StopScanning()
	return ctx.Err()
}

// Dial ...
func (d *Device) Dial(ctx context.Context, a ble.Addr) (ble.Client, error) {
	// d.HCI.Dial is a blocking call, although most of time it should return immediately.
	// But in case passing wrong device address or the device went non-connectable, it blocks.
	cln, err := d.HCI.Dial(ctx, a)
	return cln, errors.Wrap(err, "can't dial")
}

// Address returns the listener's device address.
func (d *Device) Address() ble.Addr {
	return d.HCI.Addr()
}
