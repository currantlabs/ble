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
)

// Command ...
type Command interface {
	OpCode() int
	Len() int
	Marshal([]byte) error
}

// CommandRP ...
type CommandRP interface {
	Unmarshal(b []byte) error
}

// CommandSender ...
type CommandSender interface {
	// Send sends a HCI Command and returns unserialized return parameter.
	Send(Command, CommandRP) error
}

type handlerFn func(b []byte) error

type pkt struct {
	cmd  Command
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
	evth map[int]handlerFn
	subh map[int]handlerFn

	// aclHandler
	bufSize int
	bufCnt  int

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int

	advParams  cmd.LESetAdvertisingParameters
	advData    cmd.LESetAdvertisingData
	scanResp   cmd.LESetScanResponseData
	scanParams cmd.LESetScanParameters
	connParams cmd.LECreateConnection

	advFilter  bt.AdvFilter
	advHandler bt.AdvHandler

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	pool *Pool

	// L2CAP connections
	muConns      *sync.Mutex
	conns        map[uint16]*Conn
	chMasterConn chan *Conn
	chSlaveConn  chan *Conn
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

	h.evth = map[int]handlerFn{}
	h.subh = map[int]handlerFn{}

	h.scanParams = cmd.LESetScanParameters{
		LEScanType:           0x01,   // 0x00: passive, 0x01: active
		LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public, 0x01: random
		ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
	}

	h.advParams = cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingIntervalMax:  0x0020,    // 0x0020 - 0x4000; N * 0.625 msec
		AdvertisingType:         0x00,      // 00: ADV_IND, 0x01: DIRECT(HIGH), 0x02: SCAN, 0x03: NONCONN, 0x04: DIRECT(LOW)
		OwnAddressType:          0x00,      // 0x00: public, 0x01: random
		DirectAddressType:       0x00,      // 0x00: public, 0x01: random
		DirectAddress:           [6]byte{}, // Public or Random Address of the Device to be connected
		AdvertisingChannelMap:   0x7,       // 0x07 0x01: ch37, 0x2: ch38, 0x4: ch39
		AdvertisingFilterPolicy: 0x00,
	}

	h.connParams = cmd.LECreateConnection{
		LEScanInterval:        0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		LEScanWindow:          0x0004,    // 0x0004 - 0x4000; N * 0.625 msec
		InitiatorFilterPolicy: 0x00,      // White list is not used
		PeerAddressType:       0x00,      // Public Device Address
		PeerAddress:           [6]byte{}, //
		OwnAddressType:        0x00,      // Public Device Address
		ConnIntervalMin:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnIntervalMax:       0x0006,    // 0x0006 - 0x0C80; N * 1.25 msec
		ConnLatency:           0x0000,    // 0x0000 - 0x01F3; N * 1.25 msec
		SupervisionTimeout:    0x0048,    // 0x000A - 0x0C80; N * 10 msec
		MinimumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
		MaximumCELength:       0x0000,    // 0x0000 - 0xFFFF; N * 0.625 msec
	}

	h.evth[0x3E] = h.handleLEMeta
	h.evth[evt.CommandCompleteCode] = h.handleCommandComplete
	h.evth[evt.CommandStatusCode] = h.handleCommandStatus
	h.evth[evt.DisconnectionCompleteCode] = h.handleDisconnectionComplete
	h.evth[evt.NumberOfCompletedPacketsCode] = h.handleNumberOfCompletedPackets

	h.subh[evt.LEAdvertisingReportSubCode] = h.handleLEAdvertisingReport
	h.subh[evt.LEConnectionCompleteSubCode] = h.handleLEConnectionComplete
	h.subh[evt.LEConnectionUpdateCompleteSubCode] = h.handleLEConnectionUpdateComplete
	h.subh[evt.LELongTermKeyRequestSubCode] = h.handleLELongTermKeyRequest
	// evt.EncryptionChangeCode:                     bt.todo),
	// evt.ReadRemoteVersionInformationCompleteCode: bt.todo),
	// evt.HardwareErrorCode:                        bt.todo),
	// evt.DataBufferOverflowCode:                   bt.todo),
	// evt.EncryptionKeyRefreshCompleteCode:         bt.todo),
	// evt.AuthenticatedPayloadTimeoutExpiredCode:   bt.todo),
	// evt.LEReadRemoteUsedFeaturesCompleteSubCode:   bt.todo),
	// evt.LERemoteConnectionParameterRequestSubCode: bt.todo),
	go h.cmdLoop()
	go h.mainLoop()
	h.init()

	h.muConns = &sync.Mutex{}
	h.conns = make(map[uint16]*Conn)
	h.chMasterConn = make(chan *Conn) // Peripheral accepts master connection
	h.chSlaveConn = make(chan *Conn)

	// Pre-allocate buffers with additional head room for lower layer headers.
	// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
	h.pool = NewPool(1+4+h.bufSize, h.bufCnt)

	return nil
}

// Stop ...
func (h *HCI) Stop() error { return h.skt.Close() }

// Accept returns a L2CAP master connection.
func (h *HCI) Accept() (bt.Conn, error) {
	return <-h.chSlaveConn, nil
}

// Close ...
func (h *HCI) Close() error {
	return nil
}

// Addr ...
func (h *HCI) Addr() bt.Addr { return h.addr }

// Dial ...
func (h *HCI) Dial(a bt.Addr) (bt.Conn, error) {
	b, ok := a.(net.HardwareAddr)
	if !ok {
		return nil, fmt.Errorf("invalid addr")
	}
	h.connParams.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	h.Send(&h.connParams, nil)
	c := <-h.chMasterConn
	return c, nil
}

func (h *HCI) init() error {
	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	if err := h.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP); err != nil {
		return err
	}
	a := ReadBDADDRRP.BDADDR
	h.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

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

	return nil
}

// Send ...
func (h *HCI) Send(c Command, r CommandRP) error {
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
	h.muConns.Lock()
	c, ok := h.conns[packet(b).handle()]
	h.muConns.Unlock()
	if !ok {
		return fmt.Errorf("l2cap: incoming packet for non-existing connection")
	}
	c.chInPkt <- b
	return nil
}

func (h *HCI) handleEvt(b []byte) error {
	code, plen := int(b[0]), int(b[1])
	if plen != len(b[2:]) {
		return fmt.Errorf("hci: corrupt event packet: [ % X ]", b)
	}
	if f := h.evth[code]; f != nil {
		return f(b[2:])
	}
	return fmt.Errorf("hci: unsupported event packet: [ % X ]", b)
}

func (h *HCI) handleLEMeta(b []byte) error {
	subcode := int(b[0])
	if f := h.subh[subcode]; f != nil {
		return f(b)
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

func (h *HCI) handleLEConnectionComplete(b []byte) error {
	e := evt.LEConnectionComplete(b)
	c := newConn(h, e)
	h.muConns.Lock()
	h.conns[e.ConnectionHandle()] = c
	h.muConns.Unlock()
	if e.Role() == roleMaster {
		h.chMasterConn <- c
		return nil
	}
	h.chSlaveConn <- c
	return nil
}

func (h *HCI) handleLEConnectionUpdateComplete(b []byte) error {
	return nil
}

func (h *HCI) handleDisconnectionComplete(b []byte) error {
	e := evt.DisconnectionComplete(b)
	h.muConns.Lock()
	c, found := h.conns[e.ConnectionHandle()]
	delete(h.conns, e.ConnectionHandle())
	h.muConns.Unlock()
	if !found {
		return fmt.Errorf("l2cap: disconnecting an invalid handle %04X", e.ConnectionHandle())
	}
	close(c.chInPkt)

	// When a connection disconnects, all the sent packets and weren't acked yet
	// will be recylecd. [Vol2, Part E 4.1.1]
	c.txBuffer.PutAll()
	return nil
}

func (h *HCI) handleNumberOfCompletedPackets(b []byte) error {
	e := evt.NumberOfCompletedPackets(b)
	h.muConns.Lock()
	defer h.muConns.Unlock()
	for i := 0; i < int(e.NumberOfHandles()); i++ {
		c, found := h.conns[e.ConnectionHandle(i)]
		if !found {
			continue
		}

		// Put the delivered buffers back to the pool.
		for j := 0; j < int(e.HCNumOfCompletedPackets(i)); j++ {
			c.txBuffer.Put()
		}
	}
	return nil
}

func (h *HCI) handleLELongTermKeyRequest(b []byte) error {
	e := evt.LELongTermKeyRequest(b)
	return h.Send(&cmd.LELongTermKeyRequestNegativeReply{
		ConnectionHandle: e.ConnectionHandle(),
	}, nil)
}
