package gatt

import (
	"encoding/binary"
	"log"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/cmd"
	"github.com/currantlabs/bt/evt"
)

type device struct {
	deviceHandler

	hci bt.HCI

	state State

	// All the following fields are only used peripheralManager (server) implementation.
	svcs  []*Service
	attrs *attrRange

	devID   int
	chkLE   bool
	maxConn int

	advData   *cmd.LESetAdvertisingData
	scanResp  *cmd.LESetScanResponseData
	advParam  *cmd.LESetAdvertisingParameters
	scanParam *cmd.LESetScanParameters
	connParam *cmd.LECreateConnection
}

// NewDevice ...
func NewDevice(opts ...Option) (Device, error) {
	d := &device{
		maxConn: 1,    // Support 1 connection at a time.
		devID:   -1,   // Find an available HCI device.
		chkLE:   true, // Check if the device supports LE.

		advParam: &cmd.LESetAdvertisingParameters{
			AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
			AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
			OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
			DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
			DirectAddress:           [6]byte{}, // Public or Random Address of the device to be connected
			AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
			AdvertisingFilterPolicy: 0x00,
		},
		scanParam: &cmd.LESetScanParameters{
			LEScanType:           0x01,   // [0x00]: passive, 0x01: active
			LEScanInterval:       0x0010, // [0x10]: 0.625ms * 16
			LEScanWindow:         0x0010, // [0x10]: 0.625ms * 16
			OwnAddressType:       0x00,   // [0x00]: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // [0x00]: accept all, 0x01: ignore non-white-listed.
		},
		connParam: &cmd.LECreateConnection{
			LEScanInterval:        0x0004, // N x 0.625ms
			LEScanWindow:          0x0004, // N x 0.625ms
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
		},
	}

	h, err := bt.NewHCI(d.devID, d.chkLE)
	if err != nil {
		return nil, err
	}

	d.hci = h
	return d, nil
}

func (d *device) acceptLoop() {
	for {
		l2c, err := d.hci.Accept()
		if err != nil {
			log.Fatalf("can't accept conn: %s", err)
		}
		if l2c.Parameters().Role == 0x01 {
			d.handleCentral(l2c)
			continue
		}
		d.handlePeripheral(l2c)
	}
}

func (d *device) handleCentral(l2c bt.Conn) {
	c := newCentral(d.attrs, l2c)
	if d.centralConnected != nil {
		d.centralConnected(c)
	}
	c.loop()
	if d.centralDisconnected != nil {
		d.centralDisconnected(c)
	}
	d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}, nil)
}

func (d *device) handlePeripheral(l2c bt.Conn) {
	p := newPeripheral(d, l2c)
	if d.peripheralConnected != nil {
		go d.peripheralConnected(p, nil)
	}
	p.c.Loop()
	if d.peripheralDisconnected != nil {
		d.peripheralDisconnected(p, nil)
	}
}
func (d *device) Init(f func(Device, State)) error {
	go d.acceptLoop()

	// Register our own advertising report handler.
	d.hci.SetSubeventHandler(
		evt.LEAdvertisingReportEvent{}.SubCode(),
		bt.HandlerFunc(d.handleLEAdvertisingReport))
	d.state = StatePoweredOn
	d.stateChanged = f
	go d.stateChanged(d, d.state)
	return nil
}

func (d *device) handleLEAdvertisingReport(b []byte) {
	if d.peripheralDiscovered == nil {
		return
	}
	e := &leAdvertisingReportEP{}
	if err := e.Unmarshal(b); err != nil {
		return
	}

	for _, r := range e.Reports {
		adv := &Advertisement{}
		adv.unmarshall(r.Data)
		adv.Connectable = r.EventType&0x01 == 0x01

		a := r.Address
		p := &peripheral{
			d:         d,
			adv:       adv,
			advReport: &r,
			addr:      []byte{a[5], a[4], a[3], a[2], a[1], a[0]},
		}
		go d.peripheralDiscovered(p, adv, r.RSSI)
	}
}

type advertisingReport struct {
	EventType   uint8
	AddressType uint8
	Address     [6]byte
	Data        []byte
	RSSI        int
}

type leAdvertisingReportEP struct {
	SubeventCode uint8
	Reports      []advertisingReport
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *leAdvertisingReportEP) Unmarshal(b []byte) error {
	e.SubeventCode, b = b[0], b[1:]
	n, b := int(b[0]), b[1:]

	e.Reports = make([]advertisingReport, n)
	for i := range e.Reports {
		r := &e.Reports[i]
		r.EventType = b[0]
		r.AddressType = b[1]
		copy(r.Address[:], b[2:8])
		dlen := int(b[8])
		r.Data = make([]byte, dlen)
		copy(r.Data, b[9:9+dlen])
		r.RSSI = int(b[9+dlen])
		b = b[10+dlen:]
	}
	return nil
}

func (d *device) Stop() error {
	d.state = StatePoweredOff
	defer d.stateChanged(d, d.state)
	return d.hci.Close()
}

func (d *device) AddService(s *Service) error {
	d.svcs = append(d.svcs, s)
	d.attrs = generateAttributes(d.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

func (d *device) RemoveAllServices() error {
	d.svcs = nil
	d.attrs = nil
	return nil
}

func (d *device) SetServices(s []*Service) error {
	d.RemoveAllServices()
	d.svcs = append(d.svcs, s...)
	d.attrs = generateAttributes(d.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

func (d *device) Advertise(a *AdvPacket) error {
	d.advData = &cmd.LESetAdvertisingData{
		AdvertisingDataLength: uint8(a.Len()),
		AdvertisingData:       a.Bytes(),
	}

	d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}, nil)
	return nil
}

func (d *device) AdvertiseNameAndServices(name string, uu []UUID) error {
	a := &AdvPacket{}
	a.AppendFlags(flagGeneralDiscoverable | flagLEOnly)
	a.AppendUUIDFit(uu)

	if len(a.b)+len(name)+2 < MaxEIRPacketLength {
		a.AppendName(name)
		d.scanResp = nil
	} else {
		a := &AdvPacket{}
		a.AppendName(name)
		d.scanResp = &cmd.LESetScanResponseData{
			ScanResponseDataLength: uint8(a.Len()),
			ScanResponseData:       a.Bytes(),
		}
	}

	return d.Advertise(a)
}

func (d *device) AdvertiseIBeaconData(b []byte) error {
	a := &AdvPacket{}
	a.AppendFlags(flagGeneralDiscoverable | flagLEOnly)
	a.AppendManufacturerData(0x004C, b)

	return d.Advertise(a)
}

func (d *device) AdvertiseIBeacon(u UUID, major, minor uint16, pwr int8) error {
	b := make([]byte, 23)
	b[0] = 0x02                               // Data type: iBeacon
	b[1] = 0x15                               // Data length: 21 bytes
	copy(b[2:], reverse(u))                   // Big endian
	binary.BigEndian.PutUint16(b[18:], major) // Big endian
	binary.BigEndian.PutUint16(b[20:], minor) // Big endian
	b[22] = uint8(pwr)                        // Measured Tx Power
	return d.AdvertiseIBeaconData(b)
}

func (d *device) StopAdvertising() error {
	return d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}, nil)
}

func (d *device) Scan(ss []UUID, dup bool) {
	d.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
}

func (d *device) StopScanning() {
	d.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
}

func (d *device) Connect(p Peripheral) {
	pp := p.(*peripheral)
	cmd := *d.connParam
	cmd.PeerAddressType = pp.advReport.AddressType // public or random
	cmd.PeerAddress = pp.advReport.Address         //
	d.hci.Send(&cmd, nil)
}

func (d *device) CancelConnection(p Peripheral) {
	pp := p.(*peripheral)
	d.hci.Send(&cmd.Disconnect{
		ConnectionHandle: pp.c.conn.Parameters().ConnectionHandle,
		Reason:           0x13,
	}, nil)
}
