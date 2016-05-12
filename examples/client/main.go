package main

import (
	"log"
	"time"

	"github.com/currantlabs/bt/hci"
)

func main() {
	// hci.HCI implements bt.Central.
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s\n", err)
	}

	// Overwrite default connection paramteters (optional).
	dev.SetConnParams(hci.ConnParams{
		LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		InitiatorFilterPolicy: 0x00,      // White list is not used
		PeerAddressType:       0x00,      // Public Device Address
		PeerAddress:           [6]byte{}, //
		OwnAddressType:        0x00,      // Public Device Address
		ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
		SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
		MinimumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
		MaximumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
	})

	// Create a centralManager to handle concurrent connections.
	m := newCentralManager(dev)
	m.HandleClient(ClientHandlerFunc(echo))
	m.HandleClient(ClientHandlerFunc(explorer))
	if err := m.Start(); err != nil {
		log.Fatalf("can't start central manager: %s", err)
	}

	// Wait for 10 seconds before exiting.
	time.Sleep(time.Second * 10)
	m.Stop()
}
