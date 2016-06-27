package darwin

import (
	"github.com/currantlabs/ble/darwin/xpc"
	"github.com/currantlabs/x/io/bt"
)

type adv struct {
	args xpc.Dict
	ad   xpc.Dict
}

func (a *adv) LocalName() string {
	return a.ad.GetString("kCBAdvDataLocalName", a.args.GetString("kCBMsgArgName", ""))
}

func (a *adv) ManufacturerData() []byte {
	return a.ad.GetBytes("kCBAdvDataManufacturerData", nil)
}

func (a *adv) ServiceData() []bt.ServiceData {
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

func (a *adv) Services() []bt.UUID {
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

func (a *adv) OverflowService() []bt.UUID {
	return nil // TODO
}

func (a *adv) TxPowerLevel() int {
	return a.ad.GetInt("kCBAdvDataTxPowerLevel", 0)
}

func (a *adv) SolicitedService() []bt.UUID {
	return nil // TODO
}

func (a *adv) Connectable() bool {
	return a.ad.GetInt("kCBAdvDataIsConnectable", 0) > 0
}

func (a *adv) RSSI() int {
	return a.args.GetInt("kCBMsgArgRssi", 0)
}

func (a *adv) Address() bt.Addr {
	return xpc.UUID(a.args.MustGetUUID("kCBMsgArgDeviceUUID"))
}
