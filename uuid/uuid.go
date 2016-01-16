package uuid

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// A UUID is a BLE UUID.
type UUID []byte

// UUID16 converts a uint16 (such as 0x1800) to a UUID.
func UUID16(i uint16) UUID {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, i)
	return UUID(b)
}

// Parse parses a standard-format UUID string, such
// as "1800" or "34DA3AD1-7110-41A1-B1EF-4430F509CDE7".
func Parse(s string) (UUID, error) {
	s = strings.Replace(s, "-", "", -1)
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if err := lenErr(len(b)); err != nil {
		return nil, err
	}
	return UUID(Reverse(b)), nil
}

// MustParse parses a standard-format UUID string,
// like Parse, but panics in case of error.
func MustParse(s string) UUID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// lenErr returns an error if n is an invalid UUID length.
func lenErr(n int) error {
	switch n {
	case 2, 16:
		return nil
	}
	return fmt.Errorf("UUIDs must have length 2 or 16, got %d", n)
}

// Len returns the length of the UUID, in bytes.
// BLE UUIDs are either 2 or 16 bytes.
func (u UUID) Len() int { return len(u) }

// String hex-encodes a UUID.
func (u UUID) String() string { return fmt.Sprintf("%X", Reverse(u)) }

// Equal returns a boolean reporting whether v represent the same UUID as u.
func (u UUID) Equal(v UUID) bool { return bytes.Equal(u, v) }

// Contains returns a boolean reporting whether u is in the slice s.
func Contains(s []UUID, u UUID) bool {
	if s == nil {
		return true
	}

	for _, a := range s {
		if a.Equal(u) {
			return true
		}
	}

	return false
}

// Reverse returns a reversed copy of u.
func Reverse(u []byte) []byte {
	// Special-case 16 bit UUIDS for speed.
	l := len(u)
	if l == 2 {
		return []byte{u[1], u[0]}
	}
	b := make([]byte, l)
	for i := 0; i < l/2+1; i++ {
		b[i], b[l-i-1] = u[l-i-1], u[i]
	}
	return b
}
