package darwin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/currantlabs/ble/darwin/xpc"
	"github.com/currantlabs/x/io/bt"
)

const (
	evtStateChanged               = 6
	evtAdvertisingStarted         = 16
	evtAdvertisingStopped         = 17
	evtServiceAdded               = 18
	evtReadRequest                = 19
	evtWriteRequest               = 20
	evtSubscribe                  = 21
	evtUnubscribe                 = 22
	evtConfirmation               = 23
	evtPeripheralDiscovered       = 37
	evtPeripheralConnected        = 38
	evtPeripheralDisconnected     = 40
	evtATTMTU                     = 53
	evtRSSIRead                   = 55
	evtServiceDiscovered          = 56
	evtIncludedServicesDiscovered = 63
	evtCharacteristicsDiscovered  = 64
	evtCharacteristicRead         = 71
	evtCharacteristicWritten      = 72
	evtNotificationValueSet       = 74
	evtDescriptorsDiscovered      = 76
	evtDescriptorRead             = 79
	evtDescriptorWritten          = 80
	evtSleveConnectionComplete    = 81
	evtMasterConnectionComplete   = 82
)

type message struct {
	id   int
	args xpc.Dict
	rspc chan msg
}

// Device ...
type Device struct {
	init chan bool
	xpc  xpc.XPC
	role int // 1: peripheralManager (server), 0: centralManager (client)

	rspc chan msg

	conns map[string]*conn

	// Only used in client/centralManager implementation
	advHandler bt.AdvHandler
	chConn     chan *conn

	// Only used in server/peripheralManager implementation
	chars map[int]*bt.Characteristic
	base  int
}

// NewDevice ...
func NewDevice(role int) (*Device, error) {
	d := &Device{
		role:   role,
		rspc:   make(chan msg),
		conns:  make(map[string]*conn),
		chConn: make(chan *conn),
		chars:  make(map[int]*bt.Characteristic),
		base:   1,
	}
	d.xpc = xpc.XpcConnect("com.apple.blued", d)
	return d, nil
}

// Init ...
func (d *Device) Init(f func(Device, State)) error {
	rsp := d.sendReq(1, xpc.Dict{
		"kCBMsgArgName": fmt.Sprintf("gopher-%v", time.Now().Unix()),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBInitOptionShowPowerAlert": 1,
		},
		"kCBMsgArgType": d.role,
	})
	s := State(rsp.state())
	if s != StatePoweredOn {
		return fmt.Errorf("state: %s", s)
	}
	log.Printf("state: %s", s)
	return nil
}

// AdvertiseMfgData ...
func (d *Device) AdvertiseMfgData(b []byte) error {
	rsp := d.sendReq(8, xpc.Dict{
		"kCBAdvDataAppleMfgData": b,
	})
	return result("Advertise", rsp)
}

// AdvertiseNameAndServices ...
func (d *Device) AdvertiseNameAndServices(name string, ss ...bt.UUID) error {
	rsp := d.sendReq(8, xpc.Dict{
		"kCBAdvDataLocalName":    name,
		"kCBAdvDataServiceUUIDs": uuidSlice(ss)},
	)
	return result("AdvertiseNameandService", rsp)
}

// AdvertiseIBeaconData ...
func (d *Device) AdvertiseIBeaconData(md []byte) error {
	var utsname xpc.Utsname
	xpc.Uname(&utsname)

	if utsname.Release >= "14." {
		l := len(md)
		b := []byte{byte(l + 5), 0xFF, 0x4C, 0x00, 0x02, byte(l)}
		return d.AdvertiseMfgData(append(b, md...))
	}
	rsp := d.sendReq(8, xpc.Dict{"kCBAdvDataAppleBeaconKey": md})
	return result("AdvertisingIbeaconData", rsp)
}

// AdvertiseIBeacon ...
func (d *Device) AdvertiseIBeacon(u bt.UUID, major, minor uint16, pwr int8) error {
	b := make([]byte, 21)
	copy(b, bt.Reverse(u))                    // Big endian
	binary.BigEndian.PutUint16(b[16:], major) // Big endian
	binary.BigEndian.PutUint16(b[18:], minor) // Big endian
	b[20] = uint8(pwr)                        // Measured Tx Power
	return d.AdvertiseIBeaconData(b)
}

// StopAdvertising ...
func (d *Device) StopAdvertising() error {
	return result("StopAdvertising", d.sendReq(9, nil))
}

// SetAdvHandler ...
func (d *Device) SetAdvHandler(ah bt.AdvHandler) error {
	d.advHandler = ah
	return nil
}

// Scan ...
func (d *Device) Scan(allowDup bool) error {
	return d.sendCmd(29, xpc.Dict{
		// "kCBMsgArgUUIDs": uuidSlice(ss),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBScanOptionAllowDuplicates": map[bool]int{true: 1, false: 0}[allowDup],
		},
	})
}

// StopScanning ...
func (d *Device) StopScanning() error {
	return d.sendCmd(30, nil)
}

func result(msg string, rsp msg) error {
	if res := rsp.result(); res != 0 {
		return fmt.Errorf("%s: %d", msg, res)
	}
	return nil
}

// RemoveAllServices ...
func (d *Device) RemoveAllServices() error {
	return d.sendCmd(12, nil)
}

// AddService ...
func (d *Device) AddService(s *bt.Service) error {
	// skip GATT and GAP services
	if s.UUID.Equal(bt.GAPUUID) || s.UUID.Equal(bt.GATTUUID) {
		return nil
	}

	xs := xpc.Dict{
		"kCBMsgArgAttributeID":     d.base,
		"kCBMsgArgAttributeIDs":    []int{},
		"kCBMsgArgCharacteristics": nil,
		"kCBMsgArgType":            1, // 1 => primary, 0 => excluded
		"kCBMsgArgUUID":            bt.Reverse(s.UUID),
	}
	d.base++

	xcs := xpc.Array{}
	for _, c := range s.Characteristics {
		props := 0
		perm := 0
		if c.Property&bt.CharRead != 0 {
			props |= 0x02
			if bt.CharRead&c.Secure != 0 {
				perm |= 0x04
			} else {
				perm |= 0x01
			}
		}
		if c.Property&bt.CharWriteNR != 0 {
			props |= 0x04
			if c.Secure&bt.CharWriteNR != 0 {
				perm |= 0x08
			} else {
				perm |= 0x02
			}
		}
		if c.Property&bt.CharWrite != 0 {
			props |= 0x08
			if c.Secure&bt.CharWrite != 0 {
				perm |= 0x08
			} else {
				perm |= 0x02
			}
		}
		if c.Property&bt.CharNotify != 0 {
			if c.Secure&bt.CharNotify != 0 {
				props |= 0x100
			} else {
				props |= 0x10
			}
		}
		if c.Property&bt.CharIndicate != 0 {
			if c.Secure&bt.CharIndicate != 0 {
				props |= 0x200
			} else {
				props |= 0x20
			}
		}

		xc := xpc.Dict{
			"kCBMsgArgAttributeID":              d.base,
			"kCBMsgArgUUID":                     bt.Reverse(c.UUID),
			"kCBMsgArgAttributePermissions":     perm,
			"kCBMsgArgCharacteristicProperties": props,
			"kCBMsgArgData":                     c.Value,
		}
		c.Handle = uint16(d.base)
		d.chars[d.base] = c
		d.base++

		xds := xpc.Array{}
		for _, d := range c.Descriptors {
			if d.UUID.Equal(bt.ClientCharacteristicConfigUUID) {
				// skip CCCD
				continue
			}
			xd := xpc.Dict{
				"kCBMsgArgData": d.Value,
				"kCBMsgArgUUID": bt.Reverse(d.UUID),
			}
			xds = append(xds, xd)
		}
		xc["kCBMsgArgDescriptors"] = xds
		xcs = append(xcs, xc)
	}
	xs["kCBMsgArgCharacteristics"] = xcs

	rsp := d.sendReq(10, xs)
	return result("AddServices", rsp)
}

// SetServices ...
func (d *Device) SetServices(ss []*bt.Service) error {
	d.RemoveAllServices()
	for _, s := range ss {
		d.AddService(s)
	}
	return nil
}

// Dial ...
func (d *Device) Dial(a bt.Addr) (bt.Conn, error) {
	d.sendCmd(31, xpc.Dict{
		"kCBMsgArgDeviceUUID": xpc.MakeUUID(a.String()),
		"kCBMsgArgOptions": xpc.Dict{
			"kCBConnectOptionNotifyOnDisconnection": 1,
		},
	})
	return <-d.chConn, nil
}

// HandleXpcEvent process Device events and asynchronous errors.
func (d *Device) HandleXpcEvent(event xpc.Dict, err error) {
	if err != nil {
		log.Println("error:", err)
		return
	}
	m := msg(event)
	args := msg(msg(event).args())
	// log.Printf(">> %d, %v", m.id(), m.args())

	switch m.id() {
	case // Device event
		evtStateChanged,
		evtAdvertisingStarted,
		evtAdvertisingStopped,
		evtServiceAdded:
		d.rspc <- args

	case evtPeripheralDiscovered:
		if d.advHandler == nil {
			break
		}
		a := &Advertisement{
			args: m.args(),
			ad:   args.advertisementData(),
		}
		go d.advHandler.Handle(a)

	case evtConfirmation:
		// log.Printf("confirmed: %d", args.attributeID())

	case evtATTMTU:
		d.conn(args.deviceUUID()).SetTxMTU(args.attMTU())

	case evtSleveConnectionComplete:
		fallthrough
	case evtMasterConnectionComplete:
		c := d.conn(args.deviceUUID())
		c.connInterval = args.connectionInterval()
		c.connLatency = args.connectionLatency()
		c.supervisionTimeout = args.supervisionTimeout()

	case evtReadRequest:
		aid := args.attributeID()
		char := d.chars[aid]
		v := char.Value
		if v == nil {
			c := d.conn(args.deviceUUID())
			req := bt.NewRequest(c, nil, args.offset())
			buf := bytes.NewBuffer(make([]byte, 0, c.txMTU-1))
			rsp := bt.NewResponseWriter(buf)
			char.ReadHandler.ServeRead(req, rsp)
			v = buf.Bytes()
		}

		d.sendCmd(13, xpc.Dict{
			"kCBMsgArgAttributeID":   aid,
			"kCBMsgArgData":          v,
			"kCBMsgArgTransactionID": args.transactionID(),
			"kCBMsgArgResult":        0,
		})

	case evtWriteRequest:
		for _, xxw := range args.attWrites() {
			xw := msg(xxw.(xpc.Dict))
			aid := xw.attributeID()
			char := d.chars[aid]
			req := bt.NewRequest(d.conn(args.deviceUUID()), xw.data(), xw.offset())
			char.WriteHandler.ServeWrite(req, nil)
			if xw.ignoreResponse() == 1 {
				continue
			}
			d.sendCmd(13, xpc.Dict{
				"kCBMsgArgAttributeID":   aid,
				"kCBMsgArgData":          nil,
				"kCBMsgArgTransactionID": args.transactionID(),
				"kCBMsgArgResult":        0,
			})
		}

	case evtSubscribe:
		d.conn(args.deviceUUID()).subscribed(d.chars[args.attributeID()])

	case evtUnubscribe:
		d.conn(args.deviceUUID()).unsubscribed(d.chars[args.attributeID()])

	case evtPeripheralConnected:
		d.chConn <- d.conn(args.deviceUUID())

	case evtPeripheralDisconnected:
		d.conn(args.deviceUUID()).rspc <- m
		delete(d.conns, args.deviceUUID().String())

	case evtCharacteristicRead:
		// Notification
		c := d.conn(args.deviceUUID())
		if args.isNotification() != 0 {
			sub := c.subs[uint16(args.characteristicHandle())]
			if sub == nil {
				log.Printf("notified by unsubscribed handle")
				// FIXME: should terminate the connection?
			} else {
				go sub.fn(args.data())
			}
			break
		}
		c.rspc <- m

	case // Peripheral events
		evtRSSIRead,
		evtServiceDiscovered,
		evtIncludedServicesDiscovered,
		evtCharacteristicsDiscovered,
		evtCharacteristicWritten,
		evtNotificationValueSet,
		evtDescriptorsDiscovered,
		evtDescriptorRead,
		evtDescriptorWritten:

		d.conn(args.deviceUUID()).rspc <- m

	default:
		log.Printf("Unhandled event: %#v", event)
	}
}

// Accept ...
func (d *Device) Accept() (bt.Conn, error) {
	return nil, nil
}

// Addr ...
func (d *Device) Addr() bt.Addr {
	return nil
}

// Close ...
func (d *Device) Close() error {
	return nil
}

func (d *Device) conn(a bt.Addr) *conn {
	c, ok := d.conns[a.String()]
	if !ok {
		c = newConn(d, a)
		d.conns[a.String()] = c
	}
	return c
}

func (d *Device) sendReq(id int, args xpc.Dict) msg {
	d.sendCBMsg(id, args)
	return <-d.rspc
}

func (d *Device) sendCmd(id int, args xpc.Dict) error {
	d.sendCBMsg(id, args)
	return nil
}

func (d *Device) sendCBMsg(id int, args xpc.Dict) {
	// log.Printf("<< %d, %v", id, args)
	d.xpc.Send(xpc.Dict{"kCBMsgId": id, "kCBMsgArgs": args}, false)
}
