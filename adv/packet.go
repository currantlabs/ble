package adv

import (
	"encoding/binary"

	"github.com/currantlabs/bt/uuid"
)

// Packet is an utility to craft or parse advertisment packets.
// Refer to Supplement to Bluetooth Core Specification | CSSv6, Part A
type Packet []byte

// Field returns the field data (excluding the initial length and typ byte).
// It returns nil, if the specified field is not found.
func (p Packet) Field(typ byte) []byte {
	b := p
	for len(b) > 0 {
		if len(b) < 2 {
			return nil
		}
		l, t := b[0], b[1]
		if len(b) < int(1+l) {
			return nil
		}
		if t == typ {
			return b[2 : 2+l-1]
		}
		b = b[1+l:]
	}
	return nil
}

// Flags ...
func (p Packet) Flags() (byte, bool) {
	b := p.Field(Flags)
	if len(b) < 2 {
		return 0, false
	}
	return b[2], true
}

// LocalName ...
func (p Packet) LocalName() string {
	if b := p.Field(ShortName); b != nil {
		return string(b)
	}
	return string(p.Field(CompleteName))
}

// TxPower ...
func (p Packet) TxPower() (int, bool) {
	b := p.Field(TxPower)
	if len(b) < 3 {
		return 0, false
	}
	return int(int8(b[2])), true
}

// UUIDs ...
func (p Packet) UUIDs() []uuid.UUID {
	var u []uuid.UUID
	if b := p.Field(SomeUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(AllUUID16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(SomeUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := p.Field(AllUUID32); b != nil {
		u = uuidList(u, b, 4)
	}
	if b := p.Field(SomeUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := p.Field(AllUUID128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceSol ...
func (p Packet) ServiceSol() []uuid.UUID {
	var u []uuid.UUID
	if b := p.Field(ServiceSol16); b != nil {
		u = uuidList(u, b, 2)
	}
	if b := p.Field(ServiceSol32); b != nil {
		u = uuidList(u, b, 16)
	}
	if b := p.Field(ServiceSol128); b != nil {
		u = uuidList(u, b, 16)
	}
	return u
}

// ServiceData ...
func (p Packet) ServiceData() []ServiceData {
	var s []ServiceData
	if b := p.Field(ServiceData16); b != nil {
		s = serviceDataList(s, b, 2)
	}
	if b := p.Field(ServiceData32); b != nil {
		s = serviceDataList(s, b, 4)
	}
	if b := p.Field(ServiceData128); b != nil {
		s = serviceDataList(s, b, 16)
	}
	return s
}

// ManufacturerData ...
func (p Packet) ManufacturerData() []byte {
	return p.Field(ManufacturerData)
}

// AppendField appends p BLE advertising packet field.
func (p Packet) AppendField(typ byte, b []byte) Packet {
	p = append(p, byte(len(b)+1))
	p = append(p, typ)
	return append(p, b...)
}

// AppendFlags appends p flag field to the packet.
func (p Packet) AppendFlags(f byte) Packet {
	return p.AppendField(Flags, []byte{f})
}

// AppendShortName appends p name field to the packet.
func (p Packet) AppendShortName(n string) Packet {
	return p.AppendField(ShortName, []byte(n))
}

// AppendCompleteName appends p name field to the packet.
func (p Packet) AppendCompleteName(n string) Packet {
	return p.AppendField(CompleteName, []byte(n))
}

// AppendManufacturerData appends p manufacturer data field to the packet.
func (p Packet) AppendManufacturerData(id uint16, b []byte) Packet {
	d := append([]byte{uint8(id), uint8(id >> 8)}, b...)
	return p.AppendField(ManufacturerData, d)
}

// AppendAllUUID appends p BLE advertised service UUID
func (p Packet) AppendAllUUID(u uuid.UUID) Packet {
	if u.Len() == 2 {
		return p.AppendField(AllUUID16, u)
	}
	if u.Len() == 4 {
		return p.AppendField(AllUUID32, u)
	}
	return p.AppendField(AllUUID128, u)
}

// AppendSomeUUID appends p BLE advertised service UUID
func (p Packet) AppendSomeUUID(u uuid.UUID) Packet {
	if u.Len() == 2 {
		return p.AppendField(SomeUUID16, u)
	}
	if u.Len() == 4 {
		return p.AppendField(SomeUUID32, u)
	}
	return p.AppendField(SomeUUID128, u)
}

// Data ...
func (p Packet) Data() [MaxEIRPacketLength]byte {
	b := [MaxEIRPacketLength]byte{}
	copy(b[:], p)
	return b
}

// Len ...
func (p Packet) Len() int {
	return len(p)
}

// ServiceData ...
type ServiceData struct {
	UUID uuid.UUID
	Data []byte
}

// Utility function for creating p list of uuids.
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

// IBeaconFromData returns an iBeacon advertisement with specified manufacturer data.
func IBeaconFromData(md []byte) Packet {
	if len(md) != 23 {
		return nil
	}
	p := Packet(make([]byte, 0, MaxEIRPacketLength))
	p = p.AppendFlags(FlagGeneralDiscoverable | FlagLEOnly)
	p = p.AppendManufacturerData(0x004C, md)
	return p
}

// IBeacon returns an iBeacon advertisement with specified parameters.
func IBeacon(u uuid.UUID, major, minor uint16, pwr int8) Packet {
	if u.Len() != 16 {
		return nil
	}
	md := make([]byte, 23)
	md[0] = 0x02                               // Data type: iBeacon
	md[1] = 0x15                               // Data length: 21 bytes
	copy(md[2:], uuid.Reverse(u))              // Big endian
	binary.BigEndian.PutUint16(md[18:], major) // Big endian
	binary.BigEndian.PutUint16(md[20:], minor) // Big endian
	md[22] = uint8(pwr)                        // Measured Tx Power
	return IBeaconFromData(md)
}
