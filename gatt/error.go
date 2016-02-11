package gatt

// Error is the implemtns Error Response of Attribute Protocol [Vol 3, Part F, 3.4.1.1]
type Error byte

// Error is the implemtns Error Response of Attribute Protocol [Vol 3, Part F, 3.4.1.1]
const (
	ErrSuccess           Error = 0x00 // ErrSuccess measn the operation is success.
	ErrInvalidHandle     Error = 0x01 // ErrInvalidHandle means the attribute handle given was not valid on this server.
	ErrReadNotPerm       Error = 0x02 // ErrReadNotPerm eans the attribute cannot be read.
	ErrWriteNotPerm      Error = 0x03 // ErrWriteNotPerm eans the attribute cannot be written.
	ErrInvalidPDU        Error = 0x04 // ErrInvalidPDU means the attribute PDU was invalid.
	ErrAuthentication    Error = 0x05 // ErrAuthentication means the attribute requires authentication before it can be read or written.
	ErrReqNotSupp        Error = 0x06 // ErrReqNotSupp means the attribute server does not support the request received from the client.
	ErrInvalidOffset     Error = 0x07 // ErrInvalidOffset means the specified was past the end of the attribute.
	ErrAuthorization     Error = 0x08 // ErrAuthorization means the attribute requires authorization before it can be read or written.
	ErrPrepQueueFull     Error = 0x09 // ErrPrepQueueFull means too many prepare writes have been queued.
	ErrAttrNotFound      Error = 0x0a // ErrAttrNotFound means no attribute found within the given attribute handle range.
	ErrAttrNotLong       Error = 0x0b // ErrAttrNotLong means the attribute cannot be read or written using the Read Blob Request.
	ErrInsuffEncrKeySize Error = 0x0c // ErrInsuffEncrKeySize means the Encryption Key Size used for encrypting this link is insufficient.
	ErrInvalAttrValueLen Error = 0x0d // ErrInvalAttrValueLen means the attribute value length is invalid for the operation.
	ErrUnlikely          Error = 0x0e // ErrUnlikely means the attribute request that was requested has encountered an error that was unlikely, and therefore could not be completed as requested.
	ErrInsuffEnc         Error = 0x0f // ErrInsuffEnc means the attribute requires encryption before it can be read or written.
	ErrUnsuppGrpType     Error = 0x10 // ErrUnsuppGrpType means the attribute type is not a supported grouping attribute as defined by a higher layer specification.
	ErrInsuffResources   Error = 0x11 // ErrInsuffResources means insufficient resources to complete the request.
)

func (a Error) Error() string {
	switch i := int(a); {
	case i < 0x11:
		return errName[a]
	case i >= 0x12 && i <= 0x7F: // Reserved for future use
		return "reserved error code"
	case i >= 0x80 && i <= 0x9F: // Application Error, defined by higher level
		return "reserved error code"
	case i >= 0xA0 && i <= 0xDF: // Reserved for future use
		return "reserved error code"
	case i >= 0xE0 && i <= 0xFF: // Common profile and service error codes
		return "profile or service error"
	default: // can't happen, just make compiler happy
		return "unkown error"
	}
}

var errName = map[Error]string{
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

// NewErrorResponse ...
func NewErrorResponse(op byte, h uint16, s Error) []byte {
	r := ErrorResponse(make([]byte, 5))
	r.SetAttributeOpcode()
	r.SetRequestOpcodeInError(op)
	r.SetAttributeInError(h)
	r.SetErrorCode(uint8(s))
	return r
}
