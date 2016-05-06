package gatt

import (
	"log"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/att"
)

// NewServer ...
func NewServer() *Server {
	return &Server{}
}

// Server ...
type Server struct {
	sync.Mutex

	svcs    []bt.Service
	attrs   *att.Range
	changed bool
}

// AddService ...
func (s *Server) AddService(svc bt.Service) bt.Service {
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
	s.attrs = nil
	return nil
}

// SetServices ...
func (s *Server) SetServices(svcs []bt.Service) error {
	s.Lock()
	defer s.Unlock()
	s.changed = true
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	return nil
}

// Start ...
func (s *Server) Start(p bt.Peripheral) error {
	s.Lock()
	if s.changed {
		s.changed = false
		s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	}
	s.Unlock()
	go func() {
		for {
			l2c, err := p.Accept()
			if err != nil {
				log.Printf("can't accept: %s", err)
				return
			}
			s.Lock()
			if s.changed {
				s.changed = false
				s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
			}
			s.Unlock()
			go att.NewServer(s.attrs, l2c, 1024).Loop()
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
