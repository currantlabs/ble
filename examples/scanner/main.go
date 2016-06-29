package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

func handle(a ble.Advertisement) {
	fmt.Printf("[%s]", a.Address())
	comma := ""
	if len(a.LocalName()) > 0 {
		fmt.Printf(" Name: %s", a.LocalName())
		comma = ","
	}
	if len(a.Services()) > 0 {
		fmt.Printf("%s Svcs: %v", comma, a.Services())
		comma = ","
	}
	if len(a.ManufacturerData()) > 0 {
		fmt.Printf("%s MD: %X", comma, a.ManufacturerData())
	}
	fmt.Printf("\n")
}

func main() {
	if err := gatt.SetAdvHandler(ble.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}
	if err := gatt.Scan(true); err != nil {
		log.Fatalf("can't scan: %s", err)
	}
	select {}
}
