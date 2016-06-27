package main

import (
	"log"

	"github.com/currantlabs/ble/gatt"
)

func main() {
	dev, err := gatt.NewCentral()
	if err != nil {
		log.Fatalf("can't create central: %s", err)
	}

	// Create a centralManager to handle concurrent connections.
	m := newCentralManager(dev)
	// m.HandleClient(ClientHandlerFunc(echo))
	m.HandleClient(ClientHandlerFunc(explorer))
	if err := m.Start(); err != nil {
		log.Fatalf("can't start central manager: %s", err)
	}

	// Wait for 10 seconds before exiting.
	// time.Sleep(time.Second * 10)
	// m.Stop()
	select {}
}
