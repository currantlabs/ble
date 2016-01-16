package adv

import (
	"bytes"
	"encoding/binary"

	"github.com/currantlabs/bt/uuid"
)

// ServiceData ...
type ServiceData struct {
	UUID uuid.UUID
	Data []byte
}

// Utility function for creating a list of uuids.
func uuidList(u []uuid.UUID, d []byte, w int) []uuid.UUID {
	for len(d) > 0 {
		u = append(u, uuid.UUID(d[:w]))
		d = d[w:]
	}
	return u
}

func serviceDataList(sd []ServiceData, d []byte, w int) []ServiceData {
	serviceData := ServiceData{uuid.UUID(d[:w]), make([]byte, len(d)-w)}
	copy(serviceData.Data, d[2:])
	return append(sd, serviceData)
}

// Packet is an utility to help crafting advertisment or scan response data.
type Packet struct {
	*bytes.Buffer
	m map[byte][]byte
}

// NewAdvPacket ...
func NewAdvPacket(b []byte) *Packet {
	return &Packet{Buffer: bytes.NewBuffer(b)}
}

func (a *Packet) parse() {
	if a.m != nil {
		return
	}
	a.m = make(map[byte][]byte)
	b := a.Bytes()
	for len(b) > 0 {
		if len(b) < 2 {
			return
		}
		l, t := b[0], b[1]
		if len(b) < int(1+l) {
			return
		}
		a.m[t] = b[2 : 1+l]
		b = b[1+l:]
	}
}

// Field returns the field data (excluding the initial length and tag byte).
// It returns nil, if the specified field is not found.
func (a Packet) Field(tag byte) []byte {
	a.parse()
	return a.m[tag]
}

// Flags ...
func (a Packet) Flags() (byte, bool) {
	b := a.Field(Flags)
	if len(b) < 2 {
		return 0, false
	}
	return b[2], true
}

// LocalName ...
func (a Packet) LocalName() string {
	if b := a.Field(ShortName); b != nil {
		return string(b)
	}
	return string(a.Field(CompleteName))
}

// TxPower ...
func (a Packet) TxPower() (int, bool) {
	b := a.Field(TxPower)
	if len(b) < 3 {
		return 0, false
	}
	return int(int8(b[2])), true
}

// UUIDs ...
func (a Packet) UUIDs() []uuid.UUID {
	var u []uuid.UUID
	if b := a.Field(SomeUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := a.Field(AllUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := a.Field(SomeUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := a.Field(AllUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := a.Field(SomeUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := a.Field(AllUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceSol ...
func (a Packet) ServiceSol() []uuid.UUID {
	var u []uuid.UUID
	if b := a.Field(ServiceSol16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := a.Field(ServiceSol32); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := a.Field(ServiceSol128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceData ...
func (a Packet) ServiceData() []ServiceData {
	var s []ServiceData
	if b := a.Field(ServiceData16); b != nil {
		s = serviceDataList(s, b, 2)
	}
	if b := a.Field(ServiceData32); b != nil {
		s = serviceDataList(s, b, 4)
	}
	if b := a.Field(ServiceData128); b != nil {
		s = serviceDataList(s, b, 16)
	}
	return s
}

// Packet returns an advertising packet, which is an 31-byte array.
func (a Packet) Packet() [31]byte {
	b := [31]byte{}
	copy(b[:], a.Buffer.Bytes())
	return b
}

// ManufacturerData ...
func (a Packet) ManufacturerData() []byte {
	return a.Field(ManufacturerData)
}

// AppendField appends a BLE advertising packet field.
// TODO: refuse to append field if it'd make the packet too long.
func (a *Packet) AppendField(typ byte, b []byte) *Packet {
	// A field consists of len, typ, b.
	// Len is 1 byte for typ plus len(b).
	if a.Len()+2+len(b) > MaxEIRPacketLength {
		b = b[:MaxEIRPacketLength-a.Len()-2]
	}
	binary.Write(a, binary.LittleEndian, byte(len(b)+1))
	binary.Write(a, binary.LittleEndian, typ)
	binary.Write(a, binary.LittleEndian, b)
	return a
}

// AppendFlags appends a flag field to the packet.
func (a *Packet) AppendFlags(f byte) *Packet {
	return a.AppendField(Flags, []byte{f})
}

// AppendName appends a name field to the packet.
// If the name fits in the space, it will be append as a complete name field, otherwise a short name field.
func (a *Packet) AppendName(n string) *Packet {
	typ := byte(CompleteName)
	if a.Len()+2+len(n) > MaxEIRPacketLength {
		typ = byte(ShortName)
	}
	return a.AppendField(typ, []byte(n))
}

// AppendManufacturerData appends a manufacturer data field to the packet.
func (a *Packet) AppendManufacturerData(id uint16, b []byte) *Packet {
	d := append([]byte{uint8(id), uint8(id >> 8)}, b...)
	return a.AppendField(ManufacturerData, d)
}

// AppendUUIDFit appends a BLE advertised service UUID
// packet field if it fits in the packet, and reports whether the UUID fit.
func (a *Packet) AppendUUIDFit(uu []uuid.UUID) bool {
	// Iterate all UUIDs to see if they fit in the packet or not.
	fit, l := true, a.Len()
	for _, u := range uu {
		l += 2 + u.Len()
		if l > MaxEIRPacketLength {
			fit = false
			break
		}
	}

	// Append the UUIDs until they no longer fit.
	for _, u := range uu {
		if a.Len()+2+u.Len() > MaxEIRPacketLength {
			break
		}
		switch l = u.Len(); {
		case l == 2 && fit:
			a.AppendField(AllUUID16, u)
		case l == 16 && fit:
			a.AppendField(AllUUID128, u)
		case l == 2 && !fit:
			a.AppendField(SomeUUID16, u)
		case l == 16 && !fit:
			a.AppendField(SomeUUID128, u)
		}
	}
	return fit
}
