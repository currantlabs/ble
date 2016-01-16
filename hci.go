package bt

import (
	"io"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/currantlabs/bt/cmd"
	"github.com/currantlabs/bt/evt"
)

// HCI ...
type HCI interface {
	cmd.Sender
	EventHandler

	// Accept returns L2CAP connection.
	Accept() (Conn, error)

	// LocalAddr returns the MAC address of local device.
	LocalAddr() net.HardwareAddr

	// Close stop the device.
	Close() error
}

type hci struct {
	dev io.ReadWriteCloser

	sentCmds map[int]*cmdPkt
	chCmdPkt chan *cmdPkt

	// Host to Controller command flow control [Vol 2, Part E, 4.4]
	chCmdBufs chan []byte

	// HCI Event handling
	evtHanlders    *dispatcher
	subevtHandlers *dispatcher

	// L2CAP (LE-U logical link) handling

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	bufCnt  int
	bufSize int // Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	chBufs  chan []byte

	// L2CAP connections
	muConns *sync.Mutex
	conns   map[uint16]*conn

	muACL  *sync.Mutex
	chConn chan *conn
	mps    int // Maximum PDU Payload Size (MPS)

	// Device information or status
	addr    net.HardwareAddr
	txPwrLv int
}

// NewHCI ...
func NewHCI(devID int, chk bool) (HCI, error) {
	dev, err := newDevice(devID, chk)
	if err != nil {
		return nil, err
	}

	h := &hci{
		dev: dev,

		chCmdPkt:  make(chan *cmdPkt),
		chCmdBufs: make(chan []byte, 8),
		sentCmds:  make(map[int]*cmdPkt),

		muConns: &sync.Mutex{},
		conns:   map[uint16]*conn{},

		muACL:  &sync.Mutex{},
		chConn: make(chan *conn),

		// Currently, we only supports BLE, and the sole user is ATT/GATT.
		// For BLE, the ATT_MTU has a default (and mandantory minimum) of 23 bytes,
		// a maximum of 512 bytes.
		mps: 512,
	}

	todo := func(b []byte) {
		log.Errorf("hci: unhandled (TODO) event packet: [ % X ]", b)
	}

	h.evtHanlders = &dispatcher{
		handlers: map[int]Handler{
			evt.DisconnectionCompleteEvent{}.Code():                HandlerFunc(h.handleDisconnectionComplete),
			evt.EncryptionChangeEvent{}.Code():                     HandlerFunc(todo),
			evt.ReadRemoteVersionInformationCompleteEvent{}.Code(): HandlerFunc(todo),
			evt.CommandCompleteEvent{}.Code():                      HandlerFunc(h.handleCommandComplete),
			evt.CommandStatusEvent{}.Code():                        HandlerFunc(h.handleCommandStatus),
			evt.HardwareErrorEvent{}.Code():                        HandlerFunc(todo),
			evt.NumberOfCompletedPacketsEvent{}.Code():             HandlerFunc(h.handleNumberOfCompletedPackets),
			evt.DataBufferOverflowEvent{}.Code():                   HandlerFunc(todo),
			evt.EncryptionKeyRefreshCompleteEvent{}.Code():         HandlerFunc(todo),
			0x3E: HandlerFunc(h.handleLEMeta), // FIMXE: ugliness
			evt.AuthenticatedPayloadTimeoutExpiredEvent{}.Code(): HandlerFunc(todo),
		},
	}

	h.subevtHandlers = &dispatcher{
		handlers: map[int]Handler{
			evt.LEConnectionCompleteEvent{}.SubCode():               HandlerFunc(h.handleLEConnectionComplete),
			evt.LEAdvertisingReportEvent{}.SubCode():                HandlerFunc(h.handleLEAdvertisingReport),
			evt.LEConnectionUpdateCompleteEvent{}.SubCode():         HandlerFunc(todo),
			evt.LEReadRemoteUsedFeaturesCompleteEvent{}.SubCode():   HandlerFunc(todo),
			evt.LELongTermKeyRequestEvent{}.SubCode():               HandlerFunc(todo),
			evt.LERemoteConnectionParameterRequestEvent{}.SubCode(): HandlerFunc(todo),
		},
	}

	go h.mainLoop()
	go h.cmdLoop()
	h.chCmdBufs <- make([]byte, 64)

	h.Init()

	return h, nil
}

// A Handler handles an HCI event or incoming ACL Data packets.
type Handler interface {
	Handle([]byte)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as packet or event handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(b []byte)

// Handle handles an event or ACLData packet.
func (f HandlerFunc) Handle(b []byte) {
	f(b)
}

// EventHandler ...
type EventHandler interface {
	// SetEventHandler registers the handler to handle the HCI event, and returns current handler.
	SetEventHandler(c int, h Handler) Handler

	// SetSubeventHandler registers the handler to handle the HCI subevent, and returns current handler.
	SetSubeventHandler(c int, h Handler) Handler
}

// SetEventHandler registers the handler to handle the hci event, and returns current handler.
func (h *hci) SetEventHandler(c int, f Handler) Handler {
	return h.evtHanlders.SetHandler(c, f)
}

// SetSubeventHandler registers the handler to handle the hci subevent, and returns current handler.
func (h *hci) SetSubeventHandler(c int, f Handler) Handler {
	return h.subevtHandlers.SetHandler(c, f)
}

// LocalAddr ...
func (h *hci) LocalAddr() net.HardwareAddr {
	return net.HardwareAddr([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
}

// Close ...
func (h *hci) Close() error {
	return h.dev.Close()
}

// Send sends a hci Command and returns unserialized return parameter.
func (h *hci) Send(c cmd.Command, r cmd.CommandRP) error {
	p := &cmdPkt{c, make(chan []byte)}
	h.chCmdPkt <- p
	b := <-p.done
	if r == nil {
		return nil
	}
	return r.Unmarshal(b)
}

type cmdPkt struct {
	cmd  cmd.Command
	done chan []byte
}

func (h *hci) cmdLoop() {
	for p := range h.chCmdPkt {
		b := <-h.chCmdBufs
		c := p.cmd
		b[0] = byte(pktTypeCommand) // HCI header
		b[1] = byte(c.OpCode())
		b[2] = byte(c.OpCode() >> 8)
		b[3] = byte(c.Len())
		if err := c.Marshal(b[4:]); err != nil {
			log.Errorf("hci: failed to marshal cmd")
			return
		}

		h.sentCmds[c.OpCode()] = p // TODO: lock
		if n, err := h.dev.Write(b[:4+c.Len()]); err != nil {
			log.Errorf("hci: failed to send cmd")
		} else if n != 4+c.Len() {
			log.Errorf("hci: failed to send whole cmd pkt to hci socket")
		}
	}
}

func (h *hci) mainLoop() {
	b := make([]byte, 4096)
	for {
		n, err := h.dev.Read(b)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}
		p := make([]byte, n)
		copy(p, b)
		h.handlePkt(p)
	}
}

func (h *hci) handlePkt(b []byte) {
	// Strip the HCI header, and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case pktTypeCommand:
		log.Errorf("hci: unmanaged cmd: [ % X ]", b)
	case pktTypeACLData:
		h.handleACLData(b)
	case pktTypeSCOData:
		log.Errorf("hci: unsupported sco packet: [ % X ]", b)
	case pktTypeEvent:
		go h.evtHanlders.dispatch(b)
	case pktTypeVendor:
		log.Errorf("hci: unsupported vendor packet: [ % X ]", b)
	default:
		log.Errorf("hci: invalid packet: 0x%02X [ % X ]", t, b)
	}
}

// Init ...
func (h *hci) Init() error {

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

	// Pre-allocate buffers with additional head room for lower layer headers.
	h.chBufs = make(chan []byte, h.bufCnt)
	for len(h.chBufs) < h.bufCnt {
		// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
		h.chBufs <- make([]byte, 1+4+h.bufSize)
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
