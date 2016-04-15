package gap

import (
	"sync"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
)

// Broadcaster ...
type Broadcaster interface {
	Advertise(ad []byte, sr []byte) error
	StartAdvertising() error
	StopAdvertising() error
}

// NewBroadcaster ...
func NewBroadcaster(h hci.HCI) (Broadcaster, error) {
	b := &bcast{
		hci: h,

		advOn:  cmd.LESetAdvertiseEnable{AdvertisingEnable: 1},
		advOff: cmd.LESetAdvertiseEnable{AdvertisingEnable: 0},
		advParam: cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
			OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
			DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
			DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
			AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
			AdvertisingFilterPolicy: 0x00,
		},
	}
	return b, nil
}

type bcast struct {
	sync.RWMutex

	hci hci.HCI

	advOn    cmd.LESetAdvertiseEnable
	advOff   cmd.LESetAdvertiseEnable
	advParam cmd.LESetAdvertisingParameters
	advData  cmd.LESetAdvertisingData
	scanResp cmd.LESetScanResponseData
}

func (b *bcast) Advertise(ad []byte, sr []byte) error {
	b.Lock()
	defer b.Unlock()

	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return ErrEIRPacketTooLong
	}

	b.advData.AdvertisingDataLength = uint8(len(ad))
	copy(b.advData.AdvertisingData[:], ad)

	b.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(b.scanResp.ScanResponseData[:], sr)

	// TODO: set advParam event type accordingly.

	b.hci.Send(&b.advOff, nil)
	b.hci.Send(&b.advParam, nil)
	b.hci.Send(&b.advData, nil)
	b.hci.Send(&b.scanResp, nil)
	b.hci.Send(&b.advOn, nil)

	return nil
}

func (b *bcast) StartAdvertising() error {
	return b.hci.Send(&b.advOn, nil)
}

func (b *bcast) StopAdvertising() error {
	return b.hci.Send(&b.advOff, nil)
}
