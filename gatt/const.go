package gatt

import "github.com/currantlabs/bt/uuid"

var (
	attrGAPUUID  = uuid.UUID16(0x1800)
	attrGATTUUID = uuid.UUID16(0x1801)

	attrPrimaryServiceUUID   = uuid.UUID16(0x2800)
	attrSecondaryServiceUUID = uuid.UUID16(0x2801)
	attrIncludeUUID          = uuid.UUID16(0x2802)
	attrCharacteristicUUID   = uuid.UUID16(0x2803)

	attrClientCharacteristicConfigUUID = uuid.UUID16(0x2902)
	attrServerCharacteristicConfigUUID = uuid.UUID16(0x2903)

	attrDeviceNameUUID        = uuid.UUID16(0x2A00)
	attrAppearanceUUID        = uuid.UUID16(0x2A01)
	attrPeripheralPrivacyUUID = uuid.UUID16(0x2A02)
	attrReconnectionAddrUUID  = uuid.UUID16(0x2A03)
	attrPeferredParamsUUID    = uuid.UUID16(0x2A04)
	attrServiceChangedUUID    = uuid.UUID16(0x2A05)
)

const (
	flagCCCNotify   = 0x0001
	flagCCCIndicate = 0x0002
)
