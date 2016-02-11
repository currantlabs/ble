package service

import (
	"log"

	"github.com/currantlabs/bt/gatt"
)

var (
	attrGATTUUID           = gatt.UUID16(0x1801)
	attrServiceChangedUUID = gatt.UUID16(0x2A05)
)

func NewGattService() *gatt.Service {
	s := gatt.NewService(attrGATTUUID)
	s.AddCharacteristic(attrServiceChangedUUID).HandleNotify(
		gatt.NotifyHandlerFunc(func(r gatt.Request, n gatt.Notifier) {
			go func() {
				log.Printf("TODO: indicate client when the services are changed")
			}()
		}))
	return s
}
