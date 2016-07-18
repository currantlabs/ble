package main

import (
	"fmt"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

type matcher func(a ble.Advertisement) bool

type discoverer struct {
	ble.Client

	match matcher
	ch    chan ble.Advertisement
	svcs  []*ble.Service
}

func (d *discoverer) Handle(a ble.Advertisement) {
	if !d.match(a) {
		return
	}
	gatt.StopScanning()
	d.ch <- a
}

func (d *discoverer) connect(m matcher) error {
	d.match = m
	if err := gatt.SetAdvHandler(d); err != nil {
		return fmt.Errorf("can't set adv handler: %s", err)
	}
	if err := gatt.Scan(false); err != nil {
		return fmt.Errorf("can't scan: %s", err)
	}

	a := <-d.ch

	var err error
	d.Client, err = gatt.Dial(a.Address())
	if err != nil {
		return fmt.Errorf("can't dial: %s", err)
	}
	return nil
}

func (d *discoverer) discover() ([]*ble.Service, error) {
	if d.svcs != nil {
		return d.svcs, nil
	}

	ss, err := d.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("can't discover services: %s\n", err)
	}
	for _, s := range ss {
		cs, err := d.DiscoverCharacteristics(nil, s)
		if err != nil {
			return nil, fmt.Errorf("can't discover characteristics: %s\n", err)
		}
		for _, c := range cs {
			_, err := d.DiscoverDescriptors(nil, c)
			if err != nil {
				return nil, fmt.Errorf("can't discover descriptors: %s\n", err)
			}
		}
	}
	d.svcs = ss
	return ss, nil
}

type target interface {
	// UUID() ble.UUID
}

func (d *discoverer) find(t target) target {
	for _, s := range d.svcs {
		ts, ok := t.(*ble.Service)
		if ok && s.UUID.Equal(ts.UUID) {
			return s
		}
		for _, c := range s.Characteristics {
			tc, ok := t.(*ble.Characteristic)
			if ok && c.UUID.Equal(tc.UUID) {
				return c
			}
			for _, d := range c.Descriptors {
				td, ok := t.(*ble.Descriptor)
				if ok && d.UUID.Equal(td.UUID) {
					return d
				}
			}
		}
	}
	return nil
}

func newDiscoverer() *discoverer {
	return &discoverer{
		ch: make(chan ble.Advertisement),
	}
}
