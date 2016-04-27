package l2cap

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/currantlabs/bt"
	"github.com/currantlabs/bt/hci/cmd"
	"github.com/currantlabs/bt/hci/evt"
)

// LE implements L2CAP (LE-U logical link) handling
type LE struct {
	hci       bt.HCI
	pktWriter io.Writer

	// Host to Controller Data Flow Control Packet-based Data flow control for LE-U [Vol 2, Part E, 4.1.1]
	// Minimum 27 bytes. 4 bytes of L2CAP Header, and 23 bytes Payload from upper layer (ATT)
	pool *Pool

	// L2CAP connections
	muConns      *sync.Mutex
	conns        map[uint16]*Conn
	chMasterConn chan *Conn
	chSlaveConn  chan *Conn

	connParam *cmd.LECreateConnection
}

// Init ...
func (l *LE) Init(h bt.HCI) error {
	l.hci = h

	l.muConns = &sync.Mutex{}
	l.conns = make(map[uint16]*Conn)
	l.chMasterConn = make(chan *Conn) // Peripheral accepts master connection
	l.chSlaveConn = make(chan *Conn)

	// LECreateConnection implements LE Create Connection (0x08|0x000D) [Vol 2, Part E, 7.8.12]
	// TODO: allow users to overrite the default values
	l.connParam = &cmd.LECreateConnection{
		LEScanInterval:        0x0004,    // N * 0.625 msec
		LEScanWindow:          0x0004,    // N * 0.625 msec
		InitiatorFilterPolicy: 0x00,      // White list is not used
		PeerAddressType:       0x00,      // Public Device Address
		PeerAddress:           [6]byte{}, //
		OwnAddressType:        0x00,      // Public Device Address
		ConnIntervalMin:       0x0006,    // N * 1.25 msec
		ConnIntervalMax:       0x0006,    // N * 1.25 msec
		ConnLatency:           0x0000,    // N * 1.25 msec
		SupervisionTimeout:    0x0048,    // N * 10 msec
		MinimumCELength:       0x0000,    // N * 0.625 msec
		MaximumCELength:       0x0000,    // N * 0.625 msec
	}

	// Pre-allocate buffers with additional head room for lower layer headers.
	// HCI header (1 Byte) + ACL Data Header (4 bytes) + L2CAP PDU (or fragment)
	w, size, cnt := h.SetACLHandler(bt.HandlerFunc(l.handlePacket))
	l.pktWriter = w
	l.pool = NewPool(1+4+size, cnt)

	h.SetEventHandler(evt.DisconnectionCompleteCode, bt.HandlerFunc(l.handleDisconnectionComplete))
	h.SetEventHandler(evt.NumberOfCompletedPacketsCode, bt.HandlerFunc(l.handleNumberOfCompletedPackets))

	h.SetSubeventHandler(evt.LEConnectionCompleteSubCode, bt.HandlerFunc(l.handleLEConnectionComplete))
	h.SetSubeventHandler(evt.LEConnectionUpdateCompleteSubCode, bt.HandlerFunc(l.handleLEConnectionUpdateComplete))
	h.SetSubeventHandler(evt.LELongTermKeyRequestSubCode, bt.HandlerFunc(l.handleLELongTermKeyRequest))

	return nil
}

// Accept returns a L2CAP master connection.
func (l *LE) Accept() (bt.Conn, error) {
	return <-l.chSlaveConn, nil
}

// Close ...
func (l *LE) Close() error {
	// TODO: implement HCI reference counting.
	return nil
}

// Addr ...
func (l *LE) Addr() bt.Addr {
	return l.hci.LocalAddr()
}

// Dial ...
func (l *LE) Dial(a bt.Addr) (bt.Conn, error) {
	cmd := *l.connParam
	b, ok := a.(net.HardwareAddr)
	if !ok {
		return nil, fmt.Errorf("invalid addr")
	}
	cmd.PeerAddress = [6]byte{b[5], b[4], b[3], b[2], b[1], b[0]}
	l.hci.Send(&cmd, nil)
	c := <-l.chMasterConn
	return c, nil
}

func (l *LE) handlePacket(b []byte) error {
	l.muConns.Lock()
	c, ok := l.conns[packet(b).handle()]
	l.muConns.Unlock()
	if !ok {
		return fmt.Errorf("l2cap: incoming packet for non-existing connection")
	}
	c.chInPkt <- b
	return nil
}

func (l *LE) handleLEConnectionComplete(b []byte) error {
	e := evt.LEConnectionComplete(b)
	c := newConn(l, e)
	l.muConns.Lock()
	l.conns[e.ConnectionHandle()] = c
	l.muConns.Unlock()
	if e.Role() == roleMaster {
		l.chMasterConn <- c
		return nil
	}
	l.chSlaveConn <- c
	return nil
}

func (l *LE) handleLEConnectionUpdateComplete(b []byte) error {
	// TODO: anything todo?
	return nil
}

func (l *LE) handleDisconnectionComplete(b []byte) error {
	e := evt.DisconnectionComplete(b)
	l.muConns.Lock()
	c, found := l.conns[e.ConnectionHandle()]
	delete(l.conns, e.ConnectionHandle())
	l.muConns.Unlock()
	if !found {
		return fmt.Errorf("l2cap: disconnecting an invalid handle %04X", e.ConnectionHandle())
	}
	close(c.chInPkt)

	// When a connection disconnects, all the sent packets and weren't acked yet
	// will be recylecd. [Vol2, Part E 4.1.1]
	c.txBuffer.PutAll()
	return nil
}

func (l *LE) handleNumberOfCompletedPackets(b []byte) error {
	e := evt.NumberOfCompletedPackets(b)
	l.muConns.Lock()
	defer l.muConns.Unlock()
	for i := 0; i < int(e.NumberOfHandles()); i++ {
		c, found := l.conns[e.ConnectionHandle(i)]
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

func (l *LE) handleLELongTermKeyRequest(b []byte) error {
	e := evt.LELongTermKeyRequest(b)
	return l.hci.Send(&cmd.LELongTermKeyRequestNegativeReply{
		ConnectionHandle: e.ConnectionHandle(),
	}, nil)
}
