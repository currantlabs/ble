package main

import (
	"log"

	"github.com/currantlabs/ble/examples/lib"
	"github.com/currantlabs/ble/gatt"
	"github.com/currantlabs/x/io/bt"
)

func main() {
	svr, err := gatt.NewServer()
	if err != nil {
		log.Fatalf("can't create server: %s", err)
	}

	svr.AddService(lib.NewGAPService("Gopher"))
	svr.AddService(lib.NewGATTService())

	testSvc := svr.AddService(bt.NewService(lib.TestSvcUUID))
	testSvc.AddCharacteristic(lib.NewCountChar())
	// testSvc.AddCharacteristic(lib.NewEchoChar())

	// batSvc := svr.AddService(lib.NewBatteryService())

	dev, err := gatt.NewPeripheral()
	if err != nil {
		log.Fatalf("can't create device: %s", err)
	}
	// dev.AdvertiseNameAndServices("Gopher", testSvc.UUID, batSvc.UUID)
	dev.AdvertiseNameAndServices("Gopher", testSvc.UUID)

	// Attach and starts the GATT server to the Peripheral device.
	if err := svr.Start(dev); err != nil {
		log.Fatalf("can't start gatt server: %s", err)
	}

	select {}
}
