package gatt

import (
	"encoding/binary"
	"log"

	"github.com/currantlabs/bt/adv"
	"github.com/currantlabs/bt/att"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
	"github.com/currantlabs/bt/l2cap"
	"github.com/currantlabs/bt/uuid"
)

// State ...
type State string

// State ...
const (
	StateUnknown      = "Unknown"
	StateResetting    = "Resetting"
	StateUnsupported  = "Unsupported"
	StateUnauthorized = "Unauthorized"
	StatePoweredOff   = "PoweredOff"
	StatePoweredOn    = "PoweredOn"
)

// deviceHandler is the handlers(callbacks) of the Device.
type deviceHandler struct {
	// StateChanged is called when the Device states changes.
	StateChanged func(d *Device, s State)

	// Connect is called when a remote central Device connects to the Device.
	CentralConnected func(c *Central)

	// Disconnect is called when a remote central Device disconnects to the Device.
	CentralDisconnected func(c *Central)

	// PeripheralDiscovered is called when a remote peripheral Device is found during scan procedure.
	PeripheralDiscovered func(p *Peripheral, a *adv.Packet, rssi int)

	// PeripheralConnected is called when a remote peripheral is conneted.
	PeripheralConnected func(p *Peripheral, err error)

	// PeripheralConnected is called when a remote peripheral is disconneted.
	PeripheralDisconnected func(p *Peripheral, err error)
}

// An Option is a self-referential function, which sets the option specified.
// Most Options are platform-specific, which gives more fine-grained control over the Device at a cost of losing portibility.
// See http://commandcenter.blogspot.com.au/2014/01/self-referential-functions-and-design.html for more discussion.
type Option func(*Device) error

// Option sets the options specified.
// Some options can only be set before the Device is initialized; they are best used with NewDevice instead of Option.
func (d *Device) Option(opts ...Option) error {
	var err error
	for _, opt := range opts {
		err = opt(d)
	}
	return err
}

// Device ...
type Device struct {
	deviceHandler

	hci hci.HCI
	acl *l2cap.LE

	state State

	// All the following fields are only used peripheralManager (server) implementation.
	svcs  []*Service
	attrs *att.Range

	devID   int
	maxConn int

	advData   *cmd.LESetAdvertisingData
	scanResp  *cmd.LESetScanResponseData
	advParam  *cmd.LESetAdvertisingParameters
	scanParam *cmd.LESetScanParameters
	connParam *cmd.LECreateConnection
}

// NewDevice ...
func NewDevice(opts ...Option) (*Device, error) {
	d := &Device{
		maxConn: 1,  // Support 1 connection at a time.
		devID:   -1, // Find an available HCI Device.

	}
	h, err := hci.NewHCI(d.devID)
	if err != nil {
		return nil, err
	}

	d.hci = h
	d.acl = l2cap.NewL2CAP(h)
	d.Option(opts...)
	return d, nil
}

// Init ...
func (d *Device) Init(f func(*Device, State)) error {
	go d.acceptLoop()

	// Register our own advertising report handler.
	d.hci.SetSubeventHandler(
		evt.LEAdvertisingReportSubCode,
		hci.HandlerFunc(d.handleLEAdvertisingReport))
	d.state = StatePoweredOn
	d.StateChanged = f
	go d.StateChanged(d, d.state)
	return nil
}

// Stop calls OS specific close calls
func (d *Device) Stop() error {
	d.state = StatePoweredOff
	defer d.StateChanged(d, d.state)
	return nil
	// FIXME: rework API
	// return d.hci.Close()
}

// AddService add a service to database.
func (d *Device) AddService(s *Service) *Service {
	d.svcs = append(d.svcs, s)
	d.attrs = generateAttributes(d.svcs, uint16(1)) // ble attrs start at 1
	return s
}

// RemoveAllServices removes all services that are currently in the database.
func (d *Device) RemoveAllServices() error {
	d.svcs = nil
	d.attrs = nil
	return nil
}

// SetServices set the specified service to the database.
// It removes all currently added services, if any.
func (d *Device) SetServices(s []*Service) error {
	d.RemoveAllServices()
	d.svcs = append(d.svcs, s...)
	d.attrs = generateAttributes(d.svcs, uint16(1)) // ble attrs start at 1
	return nil
}

// Advertise advertise AdvPacket
// If name doesn't fit in the advertising packet, it will be put in scan response.
func (d *Device) Advertise(a *adv.Packet) error {
	d.advData = &cmd.LESetAdvertisingData{
		AdvertisingDataLength: uint8(a.Len()),
		AdvertisingData:       a.Packet(),
	}

	d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}, nil)
	d.hci.Send(d.advData, nil)
	d.hci.Send(d.advParam, nil)
	d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}, nil)
	return nil
}

// AdvertiseNameAndServices advertises Device name, and specified service UUIDs.
// It tres to fit the UUIDs in the advertising packet as much as possible.
func (d *Device) AdvertiseNameAndServices(name string, uu []uuid.UUID) error {
	a := adv.NewAdvPacket(nil)
	a.AppendFlags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly)
	a.AppendUUIDFit(uu)

	if a.Len()+len(name)+2 < adv.MaxEIRPacketLength {
		a.AppendName(name)
		d.scanResp = nil
		log.Printf("ADV1: [ % X ]", a.Bytes())
	} else {
		a := adv.NewAdvPacket(nil)
		a.AppendName(name)
		d.scanResp = &cmd.LESetScanResponseData{
			ScanResponseDataLength: uint8(a.Len()),
			ScanResponseData:       a.Packet(),
		}
		log.Printf("ADV2: [ % X ]", a.Bytes())
	}

	log.Printf("ADV3: [ % X ]", a.Bytes())
	return d.Advertise(a)
}

// AdvertiseIBeaconData advertise iBeacon with given manufacturer data.
func (d *Device) AdvertiseIBeaconData(b []byte) error {
	a := adv.NewAdvPacket(nil)
	a.AppendFlags(adv.FlagGeneralDiscoverable | adv.FlagLEOnly)
	a.AppendManufacturerData(0x004C, b)

	return d.Advertise(a)
}

// AdvertiseIBeacon advertises iBeacon with specified parameters.
func (d *Device) AdvertiseIBeacon(u uuid.UUID, major, minor uint16, pwr int8) error {
	b := make([]byte, 23)
	b[0] = 0x02                               // Data type: iBeacon
	b[1] = 0x15                               // Data length: 21 bytes
	copy(b[2:], uuid.Reverse(u))              // Big endian
	binary.BigEndian.PutUint16(b[18:], major) // Big endian
	binary.BigEndian.PutUint16(b[20:], minor) // Big endian
	b[22] = uint8(pwr)                        // Measured Tx Power
	return d.AdvertiseIBeaconData(b)
}

// StopAdvertising stops advertising.
func (d *Device) StopAdvertising() error {
	return d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 0}, nil)
}

// Scan discovers surounding remote peripherals that have the Service UUID specified in ss.
// If ss is set to nil, all devices scanned are reported.
// dup specifies weather duplicated advertisement should be reported or not.
// When a remote peripheral is discovered, the PeripheralDiscovered Handler is called.
func (d *Device) Scan(ss []uuid.UUID, dup bool) {
	d.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
	d.hci.Send(d.scanParam, nil)
	d.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 1}, nil)
}

// StopScanning stops scanning.
func (d *Device) StopScanning() {
	d.hci.Send(&cmd.LESetScanEnable{LEScanEnable: 0}, nil)
}

// Connect connects to a remote peripheral.
func (d *Device) Connect(p *Peripheral) {
	cmd := *d.connParam
	cmd.PeerAddressType = p.advReport.AddressType // public or random
	cmd.PeerAddress = p.advReport.Address         //
	d.hci.Send(&cmd, nil)
}

// CancelConnection disconnects a remote peripheral.
func (d *Device) CancelConnection(p *Peripheral) {
	d.hci.Send(&cmd.Disconnect{
		ConnectionHandle: p.c.Conn.Parameters().ConnectionHandle(),
		Reason:           0x13,
	}, nil)
}

func (d *Device) acceptLoop() {
	for {
		l2c, err := d.acl.Accept()
		if err != nil {
			log.Fatalf("can't accept conn: %s", err)
		}
		if l2c.Parameters().Role() == 0x01 {
			d.handleCentral(l2c)
			continue
		}
		d.handlePeripheral(l2c)
	}
}

func (d *Device) handleCentral(l2c l2cap.Conn) {
	c := newCentral(d.attrs, l2c)
	if d.CentralConnected != nil {
		d.CentralConnected(c)
	}
	c.server.Loop()
	if d.CentralDisconnected != nil {
		d.CentralDisconnected(c)
	}
	d.hci.Send(&cmd.LESetAdvertiseEnable{AdvertisingEnable: 1}, nil)
}

func (d *Device) handlePeripheral(l2c l2cap.Conn) {
	p := newPeripheral(d, l2c)
	if d.PeripheralConnected != nil {
		go d.PeripheralConnected(p, nil)
	}
	p.c.Loop()
	if d.PeripheralDisconnected != nil {
		d.PeripheralDisconnected(p, nil)
	}
}

func (d *Device) handleLEAdvertisingReport(b []byte) error {
	if d.PeripheralDiscovered == nil {
		return nil
	}
	e := &leAdvertisingReportEP{}
	if err := e.Unmarshal(b); err != nil {
		return err
	}

	for _, r := range e.Reports {
		adv := adv.NewAdvPacket(r.Data)

		a := r.Address
		p := &Peripheral{
			d:         d,
			adv:       adv,
			advReport: &r,
			addr:      []byte{a[5], a[4], a[3], a[2], a[1], a[0]},
		}
		go d.PeripheralDiscovered(p, adv, r.RSSI)
	}
	return nil
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
