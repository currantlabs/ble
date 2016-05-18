package darwin

import (
	"fmt"
	"sync"

	"github.com/currantlabs/bt"
)

type request struct {
	conn   bt.Conn
	data   []byte
	offset int
}

func (r *request) Conn() bt.Conn { return r.conn }
func (r *request) Data() []byte  { return r.data }
func (r *request) Offset() int   { return r.offset }

// NewServer ...
func NewServer() *Server {
	return &Server{}
}

// Server ...
type Server struct {
	sync.Mutex

	svcs []*bt.Service
	dev  *Device
}

// AddService ...
func (s *Server) AddService(svc *bt.Service) *bt.Service {
	s.Lock()
	defer s.Unlock()
	s.svcs = append(s.svcs, svc)
	return svc
}

// RemoveAllServices ...
func (s *Server) RemoveAllServices() error {
	s.Lock()
	defer s.Unlock()
	s.svcs = nil
	return nil
}

// SetServices ...
func (s *Server) SetServices(svcs []*bt.Service) error {
	s.Lock()
	defer s.Unlock()
	s.RemoveAllServices()
	s.svcs = append(s.svcs, svcs...)
	return nil
}

// Start ...
func (s *Server) Start(p bt.Peripheral) error {
	d, ok := p.(*Device)
	if !ok {
		return fmt.Errorf("can't convert peripheral to os x device")
	}
	return d.SetServices(s.svcs)
}

// Stop ...
func (s *Server) Stop() error {
	s.Lock()
	defer s.Unlock()
	return nil
}
