package gap

import "github.com/currantlabs/bt/hci"

// Broadcaster ...
type Broadcaster interface {
	Advertise(a Advertisement) error
	StopAdvertising() error
}

// NewBroadcaster ...
func NewBroadcaster(h hci.HCI) Broadcaster {
	b := &bcast{}
	return b
}

type bcast struct {
	h hci.HCI
}

func (b *bcast) Advertise(a Advertisement) error {
	return nil
}

func (b *bcast) StopAdvertising() error {
	return nil
}
