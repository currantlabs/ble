package hci

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/linux/hci/cmd"
	"github.com/currantlabs/ble/linux/hci/evt"
	"github.com/currantlabs/ble/linux/hci/skt"
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

type handlerFn func(b []byte) error

type pkt struct {
	cmd  Command
	done chan []byte
}

// NewHCI returns a hci device.
func NewHCI(opts ...ble.Option) (*HCI, error) {
	h := &HCI{
		id: -1,

		chCmdPkt:  make(chan *pkt),
		chCmdBufs: make(chan []byte, 8),
		sent:      make(map[int]*pkt),

		evth: map[int]handlerFn{},
		subh: map[int]handlerFn{},

		states: newStates(),

		chEvt:       make(chan []byte, 64),
		chStartScan: make(chan bool),

		muConns:      &sync.Mutex{},
		conns:        make(map[uint16]*Conn),
		chMasterConn: make(chan *Conn),
		chSlaveConn:  make(chan *Conn),

		done: make(chan bool),
	}

	return h, nil
}

// HCI ...
type HCI struct {
	sync.Mutex

	skt io.ReadWriteCloser
	id  int

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chCmdPkt  chan *pkt
	chCmdBufs chan []byte
	sent      map[int]*pkt

	// evtHub
	evth map[int]handlerFn
	subh map[int]handlerFn

	// aclHandler
	bufSize int
	bufCnt  int

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int

	states *states

	chEvt       chan []byte
	chStartScan chan bool
	chAdvEvt    chan []byte
	advHandler  ble.AdvHandler

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	pool *Pool

	// L2CAP connections
	muConns      *sync.Mutex
	conns        map[uint16]*Conn
	chMasterConn chan *Conn // Dial returns master connections.
	chSlaveConn  chan *Conn // Peripheral accept slave connections.

	dialerTmo   time.Duration
	listenerTmo time.Duration

	err  error
	done chan bool
}

// Init ...
func (h *HCI) Init(opts ...Option) error {
	h.evth[0x3E] = h.handleLEMeta
	h.evth[evt.CommandCompleteCode] = h.handleCommandComplete
	h.evth[evt.CommandStatusCode] = h.handleCommandStatus
	h.evth[evt.DisconnectionCompleteCode] = h.handleDisconnectionComplete
	h.evth[evt.NumberOfCompletedPacketsCode] = h.handleNumberOfCompletedPackets

	h.subh[evt.LEAdvertisingReportSubCode] = h.handleLEAdvertisingReport
	h.subh[evt.LEConnectionCompleteSubCode] = h.handleLEConnectionComplete
	h.subh[evt.LEConnectionUpdateCompleteSubCode] = h.handleLEConnectionUpdateComplete
	h.subh[evt.LELongTermKeyRequestSubCode] = h.handleLELongTermKeyRequest
	// evt.EncryptionChangeCode:                     todo),
	// evt.ReadRemoteVersionInformationCompleteCode: todo),
	// evt.HardwareErrorCode:                        todo),
	// evt.DataBufferOverflowCode:                   todo),
	// evt.EncryptionKeyRefreshCompleteCode:         todo),
	// evt.AuthenticatedPayloadTimeoutExpiredCode:   todo),
	// evt.LEReadRemoteUsedFeaturesCompleteSubCode:   todo),
	// evt.LERemoteConnectionParameterRequestSubCode: todo),

	if err := h.Option(opts...); err != nil {
		return err
	}

	skt, err := skt.NewSocket(h.id)
	if err != nil {
		return err
	}
	h.skt = skt

	h.chCmdBufs <- make([]byte, 64)

	go h.asyncLoop()
	go h.sktLoop()
	h.init()
	h.states.init(h)

	// Pre-allocate buffers with additional head room for lower layer headers.
	// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
	h.pool = NewPool(1+4+h.bufSize, h.bufCnt-1)

	return nil
}

// Stop ...
func (h *HCI) Stop() error {
	return h.stop(nil)
}

// Error ...
func (h *HCI) Error() error {
	return h.err
}

// Option sets the options specified.
func (h *HCI) Option(opts ...Option) error {
	var err error
	for _, opt := range opts {
		err = opt(h)
	}
	return err
}

func (h *HCI) init() error {
	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	h.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP)

	a := ReadBDADDRRP.BDADDR
	h.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

	ReadBufferSizeRP := cmd.ReadBufferSizeRP{}
	h.Send(&cmd.ReadBufferSize{}, &ReadBufferSizeRP)

	// Assume the buffers are shared between ACL-U and LE-U.
	h.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	h.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	h.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP)

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		h.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		h.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
	}

	LEReadAdvertisingChannelTxPowerRP := cmd.LEReadAdvertisingChannelTxPowerRP{}
	h.Send(&cmd.LEReadAdvertisingChannelTxPower{}, &LEReadAdvertisingChannelTxPowerRP)

	h.txPwrLv = int(LEReadAdvertisingChannelTxPowerRP.TransmitPowerLevel)

	LESetEventMaskRP := cmd.LESetEventMaskRP{}
	h.Send(&cmd.LESetEventMask{LEEventMask: 0x000000000000001F}, &LESetEventMaskRP)

	SetEventMaskRP := cmd.SetEventMaskRP{}
	h.Send(&cmd.SetEventMask{EventMask: 0x3dbff807fffbffff}, &SetEventMaskRP)

	WriteLEHostSupportRP := cmd.WriteLEHostSupportRP{}
	h.Send(&cmd.WriteLEHostSupport{LESupportedHost: 1, SimultaneousLEHost: 0}, &WriteLEHostSupportRP)

	return h.err
}

// Send ...
func (h *HCI) Send(c Command, r CommandRP) error {
	b, err := h.send(c)
	if err != nil {
		return err
	}
	if r != nil {
		return r.Unmarshal(b)
	}
	return nil
}

func (h *HCI) send(c Command) ([]byte, error) {
	if h.err != nil {
		return nil, h.err
	}
	p := &pkt{c, make(chan []byte)}
	b := <-h.chCmdBufs
	b[0] = byte(pktTypeCommand) // HCI header
	b[1] = byte(c.OpCode())
	b[2] = byte(c.OpCode() >> 8)
	b[3] = byte(c.Len())
	if err := c.Marshal(b[4:]); err != nil {
		h.stop(fmt.Errorf("hci: failed to marshal cmd"))
	}

	h.sent[c.OpCode()] = p // TODO: lock
	if n, err := h.skt.Write(b[:4+c.Len()]); err != nil {
		h.stop(fmt.Errorf("hci: failed to send cmd"))
	} else if n != 4+c.Len() {
		h.stop(fmt.Errorf("hci: failed to send whole cmd pkt to hci socket"))
	}

	select {
	case <-h.done:
		return nil, h.err
	case b := <-p.done:
		return b, nil
	}
}

func (h *HCI) asyncLoop() {
	var ad []*Advertisement
	var last int
	for {
		select {
		case <-h.done:
			return
		case b := <-h.chEvt:
			code, plen := int(b[0]), int(b[1])
			if plen != len(b[2:]) {
				h.err = fmt.Errorf("hci: corrupt event packet: [ % X ]", b)
			}
			if f := h.evth[code]; f != nil {
				h.err = f(b[2:])
			}
			// err = fmt.Errorf("hci: unsupported event packet: [ % X ]", b)
		case <-h.chStartScan:
			h.chAdvEvt = make(chan []byte, 16)
			ad = make([]*Advertisement, 16)
			last = 0
		case b := <-h.chAdvEvt:
			e := evt.LEAdvertisingReport(b)
			for i := 0; i < int(e.NumReports()); i++ {
				var a *Advertisement
				switch e.EventType(i) {
				case evtTypAdvInd:
					fallthrough
				case evtTypAdvScanInd:
					a = newAdvertisement(e, i)
					ad[last] = a
					last++
					if last == len(ad) {
						last = 0
					}
				case evtTypScanRsp:
					sr := newAdvertisement(e, i)
					for idx := last - 1; idx != last; idx-- {
						if idx == -1 {
							idx = len(ad) - 1
						}
						if ad[idx] == nil {
							break
						}
						if ad[idx].Address().String() == sr.Address().String() {
							ad[idx].setScanResponse(sr)
							a = ad[idx]
							break
						}
					}
				default:
					a = newAdvertisement(e, i)
				}
				// FIXME:
				go h.advHandler.Handle(a)
			}
		}
	}
}

func (h *HCI) sktLoop() {
	b := make([]byte, 4096)
	defer close(h.done)
	for {
		n, err := h.skt.Read(b)
		if n == 0 || err != nil {
			h.err = fmt.Errorf("skt: %s", err)
			return
		}
		p := make([]byte, n)
		copy(p, b)
		if err := h.handlePkt(p); err != nil {
			h.err = fmt.Errorf("skt: %s", err)
			return
		}
	}
}

func (h *HCI) stop(err error) error {
	if err := h.skt.Close(); err != nil {
		return err
	}
	return nil
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
		return h.handleEvt(b)
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
	if code == evt.CommandCompleteCode || code == evt.CommandStatusCode {
		if f := h.evth[code]; f != nil {
			return f(b[2:])
		}
	}
	h.chEvt <- b
	return nil
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
	select {
	case h.chAdvEvt <- evt.LEAdvertisingReport(b):
	default:
		// discard the advertisement.
	}
	return nil
}

func (h *HCI) handleCommandComplete(b []byte) error {
	e := evt.CommandComplete(b)
	for i := 0; i < int(e.NumHCICommandPackets()); i++ {
		h.chCmdBufs <- make([]byte, 64)
	}

	// NOP command, used for flow control purpose [Vol 2, Part E, 4.4]
	if e.CommandOpcode() == 0x0000 {
		h.chCmdBufs = make(chan []byte, 8)
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
		h.chCmdBufs <- make([]byte, 64)
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
		if e.Status() == 0x00 {
			h.chMasterConn <- c
		} else if ErrCommand(e.Status()) == ErrConnID {
			// nil for a canceled connection.
			h.chMasterConn <- nil
		}
		return nil
	}
	if e.Status() == 0x00 {
		h.chSlaveConn <- c
	}
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
	if c.param.Role() == roleSlave {
		h.states.set(CentralDisconnected)
	} else {
		h.states.set(PeripheralDisconnected)
	}
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
