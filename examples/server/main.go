package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/examples/service"
	"github.com/currantlabs/bt/gap"
	"github.com/currantlabs/bt/gatt"
)

// A mandatory handler for monitoring device state.
func onStateChanged(d dev.Device, s dev.State) {
	fmt.Printf("State: %s\n", s)
	switch s {
	case dev.StatePoweredOn:

	default:
	}
}

func main() {
	s := gatt.NewServer()
	s.AddService(service.NewGapService("Gopher"))
	s.AddService(service.NewGattService())

	// A simple count service for demo.
	s1 := s.AddService(service.NewCountService())

	// A fake battery service for demo.
	s2 := s.AddService(service.NewBatteryService())

	// Crafting the advertising data packet
	ad := adv.Packet(nil).AppendFlags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly)
	ad = ad.AppendAllUUID(s1.UUID).AppendAllUUID(s2.UUID)

	sr := adv.Packet(nil).AppendCompleteName("Gopher")

	d, err := dev.New(-1)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
	}

	p, err := gap.NewPeripheral(d, s)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
	}

	p.Advertise(ad, sr)

	// d.Init(onStateChanged)

	select {}
}
