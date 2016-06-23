package dev

import (
	"log"

	"github.com/currantlabs/ble/darwin"
	"github.com/currantlabs/x/io/bt"
)

// NewBroadcaster ...
func NewBroadcaster() bt.Broadcaster {
	dev, err := darwin.NewDevice(1)
	if err != nil {
		log.Fatalf("can't create Broadcaster: %s", err)
	}
	if err := dev.Init(nil); err != nil {
		log.Fatalf("can't init Broadcaster: %s", err)
	}
	return dev
}

// NewObserver ...
func NewObserver() bt.Observer {
	dev, err := darwin.NewDevice(0)
	if err != nil {
		log.Fatalf("can't create Observer: %s", err)
	}
	if err := dev.Init(nil); err != nil {
		log.Fatalf("can't init Observer: %s", err)
	}
	return dev
}

// NewPeripheral ...
func NewPeripheral() bt.Peripheral {
	dev, err := darwin.NewDevice(1)
	if err != nil {
		log.Fatalf("can't create Peripheral: %s", err)
	}
	if err := dev.Init(nil); err != nil {
		log.Fatalf("can't init Peripheral: %s", err)
	}
	return dev
}

// NewCentral ...
func NewCentral() bt.Central {
	dev, err := darwin.NewDevice(0)
	if err != nil {
		log.Fatalf("can't create Central: %s", err)
	}
	if err := dev.Init(nil); err != nil {
		log.Fatalf("can't init Central: %s", err)
	}
	return dev
}

// NewGATTServer ...
func NewGATTServer() bt.Server {
	return darwin.NewServer()
}

// NewGATTClient ...
func NewGATTClient(l2c bt.Conn) bt.Client {
	return darwin.NewClient(l2c)
}
