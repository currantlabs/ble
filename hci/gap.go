package hci

import (
	"fmt"
	"log"
	"net"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/hci/cmd"
)

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

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(af bt.AdvFilter, ah bt.AdvHandler) error {
	h.advFilter, h.advHandler = af, ah
	return nil
}

// SetScanParams sets scanning parameters.
func (h *HCI) SetScanParams(p ScanParams) error {
	h.scanParams = cmd.LESetScanParameters(p)
	return h.setState(ScanningUpdated)
}

// Scan starts scanning.
func (h *HCI) Scan() error { return h.setState(Scanning) }

// StopScanning stops scanning.
func (h *HCI) StopScanning() error { return h.setState(ScanningStopped) }

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return bt.ErrEIRPacketTooLong
	}

	h.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.advData.AdvertisingData[:], ad)

	h.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.scanResp.ScanResponseData[:], sr)

	return h.setState(AdvertisingUpdated)
}

// SetAdvParams sets advertising parameters.
func (h *HCI) SetAdvParams(p AdvParams) error {
	h.advParams = cmd.LESetAdvertisingParameters(p)
	return h.setState(AdvertisingUpdated)
}

// Advertise starts advertising.
func (h *HCI) Advertise() error { return h.setState(Advertising) }

// StopAdvertising stops advertising.
func (h *HCI) StopAdvertising() error { return h.setState(AdvertisingStopped) }

// Accept starts advertising and accepts connection.
func (h *HCI) Accept() (bt.Conn, error) {
	if err := h.setState(Listening); err != nil {
		return nil, err
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chSlaveConn:
		h.setState(CentralConnected)
		return c, nil
	case <-h.chListenerTmo:
		h.setState(ListeningCanceled)
		return nil, fmt.Errorf("listner timed out")
	}
}

// Close ...
func (h *HCI) Close() error {
	if h.err != nil {
		return h.err
	}
	return nil
}

// Addr ...
func (h *HCI) Addr() bt.Addr { return h.addr }

// Dial ...
func (h *HCI) Dial(a bt.Addr) (bt.Conn, error) {
	b, ok := a.(net.HardwareAddr)
	if !ok {
		return nil, fmt.Errorf("invalid addr")
	}
	h.connParams.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	h.setState(Dialing)
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chMasterConn:
		h.setState(DialingStopped)
		return c, nil
	case <-h.chDialerTmo:
		h.setState(DialingCanceled)
		return nil, fmt.Errorf("dialer timed out")
	}
}

// SetConnParams ...
func (h *HCI) SetConnParams(p ConnParams) error {
	h.connParams = cmd.LECreateConnection(p)
	return nil
}

func sendAndChk(h *HCI, c Command) error {
	if b, err := h.send(c); err != nil {
		return err
	} else if len(b) > 0 && b[0] != 0x00 {
		return ErrCommand(b[0])
	}
	return nil
}

// State ...
type State int

type nextState struct {
	s    State
	done chan error
}

// State ...
const (
	Advertising            State = iota
	AdvertisingStopped     State = iota
	AdvertisingUpdated     State = iota
	Scanning               State = iota
	ScanningStopped        State = iota
	ScanningUpdated        State = iota
	Dialing                State = iota
	DialingStopped         State = iota
	DialingCanceled        State = iota
	DialingUpdated         State = iota
	PeripheralConnected    State = iota
	PeripheralDisconnected State = iota
	Listening              State = iota
	ListeningCanceled      State = iota
	ListeningUpdated       State = iota
	CentralConnected       State = iota
	CentralDisconnected    State = iota
)

const (
	advertising = iota
	scanning
	dialing
	listening
)

func (h *HCI) stateLoop() {
	for {
		select {
		case <-h.done:
			return
		case s := <-h.chState:
			h.handleState(s)
		}
	}
}

func (h *HCI) setState(s State) error {
	n := nextState{s: s, done: make(chan error)}
	h.chState <- n
	return <-n.done
}

func (h *HCI) handleState(n nextState) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	var err error
	defer func() { n.done <- err }()

	switch n.s {
	case Scanning:
		if h.state[scanning] {
			return
		}
		sendAndChk(h, &h.scanParams)
		if err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 1}); err == nil {
			h.state[scanning] = true
		}
	case ScanningStopped:
		if h.state[scanning] {
			return
		}
		err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 0})
		h.state[scanning] = false
	case ScanningUpdated:
		if h.state[scanning] {
			return
		}
		sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 0})
		err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 1})
		if err == nil {
			h.state[scanning] = true
		}
	case Advertising:
		if h.state[advertising] {
			return
		}
		sendAndChk(h, &h.advParams)
		sendAndChk(h, &h.advData)
		sendAndChk(h, &h.scanResp)
		if err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}); err == nil {
			h.state[advertising] = true
		}
	case AdvertisingStopped:
		if !h.state[advertising] {
			return
		}
		h.state[advertising] = false
		err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 0})
	case AdvertisingUpdated:
		if !h.state[advertising] {
			return
		}
		sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 0})
		if err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}); err == nil {
			h.state[advertising] = true
		}
	case Dialing:
		if h.state[dialing] {
			return
		}
		if err = sendAndChk(h, &h.connParams); err == nil {
			h.state[dialing] = true
		}
	case PeripheralConnected:
		h.state[dialing] = false
	case PeripheralDisconnected:
		if !h.state[dialing] {
			return
		}
		if err = sendAndChk(h, &h.connParams); err == nil {
			h.state[dialing] = true
		}
	case DialingCanceled:
		if !h.state[dialing] {
			return
		}
		h.state[dialing] = false
		err = sendAndChk(h, &cmd.LECreateConnectionCancel{})
	case DialingUpdated:
		if !h.state[dialing] {
			return
		}
		sendAndChk(h, &cmd.LECreateConnectionCancel{})
		if err = sendAndChk(h, &h.connParams); err == nil {

		}
	case Listening:
		if h.state[listening] && h.state[advertising] {
			return
		}
		h.state[listening] = true
		sendAndChk(h, &h.advParams)
		sendAndChk(h, &h.advData)
		sendAndChk(h, &h.scanResp)
		if err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}); err == nil {
			h.state[advertising] = true
		} else if e, ok := err.(ErrCommand); ok && e == 0x0C {
			err = nil
			log.Printf("reaches connection limit")
		}
	case CentralConnected:
		if !h.state[listening] {
			return
		}
		h.state[listening] = false
		h.state[advertising] = false
	case CentralDisconnected:
		if h.state[listening] && h.state[advertising] {
			return
		}
		sendAndChk(h, &h.advParams)
		sendAndChk(h, &h.advData)
		sendAndChk(h, &h.scanResp)
		if err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}); err == nil {
			h.state[advertising] = true
			log.Printf("under connection limit")
		} else if e, ok := err.(ErrCommand); ok && e == 0x0C {
			err = nil
			log.Printf("still reaches connection limit")
		}
	case ListeningCanceled:
		if !h.state[listening] {
			return
		}
		h.state[listening] = false
		h.state[advertising] = false
	case ListeningUpdated:
	}
	h.stateChanged(n.s)
}

func (h *HCI) stateChanged(s State) {
}
