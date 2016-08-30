//go:generate protoc -I proto/ proto/bled.proto --go_out=plugins=grpc:proto

package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/currantlabs/ble"
	pb "github.com/currantlabs/ble/bled/proto"
	"github.com/currantlabs/ble/examples/lib/dev"
	"github.com/currantlabs/ble/linux"
	"github.com/currantlabs/ble/linux/hci/cmd"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	defaultSocket = "/tmp/bled.sock"
	port          = ":50051"
)

var cnt int32

// server is used to implement helloworld.GreeterServer.
type server struct {
	sync.RWMutex

	scanners   map[pb.Bled_ScanServer]chan bool
	scannersMu sync.RWMutex
}

// NewCentral implements helloworld.GreeterServer
func (s *server) NewCentral(ctx context.Context, in *pb.DeviceOptions) (*pb.DeviceID, error) {
	cnt++
	return &pb.DeviceID{Id: cnt}, nil
}

func (s *server) Scan(in *pb.Dummy, stream pb.Bled_ScanServer) error {
	done := make(chan bool)
	s.scannersMu.Lock()
	s.scanners[stream] = done
	log.Printf("stream %#v", stream)
	if len(s.scanners) == 1 {
		if err := ble.Scan(true); err != nil {
			return errors.Wrap(err, "can't scan")
		}
	}
	s.scannersMu.Unlock()
	<-done

	return nil
}

func (s *server) StopScanning(ctx context.Context, dummy *pb.Dummy) (*pb.Dummy, error) {
	log.Printf("ctx %#v", ctx)
	// s.scannersMu.Lock()
	// done := s.scanners[stream]
	// delete(s.scanners, stream)
	// close(done)
	// defer s.scannersMu.Unlock()
	// if len(s.scanners) == 0 {
	// 	if err := ble.StopScanning(true); err != nil {
	// 		return errors.Wrap(err, "can't scan")
	// 	}
	// }

	return nil, nil
}

func (s *server) Handle(a ble.Advertisement) {
	s.scannersMu.Lock()
	defer s.scannersMu.Unlock()

	adv := marshalAdv(a)
	for stream := range s.scanners {
		if err := stream.Send(adv); err != nil {
			// done <- errors.Wrap(err, "scan")
			delete(s.scanners, stream)
			log.Printf("failed to send adv: %s", err)
		}
	}
	if len(s.scanners) == 0 {
		ble.StopScanning()
	}
}

func marshalAdv(a ble.Advertisement) *pb.Advertisement {
	return &pb.Advertisement{
		LocalName:        a.LocalName(),
		ManufacturerData: a.ManufacturerData(),
		// ServiceData      []*ServiceData,
		// Services         []*UUID      ,
		// OverflowService  []*UUID     ,
		TxPowerLevel: int32(a.TxPowerLevel()),
		Connectable:  false,
		// SolicitedService *UUID    ,
		RSSI:    int32(a.RSSI()),
		Address: a.Address().String(),
	}
}
func main() {
	d, err := dev.DefaultDevice()
	if err != nil {
		log.Fatalf("can't create default device: %s", err)
	}
	ble.SetDefaultDevice(d)
	if dev, ok := d.(*linux.Device); ok {
		if err = dev.HCI.Send(&cmd.LESetScanParameters{
			LEScanType:           0x01,   // 0x00: passive, 0x01: active
			LEScanInterval:       0x0004, // 0x0004 - 0x4000; N * 0.625msec
			LEScanWindow:         0x0004, // 0x0004 - 0x4000; N * 0.625msec
			OwnAddressType:       0x00,   // 0x00: public, 0x01: random
			ScanningFilterPolicy: 0x00,   // 0x00: accept all, 0x01: ignore non-white-listed.
		}, nil); err != nil {
			log.Fatalf("can't set advertising param: %s", err)
		}
	}

	// lis, err := net.Listen("tcp", port)
	lis, err := net.ListenUnix("unix", &net.UnixAddr{Name: defaultSocket, Net: "unix"})
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	cleanup := func() { os.Remove(defaultSocket) }
	defer cleanup()
	trap(cleanup)

	svr := &server{
		scanners: make(map[pb.Bled_ScanServer]chan bool),
	}
	if err := ble.SetAdvHandler(svr); err != nil {
		// return errors.Wrap(err, "can't set adv handler")
		log.Fatalf("can't set adv handler: %s", err)
	}
	s := grpc.NewServer()
	pb.RegisterBledServer(s, svr)
	s.Serve(lis)
}

// Cleanup for the interrupted case.
func trap(cleanup func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cleanup()
		os.Exit(1)
	}()
}
