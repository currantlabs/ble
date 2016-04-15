package gap

import (
	"net"

	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

// Observer ...
type Observer interface {
	Scan(f Filter, h Handler) error
	StopScanning() error
}

// Filter ...
type Filter interface {
	Filter(a Advertisement) bool
}

// FilterFunc ...
type FilterFunc func(a Advertisement) bool

// Filter ...
func (f FilterFunc) Filter(a Advertisement) bool {
	return f(a)
}

// Handler ...
type Handler interface {
	Handle(a Advertisement)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(a Advertisement)

// Handle handles an advertisement.
func (f HandlerFunc) Handle(a Advertisement) {
	f(a)
}

// Advertisement ...
type Advertisement struct {
	e evt.LEAdvertisingReport
	i int
}

// EventType ...
func (a Advertisement) EventType() uint8 { return a.e.EventType(a.i) }

// AddressType ...
func (a Advertisement) AddressType() uint8 { return a.e.AddressType(a.i) }

// RSSI ...
func (a Advertisement) RSSI() int8 { return a.e.RSSI(a.i) }

// Address ...
func (a Advertisement) Address() net.HardwareAddr {
	b := a.e.Address(a.i)
	return []byte{b[5], b[4], b[3], b[2], b[1], b[0]}
}

// Data ...
func (a Advertisement) Data() []byte { return a.e.Data(a.i) }

// NewObserver ...
func NewObserver(h hci.HCI) (Observer, error) {
	o := &observer{
		hci: h,

		scanOn:  cmd.LESetScanEnable{LEScanEnable: 1},
		scanOff: cmd.LESetScanEnable{LEScanEnable: 0},
		scanParam: cmd.LESetScanParameters{
			LEScanType:           0x01,   // [0x00]: passive, 0x01: active
			LEScanInterval:       0x0010, // [0x10]: 0.625ms * 16
			LEScanWindow:         0x0010, // [0x10]: 0.625ms * 16
			OwnAddressType:       0x00,   // [0x00]: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // [0x00]: accept all, 0x01: ignore non-white-listed.
		},
	}

	// Register our own advertising report handler.
	o.hci.SetSubeventHandler(evt.LEAdvertisingReportSubCode, hci.HandlerFunc(o.advHandler))

	return o, nil
}

type observer struct {
	hci     hci.HCI
	filter  Filter
	handler Handler

	scanOn    cmd.LESetScanEnable
	scanOff   cmd.LESetScanEnable
	scanParam cmd.LESetScanParameters
}

// Scan ...
func (o *observer) Scan(f Filter, h Handler) error {
	o.filter, o.handler = f, h
	o.hci.Send(&o.scanOff, nil)
	o.hci.Send(&o.scanParam, nil)
	o.hci.Send(&o.scanOn, nil)
	return nil
}

// StopScanning stops scanning.
func (o *observer) StopScanning() error {
	o.hci.Send(&o.scanOff, nil)
	return nil
}

func (o *observer) advHandler(b []byte) error {
	if o.handler == nil {
		return nil
	}
	e := evt.LEAdvertisingReport(b)
	for i := 0; i < int(e.NumReports()); i++ {
		a := Advertisement{e: e, i: i}
		if o.filter != nil && o.filter.Filter(a) {
			go o.handler.Handle(a)
		}
	}
	return nil
}
