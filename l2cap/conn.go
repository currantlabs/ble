package l2cap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/currantlabs/bt/hci/evt"
)

type conn struct {
	l *le

	param evt.LEConnectionComplete

	// Maximum Transmission Unit (MTU) is the maximum size of payload data
	// which the upper layer entity is capable of accepting. [Vol 3, Part A, 1.4]
	// For LE-U logical transport, the L2CAP implementations should support
	// a minimum of 23 bytes, which are also the default values before the
	// upper layer (ATT) optionally reconfigures them [Vol 3, Part A, 3.2.8].
	rxMTU int
	txMTU int

	// Maximum PDU payload Size (MPS) is the maximum size of payload data
	// which the L2CAP layer entity is capable of accepting.
	// When segmantation is not used, the MPS should be made to the same
	// values of MTUs [Vol 3, Part A, 1.4].
	rxMPS int
	txMPS int

	// Signaling MTUs are The maximum size of command information that the
	// L2CAP layer entity is capable of accepting.
	// A L2CAP implementations supporting LE-U should support at least 23 bytes.
	// Currently, we support 512 bytes, which should be more than sufficient.
	// The sigTxMTU is discovered via when we sent a signaling pkt that is
	// larger thean the remote device can handle, and get a response of "Command
	// Reject" indicating "Signaling MTU exceeded" along with the actual
	// signaling MTU [Vol 3, Part A, 4.1].
	sigRxMTU int
	sigTxMTU int

	// sigID is used to match responses with signaling requests.
	// The requesting device sets this field and the responding device uses the
	// same value in its response. Within each signalling channel a different
	// Identifier shall be used for each successive command. [Vol 3, Part A, 4]
	sigID uint8

	sigSent chan []byte

	chInPkt chan packet
	chInPDU chan pdu

	// leFrame is set to be true when the LE Credit based flow control is used.
	leFrame bool

	// Host to Controller Data Flow Control pkt-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// chSentBufs tracks the HCI buffer occupied by this connection.
	txBuffer *Client

	chDone chan bool
}

func newConn(l *le, param evt.LEConnectionComplete) *conn {
	c := &conn{
		l:     l,
		param: param,

		rxMTU: 23,
		txMTU: 23,

		rxMPS: 23,
		txMPS: 23,

		sigRxMTU: 512,
		sigTxMTU: 23,

		chInPkt: make(chan packet, 16),
		chInPDU: make(chan pdu, 16),

		txBuffer: NewClient(l.pool),

		chDone: make(chan bool),
	}

	go func() {
		for {
			if err := c.recombine(); err != nil {
				if err != io.EOF {
					log.Printf("recombine failed: %s", err)
				}
				close(c.chInPDU)
				return
			}
		}
	}()
	return c
}

// Read copies re-assembled L2CAP PDUs into sdu.
func (c *conn) Read(sdu []byte) (int, error) {
	p, ok := <-c.chInPDU
	if !ok || len(p) == 0 {
		return 0, io.ErrUnexpectedEOF
	}

	// Assume it's a B-Frame.
	slen := p.dlen()
	data := p.payload()
	if c.leFrame {
		// LE-Frame.
		slen = leFrameHdr(p).slen()
		data = leFrameHdr(p).payload()
	}
	if cap(sdu) < slen {
		return 0, io.ErrShortBuffer
	}
	buf := bytes.NewBuffer(sdu)
	buf.Reset()
	buf.Write(data)
	for buf.Len() < slen {
		p := <-c.chInPDU
		buf.Write(pdu(p).payload())
	}
	// log.Printf("Read(): %d [ % X ]", slen, sdu[:slen])
	return slen, nil
}

// Write breaks down a L2CAP SDU into segmants [Vol 3, Part A, 7.3.1]
func (c *conn) Write(sdu []byte) (int, error) {
	if len(sdu) > c.txMTU {
		return 0, io.ErrShortWrite
	}
	if len(sdu) > c.txMPS && !c.leFrame {
		return 0, io.ErrShortWrite
	}

	plen := len(sdu)
	if plen > c.txMPS {
		plen = c.txMPS
	}
	b := make([]byte, 4+plen)
	binary.LittleEndian.PutUint16(b[0:2], uint16(len(sdu)))
	binary.LittleEndian.PutUint16(b[2:4], cidLEAtt)
	if c.leFrame {
		binary.LittleEndian.PutUint16(b[4:6], uint16(len(sdu)))
		copy(b[6:], sdu)
	} else {
		copy(b[4:], sdu)
	}
	sent, err := c.writePDU(cidLEAtt, b)
	if err != nil {
		return sent, err
	}
	sdu = sdu[plen:]

	for len(sdu) > 0 {
		plen := len(sdu)
		if plen > c.txMPS {
			plen = c.txMPS
		}
		n, err := c.writePDU(cidLEAtt, sdu[:plen])
		sent += n
		if err != nil {
			return sent, err
		}
		sdu = sdu[plen:]
	}
	return sent, nil
}

// writePDU breaks down a L2CAP PDU into fragments if it's larger than the HCI buffer size. [Vol 3, Part A, 7.2.1]
func (c *conn) writePDU(cid uint16, pdu []byte) (int, error) {
	sent := 0
	flags := uint16(pbfHostToControllerStart << 4) // ACL boundary flags

	// All L2CAP fragments associated with an L2CAP PDU shall be processed for
	// transmission by the Controller before any other L2CAP PDU for the same
	// logical transport shall be processed.
	c.txBuffer.LockPool()
	defer c.txBuffer.UnlockPool()

	for len(pdu) > 0 {
		// Get a buffer from our pre-allocated and flow-controlled pool.
		pkt := c.txBuffer.Get() // ACL pkt
		flen := len(pdu)        // fragment length
		if flen > pkt.Cap()-1-4 {
			flen = pkt.Cap() - 1 - 4
		}

		// Prepare the Headers
		binary.Write(pkt, binary.LittleEndian, uint8(pktTypeACLData))                         // HCI Header: pkt Type
		binary.Write(pkt, binary.LittleEndian, uint16(c.param.ConnectionHandle()|(flags<<8))) // ACL Header: handle and flags
		binary.Write(pkt, binary.LittleEndian, uint16(flen))                                  // ACL Header: data len
		binary.Write(pkt, binary.LittleEndian, pdu[:flen])                                    // Append payload

		// Flush the pkt to HCI
		select {
		case <-c.chDone:
			return 0, io.ErrClosedPipe
		default:
		}

		if _, err := c.l.pktWriter.Write(pkt.Bytes()); err != nil {
			return sent, err
		}
		sent += flen

		flags = (pbfContinuing << 4) // Set "continuing" in the boundary flags for the rest of fragments, if any.
		pdu = pdu[flen:]             // Advence the point
	}
	return sent, nil
}

// Recombines fragments into a L2CAP PDU. [Vol 3, Part A, 7.2.2]
func (c *conn) recombine() error {
	pkt, ok := <-c.chInPkt
	if !ok {
		return io.EOF
	}

	p := pdu(pkt.data())

	// Currently, check for LE-U only. For channels that we don't recognizes,
	// re-combine them anyway, and discard them later when we dispatch the PDU
	// according to CID.
	if p.cid() == cidLEAtt && p.dlen() > c.rxMPS {
		return fmt.Errorf("fragment size (%d) larger than rxMPS (%d)", p.dlen(), c.rxMPS)
	}

	// If this pkt is not a complete PDU, and we'll be receiving more
	// fragments, re-allocate the whole PDU (including Header).
	if len(p.payload()) < p.dlen() {
		p = make([]byte, 0, 4+p.dlen())
		p = append(p, pdu(pkt.data())...)
	}
	for len(p) < 4+p.dlen() {
		if pkt, ok = <-c.chInPkt; !ok || (pkt.pbf()&pbfContinuing) == 0 {
			return io.ErrUnexpectedEOF
		}
		p = append(p, pdu(pkt.data())...)
	}

	// TODO: support dynamic or assigned channels for LE-Frames.
	switch p.cid() {
	case cidLEAtt:
		c.chInPDU <- p
	case cidLESignal:
		c.handleSignal(p)
	case cidLESMP:
		// TODO: Security Manager Protocol
	default:
		log.Printf("recombine(): unrecognized CID: 0x%04X, [%X]", p.cid(), p)
	}
	return nil
}

// Close disconnects the connection by sending hci disconnect command to the device.
func (c *conn) Close() error {
	close(c.chDone)
	return nil
}

// LocalAddr returns local device's MAC address.
func (c *conn) LocalAddr() net.HardwareAddr {
	return c.l.hci.LocalAddr()
}

// RemoteAddr returns remote device's MAC address.
func (c *conn) RemoteAddr() net.HardwareAddr {
	a := c.param.PeerAddress()
	return net.HardwareAddr([]byte{a[5], a[4], a[3], a[2], a[1], a[0]})
}

// RxMTU returns the MTU which the upper layer is capable of accepting.
func (c *conn) RxMTU() int { return c.rxMTU }

// SetRxMTU sets the MTU which the upper layer of remote device is capable of accepting.
func (c *conn) SetRxMTU(mtu int) {
	c.rxMTU = mtu
	c.rxMPS = mtu
}

// TxMTU returns the MTU which the upper layer of remote device is capable of accepting.
func (c *conn) TxMTU() int { return c.txMTU }

// SetTxMTU sets the MTU which the upper layer is capable of accepting.
func (c *conn) SetTxMTU(mtu int) {
	log.Printf("Set MTU: %d", mtu)
	c.txMTU = mtu
	c.txMPS = mtu
}

// Parameters ...
func (c *conn) Parameters() evt.LEConnectionComplete {
	return c.param
}

// pkt implements HCI ACL Data Packet [Vol 2, Part E, 5.4.2]
// Packet boundary flags , bit[5:6] of handle field's MSB
// Broadcast flags. bit[7:8] of handle field's MSB
// Not used in LE-U. Leave it as 0x00 (Point-to-Point).
// Broadcasting in LE uses ADVB logical transport.
type packet []byte

func (a packet) handle() uint16 { return uint16(a[0]) | (uint16(a[1]&0x0f) << 8) }
func (a packet) pbf() int       { return (int(a[1]) >> 4) & 0x3 }
func (a packet) bcf() int       { return (int(a[1]) >> 6) & 0x3 }
func (a packet) dlen() int      { return int(a[2]) | (int(a[3]) << 8) }
func (a packet) data() []byte   { return a[4:] }

type pdu []byte

func (p pdu) dlen() int       { return int(binary.LittleEndian.Uint16(p[0:2])) }
func (p pdu) cid() uint16     { return binary.LittleEndian.Uint16(p[2:4]) }
func (p pdu) payload() []byte { return p[4:] }

type leFrameHdr pdu

func (f leFrameHdr) slen() int       { return int(binary.LittleEndian.Uint16(f[4:6])) }
func (f leFrameHdr) payload() []byte { return f[6:] }
