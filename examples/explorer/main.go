package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/uuid"
)

type explorer struct {
	bt.Central
	found chan bool
	done  chan bool
}

func newExplorer(c bt.Central) *explorer {
	return &explorer{
		Central: c,
		found:   make(chan bool),
		done:    make(chan bool),
	}
}

func (e *explorer) Handle(a bt.Advertisement) {
	select {
	case <-e.found:
		return
	default:
		close(e.found)
	}
	e.StopScanning()
	p := adv.Packet(a.Data())
	fmt.Printf("\n%s: RSSI: %3d, Name: %s, UUIDs: %v, MD: %X\n",
		a.Address(), a.RSSI(), p.LocalName(), p.UUIDs(), p.ManufacturerData())
	go e.explore(a.Address())
}

func (e *explorer) explore(a net.HardwareAddr) {
	// FIXME: rework the API/WRAPPER
	l2c, _ := e.Dial(a)
	p := gatt.NewClient(l2c)
	if err := p.SetMTU(512); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}
	defer close(e.done)
	defer l2c.Close()
	defer p.ClearHandlers()

	// Discovery services
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}

	for _, s := range ss {
		fmt.Printf("Service: %s %s\n", s.UUID(), uuid.Name(s.UUID()))

		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
		if err != nil {
			fmt.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			fmt.Printf("  Characteristic: %s, Properties: 0x%02X, %s\n", c.UUID(), c.Properties(), uuid.Name(c.UUID()))

			// Read the characteristic, if possible.
			if (c.Properties() & bt.CharRead) != 0 {
				b, err := p.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic, err: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}

			// Discovery descriptors
			ds, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				fmt.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			for _, d := range ds {
				fmt.Printf("    Descriptor: %s, %s\n", d.UUID(), uuid.Name(d.UUID()))
				// Read descriptor (could fail, if it's not readable)
				b, err := p.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor, err: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}

			// Subscribe the characteristic, if possible.
			// Note: This can only be done after the descriptors (CCCD) are discovered.
			if (c.Properties() & bt.CharNotify) != 0 {
				f := func(b []byte) { fmt.Printf("Notified: % X | %q\n", b, b) }
				if err := p.SetNotificationHandler(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}
			if (c.Properties() & bt.CharIndicate) != 0 {
				f := func(b []byte) { fmt.Printf("Indicated: % X | %q\n", b, b) }
				if err := p.SetIndicationHandler(c, f); err != nil {
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

	flt := func(a bt.Advertisement) bool { return adv.Packet(a.Data()).LocalName() == "Gopher" }
	if flag.NArg() == 1 {
		flt = func(a bt.Advertisement) bool { return strings.EqualFold(a.Address().String(), flag.Args()[0]) }
	}

	h := &hci.HCI{}
	if err := h.Init(-1); err != nil {
		log.Fatalf("Failed to open HCI device, err: %s\n", err)
	}

	e := newExplorer(h)
	e.SetAdvHandler(bt.AdvFilterFunc(flt), e)
	e.Scan()
	e.Wait()
}
