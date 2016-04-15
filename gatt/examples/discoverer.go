// +build

package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

func onStateChanged(d *gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]uuid.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p *gatt.Peripheral, a *adv.Packet, rssi int) {
	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Printf("\npkt: %X %v, %s", a, a.UUIDs(), a.LocalName())
}

func main() {
	d, err := gatt.NewDevice(-1)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.PeripheralDiscovered = onPeriphDiscovered
	d.Init(onStateChanged)
	select {}
}
