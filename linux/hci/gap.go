package hci

import (
	"fmt"
	"net"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux/adv"
	"github.com/currantlabs/ble/linux/gatt"
	"github.com/currantlabs/ble/linux/hci/cmd"
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

// Addr ...
func (h *HCI) Addr() ble.Addr { return h.addr }

// SetAdvHandler ...
func (h *HCI) SetAdvHandler(ah ble.AdvHandler) error {
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
func (h *HCI) AdvertiseNameAndServices(name string, uuids ...ble.UUID) error {
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
func (h *HCI) AdvertiseIBeacon(u ble.UUID, major, minor uint16, pwr int8) error {
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

// Accept starts advertising and accepts connection.
func (h *HCI) Accept() (ble.Conn, error) {
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
func (h *HCI) Dial(a ble.Addr) (ble.Client, error) {
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
		return gatt.NewClient(c)
	case <-tmo:
		err := h.states.set(DialingCanceling)
		<-h.chMasterConn
		return nil, err
	}
}

// Close ...
func (h *HCI) Close() error {
	if h.err != nil {
		return h.err
	}
	return nil
}

// Advertise starts advertising.
func (h *HCI) Advertise() error {
	return h.states.set(Advertising)
}

// SetAdvertisement sets advertising data and scanResp.
func (h *HCI) SetAdvertisement(ad []byte, sr []byte) error {
	if len(ad) > adv.MaxEIRPacketLength || len(sr) > adv.MaxEIRPacketLength {
		return ble.ErrEIRPacketTooLong
	}

	h.states.Lock()
	h.states.advData.AdvertisingDataLength = uint8(len(ad))
	copy(h.states.advData.AdvertisingData[:], ad)

	h.states.scanResp.ScanResponseDataLength = uint8(len(sr))
	copy(h.states.scanResp.ScanResponseData[:], sr)
	h.states.Unlock()

	return h.states.set(AdvertisingUpdated)
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

// SetConnParams sets connection parameters.
func (h *HCI) SetConnParams(p ConnParams) error {
	h.states.Lock()
	h.states.connParams = cmd.LECreateConnection(p)
	h.states.Unlock()
	return h.states.set(DialingUpdated)
}
