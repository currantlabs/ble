package att

import "errors"

// DefaultMTU 23 defines the default MTU of ATT protocol.
const DefaultMTU = 23

// MaxMTU is maximum of ATT_MTU, which is 512 bytes of value length and 3 bytes of header.
// The maximum length of an attribute value shall be 512 octets [Vol 3, Part F, 3.2.9]
const MaxMTU = 512 + 3

var (
	// ErrInvalidArgument means one or more of the arguments are invalid.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrInvalidResponse means one or more of the response fields are invalid.
	ErrInvalidResponse = errors.New("invalid response")

	// ErrSeqProtoTimeout means the request hasn't been acknowledged in 30 seconds.
	// [Vol 3, Part F, 3.3.3]
	ErrSeqProtoTimeout = errors.New("req timeout")
)
