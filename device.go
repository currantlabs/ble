package bt

import (
	"errors"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"github.com/currantlabs/bt/gioctl"
	"github.com/currantlabs/bt/socket"
)

const (
	ioctlSize     = uintptr(4)
	hciMaxDevices = 16
	typHCI        = 72 // 'H'
)

var (
	hciUpDevice      = gioctl.IoW(typHCI, 201, ioctlSize) // HCIDEVUP
	hciDownDevice    = gioctl.IoW(typHCI, 202, ioctlSize) // HCIDEVDOWN
	hciResetDevice   = gioctl.IoW(typHCI, 203, ioctlSize) // HCIDEVRESET
	hciGetDeviceList = gioctl.IoR(typHCI, 210, ioctlSize) // HCIGETDEVLIST
	hciGetDeviceInfo = gioctl.IoR(typHCI, 211, ioctlSize) // HCIGETDEVINFO
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
type device struct {
	fd   int
	dev  int
	name string
	rmu  *sync.Mutex
	wmu  *sync.Mutex
}

func newDevice(n int, chk bool) (*device, error) {
	fd, err := socket.Socket(socket.AF_BLUETOOTH, syscall.SOCK_RAW, socket.BTPROTO_HCI)
	if err != nil {
		return nil, err
	}
	if n != -1 {
		return newSocket(fd, n, chk)
	}

	req := devListRequest{devNum: hciMaxDevices}
	if err := gioctl.Ioctl(uintptr(fd), hciGetDeviceList, uintptr(unsafe.Pointer(&req))); err != nil {
		return nil, err
	}
	for i := 0; i < int(req.devNum); i++ {
		d, err := newSocket(fd, i, chk)
		if err == nil {
			log.Printf("dev: %s opened", d.name)
			return d, err
		}
	}
	return nil, errors.New("no supported devices available")
}

func newSocket(fd, n int, chk bool) (*device, error) {
	i := hciDevInfo{id: uint16(n)}
	if err := gioctl.Ioctl(uintptr(fd), hciGetDeviceInfo, uintptr(unsafe.Pointer(&i))); err != nil {
		return nil, err
	}
	name := string(i.name[:])
	// Check the feature list returned feature list.
	if chk && i.features[4]&0x40 == 0 {
		err := errors.New("does not support LE")
		log.Printf("dev: %s %s", name, err)
		return nil, err
	}
	log.Printf("dev: %s up", name)
	if err := gioctl.Ioctl(uintptr(fd), hciUpDevice, uintptr(n)); err != nil {
		if err != syscall.EALREADY {
			return nil, err
		}
		log.Printf("dev: %s reset", name)
		if err := gioctl.Ioctl(uintptr(fd), hciResetDevice, uintptr(n)); err != nil {
			return nil, err
		}
	}
	log.Printf("dev: %s down", name)
	if err := gioctl.Ioctl(uintptr(fd), hciDownDevice, uintptr(n)); err != nil {
		return nil, err
	}

	// Attempt to use the linux 3.14 feature, if this fails with EINVAL fall back to raw access
	// on older kernels.
	sa := socket.SockaddrHCI{Dev: n, Channel: socket.HCI_CHANNEL_USER}
	if err := socket.Bind(fd, &sa); err != nil {
		if err != syscall.EINVAL {
			return nil, err
		}
		log.Printf("dev: %s can't bind to hci user channel, err: %s.", name, err)
		sa := socket.SockaddrHCI{Dev: n, Channel: socket.HCI_CHANNEL_RAW}
		if err := socket.Bind(fd, &sa); err != nil {
			log.Printf("dev: %s can't bind to hci raw channel, err: %s.", name, err)
			return nil, err
		}
	}
	return &device{
		fd:   fd,
		dev:  n,
		name: name,
		rmu:  &sync.Mutex{},
		wmu:  &sync.Mutex{},
	}, nil
}

func (d *device) Read(b []byte) (int, error) {
	d.rmu.Lock()
	defer d.rmu.Unlock()
	return syscall.Read(d.fd, b)
}

func (d *device) Write(b []byte) (int, error) {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	return syscall.Write(d.fd, b)
}

func (d *device) Close() error {
	return syscall.Close(d.fd)
}
