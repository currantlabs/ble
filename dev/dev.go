package dev

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/currantlabs/bt/dev/skt"
	"github.com/currantlabs/bt/hci"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
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

// Device ...
type Device interface {
	hci.HCI
	// LocalAddr returns the MAC address of local skt.
	LocalAddr() net.HardwareAddr

	// Stop closes the HCI socket.
	Stop() error
}

// HCI Packet types
const (
	pktTypeCommand uint8 = 0x01
	pktTypeACLData uint8 = 0x02
	pktTypeSCOData uint8 = 0x03
	pktTypeEvent   uint8 = 0x04
	pktTypeVendor  uint8 = 0xFF
)

type dev struct {
	skt io.ReadWriteCloser
	cmd *cmdSender
	evt *evtHub
	acl *aclHandler

	// Device information or status.
	addr    net.HardwareAddr
	txPwrLv int
}

// New ...
func New(id int) (Device, error) {
	skt, err := skt.NewSocket(id)
	if err != nil {
		return nil, err
	}

	d := &dev{
		skt: skt,
		cmd: newCmdSender(skt),
		acl: newACLHandler(skt),
		evt: newEvtHub(),
	}

	d.SetEventHandler(evt.CommandCompleteCode, hci.HandlerFunc(d.cmd.handleCommandComplete))
	d.SetEventHandler(evt.CommandStatusCode, hci.HandlerFunc(d.cmd.handleCommandStatus))
	go d.loop()
	return d, d.init()
}

func (d *dev) Send(c hci.Command, r hci.CommandRP) error        { return d.cmd.send(c, r) }
func (d *dev) SetEventHandler(c int, f hci.Handler) hci.Handler { return d.evt.SetEventHandler(c, f) }
func (d *dev) SetSubeventHandler(c int, f hci.Handler) hci.Handler {
	return d.evt.SetSubeventHandler(c, f)
}
func (d *dev) SetACLHandler(f hci.Handler) (w io.Writer, size int, cnt int) {
	return d.acl.setACLHandler(f)
}
func (d *dev) LocalAddr() net.HardwareAddr { return d.addr }
func (d *dev) Stop() error                 { return d.skt.Close() }

func (d *dev) loop() {
	b := make([]byte, 4096)
	for {
		n, err := d.skt.Read(b)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}
		p := make([]byte, n)
		copy(p, b)
		if err := d.handlePkt(p); err != nil {
			log.Printf("hci: %s", err)

		}
	}
}

func (d *dev) handlePkt(b []byte) error {
	// Strip the HCI header, and pass down the rest of the packet.
	t, b := b[0], b[1:]
	switch t {
	case pktTypeCommand:
		return fmt.Errorf("hci: unmanaged cmd: [ % X ]", b)
	case pktTypeACLData:
		return d.acl.handle(b)
	case pktTypeSCOData:
		return fmt.Errorf("hci: unsupported sco packet: [ % X ]", b)
	case pktTypeEvent:
		// return d.evt.handle(b)
		go d.evt.handle(b)
		return nil
	case pktTypeVendor:
		return fmt.Errorf("hci: unsupported vendor packet: [ % X ]", b)
	default:
		return fmt.Errorf("hci: invalid packet: 0x%02X [ % X ]", t, b)
	}
}

func (d *dev) init() error {
	ResetRP := cmd.ResetRP{}
	if err := d.Send(&cmd.Reset{}, &ResetRP); err != nil {
		return err
	}

	ReadBDADDRRP := cmd.ReadBDADDRRP{}
	if err := d.Send(&cmd.ReadBDADDR{}, &ReadBDADDRRP); err != nil {
		return err
	}
	a := ReadBDADDRRP.BDADDR
	d.addr = net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})

	ReadLocalSupportedCommandsRP := cmd.ReadLocalSupportedCommandsRP{}
	if err := d.Send(&cmd.ReadLocalSupportedCommands{}, &ReadLocalSupportedCommandsRP); err != nil {
		return err
	}

	ReadLocalSupportedFeaturesRP := cmd.ReadLocalSupportedFeaturesRP{}
	if err := d.Send(&cmd.ReadLocalSupportedFeatures{}, &ReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	ReadLocalVersionInformationRP := cmd.ReadLocalVersionInformationRP{}
	if err := d.Send(&cmd.ReadLocalVersionInformation{}, &ReadLocalVersionInformationRP); err != nil {
		return err
	}

	ReadBufferSizeRP := cmd.ReadBufferSizeRP{}
	if err := d.Send(&cmd.ReadBufferSize{}, &ReadBufferSizeRP); err != nil {
		return err
	}

	// Assume the buffers are shared between ACL-U and LE-U.
	ap := d.acl
	ap.bufCnt = int(ReadBufferSizeRP.HCTotalNumACLDataPackets)
	ap.bufSize = int(ReadBufferSizeRP.HCACLDataPacketLength)

	LEReadBufferSizeRP := cmd.LEReadBufferSizeRP{}
	if err := d.Send(&cmd.LEReadBufferSize{}, &LEReadBufferSizeRP); err != nil {
		return err
	}

	if LEReadBufferSizeRP.HCTotalNumLEDataPackets != 0 {
		// Okay, LE-U do have their own buffers.
		ap.bufCnt = int(LEReadBufferSizeRP.HCTotalNumLEDataPackets)
		ap.bufSize = int(LEReadBufferSizeRP.HCLEDataPacketLength)
	}

	LEReadLocalSupportedFeaturesRP := cmd.LEReadLocalSupportedFeaturesRP{}
	if err := d.Send(&cmd.LEReadLocalSupportedFeatures{}, &LEReadLocalSupportedFeaturesRP); err != nil {
		return err
	}

	LEReadSupportedStatesRP := cmd.LEReadSupportedStatesRP{}
	if err := d.Send(&cmd.LEReadSupportedStates{}, &LEReadSupportedStatesRP); err != nil {
		return err
	}

	LEReadAdvertisingChannelTxPowerRP := cmd.LEReadAdvertisingChannelTxPowerRP{}
	if err := d.Send(&cmd.LEReadAdvertisingChannelTxPower{}, &LEReadAdvertisingChannelTxPowerRP); err != nil {
		return err
	}
	d.txPwrLv = int(LEReadAdvertisingChannelTxPowerRP.TransmitPowerLevel)

	LESetEventMaskRP := cmd.LESetEventMaskRP{}
	if err := d.Send(&cmd.LESetEventMask{LEEventMask: 0x000000000000001F}, &LESetEventMaskRP); err != nil {
		return err
	}

	SetEventMaskRP := cmd.SetEventMaskRP{}
	if err := d.Send(&cmd.SetEventMask{EventMask: 0x3dbff807fffbffff}, &SetEventMaskRP); err != nil {
		return err
	}

	WriteLEHostSupportRP := cmd.WriteLEHostSupportRP{}
	if err := d.Send(&cmd.WriteLEHostSupport{LESupportedHost: 1, SimultaneousLEHost: 0}, &WriteLEHostSupportRP); err != nil {
		return err
	}

	WriteClassOfDeviceRP := cmd.WriteClassOfDeviceRP{}
	if err := d.Send(&cmd.WriteClassOfDevice{ClassOfDevice: [3]byte{0x40, 0x02, 0x04}}, &WriteClassOfDeviceRP); err != nil {
		return err
	}

	return nil
}
