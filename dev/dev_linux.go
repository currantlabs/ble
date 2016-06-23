package dev

import (
	"log"

	"github.com/currantlabs/x/io/bt"
	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci"
)

// NewPeripheral ...
func NewPeripheral() bt.Peripheral {
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s", err)
	}
	return dev
}

func NewCentral() bt.Central {
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s", err)
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
	return dev
}

func NewBroadcaster() bt.Broadcaster {
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s", err)
	}
	// // Craft a simple advertising data packet.
	// ad := adv.NewPacket(nil)
	// ad = ad.AppendFlags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly)
	// ad = ad.AppendCompleteName("Gopher")
	//
	// // Set advertising data
	// if err := dev.SetAdvertisement(ad, nil); err != nil {
	// 	log.Fatalf("can't set advertisement: %s", err)
	// }

	// Set advertising parameter
	if err := dev.SetAdvParams(hci.AdvParams{
		AdvertisingIntervalMin:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingIntervalMax:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingType:         0x00,      // 00: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // 0x00: public, 0x01: random
		DirectAddressType:       0x00,      // 0x00: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // 0x07 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}); err != nil {
		log.Fatalf("can't set advertising parameters: %s", err)
	}
	return dev

}

func NewObserver() bt.Observer {
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %s", err)
	}

	// Overwrite default scanning parameters (optional).
	if err := dev.SetScanParams(hci.ScanParams{
		LEScanType:           0x01,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}); err != nil {
		log.Fatalf("can't set scan params: %s", err)
	}

	return dev
}

func NewGATTServer() bt.Server {
	return gatt.NewServer()
}

func NewGATTClient(l2c bt.Conn) bt.Client {
	return gatt.NewClient(l2c)
}
