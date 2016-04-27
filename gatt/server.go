package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/gap"
)

// Server ...
type Server struct {
	svcs  []*Service
	attrs *att.Range
}

// AddService add a service to database.
func (s *Server) AddService(svc *Service) *Service {
	s.svcs = append(s.svcs, svc)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return svc
}

// RemoveAllServices removes all services that are currently in the database.
func (s *Server) RemoveAllServices() error {
	s.svcs = nil
	s.attrs = nil
	return nil
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func (s *Server) SetServices(svcs []*Service) error {
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

// Init ...
func (s *Server) Init(p *gap.Peripheral) {
	go func() {
		for {
			l2c, err := p.Accept()
			if err != nil {
				break
			}
			att.NewServer(s.attrs, l2c, 1024).Loop()
			p.StartAdvertising()
		}
	}()
}
