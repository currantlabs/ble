package lib

import "github.com/currantlabs/bt"

// https://developer.bluetooth.org/gatt/characteristics/Pages/CharacteristicViewer.aspx?u=org.bluetooth.characteristic.bt.appearance.xml
var gapCharAppearanceGenericComputer = []byte{0x00, 0x80}

// NewGAPService ...
func NewGAPService(name string) *bt.Service {
	s := bt.NewService(bt.GAPUUID)
	s.NewCharacteristic(bt.DeviceNameUUID).SetValue([]byte(name))
	s.NewCharacteristic(bt.AppearanceUUID).SetValue(gapCharAppearanceGenericComputer)
	s.NewCharacteristic(bt.PeripheralPrivacyUUID).SetValue([]byte{0x00})
	s.NewCharacteristic(bt.ReconnectionAddrUUID).SetValue([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	s.NewCharacteristic(bt.PeferredParamsUUID).SetValue([]byte{0x06, 0x00, 0x06, 0x00, 0x00, 0x00, 0xd0, 0x07})
	return s
}
