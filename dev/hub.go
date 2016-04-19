package dev

import (
	"fmt"
	"log"
	"sync"

	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/evt"
)

func newEvtHub() *evtHub {
	todo := func(b []byte) error { return fmt.Errorf("hci: unhandled (TODO) event packet: [ % X ]", b) }

	h := &evtHub{
		evth: map[int]hci.Handler{
			evt.EncryptionChangeCode:                     hci.HandlerFunc(todo),
			evt.ReadRemoteVersionInformationCompleteCode: hci.HandlerFunc(todo),
			evt.HardwareErrorCode:                        hci.HandlerFunc(todo),
			evt.DataBufferOverflowCode:                   hci.HandlerFunc(todo),
			evt.EncryptionKeyRefreshCompleteCode:         hci.HandlerFunc(todo),
			evt.AuthenticatedPayloadTimeoutExpiredCode:   hci.HandlerFunc(todo),
		},
		subh: map[int]hci.Handler{
			evt.LEReadRemoteUsedFeaturesCompleteSubCode:   hci.HandlerFunc(todo),
			evt.LERemoteConnectionParameterRequestSubCode: hci.HandlerFunc(todo),
		},
	}
	h.SetEventHandler(0x3E, hci.HandlerFunc(h.handleLEMeta))
	h.SetSubeventHandler(evt.LEAdvertisingReportSubCode, hci.HandlerFunc(h.handleLEAdvertisingReport))
	return h
}

type evtHub struct {
	sync.Mutex
	evth map[int]hci.Handler
	subh map[int]hci.Handler
}

func (h *evtHub) EventHandler(c int) hci.Handler {
	h.Lock()
	defer h.Unlock()
	return h.evth[c]
}

func (h *evtHub) SetEventHandler(c int, f hci.Handler) hci.Handler {
	h.Lock()
	defer h.Unlock()
	old := h.evth[c]
	h.evth[c] = f
	return old
}

func (h *evtHub) SubeventHandler(c int) hci.Handler {
	h.Lock()
	defer h.Unlock()
	return h.subh[c]
}

func (h *evtHub) SetSubeventHandler(c int, f hci.Handler) hci.Handler {
	h.Lock()
	defer h.Unlock()
	old := h.subh[c]
	h.subh[c] = f
	return old
}

func (h *evtHub) handle(b []byte) error {
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		return fmt.Errorf("hci: corrupt event packet: [ % X ]", b)
	}
	if f := h.EventHandler(code); f != nil {
		return f.Handle(b[2:])
	}
	return fmt.Errorf("hci: unsupported event packet: [ % X ]", b)
}

func (h *evtHub) handleLEMeta(b []byte) error {
	subcode := int(b[0])
	if f := h.SubeventHandler(subcode); f != nil {
		return f.Handle(b)
	}
	return fmt.Errorf("hci: unsupported LE event: [ % X ]", b)
}

// Default dummy advertising packet handler.
func (h *evtHub) handleLEAdvertisingReport(b []byte) error {
	e := evt.LEAdvertisingReport(b)
	f := func(a [6]byte) string {
		return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", a[5], a[4], a[3], a[2], a[1], a[0])
	}
	for i := 0; i < int(e.NumReports()); i++ {
		log.Printf("%d, %d, %s, %d, [% X]",
			e.EventType(i), e.AddressType(i), f(e.Address(i)), e.RSSI(i), e.Data(i))
	}
	return nil
}
