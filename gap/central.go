package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/l2cap"
)

// Central ...
type Central struct {
	Observer
	l2cap.Dialer
}

// Init ...
func (c *Central) Init(d dev.Device) error {
	if err := c.Observer.Init(d); err != nil {
		return err
	}

	l, err := l2cap.Dial(d)
	if err != nil {
		return err
	}
	c.Dialer = l

	return nil
}
