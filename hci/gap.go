package hci

import (
	"fmt"
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
	return h.update(scanUpdate)
}

// Scan starts scanning.
func (h *HCI) Scan() error { return h.update(scanEnable) }

// StopScanning stops scanning.
func (h *HCI) StopScanning() error { return h.update(scanDisable) }

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return bt.ErrEIRPacketTooLong
	}

	h.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.advData.AdvertisingData[:], ad)

	h.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.scanResp.ScanResponseData[:], sr)

	return h.update(advUpdate)
}

// SetAdvParams sets advertising parameters.
func (h *HCI) SetAdvParams(p AdvParams) error {
	h.advParams = cmd.LESetAdvertisingParameters(p)
	return h.update(advUpdate)
}

// Advertise starts advertising.
func (h *HCI) Advertise() error { return h.update(advEnable) }

// StopAdvertising stops advertising.
func (h *HCI) StopAdvertising() error { return h.update(advDisable) }

// Accept starts advertising and accepts connection.
func (h *HCI) Accept() (bt.Conn, error) {
	if err := h.update(listen); err != nil {
		if e, ok := err.(ErrCommand); !ok || e != 0x0C {
			return nil, err
		}
	}
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chSlaveConn:
		h.stateMu.Lock()
		h.state[advertising] = false
		h.state[listening] = false
		h.stateMu.Unlock()
		return c, nil
	case <-h.chListenerTmo:
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
	h.update(dial)
	defer h.update(dialCancel)
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chMasterConn:
		return c, nil
	case <-h.chDialerTmo:
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

const (
	advEnable = iota
	advDisable
	advUpdate
	scanEnable
	scanDisable
	scanUpdate
	dial
	dialCancel
	dialUpdate
	listen
	listenCancel
	listenUpdate
)

const (
	advertising = iota
	scanning
	dialing
	listening
)

func (h *HCI) update(op int) error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	var err error
	switch {
	case op == scanEnable && !h.state[scanning]:
		sendAndChk(h, &h.scanParams)
		err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 1})
		if err == nil {
			h.state[scanning] = true
		}
	case op == scanDisable && h.state[scanning]:
		h.state[scanning] = false
		err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 0})
	case op == scanUpdate && h.state[scanning]:
		sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 0})
		err = sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 1})
		if err == nil {
			h.state[scanning] = true
		}
	case op == advEnable && !h.state[advertising]:
		sendAndChk(h, &h.advParams)
		sendAndChk(h, &h.advData)
		sendAndChk(h, &h.scanResp)
		err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1})
		if err == nil {
			h.state[advertising] = true
		}
	case op == advDisable && h.state[advertising]:
		h.state[advertising] = false
		err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 0})
	case op == advUpdate && h.state[advertising]:
		sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 0})
		err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1})
		if err == nil {
			h.state[advertising] = true
		}
	case op == dial && !h.state[dialing]:
		h.state[dialing] = true
		err = sendAndChk(h, &h.connParams)
	case op == dialCancel && h.state[dialing]:
		h.state[dialing] = false
		err = sendAndChk(h, &cmd.LECreateConnectionCancel{})
	case op == dialUpdate && h.state[dialing]:
		sendAndChk(h, &cmd.LECreateConnectionCancel{})
		err = sendAndChk(h, &h.connParams)
	case op == listen && !h.state[advertising]:
		h.state[listening] = true
		sendAndChk(h, &h.advParams)
		sendAndChk(h, &h.advData)
		sendAndChk(h, &h.scanResp)
		err = sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1})
		if err == nil {
			h.state[advertising] = true
		}
	case op == listenCancel && h.state[listening]:
		h.state[listening] = false
	case op == listenUpdate && h.state[listening]:
	}
	return err
}
