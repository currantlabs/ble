package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bt/dev"
)

func main() {
	dev := dev.NewBroadcaster()
	if err := dev.AdvertiseNameAndServices("Hello"); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	fmt.Printf("Advertising...\n")
	select {} // Prevent program from exiting
}
