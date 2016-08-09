package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/ble/examples/lib/gatt"
	"github.com/currantlabs/ble/linux/hci"
	"github.com/currantlabs/ble/linux/hci/cmd"
)

func main() {

	// Set advertising parameters. Only supported on Linux platform.
	d := gatt.DefaultDevice()
	if h, ok := d.(*hci.HCI); ok {
		if err := h.Send(&cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
			AdvertisingIntervalMax:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
			AdvertisingType:         0x00,      // 00: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
			OwnAddressType:          0x00,      // 0x00: public, 0x01: random
			DirectAddressType:       0x00,      // 0x00: public, 0x01: random
			DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
			AdvertisingChannelMap:   0x7,       // 0x07 0x01: ch37, 0x2: ch38, 0x4: ch39
			AdvertisingFilterPolicy: 0x00,
		}, nil); err != nil {
			log.Fatalf("can't set advertising param: %s", err)
		}
	}

	if err := gatt.AdvertiseNameAndServices("Hello"); err != nil {
		log.Fatalf("can't advertise: %s", err)
	}

	fmt.Printf("Advertising...\n")
	select {} // Prevent program from exiting
}
