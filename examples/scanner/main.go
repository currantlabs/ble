package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/dev"
	"github.com/currantlabs/x/io/bt"
)

// func handle(a bt.Advertisement) {
// 	// Show event info, and raw data.
// 	fmt.Printf("%s: EvtType %d, AddrType %d, RSSI %d, Data [%X]\n",
// 		a.Address(), a.EventType(), a.AddressType(), a.RSSI(), a.Data())
//
// 	// Decode the raw data
// 	p := adv.Packet(a.Data())
// 	fmt.Printf("Name: %s, UUIDs: %v\n\n", p.LocalName(), p.UUIDs())
// }

func handle(a bt.Advertisement) {
	fmt.Printf("%s (%s): Services: %v [ %x ]\n", a.Address(), a.LocalName(), a.Services(), a.ManufacturerData())
	// aa := a.(*hci.Advertisement)
	// log.Printf("% X ", aa.Data())
}

func main() {
	dev := dev.NewObserver()
	if err := dev.Scan(true); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	if err := dev.SetAdvHandler(bt.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}
	select {}
}
