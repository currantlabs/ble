// +build linux

package skt

import (
	"errors"
	"io"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func ioR(t, nr, size uintptr) uintptr {
	return (2 << 30) | (t << 8) | nr | (size << 16)
}

func ioW(t, nr, size uintptr) uintptr {
	return (1 << 30) | (t << 8) | nr | (size << 16)
}

func ioctl(fd, op, arg uintptr) error {
	if _, _, ep := unix.Syscall(unix.SYS_IOCTL, fd, op, arg); ep != 0 {
		return syscall.Errno(ep)
	}
	return nil
}

const (
	ioctlSize     = 4
	hciMaxDevices = 16
	typHCI        = 72 // 'H'
)

var (
	hciUpDevice      = ioW(typHCI, 201, ioctlSize) // HCIDEVUP
	hciDownDevice    = ioW(typHCI, 202, ioctlSize) // HCIDEVDOWN
	hciResetDevice   = ioW(typHCI, 203, ioctlSize) // HCIDEVRESET
	hciGetDeviceList = ioR(typHCI, 210, ioctlSize) // HCIGETDEVLIST
	hciGetDeviceInfo = ioR(typHCI, 211, ioctlSize) // HCIGETDEVINFO
)

type devRequest struct {
	id  uint16
	opt uint32
}

type devListRequest struct {
	devNum     uint16
	devRequest [hciMaxDevices]devRequest
}

type hciDevInfo struct {
	id         uint16
	name       [8]byte
	bdaddr     [6]byte
	flags      uint32
	devType    uint8
	features   [8]uint8
	pktType    uint32
	linkPolicy uint32
	linkMode   uint32
	aclMtu     uint16
	aclPkts    uint16
	scoMtu     uint16
	scoPkts    uint16

	stats hciDevStats
}

type hciDevStats struct {
	errRx  uint32
	errTx  uint32
	cmdTx  uint32
	evtRx  uint32
	aclTx  uint32
	aclRx  uint32
	scoTx  uint32
	scoRx  uint32
	byteRx uint32
	byteTx uint32
}

type skt struct {
	fd   int
	dev  int
	name string
	rmu  *sync.Mutex
	wmu  *sync.Mutex
}

// NewSocket ...
func NewSocket(n int) (io.ReadWriteCloser, error) {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return nil, err
	}
	if n != -1 {
		return open(fd, n)
	}

	req := devListRequest{devNum: hciMaxDevices}
	if err := ioctl(uintptr(fd), hciGetDeviceList, uintptr(unsafe.Pointer(&req))); err != nil {
		return nil, err
	}
	for i := 0; i < int(req.devNum); i++ {
		s, err := open(fd, i)
		if err == nil {
			log.Printf("dev: %s opened", s.name)
			return s, err
		}
	}
	return nil, errors.New("no supported devices available")
}

func open(fd, n int) (*skt, error) {
	i := hciDevInfo{id: uint16(n)}
	if err := ioctl(uintptr(fd), hciGetDeviceInfo, uintptr(unsafe.Pointer(&i))); err != nil {
		return nil, err
	}
	name := string(i.name[:])
	log.Printf("dev: %s up", name)
	if err := ioctl(uintptr(fd), hciUpDevice, uintptr(n)); err != nil {
		if err != unix.EALREADY {
			return nil, err
		}
		log.Printf("dev: %s reset", name)
		if err := ioctl(uintptr(fd), hciResetDevice, uintptr(n)); err != nil {
			return nil, err
		}
	}
	log.Printf("dev: %s down", name)
	if err := ioctl(uintptr(fd), hciDownDevice, uintptr(n)); err != nil {
		return nil, err
	}

	sa := unix.SockaddrHCI{Dev: uint16(n), Channel: unix.HCI_CHANNEL_USER}
	if err := unix.Bind(fd, &sa); err != nil {
		log.Printf("dev: %s can't bind to hci user channel, err: %s.", name, err)
		return nil, err
	}
	return &skt{
		fd:   fd,
		dev:  n,
		name: name,
		rmu:  &sync.Mutex{},
		wmu:  &sync.Mutex{},
	}, nil
}

func (s *skt) Read(b []byte) (int, error) {
	s.rmu.Lock()
	defer s.rmu.Unlock()
	return unix.Read(s.fd, b)
}

func (s *skt) Write(b []byte) (int, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return unix.Write(s.fd, b)
}

func (s *skt) Close() error {
	return unix.Close(s.fd)
}
