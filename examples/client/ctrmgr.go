package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/ble/gatt"
	"github.com/currantlabs/x/io/bt"
)

// ClientHandler is invoked when a connection is established.
type ClientHandler interface {
	Handle(bt.Advertisement, bt.Client)
}

// ClientHandlerFunc is an adapter to convert a function, with expected signature, to a ClientHandler.
func ClientHandlerFunc(f func(bt.Advertisement, bt.Client)) ClientHandler {
	return &clientHandlerFunc{f}
}

type clientHandlerFunc struct {
	f func(bt.Advertisement, bt.Client)
}

func (c clientHandlerFunc) Handle(a bt.Advertisement, cln bt.Client) {
	c.f(a, cln)
}

type centralManager struct {
	bt.Central

	chAdv       chan bt.Advertisement
	chDone      chan bool
	visited     map[string]bool
	visitedLock sync.RWMutex
	handler     map[ClientHandler]bool
}

func newCentralManager(c bt.Central) *centralManager {
	return &centralManager{
		Central: c,

		chAdv:   make(chan bt.Advertisement),
		chDone:  make(chan bool),
		visited: make(map[string]bool),
		handler: make(map[ClientHandler]bool),
	}
}

// HandleClient registers ClientHandler to the centralManager.
func (m *centralManager) HandleClient(h ClientHandler) {
	m.handler[h] = true
}

// Start starts the centralManager.
func (m *centralManager) Start() error {
	m.SetAdvHandler(bt.AdvHandlerFunc(m.advHandle))
	m.Scan(false)
	go func() {
		for {
			select {
			case <-m.chDone:
			case a := <-m.chAdv:
				// Stop scanning before dialing to the device.
				if err := m.StopScanning(); err != nil {
					log.Fatalf("can't stop dialing: %s", err)
				}

				// Dial connects to the remote device.
				l2c, err := m.Dial(a.Address())
				if err != nil {
					log.Fatalf("can't dial: %s", err)
				}

				// Mark the device visited.
				m.visitedLock.Lock()
				m.visited[a.Address().String()] = true
				m.visitedLock.Unlock()

				// Attach a GATT client to the connection.
				cln, _ := newClient(l2c)

				// Spawn a goroutine to handle the connection.
				go func() {
					for h := range m.handler {
						h.Handle(a, cln)
					}
					cln.CancelConnection()
				}()

				// Continuing scanning.
				if err := m.Scan(false); err != nil {
					log.Fatalf("can't stop dialing: %s", err)
				}
			}
		}
	}()
	return nil
}

// Stop stops the centralManager.
func (m *centralManager) Stop() {
	close(m.chDone)
}

func newClient(l2c bt.Conn) (bt.Client, error) {
	cln := gatt.NewGATTClient(l2c)
	txMTU, err := cln.ExchangeMTU(bt.MaxMTU)
	if err != nil {
		log.Printf("can't set MTU: %s\n", err)
		// return nil, err
	}

	// Perform services/characteristics/descriptors discovery.
	if err := discover(cln); err != nil {
		log.Fatalf("can't discover: %s", err)
	}

	return &centralManagerCilent{Client: cln, txMTU: txMTU}, nil
}

type centralManagerCilent struct {
	bt.Client
	txMTU int
	rxMTU int
}

func (c centralManagerCilent) ExchangeMTU(rxMTU int) (int, error) {
	return c.txMTU, nil
}

func (m *centralManager) advHandle(a bt.Advertisement) {
	if a.LocalName() != "Gopher" {
		return
	}
	// m.visitedLock.RLock()
	// defer m.visitedLock.RUnlock()
	// return !m.visited[a.Address().String()]
	select {
	case m.chAdv <- a:
	default:
	}
}

func discover(cln bt.Client) error {
	ss, err := cln.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("can't discover services: %s\n", err)
	}
	for _, s := range ss {
		cs, err := cln.DiscoverCharacteristics(nil, s)
		if err != nil {
			return fmt.Errorf("can't discover characteristics: %s\n", err)
		}
		for _, c := range cs {
			_, err := cln.DiscoverDescriptors(nil, c)
			if err != nil {
				return fmt.Errorf("can't discover descriptors: %s\n", err)
			}
		}
	}
	return nil
}

func findChar(cln bt.Client, u bt.UUID) *bt.Characteristic {
	for _, s := range cln.Services() {
		for _, c := range s.Characteristics {
			if c.UUID.Equal(u) {
				return c
			}
		}
	}
	return nil
}
