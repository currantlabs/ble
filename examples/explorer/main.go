package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

var (
	name = flag.String("name", "Gopher", "name of remote peripheral")
	addr = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	sub  = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
)

// matcher returns true if the advertisement matches our search criteria.
type matcher func(a ble.Advertisement) bool

// explorer connects to a remote peripheral and explores its GATT server.
type explorerer struct {
	match matcher
	ch    chan ble.Advertisement
}

func (e *explorerer) Handle(a ble.Advertisement) {
	if e.match(a) {
		gatt.StopScanning()
		e.ch <- a
	}
}

func explorer(cln ble.Client) error {
	fmt.Printf("Exploring Peripheral [ %s ] ...\n", cln.Address())

	ss, err := cln.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("can't discover services: %s\n", err)
	}
	for _, s := range ss {
		fmt.Printf("Service: %s %s, Handle (0x%02X)\n", s.UUID.String(), ble.Name(s.UUID), s.Handle)

		cs, err := cln.DiscoverCharacteristics(nil, s)
		if err != nil {
			return fmt.Errorf("can't discover characteristics: %s\n", err)
		}
		for _, c := range cs {
			fmt.Printf("  Characteristic: %s, Property: 0x%02X (%s), %s, Handle(0x%02X), VHandle(0x%02X)\n",
				c.UUID, c.Property, propString(c.Property), ble.Name(c.UUID), c.Handle, c.ValueHandle)
			if (c.Property & ble.CharRead) != 0 {
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					fmt.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}

			ds, err := cln.DiscoverDescriptors(nil, c)
			if err != nil {
				return fmt.Errorf("can't discover descriptors: %s\n", err)
			}
			for _, d := range ds {
				fmt.Printf("    Descriptor: %s, %s, Handle(0x%02x)\n", d.UUID, ble.Name(d.UUID), d.Handle)
				b, err := cln.ReadDescriptor(d)
				if err != nil {
					fmt.Printf("Failed to read descriptor: %s\n", err)
					continue
				}
				fmt.Printf("    Value         %x | %q\n", b, b)
			}
			if *sub != 0 {
				// Don't bother to subscribe the Service Changed characteristics.
				if c.UUID.Equal(ble.ServiceChangedUUID) {
					continue
				}

				// Don't touch the Apple-specific Service/Characteristic.
				// Service: D0611E78BBB44591A5F8487910AE4366
				// Characteristic: 8667556C9A374C9184ED54EE27D90049, Property: 0x18 (WN),
				//   Descriptor: 2902, Client Characteristic Configuration
				//   Value         0000 | "\x00\x00"
				if c.UUID.Equal(ble.MustParse("8667556C9A374C9184ED54EE27D90049")) {
					continue
				}

				if (c.Property & ble.CharNotify) != 0 {
					fmt.Printf("\n-- Subscribe to notification for %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Notified: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, false, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, false); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to notification --\n")
				}
				if (c.Property & ble.CharIndicate) != 0 {
					fmt.Printf("\n-- Subscribe to indication of %s --\n", *sub)
					h := func(req []byte) { fmt.Printf("Indicated: %q [ % X ]\n", string(req), req) }
					if err := cln.Subscribe(c, true, h); err != nil {
						log.Fatalf("subscribe failed: %s", err)
					}
					time.Sleep(*sub)
					if err := cln.Unsubscribe(c, true); err != nil {
						log.Fatalf("unsubscribe failed: %s", err)
					}
					fmt.Printf("-- Unsubscribe to indication --\n")
				}
			}

		}
		fmt.Printf("\n")
	}
	return nil
}

func propString(p ble.Property) string {
	var s string
	for k, v := range map[ble.Property]string{
		ble.CharBroadcast:   "B",
		ble.CharRead:        "R",
		ble.CharWriteNR:     "w",
		ble.CharWrite:       "W",
		ble.CharNotify:      "N",
		ble.CharIndicate:    "I",
		ble.CharSignedWrite: "S",
		ble.CharExtended:    "E",
	} {
		if p&k != 0 {
			s += v
		}
	}
	return s
}

func main() {
	flag.Parse()

	// Default to search device with name of Gopher (or specified by user).
	match := func(a ble.Advertisement) bool {
		return strings.ToUpper(a.LocalName()) == strings.ToUpper(*name)
	}

	// If addr is specified, search for addr instead.
	if len(*addr) != 0 {
		match = func(a ble.Advertisement) bool {
			return strings.ToUpper(a.Address().String()) == strings.ToUpper(*addr)
		}
	}

	exp := &explorerer{
		ch:    make(chan ble.Advertisement),
		match: match,
	}

	if err := gatt.SetAdvHandler(exp); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}

	if err := gatt.Scan(false); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	// Wait for the exploration is done.
	a := <-exp.ch

	// Dial connects to the remote device.
	cln, err := gatt.Dial(a.Address())
	if err != nil {
		log.Fatalf("can't dial: %s", err)
	}

	// Start the exploration.
	explorer(cln)

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Address())
	cln.CancelConnection()
}
