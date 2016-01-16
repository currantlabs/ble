package option

import (
	"github.com/currantlabs/bt/gatt"
	"github.com/currantlabs/bt/hci/cmd"
)

// DefaultClientOptions ...
var DefaultClientOptions = []gatt.Option{
	gatt.LnxMaxConnections(1),
	gatt.LnxDeviceID(-1),
	gatt.LnxSetScanParameters(
		&cmd.LESetScanParameters{
			LEScanType:           0x01,   // [0x00]: passive, 0x01: active
			LEScanInterval:       0x0010, // [0x10]: 0.625ms * 16
			LEScanWindow:         0x0010, // [0x10]: 0.625ms * 16
			OwnAddressType:       0x00,   // [0x00]: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // [0x00]: accept all, 0x01: ignore non-white-listed.
		}),
	gatt.LnxSetConnectionParameters(
		&cmd.LECreateConnection{
			LEScanInterval:        0x0010, // N x 0.625ms
			LEScanWindow:          0x0010, // N x 0.625ms
			InitiatorFilterPolicy: 0x00,   // white list not used
			OwnAddressType:        0x00,   // public
			ConnIntervalMin:       0x0006, // N x 0.125ms
			ConnIntervalMax:       0x0006, // N x 0.125ms
			ConnLatency:           0x0000, //
			SupervisionTimeout:    0x0048, // N x 10ms
			MinimumCELength:       0x0000, // N x 0.625ms
			MaximumCELength:       0x0000, // N x 0.625ms
			// PeerAddressType:       pd.AddressType, // public or random
			// PeerAddress:           pd.Address,     //
		}),
}

// DefaultServerOptions include default options.
var DefaultServerOptions = []gatt.Option{
	gatt.LnxMaxConnections(1),
	gatt.LnxDeviceID(-1),
	gatt.LnxSetAdvertisingParameters(
		&cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
			OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
			DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
			DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
			AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
			AdvertisingFilterPolicy: 0x00,
		}),
}
