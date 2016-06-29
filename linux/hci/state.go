package hci

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/currantlabs/ble/linux/hci/cmd"
	"github.com/mgutz/logxi/v1"
)

var logger = log.New("state")

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
	done    chan bool
	err     error

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
}

func newStates() *states {
	return &states{
		chState: make(chan nextState, 10),
		done:    make(chan bool),

		scanEnable:  cmd.LESetScanEnable{LEScanEnable: 1},
		scanDisable: cmd.LESetScanEnable{LEScanEnable: 0},
		advEnable:   cmd.LESetAdvertiseEnable{AdvertisingEnable: 1},
		advDisable:  cmd.LESetAdvertiseEnable{AdvertisingEnable: 0},
		scanParams: cmd.LESetScanParameters{
			LEScanType:           0x01,   // 0x00: passive, 0x01: active
			LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
			LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
			OwnAddressType:       0x00,   // 0x00: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
		},
		advParams: cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
			AdvertisingIntervalMax:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
			AdvertisingType:         0x00,      // 00: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
			OwnAddressType:          0x00,      // 0x00: public, 0x01: random
			DirectAddressType:       0x00,      // 0x00: public, 0x01: random
			DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
			AdvertisingChannelMap:   0x7,       // 0x07 0x01: ch37, 0x2: ch38, 0x4: ch39
			AdvertisingFilterPolicy: 0x00,
		},
		connParams: cmd.LECreateConnection{
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
		},
	}
}

func (s *states) init(h *HCI) {
	s.hci = h
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
		// if s.isListening {
		// 	s.err = errors.Wrapf(ErrBusyListening, "advertise")
		// 	return
		// }
		if s.send(&s.advEnable) == nil {
			s.isAdvertising = true
		}
		if s.err == nil || s.err == ErrDisallowed {
			s.err = nil
			return
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
