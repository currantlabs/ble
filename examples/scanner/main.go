package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/hci"
)

func filter(a bt.Advertisement) bool {
	p := adv.Packet(a.Data())
	if p.LocalName() == "Gopher" {
		return true
	}
	fmt.Printf("%s: filtered ...\n", a.Address())
	return false
}

func handle(a bt.Advertisement) {
	// Show event info, and raw data.
	fmt.Printf("%s: EvtType %d, AddrType %d, RSSI %d, Data [%X]\n",
		a.Address(), a.EventType(), a.AddressType(), a.RSSI(), a.Data())

	// Decode the raw data
	p := adv.Packet(a.Data())
	fmt.Printf("Name: %s, UUIDs: %v\n\n", p.LocalName(), p.UUIDs())
}

func main() {
	// hci.HCI implements bt.Scanner.
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s\n", err)
	}

	// Overwrite default scanning parameters (optional).
	if err := dev.SetScanParams(hci.ScanParams{
		LEScanType:           0x01,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}); err != nil {
		log.Fatalf("can't set scan params: %s", err)
	}

	if err := dev.SetAdvHandler(bt.AdvFilterFunc(filter), bt.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}

	if err := dev.Scan(false); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	select {}
}
