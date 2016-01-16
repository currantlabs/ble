package main

import (
	"log"

	"github.com/currantlabs/bluetooth/hci"
	"github.com/currantlabs/bluetooth/hci/cmd"
)

func main() {
	h, err := hci.NewHCI(-1, false)
	if err != nil {
		log.Fatalf("failed to create HCI. %s", err)
	}

	rp := cmd.LESetAdvertisingParametersRP{}
	c := cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0,
		AdvertisingIntervalMax:  0,
		AdvertisingType:         0,
		OwnAddressType:          0,
		DirectAddressType:       0,
		DirectAddress:           [6]byte{},
		AdvertisingChannelMap:   0,
		AdvertisingFilterPolicy: 0,
	}
	if err := h.Send(&c, &rp); err != nil {
		log.Printf("failed to send command. err %s", err)
	}
	log.Printf("%#v", rp)
}
