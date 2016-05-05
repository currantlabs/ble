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

func discovered(a bt.Advertisement) {
	// Show event level info, and raw data.
	fmt.Printf("%s: EvtType %d, AddrType %d, RSSI %d, Data [%X]\n",
		a.Address(), a.EventType(), a.AddressType(), a.RSSI(), a.Data())

	// Decode the raw data
	p := adv.Packet(a.Data())
	fmt.Printf("Name: %s, UUIDs: %v\n\n", p.LocalName(), p.UUIDs())
}

func main() {
	h := &hci.HCI{}
	if err := h.Init(-1); err != nil {
		log.Fatalf("can't open HCI device, err: %s\n", err)
	}

	if err := h.SetAdvHandler(bt.AdvFilterFunc(filter), bt.AdvHandlerFunc(discovered)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}

	if err := h.SetScanParams(hci.ScanParams{
		LEScanType:           0x01,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}); err != nil {
		log.Fatalf("can't set scan params: %s", err)
	}

	if err := h.Scan(); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	select {}
}
