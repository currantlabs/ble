package gatt

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/linux/att"
)

// NewServer ...
func NewServer() *Server {
	return &Server{}
}

// Server ...
type Server struct {
	sync.Mutex

	svcs    []*bt.Service
	db      *att.DB
	changed bool
}

// AddService ...
func (s *Server) AddService(svc *bt.Service) *bt.Service {
	s.Lock()
	defer s.Unlock()
	s.changed = true
	s.svcs = append(s.svcs, svc)
	return svc
}

// RemoveAllServices ...
func (s *Server) RemoveAllServices() error {
	s.Lock()
	defer s.Unlock()
	s.changed = true
	s.svcs = nil
	s.db = nil
	return nil
}

// SetServices ...
func (s *Server) SetServices(svcs []*bt.Service) error {
	s.Lock()
	defer s.Unlock()
	s.changed = true
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	return nil
}

// Start ...
func (s *Server) Start(p bt.Peripheral) error {
	mtu := bt.DefaultMTU
	mtu = bt.MaxMTU // TODO: get this from user using Option.
	if mtu > bt.MaxMTU {
		return fmt.Errorf("maximum ATT_MTU is %d", bt.MaxMTU)
	}
	s.Lock()
	if s.changed {
		s.db = att.NewDB(s.svcs, uint16(1)) // ble attrs start at 1
		s.changed = false
	}
	s.Unlock()
	go func() {
		for {
			p.StopAdvertising()
			l2c, err := p.Accept()
			if err != nil {
				log.Printf("can't accept: %s", err)
				return
			}

			// Initialize the per-connection cccd values.
			l2c.SetContext(context.WithValue(l2c.Context(), "ccc", make(map[uint16]uint16)))
			l2c.SetRxMTU(mtu)

			// Re-generate attributes if the services has been changed
			s.Lock()
			if s.changed {
				s.changed = false
				s.db = att.NewDB(s.svcs, uint16(1)) // ble attrs start at 1
			}
			s.Unlock()

			as, err := att.NewServer(s.db, l2c)
			if err != nil {
				log.Printf("can't create ATT server: %s", err)
				continue

			}
			go as.Loop()
		}
	}()
	return nil
}

// Stop ...
func (s *Server) Stop() error {
	s.Lock()
	defer s.Unlock()
	return nil
}
