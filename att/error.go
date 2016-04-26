package att

import "github.com/currantlabs/bt"

// NewErrorResponse ...
func NewErrorResponse(op byte, h uint16, s bt.AttError) []byte {
	r := ErrorResponse(make([]byte, 5))
	r.SetAttributeOpcode()
	r.SetRequestOpcodeInError(op)
	r.SetAttributeInError(h)
	r.SetErrorCode(uint8(s))
	return r
}
