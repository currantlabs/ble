package main

import (
	"bytes"
	"crypto/rand"
	"log"
	"os"
	"time"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/examples/lib"
)

const echoDataSize = 1024 * 16

func echo(a bt.Advertisement, cln bt.Client) {
	l := log.New(os.Stdout, "["+cln.Address().String()+"] ", log.Lmicroseconds)
	c := findChar(cln, lib.EchoCharUUID)
	if c == nil {
		l.Printf("skip echo test: can't find char %s", lib.EchoCharUUID.String())
		return
	}
	ec := newEchoClient(cln)

	// Synchronous transfer (write with response, echoed by indicattion)
	ec.test(l, c, false, true)

	// Asynchronous transfer (write without response, echoed by notification)
	ec.test(l, c, true, false)
	// ec.ClearSubscriptions()
}

type echoClient struct {
	bt.Client
	data   []byte
	rx     chan []byte
	result chan bool
}

func newEchoClient(cln bt.Client) *echoClient {
	ec := &echoClient{
		Client: cln,
		data:   make([]byte, echoDataSize),
		rx:     make(chan []byte, 16),
		result: make(chan bool),
	}
	if _, err := rand.Read(ec.data); err != nil {
	}
	return ec
}

func (ec *echoClient) rxHandler(b []byte) {
	ec.rx <- b
}

func (ec *echoClient) test(l *log.Logger, c *bt.Characteristic, noRsp bool, ind bool) {
	if err := ec.Subscribe(c, ind, ec.rxHandler); err != nil {
		l.Printf("can't subscribe: %s", err)
	}
	defer ec.ClearSubscriptions()

	go func() {
		rx := ec.data
		for len(rx) > 0 {
			select {
			case b := <-ec.rx:
				if bytes.Compare(b, rx[:len(b)]) != 0 {
					ec.result <- false
					return
				}
				rx = rx[len(b):]
				// l.Printf("%d/%d\n", echoDataSize-len(rx), echoDataSize)
			case <-time.After(time.Second * 10):
				ec.result <- false
				l.Printf("timeout")
				return
			}
		}
		ec.result <- true
	}()

	tx := ec.data
	for len(tx) > 0 {
		n, err := ec.ExchangeMTU(bt.MaxMTU)
		if err != nil {
			return
		}
		n -= 3 // deduct 3 bytes of ATT header
		if n > len(tx) {
			n = len(tx)
		}
		if err := ec.WriteCharacteristic(c, tx[:n], noRsp); err != nil {
			l.Printf("can't wtite char: %s", err)
		}
		tx = tx[n:]
	}
	l.Printf("test success: %t\n", <-ec.result)
}
