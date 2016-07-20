package main

import (
	"flag"
	"log"
	"net"
	"strings"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib/gatt"
)

var (
	charPair = ble.NewCharacteristic(ble.MustParse("000102030405060708090A0B0C0D1914"))
	charCmd  = ble.NewCharacteristic(ble.MustParse("000102030405060708090A0B0C0D1912"))
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
	ble.Client

	cnt uint16
	mac []byte
	sk  []byte
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
	c, ok := b.Profile().Find(charCmd).(*ble.Characteristic)
	if !ok {
		log.Fatalf("can't find command char")
	}
	log.Printf("found command char %s, h: 0x%02X, vh: 0x%02X", c.UUID, c.Handle, c.ValueHandle)
	b.WriteCharacteristic(c, p, false)
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

	cln, err := gatt.Discover(gatt.MatcherFunc(match))
	p, err := cln.DiscoverProfile(true)
	if err != nil {
		log.Fatalf("can't discover profile: %s", err)
	}

	c, ok := p.Find(charPair).(*ble.Characteristic)
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
	cln.WriteCharacteristic(c, pkt, false)
	rsp, err := cln.ReadCharacteristic(c)
	if err != nil {
		log.Fatalf("can't read pair char: %s", err)
	}
	log.Printf("pairing rsp: % X", rsp)

	// generate session key.
	rsp = append([]byte{0x00}, rsp...) // preppend a byte of zero?
	sk := encrypt(key, append(data[:8], rsp[:8]...))
	log.Printf("sk: % X", sk)

	// read and reverse the mac address.
	mac, err := net.ParseMAC(cln.Address().String())
	if err != nil {
		log.Fatalf("can't parse mac (%s): %s", cln.Address().String(), err)
	}
	mac = []byte{mac[5], mac[4], mac[3], mac[2], mac[1], mac[0]}

	b := &bulb{
		Client: cln,
		mac:    mac,
		cnt:    2012,
		sk:     sk,
	}
	b.setState(0xff, 0x00, 0x00, 0x80)
	cln.CancelConnection()
}
