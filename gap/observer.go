package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

// Observer ...
type Observer struct {
	dev dev.Device

	filter  AdvFilter
	handler AdvHandler

	scanOn    cmd.LESetScanEnable
	scanOff   cmd.LESetScanEnable
	scanParam cmd.LESetScanParameters
}

// Init ...
func (o *Observer) Init(d dev.Device) error {
	o.dev = d

	o.scanOn = cmd.LESetScanEnable{LEScanEnable: 1}
	o.scanOff = cmd.LESetScanEnable{LEScanEnable: 0}
	o.scanParam = cmd.LESetScanParameters{
		LEScanType:           0x01,   // [0x00]: passive, 0x01: active
		LEScanInterval:       0x0010, // [0x10]: 0.625ms * 16
		LEScanWindow:         0x0010, // [0x10]: 0.625ms * 16
		OwnAddressType:       0x00,   // [0x00]: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // [0x00]: accept all, 0x01: ignore non-white-listed.
	}

	// Register our own advertising report handler.
	o.dev.SetSubeventHandler(evt.LEAdvertisingReportSubCode, hci.HandlerFunc(o.advHandler))

	return nil
}

// Scan ...
func (o *Observer) Scan(f AdvFilter, h AdvHandler) error {
	o.filter, o.handler = f, h

	o.dev.Send(&o.scanOff, nil)
	o.dev.Send(&o.scanParam, nil)
	o.dev.Send(&o.scanOn, nil)
	return nil
}

// StopScanning stops scanning.
func (o *Observer) StopScanning() error {
	o.dev.Send(&o.scanOff, nil)
	return nil
}

func (o *Observer) advHandler(b []byte) error {
	if o.handler == nil {
		return nil
	}
	e := evt.LEAdvertisingReport(b)
	for i := 0; i < int(e.NumReports()); i++ {
		a := Advertisement{e: e, i: i}
		if o.filter != nil && o.filter.AdvFilter(a) {
			go o.handler.Handle(a)
		}
	}
	return nil
}
