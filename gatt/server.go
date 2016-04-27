package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/gap"
)

// NewServer ...
func NewServer() Server {
	return &server{}
}

type server struct {
	svcs  []*Service
	attrs *att.Range
}

func (s *server) AddService(svc *Service) *Service {
	s.svcs = append(s.svcs, svc)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return svc
}

func (s *server) RemoveAllServices() error {
	s.svcs = nil
	s.attrs = nil
	return nil
}

func (s *server) SetServices(svcs []*Service) error {
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	s.attrs = genAttr(s.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

func (s *server) Start(p *gap.Peripheral) {
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
