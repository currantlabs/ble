package main

import (
	"fmt"
	"log"
	"time"

	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

func main() {
	h, err := hci.New(-1)
	if err != nil {
		log.Fatalf("Filed to open HCI device: %s", err)
	}

	du := time.Duration(3 * time.Second)

	h.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
	fmt.Printf("Start scanning for %s ...\n", du)
	time.Sleep(du)
	h.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)

	fmt.Printf("\nRegister our customized advertising report handler ...\n")
	h.SetSubeventHandler(
		evt.LEAdvertisingReportSubCode,
		hci.HandlerFunc(handleLEAdvertisingReport))

	fmt.Printf("Start scanning for %s ...\n", du)
	h.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
	time.Sleep(du)
	h.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)

	fmt.Printf("Stopped.\n")
}

func handleLEAdvertisingReport(b []byte) error {
	e := evt.LEAdvertisingReport(b)
	f := func(a [6]byte) string {
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", a[5], a[4], a[3], a[2], a[1], a[0])
	}
	for i := 0; i < int(e.NumReports()); i++ {
		fmt.Printf("%s: EvtType %d, AddrType %d, RSSI %d, Data [%X]\n",
			f(e.Address(i)), e.EventType(i), e.AddressType(i), e.RSSI(i), e.Data(i))
	}
	return nil
}
