package gap

import (
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

// Observer ...
type Observer interface {
	Scan(f AdvertisementFilter, h AdvertisementHandler) error
	StopScanning() error
}

// NewObserver ...
func NewObserver(h hci.HCI) Observer {
	o := &observer{
		hci: h,
	}

	// Register our own advertising report handler.
	o.hci.SetSubeventHandler(evt.LEAdvertisingReportSubCode, hci.HandlerFunc(o.adHandler))

	return o
}

type observer struct {
	hci hci.HCI
	f   AdvertisementFilter
	h   AdvertisementHandler

	scanParam *cmd.LESetScanParameters
}

// Scan ...
func (o *observer) Scan(f AdvertisementFilter, h AdvertisementHandler) error {
	o.f, o.h = f, h
	o.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
	o.hci.Send(o.scanParam, nil)
	o.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
	return nil
}

// StopScanning stops scanning.
func (o *observer) StopScanning() error {
	o.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
	return nil
}

func (o *observer) adHandler(b []byte) error {
	return nil
}
