package main

import (
	"fmt"

	"github.com/currantlabs/ble"
)

func advHandler(a ble.Advertisement) {
	curr.addr = a.Address()
	if a.Connectable() {
		fmt.Printf("[%s] C %3d:", a.Address(), a.RSSI())
	} else {
		fmt.Printf("[%s] N %3d:", a.Address(), a.RSSI())
	}
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

// ServiceData() []ServiceData
// OverflowService() []UUID
// TxPowerLevel() int
// SolicitedService() []UUID
