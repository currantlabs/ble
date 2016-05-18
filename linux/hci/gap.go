package hci

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/linux/adv"
	"github.com/currantlabs/bt/linux/hci/cmd"
	"github.com/mgutz/logxi/v1"

	"github.com/pkg/errors"
)

var logger = log.New("state")

// ScanParams implements LE Set Scan Parameters (0x08|0x000B) [Vol 2, Part E, 7.8.10]
type ScanParams struct {
	LEScanType           uint8
	LEScanInterval       uint16
	LEScanWindow         uint16
	OwnAddressType       uint8
	ScanningFilterPolicy uint8
}

// AdvParams implements LE Set Advertising Parameters (0x08|0x0006) [Vol 2, Part E, 7.8.5]
type AdvParams struct {
	AdvertisingIntervalMin  uint16
	AdvertisingIntervalMax  uint16
	AdvertisingType         uint8
	OwnAddressType          uint8
	DirectAddressType       uint8
	DirectAddress           [6]byte
	AdvertisingChannelMap   uint8
	AdvertisingFilterPolicy uint8
}

// ConnParams implements LE Create Connection (0x08|0x000D) [Vol 2, Part E, 7.8.12]
type ConnParams struct {
	LEScanInterval        uint16
	LEScanWindow          uint16
	InitiatorFilterPolicy uint8
	PeerAddressType       uint8
	PeerAddress           [6]byte
	OwnAddressType        uint8
	ConnIntervalMin       uint16
	ConnIntervalMax       uint16
	ConnLatency           uint16
	SupervisionTimeout    uint16
	MinimumCELength       uint16
	MaximumCELength       uint16
}

// Addr ...
func (h *HCI) Addr() bt.Addr { return h.addr }

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(ah bt.AdvHandler) error {
	h.advHandler = ah
	return nil
}

// Scan starts scanning.
func (h *HCI) Scan(allowDup bool) error {
	h.states.Lock()
	h.states.scanEnable.FilterDuplicates = 1
	if allowDup {
		h.states.scanEnable.FilterDuplicates = 0
	}
	h.states.Unlock()
	return h.states.set(Scanning)
}

// StopScanning stops scanning.
func (h *HCI) StopScanning() error {
	return h.states.set(StopScanning)
}

// AdvertiseNameAndServices advertises device name, and specified service UUIDs.
// It tries to fit the UUIDs in the advertising data as much as possible.
// If name doesn't fit in the advertising data, it will be put in scan response.
func (h *HCI) AdvertiseNameAndServices(name string, uuids ...bt.UUID) error {
	ad, err := adv.NewPacket(adv.Flags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly))
	if err != nil {
		return err
	}
	f := adv.AllUUID

	// Current length of ad packet plus two bytes of length and tag.
	l := ad.Len() + 1 + 1
	for _, u := range uuids {
		l += u.Len()
	}
	if l > adv.MaxEIRPacketLength {
		f = adv.SomeUUID
	}
	for _, u := range uuids {
		if err := ad.Append(f(u)); err != nil {
			if err == adv.ErrNotFit {
				break
			}
			return err
		}
	}
	sr, _ := adv.NewPacket()
	switch {
	case ad.Append(adv.CompleteName(name)) == nil:
	case sr.Append(adv.CompleteName(name)) == nil:
	case sr.Append(adv.ShortName(name)) == nil:
	}
	if err := h.SetAdvertisement(ad.Bytes(), sr.Bytes()); err != nil {
		return nil
	}
	return h.Advertise()
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func (h *HCI) AdvertiseIBeaconData(md []byte) error {
	ad, err := adv.NewPacket(adv.IBeaconData(md))
	if err != nil {
		return err
	}
	if err := h.SetAdvertisement(ad.Bytes(), nil); err != nil {
		return nil
	}
	return h.Advertise()
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func (h *HCI) AdvertiseIBeacon(u bt.UUID, major, minor uint16, pwr int8) error {
	ad, err := adv.NewPacket(adv.IBeacon(u, major, minor, pwr))
	if err != nil {
		return err
	}
	if err := h.SetAdvertisement(ad.Bytes(), nil); err != nil {
		return nil
	}
	return h.Advertise()
}

// StopAdvertising stops advertising.
func (h *HCI) StopAdvertising() error {
	return h.states.set(StopAdvertising)
}

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return bt.ErrEIRPacketTooLong
	}

	h.states.Lock()
	h.states.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.states.advData.AdvertisingData[:], ad)

	h.states.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.states.scanResp.ScanResponseData[:], sr)
	h.states.Unlock()

	return h.states.set(AdvertisingUpdated)
}

// Advertise starts advertising.
func (h *HCI) Advertise() error {
	return h.states.set(Advertising)
}

// Accept starts advertising and accepts connection.
func (h *HCI) Accept() (bt.Conn, error) {
	if err := h.states.set(Listening); err != nil {
		return nil, err
	}
	var tmo <-chan time.Time
	if h.listenerTmo != time.Duration(0) {
		tmo = time.After(h.listenerTmo)
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chSlaveConn:
		h.states.set(CentralConnected)
		return c, nil
	case <-tmo:
		h.states.set(StopListening)
		return nil, fmt.Errorf("listner timed out")
	}
}

// Dial ...
func (h *HCI) Dial(a bt.Addr) (bt.Conn, error) {
	b, err := net.ParseMAC(a.String())
	if err != nil {
		return nil, ErrInvalidAddr
	}
	h.states.Lock()
	h.states.connParams.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	h.states.Unlock()
	if err := h.states.set(Dialing); err != nil {
		return nil, err
	}
	defer h.states.set(StopDialing)
	var tmo <-chan time.Time
	if h.dialerTmo != time.Duration(0) {
		tmo = time.After(h.dialerTmo)
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chMasterConn:
		return c, nil
	case <-tmo:
		if err := h.states.set(DialingCanceling); err == nil {
			return <-h.chMasterConn, nil
		}
		return <-h.chMasterConn, fmt.Errorf("dialer timed out")
	}
}

// Close ...
func (h *HCI) Close() error {
	if h.err != nil {
		return h.err
	}
	return nil
}

// SetScanParams sets scanning parameters.
func (h *HCI) SetScanParams(p ScanParams) error {
	h.states.Lock()
	h.states.scanParams = cmd.LESetScanParameters(p)
	h.states.Unlock()
	return h.states.set(ScanningUpdated)
}

// SetAdvParams sets advertising parameters.
func (h *HCI) SetAdvParams(p AdvParams) error {
	h.states.Lock()
	h.states.advParams = cmd.LESetAdvertisingParameters(p)
	h.states.Unlock()
	return h.states.set(AdvertisingUpdated)
}

// SetConnParams ...
func (h *HCI) SetConnParams(p ConnParams) error {
	h.states.Lock()
	h.states.connParams = cmd.LECreateConnection(p)
	h.states.Unlock()
	return h.states.set(DialingUpdated)
}

// State ...
type State string

type nextState struct {
	s    State
	done chan error
}

// State ...
const (
	Advertising            State = "Advertising"
	StopAdvertising        State = "StopAdvertising"
	AdvertisingUpdated     State = "AdvertisingUpdated"
	Scanning               State = "Scanning"
	StopScanning           State = "StopScanning"
	ScanningUpdated        State = "ScanningUpdated"
	Dialing                State = "Dialing"
	DialingCanceling       State = "DialingCanceling"
	StopDialing            State = "StopDialing"
	DialingUpdated         State = "DialingUpdated"
	PeripheralConnected    State = "PeripheralConnected"
	PeripheralDisconnected State = "PeripheralDisconnected"
	Listening              State = "Listening"
	StopListening          State = "StopListening"
	ListeningUpdated       State = "ListeningUpdated"
	CentralConnected       State = "CentralConnected"
	CentralDisconnected    State = "CentralDisconnected"
)

type states struct {
	sync.Mutex

	hci *HCI

	isAdvertising bool
	isScanning    bool
	isDialing     bool
	isListening   bool

	chState chan nextState

	advEnable   cmd.LESetAdvertiseEnable
	advDisable  cmd.LESetAdvertiseEnable
	scanEnable  cmd.LESetScanEnable
	scanDisable cmd.LESetScanEnable
	connCancel  cmd.LECreateConnectionCancel

	advData    cmd.LESetAdvertisingData
	scanResp   cmd.LESetScanResponseData
	advParams  cmd.LESetAdvertisingParameters
	scanParams cmd.LESetScanParameters
	connParams cmd.LECreateConnection

	done chan bool

	err error
}

func (s *states) init(h *HCI) {
	s.hci = h
	s.chState = make(chan nextState, 10)

	s.scanEnable = cmd.LESetScanEnable{LEScanEnable: 1}
	s.scanDisable = cmd.LESetScanEnable{LEScanEnable: 0}
	s.advEnable = cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}
	s.advDisable = cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}
	s.scanParams = cmd.LESetScanParameters{
		LEScanType:           0x01,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}
	s.advParams = cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingIntervalMax:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingType:         0x00,      // 00: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // 0x00: public, 0x01: random
		DirectAddressType:       0x00,      // 0x00: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // 0x07 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}
	s.connParams = cmd.LECreateConnection{
		LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		InitiatorFilterPolicy: 0x00,      // White list is not used
		PeerAddressType:       0x00,      // Public Device Address
		PeerAddress:           [6]byte{}, //
		OwnAddressType:        0x00,      // Public Device Address
		ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
		SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
		MinimumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
		MaximumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
	}

	s.done = make(chan bool)
	go s.loop()
	s.set(AdvertisingUpdated)
	s.set(ScanningUpdated)
}

func (s *states) close() {
	close(s.done)
}

func (s *states) loop() {
	for {
		select {
		case <-s.done:
			return
		case next := <-s.chState:
			s.handle(next)
		}
	}
}

func (s *states) set(next State) error {
	n := nextState{s: next, done: make(chan error)}
	s.chState <- n
	return <-n.done
}

func (s *states) send(c Command) error {
	if s.err != nil {
		return s.err
	}
	if b, err := s.hci.send(c); err != nil {
		s.err = err
	} else if len(b) > 0 && b[0] != 0x00 {
		s.err = ErrCommand(b[0])
	}
	return s.err
}

func (s *states) handle(n nextState) {
	s.err = nil
	logger.Info(string(n.s) + " +")
	defer func() {
		logger.Info(string(n.s) + " -")
		n.done <- s.err
	}()
	switch n.s {
	case Scanning:
		if s.isScanning {
			return
		}
		if s.isDialing {
			s.err = errors.Wrapf(ErrBusyScanning, "scan")
		}
		s.hci.chStartScan <- true
		if s.send(&s.scanEnable) == nil {
			s.isScanning = true
		}
		if s.err == ErrDisallowed {
			logger.Info("scan: over maximum connections.")
			s.err = nil
		}
	case StopScanning:
		if !s.isScanning {
			return
		}
		s.isScanning = false
		s.send(&s.scanDisable)
	case ScanningUpdated:
		if s.isScanning {
			s.send(&s.scanDisable)
		}
		s.send(&s.scanParams)
		if s.isScanning {
			s.send(&s.scanEnable)
		}
	case Advertising:
		if s.isAdvertising {
			return
		}
		if s.isListening {
			s.err = errors.Wrapf(ErrBusyListening, "advertise")
			return
		}
		if s.send(&s.advEnable) == nil {
			s.isAdvertising = true
		}
	case StopAdvertising:
		if !s.isAdvertising {
			return
		}
		s.isAdvertising = false
		s.send(&s.advDisable)
	case AdvertisingUpdated:
		if s.isAdvertising {
			s.send(&s.advDisable)
		}
		s.send(&s.advParams)
		s.send(&s.advData)
		s.send(&s.scanResp)
		if s.isAdvertising {
			s.send(&s.advEnable)
		}
	case Dialing:
		if s.isScanning {
			s.err = errors.Wrapf(ErrBusyScanning, "dial")
			return
		}
		if s.isDialing {
			s.err = errors.Wrapf(ErrBusyDialing, "dial")
			return
		}
		s.send(&s.connParams)
		if s.err == nil || s.err == ErrDisallowed {
			s.err = nil
			s.isDialing = true
			return
		}
	case StopDialing:
		s.isDialing = false
	case DialingCanceling:
		s.isDialing = false
		s.send(&s.connCancel)
	case DialingUpdated:
		if !s.isDialing {
			return
		}
		if s.send(&s.connCancel) == ErrDisallowed {
			s.err = nil
		}
		s.send(&s.connParams)
	case PeripheralConnected:
	case PeripheralDisconnected:
		if !s.isDialing {
			return
		}
		if s.send(&s.connParams) == ErrDisallowed {
			s.err = nil
		}
	case Listening:
		if s.isListening {
			s.err = errors.Wrapf(ErrBusyListening, "listen")
			return
		}
		if s.isAdvertising {
			s.err = errors.Wrapf(ErrBusyAdvertising, "listen")
			return
		}
		s.isListening = true
		if s.send(&s.advEnable) == ErrDisallowed {
			s.err = nil
			logger.Info("listen: over maximum connections.")
		}
	case CentralConnected:
		s.isListening = false
	case CentralDisconnected:
		if !s.isListening {
			return
		}
		if s.send(&s.advEnable) == nil {
			logger.Info("listen: under maximum connections.")
		} else if s.err == ErrDisallowed {
			s.err = nil
			logger.Info("listen: over maximum connections.")
		}
	case StopListening:
		s.isListening = false
		if s.send(&s.advDisable) == ErrDisallowed {
			s.err = nil
		}
	case ListeningUpdated:
	}
}
