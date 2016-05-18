package lib

import "github.com/currantlabs/bt"

// NewGATTService ...
func NewGATTService() *bt.Service {
	s := bt.NewService(bt.GATTUUID)
	// s.AddCharacteristic(bt.ServiceChangedUUID).HandleNotify(
	// 	gatt.NotifyHandlerFunc(func(r gatt.Request, n *gatt.Notifier) {
	// 		go func() {
	// 			log.Printf("TODO: indicate client when the services are changed")
	// 		}()
	// 	}))
	return s
}
