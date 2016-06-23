package lib

import (
	"fmt"
	"log"
	"time"

	"github.com/currantlabs/x/io/bt"
)

// NewCountChar ...
func NewCountChar() *bt.Characteristic {
	n := 0
	c := bt.NewCharacteristic(CountCharUUID)
	c.HandleRead(bt.ReadHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
		fmt.Fprintf(rsp, "count: Read %d", n)
		log.Printf("count: Read %d", n)
		n++
	}))

	c.HandleWrite(bt.WriteHandlerFunc(func(req bt.Request, rsp bt.ResponseWriter) {
		log.Printf("count: Wrote %s", string(req.Data()))
	}))

	f := func(req bt.Request, n bt.Notifier) {
		cnt := 0
		log.Printf("count: Notification subscribed")
		for {
			select {
			case <-n.Context().Done():
				log.Printf("count: Notification unsubscribed")
				return
			case <-time.After(time.Second):
				log.Printf("count: Notify: %d", cnt)
				if _, err := fmt.Fprintf(n, "Count: %d", cnt); err != nil {
					// Client disconnected prematurely before unsubscription.
					log.Printf("count: Failed to notify : %s", err)
					return
				}
				cnt++
			}
		}
	}

	c.HandleNotify(bt.NotifyHandlerFunc(f))
	c.HandleIndicate(bt.NotifyHandlerFunc(f))
	return c
}
