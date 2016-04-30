package hci

import (
	"fmt"
	"net"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/hci/cmd"
)

// State describe the operation of the Link Layer.
type State string

// Link Layer States [Vol 6, Part B, 1.1]
const (
	Unknown     State = "Unknown"
	Standby     State = "Standby"
	Advertising State = "Advertising"
	Scanning    State = "Scanning"
	Initiating  State = "Initiating"
	Connection  State = "Connection"
)

// State returns current state of the HCI device.
func (h *HCI) State() State {
	return h.state
}

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

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(af bt.AdvFilter, ah bt.AdvHandler) error {
	h.advFilter, h.advHandler = af, ah
	return nil
}

// SetScanParams sets scanning parameters.
func (h *HCI) SetScanParams(p ScanParams) error {
	h.scanParams = cmd.LESetScanParameters(p)
	return sendAndChk(h, &h.scanParams)
}

// Scan starts scanning.
func (h *HCI) Scan() error {
	sendAndChk(h, &h.scanParams)
	return sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 1})
}

// StopScanning stops scanning.
func (h *HCI) StopScanning() error {
	return sendAndChk(h, &cmd.LESetScanEnable{LEScanEnable: 0})
}

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return bt.ErrEIRPacketTooLong
	}

	h.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.advData.AdvertisingData[:], ad)

	h.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.scanResp.ScanResponseData[:], sr)

	if err := sendAndChk(h, &h.advData); err != nil {
		return err
	}

	return sendAndChk(h, &h.scanResp)
}

// SetAdvParams sets advertising parameters to the controller.
func (h *HCI) SetAdvParams(p AdvParams) error {
	h.advParams = cmd.LESetAdvertisingParameters(p)
	return sendAndChk(h, &h.advParams)
}

// Advertise starts advertising if the device wasn't in advertising state.
func (h *HCI) Advertise() error {
	if err := sendAndChk(h, &h.advParams); err != nil {
		return err
	}
	return sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 1})
}

// StopAdvertising stops advertising if the device was in advertising state.
func (h *HCI) StopAdvertising() error {
	return sendAndChk(h, &cmd.LESetAdvertiseEnable{AdvertisingEnable: 0})
}

// Accept returns a L2CAP master connection.
func (h *HCI) Accept() (bt.Conn, error) {
	select {
	case <-h.done:
		return nil, h.err
	case c := <-h.chSlaveConn:
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
func (h *HCI) Addr() bt.Addr {
	return h.addr
}

// Dial ...
func (h *HCI) Dial(a bt.Addr) (bt.Conn, error) {
	if h.err != nil {
		return nil, h.err
	}
	b, ok := a.(net.HardwareAddr)
	if !ok {
		return nil, fmt.Errorf("invalid addr")
	}
	h.connParams.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	sendAndChk(h, &h.connParams)
	c := <-h.chMasterConn
	return c, nil
}

func sendAndChk(h *HCI, c Command) error {
	b, err := h.send(c)
	if err != nil {
		return err
	}
	if len(b) > 0 && b[0] != 0x00 {
		return ErrCommand(b[0])
	}
	return nil
}
