package gap

import (
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/l2cap"
)

// Mode ...
type Mode int

// Mode ...
const (
	NonDiscoverable     Mode = iota // [Vol 3, Part C, 9.2.2]
	LimitedDiscoverable             // [Vol 3, Part C, 9.2.3]
	GeneralDiscoverable             // [Vol 3, Part C, 9.2.4]
)

// Peripheral ...
type Peripheral interface {
	Broadcaster
	l2cap.Listener
}

// NewPeripheral ...
func NewPeripheral(h hci.HCI, s *gatt.Server) (Peripheral, error) {
	b, err := NewBroadcaster(h)
	if err != nil {
		return nil, err
	}
	l, err := l2cap.Listen(h)
	if err != nil {
		return nil, err
	}
	p := &peripheral{
		Broadcaster: b,
		Listener:    l,
		s:           s,
	}
	go p.loop()
	return p, nil
}

type peripheral struct {
	Broadcaster
	l2cap.Listener
	s *gatt.Server
}

func (p *peripheral) loop() {
	for {
		c, _ := p.Accept()
		p.s.Loop(c)
		p.StartAdvertising()
	}
}
