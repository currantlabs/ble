package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/gatt"
)

func main() {
	dev := gatt.NewBroadcaster()
	if err := dev.AdvertiseNameAndServices("Hello"); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	fmt.Printf("Advertising...\n")
	select {} // Prevent program from exiting
}
