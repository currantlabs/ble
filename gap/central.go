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
func NewCentral(h hci.HCI) Central {
	c := &central{
		Observer: NewObserver(h),
		Dialer:   l2cap.Dial(h),
	}
	return c
}

type central struct {
	Observer
	l2cap.Dialer

	connParam *cmd.LECreateConnection
}
