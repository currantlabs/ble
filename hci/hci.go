package hci

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
	"github.com/currantlabs/bt/hci/skt"
)

// HCI Packet types
const (
	pktTypeCommand uint8 = 0x01
	pktTypeACLData uint8 = 0x02
	pktTypeSCOData uint8 = 0x03
	pktTypeEvent   uint8 = 0x04
	pktTypeVendor  uint8 = 0xFF
)

type hci struct {
	skt io.ReadWriteCloser
	cmd *cmdSender
	evt *evtHub
	acl *aclHandler

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int
}

// NewHCI ...
func NewHCI(id int) (HCI, error) {
	skt, err := skt.NewSocket(id)
	if err != nil {
		return nil, err
	}

	h := &hci{
		skt: skt,
		cmd: newCmdSender(skt),
		acl: newACLHandler(skt),
		evt: newEvtHub(),
	}

	h.SetEventHandler(evt.CommandCompleteCode, HandlerFunc(h.cmd.handleCommandComplete))
	h.SetEventHandler(evt.CommandStatusCode, HandlerFunc(h.cmd.handleCommandStatus))
	go h.loop()
	return h, h.init()
}

func (h *hci) Send(c Command, r CommandRP) error                        { return h.cmd.send(c, r) }
func (h *hci) SetEventHandler(c int, f Handler) Handler                 { return h.evt.SetEventHandler(c, f) }
func (h *hci) SetSubeventHandler(c int, f Handler) Handler              { return h.evt.SetSubeventHandler(c, f) }
func (h *hci) SetACLHandler(f Handler) (w io.Writer, size int, cnt int) { return h.acl.setACLHandler(f) }
func (h *hci) LocalAddr() net.HardwareAddr                              { return h.addr }
func (h *hci) Stop() error                                              { return h.skt.Close() }

func (h *hci) loop() {
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
			log.Printf("hci: %s", err)

		}
	}
}

func (h *hci) handlePkt(b []byte) error {
	// Strip the HCI header, and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case pktTypeCommand:
		return fmt.Errorf("hci: unmanaged cmd: [ % X ]", b)
	case pktTypeACLData:
		return h.acl.handle(b)
	case pktTypeSCOData:
		return fmt.Errorf("hci: unsupported sco packet: [ % X ]", b)
	case pktTypeEvent:
		return h.evt.handle(b)
	case pktTypeVendor:
		return fmt.Errorf("hci: unsupported vendor packet: [ % X ]", b)
	default:
		return fmt.Errorf("hci: invalid packet: 0x%02X [ % X ]", t, b)
	}
}

func (h *hci) init() error {
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
	ap := h.acl
	ap.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	ap.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	if err := h.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP); err != nil {
		return err
	}

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		ap.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		ap.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
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
