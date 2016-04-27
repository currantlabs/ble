package gap

import (
	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/dev"
	"github.com/currantlabs/bt/hci/cmd"
)

// Broadcaster ...
type Broadcaster struct {
	dev dev.Device

	advOn    cmd.LESetAdvertiseEnable
	advOff   cmd.LESetAdvertiseEnable
	advParam cmd.LESetAdvertisingParameters
	advData  cmd.LESetAdvertisingData
	scanResp cmd.LESetScanResponseData
}

// Init ...
func (b *Broadcaster) Init(d dev.Device) error {
	b.dev = d
	b.advOn = cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}
	b.advOff = cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}
	b.advParam = cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
		DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}
	return nil
}

// Advertise ...
func (b *Broadcaster) Advertise(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return ErrEIRPacketTooLong
	}

	b.advData.AdvertisingDataLength = uint8(len(ad))
	copy(b.advData.AdvertisingData[:], ad)

	b.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(b.scanResp.ScanResponseData[:], sr)

	// TODO: set advParam event type accordingly.

	b.dev.Send(&b.advOff, nil)
	b.dev.Send(&b.advParam, nil)
	b.dev.Send(&b.advData, nil)
	b.dev.Send(&b.scanResp, nil)
	b.dev.Send(&b.advOn, nil)

	return nil
}

// StartAdvertising ...
func (b *Broadcaster) StartAdvertising() error {
	return b.dev.Send(&b.advOn, nil)
}

// StopAdvertising ...
func (b *Broadcaster) StopAdvertising() error {
	return b.dev.Send(&b.advOff, nil)
}
