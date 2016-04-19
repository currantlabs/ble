package gap

import (
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/l2cap"
)

// Central ...
type Central interface {
	Observer
	l2cap.Dialer
}

// NewCentral ...
func NewCentral(d dev.Device) (Central, error) {
	o, err := NewObserver(d)
	if err != nil {
		return nil, err
	}

	dl, err := l2cap.Dial(d)
	if err != nil {
		return nil, err
	}

	c := &central{
		Observer: o,
		Dialer:   dl,
	}

	return c, nil
}

type central struct {
	Observer
	l2cap.Dialer

	connParam *cmd.LECreateConnection
}
