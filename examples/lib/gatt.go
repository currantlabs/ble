package lib

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/uuid"
)

var (
	attrGATTUUID           = uuid.UUID16(0x1801)
	attrServiceChangedUUID = uuid.UUID16(0x2A05)
)

// NewGattService ...
func NewGattService() bt.Service {
	s := gatt.NewService(attrGATTUUID)
	// s.AddCharacteristic(attrServiceChangedUUID).HandleNotify(
	// 	gatt.NotifyHandlerFunc(func(r gatt.Request, n *gatt.Notifier) {
	// 		go func() {
	// 			log.Printf("TODO: indicate client when the services are changed")
	// 		}()
	// 	}))
	return s
}
