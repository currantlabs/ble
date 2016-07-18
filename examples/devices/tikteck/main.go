package main

import (
	"flag"
	"log"
	"net"
	"strings"

	"github.com/currantlabs/ble"
)

var (
	name = flag.String("name", "Smart Light", "name of remote peripheral")
	addr = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
)

func encPacket(sk []byte, m []byte, p []byte) []byte {
	t := encrypt(sk, pad([]byte{m[0], m[1], m[2], m[3], 0x01, p[0], p[1], p[2], 15}))

	for i := 0; i < 15; i++ {
		t[i] ^= p[i+5]
	}

	t = encrypt(sk, t)

	for i := 0; i < 2; i++ {
		p[i+3] = t[i]
	}

	t2 := encrypt(sk, pad([]byte{0, m[0], m[1], m[2], m[3], 0x01, p[0], p[1], p[2]}))
	for i := 0; i < 15; i++ {
		p[i+5] ^= t2[i]
	}
	return p
}

type bulb struct {
	cnt uint16
	mac []byte
	sk  []byte
	d   *discoverer
	cmd *ble.Characteristic
}

func (b *bulb) sendPacket(msgid uint16, cmd byte, data []byte) {
	p := make([]byte, 20)
	p[0] = byte(b.cnt & 0xff)
	p[1] = byte(b.cnt >> 8 & 0xff)
	p[5] = byte(msgid & 0xff)
	p[6] = byte(msgid&0xff | 0x80)
	p[7] = cmd
	p[8] = 0x69
	p[9] = 0x69
	p[10] = data[0]
	p[11] = data[1]
	p[12] = data[2]
	p[13] = data[3]
	b.cnt++
	log.Printf("packet: [ % X ]", p)
	ep := encPacket(b.sk, b.mac, p)
	log.Printf("enc_packet: [ % X ]", ep)
	b.send(ep)
}

func (b *bulb) send(p []byte) {
	c, ok := b.d.find(ble.NewCharacteristic(ble.MustParse("000102030405060708090A0B0C0D1912"))).(*ble.Characteristic)
	if !ok {
		log.Fatalf("can't find command char")
	}
	log.Printf("found command char %s, h: 0x%02X, vh: 0x%02X", c.UUID, c.Handle, c.ValueHandle)
	b.d.WriteCharacteristic(c, p, false)
}

func (b *bulb) setState(red, green, blue, brightness byte) {
	b.sendPacket(0xffff, 0xc1, []byte{red, green, blue, brightness})
}

func main() {
	flag.Parse()

	// Search the light bulb by name, or by address.
	match := func(a ble.Advertisement) bool {
		return strings.ToUpper(a.LocalName()) == strings.ToUpper(*name)
	}
	if len(*addr) != 0 {
		match = func(a ble.Advertisement) bool {
			return strings.ToUpper(a.Address().String()) == strings.ToUpper(*addr)
		}
	}

	d := newDiscoverer()

	// Connect to the light bulb, and perform service/characteristic discovery.
	if err := d.connect(match); err != nil {
		log.Fatalf("can't conect: %s", err)
	}
	_, err := d.discover()
	if err != nil {
		log.Fatalf("can't discover: %s", err)
	}

	// Service: 000102030405060708090A0B0C0D1910 , Handle (0x10)
	//   Characteristic: 000102030405060708090A0B0C0D1911, Property: 0x1A (RWN), , Handle(0x11), VHandle(0x12)
	//     Value         01 | "\x01"
	//     Descriptor: 2901, Characteristic User Description, Handle(0x13)
	//     Value         537461747573 | "Status"
	//
	//   Characteristic: 000102030405060708090A0B0C0D1912, Property: 0x0E (wWR), , Handle(0x14), VHandle(0x15)
	//     Value         00000000000000000000000000000000 | "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	//     Descriptor: 2901, Characteristic User Description, Handle(0x16)
	//     Value         436f6d6d616e64 | "Command"
	//
	//   Characteristic: 000102030405060708090A0B0C0D1913, Property: 0x06 (Rw), , Handle(0x17), VHandle(0x18)
	//     Value         e0000000000000000000000000000000 | "\xe0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	//     Descriptor: 2901, Characteristic User Description, Handle(0x19)
	//     Value         4f5441 | "OTA"
	//
	//   Characteristic: 000102030405060708090A0B0C0D1914, Property: 0x0A (RW), , Handle(0x1A), VHandle(0x1B)
	//     Value         007857b58d2af70eb269e149192262b66b | "\x00xW\xb5\x8d*\xf7\x0e\xb2i\xe1I\x19\"b\xb6k"
	//     Descriptor: 2901, Characteristic User Description, Handle(0x1c)
	//     Value         50616972 | "Pair"
	//
	// Service: 19200D0C0B0A09080706050403020100 , Handle (0x1D)
	//   Characteristic: 19210D0C0B0A09080706050403020100, Property: 0x0A (RW), , Handle(0x1E), VHandle(0x1F)
	//     Value         d0000000000000000000000000000000 | "\xd0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	//     Descriptor: 2901, Characteristic User Description, Handle(0x20)
	//     Value         534c434d44 | "SLCMD"

	c, ok := d.find(ble.NewCharacteristic(ble.MustParse("000102030405060708090A0B0C0D1914"))).(*ble.Characteristic)
	if !ok {
		log.Fatalf("can't find pair char: %s", err)
	}
	log.Printf("found pair char %s, h: 0x%02X, vh: 0x%02X", c.UUID, c.Handle, c.ValueHandle)

	// generate pairing request.
	key := genKey([]byte("Smart Light"), []byte("339829388"))
	data := []byte{00, 01, 02, 03, 04, 05, 06, 07, 00, 00, 00, 00, 00, 00, 00, 00}
	req := encrypt(data, key)
	log.Printf("pairing req: % X", req)

	pkt := []byte{0x0c}
	pkt = append(pkt, data[:8]...)
	pkt = append(pkt, req[:8]...)
	log.Printf("pairing pkt: % X", pkt)
	d.WriteCharacteristic(c, pkt, false)
	rsp, err := d.ReadCharacteristic(c)
	if err != nil {
		log.Fatalf("can't read pair char: %s", err)
	}
	log.Printf("pairing rsp: % X", rsp)

	// generate session key.
	rsp = append([]byte{0x00}, rsp...) // preppend a byte of zero?
	sk := encrypt(key, append(data[:8], rsp[:8]...))
	log.Printf("sk: % X", sk)

	// read and reverse the mac address.
	mac, err := net.ParseMAC(d.Address().String())
	if err != nil {
		log.Fatalf("can't parse mac (%s): %s", d.Address().String(), err)
	}
	mac = []byte{mac[5], mac[4], mac[3], mac[2], mac[1], mac[0]}

	b := &bulb{
		mac: mac,
		cnt: 2012,
		sk:  sk,
		d:   d,
	}
	b.setState(0xff, 0x00, 0x00, 0x80)
	d.CancelConnection()
}
