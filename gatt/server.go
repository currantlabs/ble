package gatt

import (
	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/att"
)

// NewServer ...
func NewServer() *Server {
	return &Server{}
}

// Server ...
type Server struct {
	svcs  []bt.Service
	attrs *att.Range
}

// AddService ...
func (s *Server) AddService(svc bt.Service) bt.Service {
	s.svcs = append(s.svcs, svc)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return svc
}

// RemoveAllServices ...
func (s *Server) RemoveAllServices() error {
	s.svcs = nil
	s.attrs = nil
	return nil
}

// SetServices ...
func (s *Server) SetServices(svcs []bt.Service) error {
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

// Start ...
func (s *Server) Start(p bt.Peripheral) error {
	go func() {
		for {
			l2c, err := p.Accept()
			if err != nil {
				break
			}
			att.NewServer(s.attrs, l2c, 1024).Loop()
			p.Advertise()
		}
	}()
	return nil
}

// Stop ...
func (s *Server) Stop() error {
	return nil
}
