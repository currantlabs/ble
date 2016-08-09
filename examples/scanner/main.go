package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
	"github.com/currantlabs/ble/linux/hci"
	"github.com/currantlabs/ble/linux/hci/cmd"
)

func handle(a ble.Advertisement) {
	fmt.Printf("[%s]", a.Address())
	comma := ""
	if len(a.LocalName()) > 0 {
		fmt.Printf(" Name: %s", a.LocalName())
		comma = ","
	}
	if len(a.Services()) > 0 {
		fmt.Printf("%s Svcs: %v", comma, a.Services())
		comma = ","
	}
	if len(a.ManufacturerData()) > 0 {
		fmt.Printf("%s MD: %X", comma, a.ManufacturerData())
	}
	fmt.Printf("\n")
}

func main() {
	// Set scan parameters. Only supported on Linux platform.
	d := gatt.DefaultDevice()
	if h, ok := d.(*hci.HCI); ok {
		if err := h.Send(&cmd.LESetScanParameters{
			LEScanType:           0x01,   // 0x00: passive, 0x01: active
			LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
			LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
			OwnAddressType:       0x00,   // 0x00: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
		}, nil); err != nil {
			log.Fatalf("can't set advertising param: %s", err)
		}
	}

	if err := gatt.SetAdvHandler(ble.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}
	if err := gatt.Scan(true); err != nil {
		log.Fatalf("can't scan: %s", err)
	}
	select {}
}
