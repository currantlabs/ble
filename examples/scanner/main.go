package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/gatt"
	"github.com/currantlabs/x/io/bt"
)

func handle(a bt.Advertisement) {
	fmt.Printf("%s (%s): Services: %v [ %x ]\n", a.Address(), a.LocalName(), a.Services(), a.ManufacturerData())
	// aa := a.(*hci.Advertisement)
	// log.Printf("% X ", aa.Data())
}

func main() {
	dev := gatt.NewObserver()
	if err := dev.Scan(true); err != nil {
		log.Fatalf("can't scan: %s", err)
	}

	if err := dev.SetAdvHandler(bt.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}
	select {}
}
