package main

import (
	"log"
	"os"

	"github.com/currantlabs/x/io/bt"
)

func explorer(a bt.Advertisement, cln bt.Client) {
	l := log.New(os.Stdout, "["+cln.Address().String()+"] ", log.Lmicroseconds)

	l.Printf("RSSI: %d", cln.ReadRSSI())
	for _, s := range cln.Services() {
		l.Printf("Service: %s %s\n", s.UUID.String(), bt.Name(s.UUID))
		for _, c := range s.Characteristics {
			l.Printf("  Characteristic: %s, Property: 0x%02X, %s\n", c.UUID, c.Property, bt.Name(c.UUID))
			if (c.Property & bt.CharRead) != 0 {
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					l.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				l.Printf("    Value         %x | %q\n", b, b)
			}

			for _, d := range c.Descriptors {
				l.Printf("    Descriptor: %s, %s\n", d.UUID, bt.Name(d.UUID))
				b, err := cln.ReadDescriptor(d)
				if err != nil {
					l.Printf("Failed to read descriptor: %s\n", err)
					continue
				}
				l.Printf("    Value         %x | %q\n", b, b)
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
}
