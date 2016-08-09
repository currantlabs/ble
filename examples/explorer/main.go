package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
	"github.com/currantlabs/ble/linux/hci"
	"github.com/currantlabs/ble/linux/hci/cmd"
)

var (
	name = flag.String("name", "Gopher", "name of remote peripheral")
	addr = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	sub  = flag.Duration("sub", 0, "subscribe to notification and indication for a specified period")
)

func explorer(cln ble.Client) error {
	fmt.Printf("Exploring Peripheral [ %s ] ...\n", cln.Address())

	p, err := cln.DiscoverProfile(true)
	if err != nil {
		return fmt.Errorf("can't discover services: %s\n", err)
	}
	for _, s := range p.Services {
		fmt.Printf("Service: %s %s, Handle (0x%02X)\n", s.UUID.String(), ble.Name(s.UUID), s.Handle)

		for _, c := range s.Characteristics {
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

			for _, d := range c.Descriptors {
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

	// Set connection parameters. Only supported on Linux platform.
	d := gatt.DefaultDevice()
	if h, ok := d.(*hci.HCI); ok {
		if err := h.Option(hci.OptConnParam(
			cmd.LECreateConnection{
				LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
				LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
				InitiatorFilterPolicy: 0x00,      // White list is not used
				PeerAddressType:       0x00,      // Public Device Address
				PeerAddress:           [6]byte{}, //
				OwnAddressType:        0x00,      // Public Device Address
				ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
				ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
				ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
				SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
				MinimumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
				MaximumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
			})); err != nil {
			log.Fatalf("can't set advertising param: %s", err)
		}
	}

	cln, err := gatt.Discover(gatt.MatcherFunc(match))
	if err != nil {
		log.Fatalf("can't discover: %s", err)
	}

	// Start the exploration.
	explorer(cln)

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Address())
	cln.CancelConnection()
}
