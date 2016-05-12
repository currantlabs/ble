package main

import (
	"log"
	"os"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/uuid"
)

func explorer(a bt.Advertisement, cln bt.Client) {
	l := log.New(os.Stdout, "["+cln.Address().String()+"] ", log.Lmicroseconds)

	for _, s := range cln.Services() {
		l.Printf("Service: %s %s\n", s.UUID(), uuid.Name(s.UUID()))
		for _, c := range s.Characteristics() {
			l.Printf("  Characteristic: %s, Properties: 0x%02X, %s\n", c.UUID(), c.Properties(), uuid.Name(c.UUID()))
			if (c.Properties() & bt.CharRead) != 0 {
				b, err := cln.ReadCharacteristic(c)
				if err != nil {
					l.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				l.Printf("    Value         %x | %q\n", b, b)
			}

			for _, d := range c.Descriptors() {
				l.Printf("    Descriptor: %s, %s\n", d.UUID(), uuid.Name(d.UUID()))
				b, err := cln.ReadDescriptor(d)
				if err != nil {
					l.Printf("Failed to read descriptor: %s\n", err)
					continue
				}
				l.Printf("    Value         %x | %q\n", b, b)
			}
		}
		l.Printf("\n")
	}
}
