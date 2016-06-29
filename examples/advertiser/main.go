package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/examples/lib/gatt"
)

func main() {
	if err := gatt.AdvertiseNameAndServices("Hello"); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	fmt.Printf("Advertising...\n")
	select {} // Prevent program from exiting
}
