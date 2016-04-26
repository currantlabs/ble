package hci

import (
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

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(af bt.AdvFilter, ah bt.AdvHandler) error {
	h.advFilter, h.advHandler = af, ah
	return nil
}

// SetScanParams ...
func (h *HCI) SetScanParams(p ScanParams) error {
	h.scanParams = cmd.LESetScanParameters(p)
	return nil
}

// Scan starts scanning.
func (h *HCI) Scan() error {
	h.Send(&h.scanParams, nil)
	return h.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
}

// StopScanning stops scanning.
func (h *HCI) StopScanning() error {
	return h.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
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

// SetAdvertisement ...
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return bt.ErrEIRPacketTooLong
	}

	h.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.advData.AdvertisingData[:], ad)

	h.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.scanResp.ScanResponseData[:], sr)

	return nil
}

// SetAdvParams ...
func (h *HCI) SetAdvParams(p AdvParams) error {
	h.advParams = cmd.LESetAdvertisingParameters(p)
	return nil
}

// Advertise ...
func (h *HCI) Advertise() error {
	h.Send(&h.advData, nil)
	h.Send(&h.scanResp, nil)
	h.Send(&h.advParams, nil)
	return h.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}, nil)
}

// StopAdvertising ...
func (h *HCI) StopAdvertising() error {
	return h.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}, nil)
}
