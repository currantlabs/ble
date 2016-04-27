package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/l2cap"
)

// Peripheral ...
type Peripheral struct {
	Broadcaster
	l2cap.LE
}

// Init ...
func (p *Peripheral) Init(d dev.Device) error {
	if err := p.Broadcaster.Init(d); err != nil {
		return err
	}

	if err := p.LE.Init(d); err != nil {
		return err
	}

	return nil
}
