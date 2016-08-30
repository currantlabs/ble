package bled

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/currantlabs/ble"
	pb "github.com/currantlabs/ble/bled/proto"
)

const (
	address       = "localhost:50051"
	defaultSocket = "/tmp/bled.sock"
)

// Device ...
type Device struct {
	bled pb.BledClient

	advHandler ble.AdvHandler

	cancelScan func()
}

// NewDevice ...
func NewDevice() (*Device, error) {
	// Set up a connection to the server.
	dialer := func(a string, t time.Duration) (net.Conn, error) {
		return net.Dial("unix", a)
	}
	// conn, err := grpc.Dial(address, grpc.WithInsecure())
	conn, err := grpc.Dial(defaultSocket, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		return nil, errors.Wrap(err, "can't connect")
	}
	c := pb.NewBledClient(conn)
	d := &Device{
		bled: c,
	}
	return d, nil
}

// AddService adds a service to database.
func (d *Device) AddService(svc *ble.Service) error {
	return nil
}

// RemoveAllServices removes all services that are currently in the database.
func (d *Device) RemoveAllServices() error {
	return nil
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func (d *Device) SetServices(svcs []*ble.Service) error {
	return nil
}

// Stop stops gatt server.
func (d *Device) Stop() error {
	// return d.Server.Stop()
	return nil
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
// If name doesn't fit in the advertising packet, it will be put in scan response.
func (d *Device) AdvertiseNameAndServices(ctx context.Context, name string, uuids ...ble.UUID) error {
	return nil
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func (d *Device) AdvertiseIBeaconData(ctx context.Context, b []byte) error {
	return nil
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func (d *Device) AdvertiseIBeacon(ctx context.Context, u ble.UUID, major, minor uint16, pwr int8) error {
	return nil
}

// Scan starts scanning. Duplicated advertisements will be filtered out if allowDup is set to false.
func (d *Device) Scan(ctx context.Context, allowDup bool, h ble.AdvHandler) error {
	d.advHandler = h
	stream, err := d.bled.Scan(ctx, &pb.Dummy{})
	if err != nil {
		log.Fatalf("%v.Scan(_) = _, %v", d, err)
	}

	for {
		adv, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("%v.Scan(_) = _, %v, %T", d, err, err)
		}

		a := &Advertisement{adv}
		d.advHandler(a)
	}
	return nil
}

// Close closes the listner.
// Any blocked Accept operations will be unblocked and return errors.
func (d *Device) Close() error {
	return nil
}

// Address returns the listener's device address.
func (d *Device) Address() ble.Addr {
	return nil
}

// Dial ...
func (d *Device) Dial(ctx context.Context, a ble.Addr) (ble.Client, error) {
	return nil, nil
}
