package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/gap"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

type explorer struct {
	gap.Central
	found chan bool
	done  chan bool
}

func (e *explorer) Handle(a gap.Advertisement) {
	select {
	case <-e.found:
		return
	default:
		close(e.found)
	}
	e.StopScanning()
	gc := adv.Packet(a.Data())
	fmt.Printf("\n%s: RSSI: %3d, Name: %s, UUIDs: %v, MD: %X\n",
		a.Address(), a.RSSI(), gc.LocalName(), gc.UUIDs(), gc.ManufacturerData())
	go e.explore(a.Address())
}

func (e *explorer) explore(a net.HardwareAddr) {
	l2c, _ := e.Dial(a)
	gc := &gatt.Client{}
	if err := gc.Init(l2c); err != nil {
		log.Fatalf("Failed to initiate gatt client, err: %s", err)
	}

	if err := gc.SetMTU(512); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	defer close(e.done)
	defer l2c.Close()
	defer gc.ClearHandlers()

	// Discovery services
	ss, err := gc.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}

	for _, s := range ss {
		fmt.Printf("Service: %s %s\n", s.UUID(), uuid.Name(s.UUID()))

		// Discovery characteristics
		cs, err := gc.DiscoverCharacteristics(nil, s)
		if err != nil {
			fmt.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			fmt.Printf("  Characteristic: %s, Properties: 0x%02X, %s\n", c.UUID(), c.Properties(), uuid.Name(c.UUID()))

			// Read the characteristic, if possible.
			if (c.Properties() & gatt.CharRead) != 0 {
				b, err := gc.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic, err: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}

			// Discovery descriptors
			ds, err := gc.DiscoverDescriptors(nil, c)
			if err != nil {
				fmt.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			for _, d := range ds {
				fmt.Printf("    Descriptor: %s, %s\n", d.UUID(), uuid.Name(d.UUID()))
				// Read descriptor (could fail, if it's not readable)
				b, err := gc.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor, err: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}

			// Subscribe the characteristic, if possible.
			// Note: This can only be done after the descriptors (CCCD) are discovered.
			if (c.Properties() & gatt.CharNotify) != 0 {
				f := func(b []byte) { fmt.Printf("Notified: % X | %q\n", b, b) }
				if err := gc.SetNotificationHandler(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}
			if (c.Properties() & gatt.CharIndicate) != 0 {
				f := func(b []byte) { fmt.Printf("Indicated: % X | %q\n", b, b) }
				if err := gc.SetIndicationHandler(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}
		}
		fmt.Println()
	}

	du := 3 * time.Second
	fmt.Printf("Waiting for %s to get some notifiations, if any.\n", du)
	time.Sleep(du)
}

func (e *explorer) Wait() {
	<-e.done
}

func main() {
	flag.Parse()

	flt := func(a gap.Advertisement) bool { return adv.Packet(a.Data()).LocalName() == "Gopher" }
	if flag.NArg() == 1 {
		flt = func(a gap.Advertisement) bool { return strings.EqualFold(a.Address().String(), flag.Args()[0]) }
	}

	d, err := dev.New(-1)
	if err != nil {
		log.Fatalf("Failed to open HCI device, err: %s\n", err)
	}

	e := &explorer{
		found: make(chan bool),
		done:  make(chan bool),
	}
	if err := e.Central.Init(d); err != nil {
		log.Fatalf("Failed to create a central, err: %s\n", err)
	}

	e.Scan(gap.AdvFilterFunc(flt), e)
	e.Wait()
}
