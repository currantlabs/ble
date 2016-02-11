package gatt

// This file includes constants from the BLE spec.

const (
	attMTUDefault = 23
	attMTUMax     = 512
)

var (
	attrGAPUUID  = UUID16(0x1800)
	attrGATTUUID = UUID16(0x1801)

	attrPrimaryServiceUUID   = UUID16(0x2800)
	attrSecondaryServiceUUID = UUID16(0x2801)
	attrIncludeUUID          = UUID16(0x2802)
	attrCharacteristicUUID   = UUID16(0x2803)

	attrClientCharacteristicConfigUUID = UUID16(0x2902)
	attrServerCharacteristicConfigUUID = UUID16(0x2903)

	attrDeviceNameUUID        = UUID16(0x2A00)
	attrAppearanceUUID        = UUID16(0x2A01)
	attrPeripheralPrivacyUUID = UUID16(0x2A02)
	attrReconnectionAddrUUID  = UUID16(0x2A03)
	attrPeferredParamsUUID    = UUID16(0x2A04)
	attrServiceChangedUUID    = UUID16(0x2A05)
)

const (
	gattCCCNotifyFlag   = 0x0001
	gattCCCIndicateFlag = 0x0002
)
