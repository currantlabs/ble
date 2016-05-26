package bt

import "errors"

// ErrEIRPacketTooLong is the error returned when an AdvertisingPacket
// or ScanResponsePacket is too long.
var ErrEIRPacketTooLong = errors.New("max packet length is 31")

// AttError is the implemtns AttError Response of Attribute Protocol [Vol 3, Part F, 3.4.1.1]
type AttError byte

// AttError is the implemtns AttError Response of Attribute Protocol [Vol 3, Part F, 3.4.1.1]
const (
	ErrSuccess           AttError = 0x00 // ErrSuccess measn the operation is success.
	ErrInvalidHandle     AttError = 0x01 // ErrInvalidHandle means the attribute handle given was not valid on this server.
	ErrReadNotPerm       AttError = 0x02 // ErrReadNotPerm eans the attribute cannot be read.
	ErrWriteNotPerm      AttError = 0x03 // ErrWriteNotPerm eans the attribute cannot be written.
	ErrInvalidPDU        AttError = 0x04 // ErrInvalidPDU means the attribute PDU was invalid.
	ErrAuthentication    AttError = 0x05 // ErrAuthentication means the attribute requires authentication before it can be read or written.
	ErrReqNotSupp        AttError = 0x06 // ErrReqNotSupp means the attribute server does not support the request received from the client.
	ErrInvalidOffset     AttError = 0x07 // ErrInvalidOffset means the specified was past the end of the attribute.
	ErrAuthorization     AttError = 0x08 // ErrAuthorization means the attribute requires authorization before it can be read or written.
	ErrPrepQueueFull     AttError = 0x09 // ErrPrepQueueFull means too many prepare writes have been queued.
	ErrAttrNotFound      AttError = 0x0a // ErrAttrNotFound means no attribute found within the given attribute handle range.
	ErrAttrNotLong       AttError = 0x0b // ErrAttrNotLong means the attribute cannot be read or written using the Read Blob Request.
	ErrInsuffEncrKeySize AttError = 0x0c // ErrInsuffEncrKeySize means the Encryption Key Size used for encrypting this link is insufficient.
	ErrInvalAttrValueLen AttError = 0x0d // ErrInvalAttrValueLen means the attribute value length is invalid for the operation.
	ErrUnlikely          AttError = 0x0e // ErrUnlikely means the attribute request that was requested has encountered an error that was unlikely, and therefore could not be completed as requested.
	ErrInsuffEnc         AttError = 0x0f // ErrInsuffEnc means the attribute requires encryption before it can be read or written.
	ErrUnsuppGrpType     AttError = 0x10 // ErrUnsuppGrpType means the attribute type is not a supported grouping attribute as defined by a higher layer specification.
	ErrInsuffResources   AttError = 0x11 // ErrInsuffResources means insufficient resources to complete the request.
)

func (a AttError) Error() string {
	switch i := int(a); {
	case i < 0x11:
		return errName[a]
	case (i >= 0x12 && i <= 0x7F) || // Reserved for future use
		(i >= 0x80 && i <= 0x9F) || // Application AttError, defined by higher level
		(i >= 0xA0 && i <= 0xDF): // Reserved for future use
		return "reserved error code"
	case i >= 0xE0 && i <= 0xFF: // Common profile and service error codes
		return "profile or service error"
	default: // can't happen, just make compiler happy
		return "unkown error"
	}
}

var errName = map[AttError]string{
	ErrSuccess:           "success",
	ErrInvalidHandle:     "invalid handle",
	ErrReadNotPerm:       "read not permitted",
	ErrWriteNotPerm:      "write not permitted",
	ErrInvalidPDU:        "invalid PDU",
	ErrAuthentication:    "insufficient authentication",
	ErrReqNotSupp:        "request not supported",
	ErrInvalidOffset:     "invalid offset",
	ErrAuthorization:     "insufficient authorization",
	ErrPrepQueueFull:     "prepare queue full",
	ErrAttrNotFound:      "attribute not found",
	ErrAttrNotLong:       "attribute not long",
	ErrInsuffEncrKeySize: "insufficient encryption key size",
	ErrInvalAttrValueLen: "invalid attribute value length",
	ErrUnlikely:          "unlikely error",
	ErrInsuffEnc:         "insufficient encryption",
	ErrUnsuppGrpType:     "unsupported group type",
	ErrInsuffResources:   "insufficient resources",
}
