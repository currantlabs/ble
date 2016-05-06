package service

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

// NewBatteryService ...
func NewBatteryService() bt.Service {
	lv := byte(100)
	s := gatt.NewService(uuid.UUID16(0x180F))
	c := s.NewCharacteristic(uuid.UUID16(0x2A19))
	c.HandleRead(
		bt.ReadHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
			rsp.Write([]byte{lv})
			lv--
		}))

	// Characteristic User Description
	c.NewDescriptor(uuid.UUID16(0x2901)).SetValue([]byte("Battery level between 0 and 100 percent"))

	// Characteristic Presentation Format
	c.NewDescriptor(uuid.UUID16(0x2904)).SetValue([]byte{4, 1, 39, 173, 1, 0, 0})

	return s
}
