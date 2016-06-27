package darwin

import (
	"fmt"
	"sync"

	"github.com/currantlabs/x/io/bt"
)

// NewServer returns a GATT server.
func NewServer() (*Server, error) {
	return &Server{}, nil
}

// A Server is a GATT server.
type Server struct {
	sync.Mutex

	svcs []*bt.Service
	dev  *Device
}

// AddService adds a service to database.
func (svr *Server) AddService(svc *bt.Service) *bt.Service {
	svr.Lock()
	defer svr.Unlock()
	svr.svcs = append(svr.svcs, svc)
	return svc
}

// RemoveAllServices removes all services that are currently in the database.
func (svr *Server) RemoveAllServices() error {
	svr.Lock()
	defer svr.Unlock()
	svr.svcs = nil
	return nil
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func (svr *Server) SetServices(svcs []*bt.Service) error {
	svr.Lock()
	defer svr.Unlock()
	svr.RemoveAllServices()
	svr.svcs = append(svr.svcs, svcs...)
	return nil
}

// Start attach the GATT server to a peripheral device.
func (svr *Server) Start(p bt.Peripheral) error {
	d, ok := p.(*Device)
	if !ok {
		return fmt.Errorf("can't convert peripheral to os x device")
	}
	return d.SetServices(svr.svcs)
}

// Stop detatch the GATT server from a peripheral device.
func (svr *Server) Stop() error {
	svr.Lock()
	defer svr.Unlock()
	return nil
}
