package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/gap"
	"github.com/currantlabs/bt/hci"
)

func filter(a gap.Advertisement) bool {
	p := adv.Packet(a.Data())
	if p.LocalName() == "Gopher" {
		return true
	}
	fmt.Printf("filtered ...\n")
	return false
}

func discovered(a gap.Advertisement) {
	p := adv.Packet(a.Data())
	t := "AD" // Advertising Data
	if a.EventType()&0x02 == 0x02 {
		t = "SR" // Scan Response
	}
	fmt.Printf("%s (%s): RSSI: %3d, Name: %s, UUIDs: %v\n", a.Address(), t, a.RSSI(), p.LocalName(), p.UUIDs())
}

func main() {
	// Find an available HCI device
	h, err := hci.New(-1)
	if err != nil {
		log.Fatalf("Failed to open HCI device, err: %s\n", err)
	}

	o, err := gap.NewObserver(h)
	if err != nil {
		log.Fatalf("Failed to create an observer, err: %s\n", err)
	}
	o.Scan(gap.FilterFunc(filter), gap.HandlerFunc(discovered))

	select {}
}
