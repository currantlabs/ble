package bled

import (
	"github.com/currantlabs/ble"
	pb "github.com/currantlabs/ble/bled/proto"
)

// RandomAddress is a Random Device Address.
type RandomAddress struct {
	ble.Addr
}

// [Vol 6, Part B, 4.4.2] [Vol 3, Part C, 11]
const (
	evtTypAdvInd        = 0x00 // Connectable undirected advertising (ADV_IND).
	evtTypAdvDirectInd  = 0x01 // Connectable directed advertising (ADV_DIRECT_IND).
	evtTypAdvScanInd    = 0x02 // Scannable undirected advertising (ADV_SCAN_IND).
	evtTypAdvNonconnInd = 0x03 // Non connectable undirected advertising (ADV_NONCONN_IND).
	evtTypScanRsp       = 0x04 // Scan Response (SCAN_RSP).
)

// Advertisement implements ble.Advertisement and other functions that are only
// available on Linux.
type Advertisement struct {
	*pb.Advertisement
}

// LocalName returns the LocalName of the remote peripheral.
func (a *Advertisement) LocalName() string {
	return a.Advertisement.LocalName
}

// ManufacturerData returns the ManufacturerData of the advertisement.
func (a *Advertisement) ManufacturerData() []byte {
	return a.Advertisement.ManufacturerData
}

// ServiceData returns the service data of the advertisement.
func (a *Advertisement) ServiceData() []ble.ServiceData {
	// return a.Advertisement.ServiceData
	return nil
}

// Services returns the service UUIDs of the advertisement.
func (a *Advertisement) Services() []ble.UUID {
	var uu []ble.UUID
	for _, u := range a.Advertisement.Services {
		uu = append(uu, ble.UUID(u.UUID))
	}
	return uu
}

// OverflowService returns the UUIDs of overflowed service.
func (a *Advertisement) OverflowService() []ble.UUID {
	// return a.Advertisement.OverflowService
	return nil
}

// TxPowerLevel returns the tx power level of the remote peripheral.
func (a *Advertisement) TxPowerLevel() int {
	return int(a.Advertisement.TxPowerLevel)
}

// SolicitedService returns UUIDs of solicited services.
func (a *Advertisement) SolicitedService() []ble.UUID {
	// return a.Advertisement.GetSolicitedService()
	return nil
}

// Connectable indicates weather the remote peripheral is connectable.
func (a *Advertisement) Connectable() bool {
	return true
}

// RSSI returns RSSI signal strength.
func (a *Advertisement) RSSI() int {
	return int(a.Advertisement.RSSI)
}

type addr string

func (a addr) String() string { return string(a) }

// Address returns the address of the remote peripheral.
func (a *Advertisement) Address() ble.Addr {
	return addr(a.Advertisement.Address)
}

// EventType returns the event type of Advertisement.
// This is linux sepcific.
func (a *Advertisement) EventType() uint8 {
	return 0
}

// AddressType returns the address type of the Advertisement.
// This is linux sepcific.
func (a *Advertisement) AddressType() uint8 {
	return 0
}

// Data returns the advertising data of the packet.
// This is linux sepcific.
// func (a *Advertisement) Data() []byte {
// 	return nil
// }

// ScanResponse returns the scan response of the packet, if it presents.
// This is linux sepcific.
func (a *Advertisement) ScanResponse() []byte {
	return nil
}
