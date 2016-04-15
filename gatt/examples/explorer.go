// +build

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

var done = make(chan struct{})

func onStateChanged(d *gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("Scanning...")
		d.Scan([]uuid.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p *gatt.Peripheral, a *adv.Packet, rssi int) {
	id := strings.ToUpper(flag.Args()[0])
	if strings.ToUpper(p.ID()) != id {
		return
	}

	// Stop scanning once we've got the peripheral we're looking for.
	p.Device().StopScanning()

	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Printf("  Local Name        = %s\n", a.LocalName())
	fmt.Printf("  Manufacturer Data = %X\n", a.ManufacturerData())
	fmt.Printf("  UUID              = %v\n", a.UUIDs())
	fmt.Printf("  RAW               = %X\n", a.Bytes())
	fmt.Println("")

	p.Device().Connect(p)
}

func onPeriphConnected(p *gatt.Peripheral, err error) {
	fmt.Println("Connected")
	defer p.Device().CancelConnection(p)

	if err := p.SetMTU(512); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}

	for _, s := range ss {
		msg := "Service: " + s.UUID.String()
		if len(s.Name()) > 0 {
			msg += " (" + s.Name() + ")"
		}
		fmt.Println(msg)

		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
		if err != nil {
			fmt.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			fmt.Printf("  Characteristic: %s, Property: 0x%02X, %s\n", c.UUID, c.Property, c.Name())

			// Read the characteristic, if possible.
			if (c.Property & gatt.CharRead) != 0 {
				b, err := p.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic, err: %s\n", err)
					continue
				}
				fmt.Printf("    value         %x | %q\n", b, b)
			}

			// Discovery descriptors
			ds, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				fmt.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			for _, d := range ds {
				fmt.Printf("    Descriptor: %s, %s\n", d.UUID, d.Name())
				// Read descriptor (could fail, if it's not readable)
				b, err := p.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor, err: %s\n", err)
					continue
				}
				fmt.Printf("    value         %x | %q\n", b, b)
			}

			// Subscribe the characteristic, if possible.
			if (c.Property & gatt.CharNotify) != 0 {
				f := func(b []byte) {
					fmt.Printf("Notified: % X | %q\n", b, b)
				}
				if err := p.SetNotificationHandler(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}
			if (c.Property & gatt.CharIndicate) != 0 {
				f := func(b []byte) {
					fmt.Printf("Indicated: % X | %q\n", b, b)
				}
				if err := p.SetIndicationHandler(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}
		fmt.Println()
	}

	fmt.Printf("Waiting for 2 seconds to get some notifiations, if any.\n")
	time.Sleep(2 * time.Second)
	p.ClearHandlers()
}

func onPeriphDisconnected(p *gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
	close(done)
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("usage: %s [options] peripheral-id\n", os.Args[0])
	}

	d, err := gatt.NewDevice(-1)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.PeripheralDiscovered = onPeriphDiscovered
	d.PeripheralConnected = onPeriphConnected
	d.PeripheralDisconnected = onPeriphDisconnected

	d.Init(onStateChanged)
	<-done
	fmt.Println("Done")
}
