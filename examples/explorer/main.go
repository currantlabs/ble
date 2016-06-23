package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/currantlabs/ble/gatt"
	"github.com/currantlabs/x/io/bt"
)

var (
	name = flag.String("name", "Gopher", "name of remote peripheral")
	addr = flag.String("addr", "", "addr (MAC on Linux, UUID on OS X) of remote peripheral")
)

// matcher returns true if the advertisement matches our search criteria.
type matcher func(a bt.Advertisement) bool

// explorer connects to a remote peripheral and explores its GATT server.
type explorerer struct {
	bt.Central

	match matcher
	ch    chan bt.Advertisement
}

func (e *explorerer) Handle(a bt.Advertisement) {
	if e.match(a) {
		e.StopScanning()
		e.ch <- a
	}
}

func explorer(cln bt.Client) error {
	l := log.New(os.Stdout, "["+cln.Address().String()+"] ", log.Lmicroseconds)

	ss, err := cln.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("can't discover services: %s\n", err)
	}
	for _, s := range ss {
		l.Printf("Service: %s %s\n", s.UUID.String(), bt.Name(s.UUID))

		cs, err := cln.DiscoverCharacteristics(nil, s)
		if err != nil {
			return fmt.Errorf("can't discover characteristics: %s\n", err)
		}
		for _, c := range cs {
			l.Printf("  Characteristic: %s, Property: 0x%02X, %s\n", c.UUID, c.Property, bt.Name(c.UUID))
			if (c.Property & bt.CharRead) != 0 {
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					l.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				l.Printf("    Value         %x | %q\n", b, b)
			}

			for _, c := range cs {
				ds, err := cln.DiscoverDescriptors(nil, c)
				if err != nil {
					return fmt.Errorf("can't discover descriptors: %s\n", err)
				}
				for _, d := range ds {
					l.Printf("    Descriptor: %s, %s\n", d.UUID, bt.Name(d.UUID))
					b, err := cln.ReadDescriptor(d)
					if err != nil {
						l.Printf("Failed to read descriptor: %s\n", err)
						continue
					}
					l.Printf("    Value         %x | %q\n", b, b)
				}
			}
			// if (c.Property & bt.CharNotify) != 0 {
			// 	h := func(req []byte) { l.Printf("Notified: %q [ % X ]", string(req), req) }
			// 	cln.Subscribe(c, false, h)
			// 	time.Sleep(3 * time.Second)
			// 	cln.Unsubscribe(c, false)
			// }
		}
		l.Printf("\n")
	}
	return nil
}

func main() {
	flag.Parse()

	// Default to search device with name of Gopher (or specified by user).
	match := func(a bt.Advertisement) bool {
		return strings.ToUpper(a.LocalName()) == strings.ToUpper(*name)
	}

	// If addr is specified, search for addr instead.
	if len(*addr) != 0 {
		match = func(a bt.Advertisement) bool {
			return strings.ToUpper(a.Address().String()) == strings.ToUpper(*addr)
		}
	}

	dev := gatt.NewCentral()
	exp := &explorerer{
		Central: dev,
		ch:      make(chan bt.Advertisement),
		match:   match,
	}

	if err := dev.SetAdvHandler(exp); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}

	if err := dev.Scan(false); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	// Wait for the exploration is done.
	a := <-exp.ch

	// Dial connects to the remote device.
	c, err := exp.Dial(a.Address())
	if err != nil {
		log.Fatalf("can't dial: %s", err)
	}

	// Create and attach a GATT client to the connection.
	cln := gatt.NewClient(c)
	defer cln.CancelConnection()

	if _, err := cln.ExchangeMTU(bt.MaxMTU); err != nil {
		log.Printf("can't set MTU: %s\n", err)
	}

	// Start the exploration.
	explorer(cln)
}
