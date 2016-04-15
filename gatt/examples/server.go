// +build

package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/gatt/examples/service"
	"github.com/currantlabs/bt/uuid"
)

func main() {
	d, err := gatt.NewDevice(-1)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s", err)
	}

	// Register optional handlers.
	d.CentralConnected = func(c *gatt.Central) { fmt.Println("Connect: ", c.ID()) }
	d.CentralDisconnected = func(c *gatt.Central) { fmt.Println("Disconnect: ", c.ID()) }

	// A mandatory handler for monitoring device state.
	onStateChanged := func(d *gatt.Device, s gatt.State) {
		fmt.Printf("State: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:
			d.AddService(service.NewGapService("Gopher"))
			d.AddService(service.NewGattService())

			// A simple count service for demo.
			s1 := d.AddService(service.NewCountService())

			// A fake battery service for demo.
			s2 := d.AddService(service.NewBatteryService())

			// Advertise device name and service's UUIDs.
			d.AdvertiseNameAndServices("Gopher", []uuid.UUID{s1.UUID, s2.UUID})

			// Advertise as an OpenBeacon iBeacon
			//d.AdvertiseIBeacon(uuid.MustParse("AA6062F098CA42118EC4193EB73CCEB6"), 1, 2, -59)
		default:
		}
	}

	d.Init(onStateChanged)
	select {}
}
