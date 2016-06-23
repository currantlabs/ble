package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/gatt"
	"github.com/currantlabs/x/io/bt"
)

func handle(a bt.Advertisement) {
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
	dev := gatt.NewObserver()
	if err := dev.SetAdvHandler(bt.AdvHandlerFunc(handle)); err != nil {
		log.Fatalf("can't set adv handler: %s", err)
	}
	if err := dev.Scan(true); err != nil {
		log.Fatalf("can't scan: %s", err)
	}
	select {}
}
