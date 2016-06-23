package darwin

import (
	"github.com/currantlabs/ble/darwin/xpc"
	"github.com/currantlabs/x/io/bt"
)

// Advertisement ...
type Advertisement struct {
	args xpc.Dict
	ad   xpc.Dict
}

// LocalName returns the LocalName of the remote peripheral.
func (a *Advertisement) LocalName() string {
	return a.ad.GetString("kCBAdvDataLocalName", a.args.GetString("kCBMsgArgName", ""))
}

// ManufacturerData returns the ManufacturerData of the advertisement.
func (a *Advertisement) ManufacturerData() []byte {
	return a.ad.GetBytes("kCBAdvDataManufacturerData", nil)
}

// ServiceData returns the service data of the advertisement.
func (a *Advertisement) ServiceData() []bt.ServiceData {
	xSDs, ok := a.ad["kCBAdvDataServiceData"]
	if !ok {
		return nil
	}

	xSD := xSDs.(xpc.Array)
	var sd []bt.ServiceData
	for i := 0; i < len(xSD); i += 2 {
		sd = append(
			sd, bt.ServiceData{
				UUID: bt.UUID(xSD[i].([]byte)),
				Data: xSD[i+1].([]byte),
			})
	}
	return sd
}

// Services returns the service UUIDs of the advertisement.
func (a *Advertisement) Services() []bt.UUID {
	xUUIDs, ok := a.ad["kCBAdvDataServiceUUIDs"]
	if !ok {
		return nil
	}
	var uuids []bt.UUID
	for _, xUUID := range xUUIDs.(xpc.Array) {
		uuids = append(uuids, bt.UUID(bt.Reverse(xUUID.([]byte))))
	}
	return uuids
}

// OverflowService returns the UUIDs of overflowed service.
func (a *Advertisement) OverflowService() []bt.UUID {
	return nil // TODO
}

// TxPowerLevel returns the tx power level of the remote peripheral.
func (a *Advertisement) TxPowerLevel() int {
	return a.ad.GetInt("kCBAdvDataTxPowerLevel", 0)
}

// SolicitedService returns UUIDs of solicited services.
func (a *Advertisement) SolicitedService() []bt.UUID {
	return nil // TODO
}

// Connectable indicates weather the remote peripheral is connectable.
func (a *Advertisement) Connectable() bool {
	return a.ad.GetInt("kCBAdvDataIsConnectable", 0) > 0
}

// RSSI returns RSSI signal strength.
func (a *Advertisement) RSSI() int {
	return a.args.GetInt("kCBMsgArgRssi", 0)
}

// Address returns the address of the remote peripheral.
func (a *Advertisement) Address() bt.Addr {
	return xpc.UUID(a.args.MustGetUUID("kCBMsgArgDeviceUUID"))
}
