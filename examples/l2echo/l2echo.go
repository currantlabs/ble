package main

import (
	"fmt"
	"log"

	"github.com/currantlabs/bluetooth/hci"
	"github.com/currantlabs/bluetooth/l2cap"
)

func main() {
	h, err := hci.NewHCI(-1, false)
	if err != nil {
		log.Fatalf("failed to create HCI. %s", err)
	}

	const rxMTU = 23

	l := l2cap.NewL2CAP(h, rxMTU)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("failed to accept L2CAP Conn. %s", err)
		}
		go handleConn(c, rxMTU)
	}
}

func handleConn(c l2cap.Conn, rxMTU int) {
	fmt.Printf("[%s] connected\n", c.RemoteAddr())
	b := make([]byte, rxMTU)
	for {
		n, err := c.Read(b)
		if err != nil {
			break
		}
		fmt.Printf("[%s] recv %2d bytes: [ % X ]\n", c.RemoteAddr(), n, b[n:])
		n, err = c.Write(b)
		if err != nil {
			break
		}
		fmt.Printf("[%s] sent %2d bytes: [ % X ]\n", c.RemoteAddr(), n, b[n:])
	}
	fmt.Printf("[]%s] disconnected\n", c.RemoteAddr())
}
