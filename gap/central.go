package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/l2cap"
)

// Central ...
type Central struct {
	Observer
	l2cap.LE
}

// Init ...
func (c *Central) Init(d dev.Device) error {
	if err := c.Observer.Init(d); err != nil {
		return err
	}

	if err := c.LE.Init(d); err != nil {
		return err
	}

	return nil
}
