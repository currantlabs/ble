package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/l2cap"
)

// Peripheral ...
type Peripheral struct {
	Broadcaster
	l2cap.Listener
}

// Init ...
func (p *Peripheral) Init(d dev.Device) error {
	if err := p.Broadcaster.Init(d); err != nil {
		return err
	}

	l, err := l2cap.Listen(d)
	if err != nil {
		return err
	}

	p.Listener = l

	return nil
}
