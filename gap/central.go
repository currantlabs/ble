package gap

import (
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/l2cap"
)

// Central ...
type Central interface {
	Observer
	l2cap.Dialer
}

// NewCentral ...
func NewCentral(h hci.HCI) (Central, error) {
	o, err := NewObserver(h)
	if err != nil {
		return nil, err
	}

	d, err := l2cap.Dial(h)
	if err != nil {
		return nil, err
	}

	c := &central{
		Observer: o,
		Dialer:   d,
	}

	return c, nil
}

type central struct {
	Observer
	l2cap.Dialer

	connParam *cmd.LECreateConnection
}
