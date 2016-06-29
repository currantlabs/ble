package main

import (
	"log"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

func main() {
	testSvc := ble.NewService(lib.TestSvcUUID)
	testSvc.AddCharacteristic(lib.NewCountChar())
	testSvc.AddCharacteristic(lib.NewEchoChar())
	if err := gatt.AddService(testSvc); err != nil {
		log.Fatalf("can't add service: %s", err)
	}

	if err := gatt.AdvertiseNameAndServices("Gopher", testSvc.UUID); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	select {}
}
