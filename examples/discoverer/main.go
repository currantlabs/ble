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
	fmt.Printf("filtered ...\n")
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
		log.Fatalf("Failed to open HCI device, err: %s\n", err)
	}

	h.SetAdvHandler(bt.AdvFilterFunc(filter), bt.AdvHandlerFunc(discovered))
	h.Scan()

	select {}
}
