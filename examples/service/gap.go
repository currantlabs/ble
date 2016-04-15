package service

import (
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

var (
	attrGAPUUID = uuid.UUID16(0x1800)

	attrDeviceNameUUID        = uuid.UUID16(0x2A00)
	attrAppearanceUUID        = uuid.UUID16(0x2A01)
	attrPeripheralPrivacyUUID = uuid.UUID16(0x2A02)
	attrReconnectionAddrUUID  = uuid.UUID16(0x2A03)
	attrPeferredParamsUUID    = uuid.UUID16(0x2A04)
)

// https://developer.bluetooth.org/gatt/characteristics/Pages/CharacteristicViewer.aspx?u=org.bluetooth.characteristic.gap.appearance.xml
var gapCharAppearanceGenericComputer = []byte{0x00, 0x80}

// NewGapService ...
func NewGapService(name string) *gatt.Service {
	s := gatt.NewService(attrGAPUUID)
	s.AddCharacteristic(attrDeviceNameUUID).SetValue([]byte(name))
	s.AddCharacteristic(attrAppearanceUUID).SetValue(gapCharAppearanceGenericComputer)
	s.AddCharacteristic(attrPeripheralPrivacyUUID).SetValue([]byte{0x00})
	s.AddCharacteristic(attrReconnectionAddrUUID).SetValue([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	s.AddCharacteristic(attrPeferredParamsUUID).SetValue([]byte{0x06, 0x00, 0x06, 0x00, 0x00, 0x00, 0xd0, 0x07})
	return s
}
