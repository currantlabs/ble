package main

import (
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/examples/service"
	"github.com/currantlabs/bt/gap"
	"github.com/currantlabs/bt/gatt"
)

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
	ad = ad.AppendAllUUID(s1.UUID()).AppendAllUUID(s2.UUID())

	sr := adv.Packet(nil).AppendCompleteName("Gopher")

	d, err := dev.New(-1)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
	}

	p := &gap.Peripheral{}
	if err := p.Init(d); err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
	}
	s.Start(p)
	p.Advertise(ad, sr)

	select {}
}
