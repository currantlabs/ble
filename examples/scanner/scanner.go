package main

import (
	"fmt"
	"log"
	"time"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/cmd"
	"github.com/currantlabs/bt/evt"
)

func main() {
	h, err := bt.NewHCI(-1, false)
	if err != nil {
		log.Printf("filed to new bt: %s", err)
	}

	h.Send(&cmd.LESetScanEnable{
		LEScanEnable: 1,
	}, nil)

	fmt.Printf("Start scanning for 1 second ...\n")
	time.Sleep(1 * time.Second)

	// Register our own advertising report handler.
	h.SetSubeventHandler(
		evt.LEAdvertisingReportEvent{}.SubCode(),
		bt.HandlerFunc(handleLEAdvertisingReport))

	fmt.Printf("\nStart scanning for another second with customized advertising report handler ...\n")
	time.Sleep(1 * time.Second)

	h.Send(&cmd.LESetScanEnable{
		LEScanEnable: 0,
	}, nil)
	fmt.Printf("Stopped\n")

	// h.Close()

}

func handleLEAdvertisingReport(b []byte) {
	e := &evt.LEAdvertisingReportEvent{}
	if err := e.Unmarshal(b); err != nil {
		return
	}
	f := func(a [6]byte) string {
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", a[5], a[4], a[3], a[2], a[1], a[0])
	}
	for i := 0; i < int(e.NumReports); i++ {
		fmt.Printf("EventType %d, AddressType %d, Address %s, RSSI %d, Data [% X]\n",
			e.EventType[i], e.AddressType[i], f(e.Address[i]), e.RSSI[i], e.Data[i])
	}
}
