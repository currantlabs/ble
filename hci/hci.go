package hci

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
	"github.com/currantlabs/bt/hci/skt"
	"github.com/currantlabs/bt/l2cap"
)

// HCI Packet types
const (
	pktTypeCommand uint8 = 0x01
	pktTypeACLData uint8 = 0x02
	pktTypeSCOData uint8 = 0x03
	pktTypeEvent   uint8 = 0x04
	pktTypeVendor  uint8 = 0xFF
)

type pkt struct {
	cmd  bt.Command
	done chan []byte
}

// HCI ...
type HCI struct {
	sync.Mutex
	skt io.ReadWriteCloser

	// cmdSender
	sent  map[int]*pkt
	chPkt chan *pkt

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chBufs chan []byte

	// evtHub
	evth map[int]bt.Handler
	subh map[int]bt.Handler

	// aclHandler
	bufSize    int
	bufCnt     int
	aclHandler bt.Handler

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int

	// broadcaster
	advParams cmd.LESetAdvertisingParameters
	advData   cmd.LESetAdvertisingData
	scanResp  cmd.LESetScanResponseData

	// observer
	scanParams cmd.LESetScanParameters

	advFilter  bt.AdvFilter
	advHandler bt.AdvHandler

	// peripheral & central
	l2cap.LE
}

// Init ...
func (h *HCI) Init(id int) error {
	skt, err := skt.NewSocket(id)
	if err != nil {
		return err
	}
	h.skt = skt

	h.chPkt = make(chan *pkt)
	h.chBufs = make(chan []byte, 8)
	h.sent = make(map[int]*pkt)

	h.evth = map[int]bt.Handler{}
	h.subh = map[int]bt.Handler{}

	h.scanParams = cmd.LESetScanParameters{
		LEScanType:           0x01,   // [0x00]: passive, 0x01: active
		LEScanInterval:       0x0010, // [0x10]: 0.625ms * 16
		LEScanWindow:         0x0010, // [0x10]: 0.625ms * 16
		OwnAddressType:       0x00,   // [0x00]: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // [0x00]: accept all, 0x01: ignore non-white-listed.
	}

	h.advParams = cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingIntervalMax:  0x010,     // [0x0800]: 0.625 ms * 0x0800 = 1280.0 ms
		AdvertisingType:         0x00,      // [0x00]: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // [0x00]: public, 0x01: random
		DirectAddressType:       0x00,      // [0x00]: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // [0x07] 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}

	// Register our own advertising report advHandler.
	h.SetEventHandler(0x3E, bt.HandlerFunc(h.handleLEMeta))
	h.SetSubeventHandler(evt.LEAdvertisingReportSubCode, bt.HandlerFunc(h.handleLEAdvertisingReport))
	h.SetEventHandler(evt.CommandCompleteCode, bt.HandlerFunc(h.handleCommandComplete))
	h.SetEventHandler(evt.CommandStatusCode, bt.HandlerFunc(h.handleCommandStatus))
	// evt.EncryptionChangeCode:                     bt.HandlerFunc(todo),
	// evt.ReadRemoteVersionInformationCompleteCode: bt.HandlerFunc(todo),
	// evt.HardwareErrorCode:                        bt.HandlerFunc(todo),
	// evt.DataBufferOverflowCode:                   bt.HandlerFunc(todo),
	// evt.EncryptionKeyRefreshCompleteCode:         bt.HandlerFunc(todo),
	// evt.AuthenticatedPayloadTimeoutExpiredCode:   bt.HandlerFunc(todo),
	// evt.LEReadRemoteUsedFeaturesCompleteSubCode:   bt.HandlerFunc(todo),
	// evt.LERemoteConnectionParameterRequestSubCode: bt.HandlerFunc(todo),
	go h.cmdLoop()
	go h.mainLoop()
	h.init()
	h.LE.Init(h)
	return nil
}

// SetACLHandler ...
func (h *HCI) SetACLHandler(f bt.Handler) (w io.Writer, size int, cnt int) {
	h.aclHandler = f
	return h.skt, h.bufSize, h.bufCnt
}

// LocalAddr ...
func (h *HCI) LocalAddr() bt.Addr { return h.addr }

// Stop ...
func (h *HCI) Stop() error { return h.skt.Close() }

func (h *HCI) init() error {
	ResetRP := cmd.ResetRP{}
	if err := h.Send(&cmd.Reset{}, &ResetRP); err != nil {
		return err
	}

	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	if err := h.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP); err != nil {
		return err
	}
	a := ReadBDADDRRP.BDADDR
	h.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

	ReadLocalSupportedCommandsRP := cmd.ReadLocalSupportedCommandsRP{}
	if err := h.Send(&cmd.ReadLocalSupportedCommands{}, &ReadLocalSupportedCommandsRP); err != nil {
		return err
	}

	ReadLocalSupportedFeaturesRP := cmd.ReadLocalSupportedFeaturesRP{}
	if err := h.Send(&cmd.ReadLocalSupportedFeatures{}, &ReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	ReadLocalVersionInformationRP := cmd.ReadLocalVersionInformationRP{}
	if err := h.Send(&cmd.ReadLocalVersionInformation{}, &ReadLocalVersionInformationRP); err != nil {
		return err
	}

	ReadBufferSizeRP := cmd.ReadBufferSizeRP{}
	if err := h.Send(&cmd.ReadBufferSize{}, &ReadBufferSizeRP); err != nil {
		return err
	}

	// Assume the buffers are shared between ACL-U and LE-U.
	h.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	h.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	if err := h.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP); err != nil {
		return err
	}

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		h.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		h.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
	}

	LEReadLocalSupportedFeaturesRP := cmd.LEReadLocalSupportedFeaturesRP{}
	if err := h.Send(&cmd.LEReadLocalSupportedFeatures{}, &LEReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	LEReadSupportedStatesRP := cmd.LEReadSupportedStatesRP{}
	if err := h.Send(&cmd.LEReadSupportedStates{}, &LEReadSupportedStatesRP); err != nil {
		return err
	}

	LEReadAdvertisingChannelTxPowerRP := cmd.LEReadAdvertisingChannelTxPowerRP{}
	if err := h.Send(&cmd.LEReadAdvertisingChannelTxPower{}, &LEReadAdvertisingChannelTxPowerRP); err != nil {
		return err
	}
	h.txPwrLv = int(LEReadAdvertisingChannelTxPowerRP.TransmitPowerLevel)

	LESetEventMaskRP := cmd.LESetEventMaskRP{}
	if err := h.Send(&cmd.LESetEventMask{LEEventMask: 0x000000000000001F}, &LESetEventMaskRP); err != nil {
		return err
	}

	SetEventMaskRP := cmd.SetEventMaskRP{}
	if err := h.Send(&cmd.SetEventMask{EventMask: 0x3dbff807fffbffff}, &SetEventMaskRP); err != nil {
		return err
	}

	WriteLEHostSupportRP := cmd.WriteLEHostSupportRP{}
	if err := h.Send(&cmd.WriteLEHostSupport{LESupportedHost: 1, SimultaneousLEHost: 0}, &WriteLEHostSupportRP); err != nil {
		return err
	}

	WriteClassOfDeviceRP := cmd.WriteClassOfDeviceRP{}
	if err := h.Send(&cmd.WriteClassOfDevice{ClassOfDevice: [3]byte{0x40, 0x02, 0x04}}, &WriteClassOfDeviceRP); err != nil {
		return err
	}

	return nil
}

// EventHandler ...
func (h *HCI) EventHandler(c int) bt.Handler {
	h.Lock()
	defer h.Unlock()
	return h.evth[c]
}

// SetEventHandler ...
func (h *HCI) SetEventHandler(c int, f bt.Handler) bt.Handler {
	h.Lock()
	defer h.Unlock()
	old := h.evth[c]
	h.evth[c] = f
	return old
}

// SubeventHandler ...
func (h *HCI) SubeventHandler(c int) bt.Handler {
	h.Lock()
	defer h.Unlock()
	return h.subh[c]
}

// SetSubeventHandler ...
func (h *HCI) SetSubeventHandler(c int, f bt.Handler) bt.Handler {
	h.Lock()
	defer h.Unlock()
	old := h.subh[c]
	h.subh[c] = f
	return old
}

// Send ...
func (h *HCI) Send(c bt.Command, r bt.CommandRP) error {
	p := &pkt{c, make(chan []byte)}
	h.chPkt <- p
	b := <-p.done
	if r == nil {
		return nil
	}
	return r.Unmarshal(b)
}

func (h *HCI) cmdLoop() {
	h.chBufs <- make([]byte, 64)
	for p := range h.chPkt {
		b := <-h.chBufs
		c := p.cmd
		b[0] = byte(pktTypeCommand) // HCI header
		b[1] = byte(c.OpCode())
		b[2] = byte(c.OpCode() >> 8)
		b[3] = byte(c.Len())
		if err := c.Marshal(b[4:]); err != nil {
			log.Printf("hci: failed to marshal cmd")
			return
		}

		h.sent[c.OpCode()] = p // TODO: lock
		if n, err := h.skt.Write(b[:4+c.Len()]); err != nil {
			log.Printf("hci: failed to send cmd")
		} else if n != 4+c.Len() {
			log.Printf("hci: failed to send whole cmd pkt to hci socket")
		}
	}
}

func (h *HCI) mainLoop() {
	b := make([]byte, 4096)
	for {
		n, err := h.skt.Read(b)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}
		p := make([]byte, n)
		copy(p, b)
		if err := h.handlePkt(p); err != nil {
			log.Printf("HCI: %s", err)

		}
	}
}

func (h *HCI) handlePkt(b []byte) error {
	// Strip the HCI header, and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case pktTypeCommand:
		return fmt.Errorf("HCI: unmanaged cmd: [ % X ]", b)
	case pktTypeACLData:
		return h.handleACL(b)
	case pktTypeSCOData:
		return fmt.Errorf("HCI: unsupported sco packet: [ % X ]", b)
	case pktTypeEvent:
		// return h.evt.handleEvt(b)
		go h.handleEvt(b)
		return nil
	case pktTypeVendor:
		return fmt.Errorf("HCI: unsupported vendor packet: [ % X ]", b)
	default:
		return fmt.Errorf("HCI: invalid packet: 0x%02X [ % X ]", t, b)
	}
}

func (h *HCI) handleACL(b []byte) error {
	if h.aclHandler == nil {
		return fmt.Errorf("hci: unhandled ACL packet: % X", b)
	}
	return h.aclHandler.Handle(b)
}

func (h *HCI) handleEvt(b []byte) error {
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		return fmt.Errorf("hci: corrupt event packet: [ % X ]", b)
	}
	if f := h.EventHandler(code); f != nil {
		return f.Handle(b[2:])
	}
	return fmt.Errorf("hci: unsupported event packet: [ % X ]", b)
}

func (h *HCI) handleLEMeta(b []byte) error {
	subcode := int(b[0])
	if f := h.SubeventHandler(subcode); f != nil {
		return f.Handle(b)
	}
	return fmt.Errorf("hci: unsupported LE event: [ % X ]", b)
}

func (h *HCI) handleLEAdvertisingReport(b []byte) error {
	if h.advHandler == nil {
		return nil
	}
	e := evt.LEAdvertisingReport(b)
	for i := 0; i < int(e.NumReports()); i++ {
		a := bt.NewAdvertisement(e, i)
		if h.advFilter != nil && h.advFilter.Filter(*a) {
			go h.advHandler.Handle(*a)
		}
	}
	return nil
}

func (h *HCI) handleCommandComplete(b []byte) error {
	e := evt.CommandComplete(b)

	for i := 0; i < int(e.NumHCICommandPackets()); i++ {
		h.chBufs <- make([]byte, 64)
	}

	// NOP command, used for flow control purpose [Vol 2, Part E, 4.4]
	if e.CommandOpcode() == 0x0000 {
		return nil
	}
	p, found := h.sent[int(e.CommandOpcode())]
	if !found {
		return fmt.Errorf("hci: can't find the cmd for CommandCompleteEP: % X", e)
	}
	p.done <- e.ReturnParameters()
	return nil
}

func (h *HCI) handleCommandStatus(b []byte) error {
	e := evt.CommandStatus(b)

	for i := 0; i < int(e.NumHCICommandPackets()); i++ {
		h.chBufs <- make([]byte, 64)
	}

	p, found := h.sent[int(e.CommandOpcode())]
	if !found {
		return fmt.Errorf("hci: can't find the cmd for CommandStatusEP: % X", e)
	}
	close(p.done)
	return nil
}
