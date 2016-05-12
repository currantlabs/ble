package main

import (
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/examples/lib"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/hci"
)

func main() {
	svr := gatt.NewServer()
	svr.AddService(lib.NewGapService("Gopher"))
	svr.AddService(lib.NewGattService())

	testSvc := svr.AddService(gatt.NewService(lib.TestSvcUUID))
	testSvc.AddCharacteristic(lib.NewCountChar())
	testSvc.AddCharacteristic(lib.NewEchoChar())

	batSvc := svr.AddService(lib.NewBatteryService())

	// Crafting advertising data and response data packets.
	ad := adv.Packet(nil).AppendFlags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly)
	ad = ad.AppendAllUUID(testSvc.UUID()).AppendAllUUID(batSvc.UUID())
	sr := adv.Packet(nil).AppendCompleteName("Gopher")

	// hci.HCI implements bt.Peripheral.
	dev := new(hci.HCI)
	if err := dev.Init(-1); err != nil {
		log.Fatalf("can't open HCI device: %svr\n", err)
	}

	if err := dev.SetAdvertisement(ad, sr); err != nil {
		log.Fatalf("can't set advertisement: %s", err)
	}

	// Attach and starts the GATT server to the Peripheral device.
	if err := svr.Start(dev); err != nil {
		log.Fatalf("can't start gatt server: %s", err)
	}

	select {}
}
