package gatt

import (
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/l2cap"
	"golang.org/x/net/context"
)

// Server ...
type Server struct {
	svcs  []*Service
	attrs *att.Range
	as    *att.Server
}

// NewServer ...
func NewServer() *Server {
	s := &Server{}
	return s
}

// AddService add a service to database.
func (s *Server) AddService(svc *Service) *Service {
	s.svcs = append(s.svcs, svc)
	s.attrs = generateAttributes(s.svcs, uint16(1)) // ble attrs start at 1
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
	s.attrs = generateAttributes(s.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

// Loop ...
func (s *Server) Loop(l2c l2cap.Conn) {
	ctx := context.WithValue(context.Background(), keyServer, s)
	as := att.NewServer(ctx, s.attrs, l2c, 1024)
	s.as = as
	as.Loop()
}
