package gap

import (
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/l2cap"
)

// Peripheral ...
type Peripheral interface {
	Broadcaster
	l2cap.Listener
}

// NewPeripheral ...
func NewPeripheral(h hci.HCI) Peripheral {
	p := &peripheral{
		Broadcaster: NewBroadcaster(h),
		Listener:    l2cap.Listen(h),
	}
	return p
}

type peripheral struct {
	Broadcaster
	l2cap.Listener
}
