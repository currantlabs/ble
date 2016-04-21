package service

import (
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

// NewBatteryService ...
func NewBatteryService() *gatt.Service {
	lv := byte(100)
	s := gatt.NewService(uuid.UUID16(0x180F))
	c := s.AddCharacteristic(uuid.UUID16(0x2A19))
	c.HandleRead(
		gatt.ReadHandlerFunc(func(req gatt.Request, rsp gatt.ResponseWriter) {
			rsp.Write([]byte{lv})
			lv--
		}))

	// Characteristic User Description
	c.AddDescriptor(uuid.UUID16(0x2901)).SetValue([]byte("Battery level between 0 and 100 percent"))

	// Characteristic Presentation Format
	c.AddDescriptor(uuid.UUID16(0x2904)).SetValue([]byte{4, 1, 39, 173, 1, 0, 0})

	return s
}
