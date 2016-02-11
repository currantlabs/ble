package gatt

import "encoding/binary"

const ErrorResponseCode = 0x01

// ErrorResponse implements Error Response (0x01) [Vol 3, Part E, 3.4.1.1].
type ErrorResponse []byte

func (r ErrorResponse) AttributeOpcode() uint8          { return r[0] }
func (r ErrorResponse) SetAttributeOpcode()             { r[0] = 0x01 }
func (r ErrorResponse) RequestOpcodeInError() uint8     { return r[1] }
func (r ErrorResponse) SetRequestOpcodeInError(v uint8) { r[1] = v }
func (r ErrorResponse) AttributeInError() uint16        { return binary.LittleEndian.Uint16(r[2:]) }
func (r ErrorResponse) SetAttributeInError(v uint16)    { binary.LittleEndian.PutUint16(r[2:], v) }
func (r ErrorResponse) ErrorCode() uint8                { return r[4] }
func (r ErrorResponse) SetErrorCode(v uint8)            { r[4] = v }

const ExchangeMTURequestCode = 0x02

// ExchangeMTURequest implements Exchange MTU Request (0x02) [Vol 3, Part E, 3.4.2.1].
type ExchangeMTURequest []byte

func (r ExchangeMTURequest) AttributeOpcode() uint8  { return r[0] }
func (r ExchangeMTURequest) SetAttributeOpcode()     { r[0] = 0x02 }
func (r ExchangeMTURequest) ClientRxMTU() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r ExchangeMTURequest) SetClientRxMTU(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

const ExchangeMTUResponseCode = 0x03

// ExchangeMTUResponse implements Exchange MTU Response (0x03) [Vol 3, Part E, 3.4.2.2].
type ExchangeMTUResponse []byte

func (r ExchangeMTUResponse) AttributeOpcode() uint8  { return r[0] }
func (r ExchangeMTUResponse) SetAttributeOpcode()     { r[0] = 0x03 }
func (r ExchangeMTUResponse) ServerRxMTU() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r ExchangeMTUResponse) SetServerRxMTU(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

const FindInformationRequestCode = 0x04

// FindInformationRequest implements Find Information Request (0x04) [Vol 3, Part E, 3.4.3.1].
type FindInformationRequest []byte

func (r FindInformationRequest) AttributeOpcode() uint8     { return r[0] }
func (r FindInformationRequest) SetAttributeOpcode()        { r[0] = 0x04 }
func (r FindInformationRequest) StartingHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r FindInformationRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r FindInformationRequest) EndingHandle() uint16       { return binary.LittleEndian.Uint16(r[3:]) }
func (r FindInformationRequest) SetEndingHandle(v uint16)   { binary.LittleEndian.PutUint16(r[3:], v) }

const FindInformationResponseCode = 0x05

// FindInformationResponse implements Find Information Response (0x05) [Vol 3, Part E, 3.4.3.2].
type FindInformationResponse []byte

func (r FindInformationResponse) AttributeOpcode() uint8      { return r[0] }
func (r FindInformationResponse) SetAttributeOpcode()         { r[0] = 0x05 }
func (r FindInformationResponse) Format() uint8               { return r[1] }
func (r FindInformationResponse) SetFormat(v uint8)           { r[1] = v }
func (r FindInformationResponse) InformationData() []byte     { return r[2:] }
func (r FindInformationResponse) SetInformationData(v []byte) { copy(r[2:], v) }

const FindByTypeValueRequestCode = 0x06

// FindByTypeValueRequest implements Find By Type Value Request (0x06) [Vol 3, Part E, 3.4.3.3].
type FindByTypeValueRequest []byte

func (r FindByTypeValueRequest) AttributeOpcode() uint8     { return r[0] }
func (r FindByTypeValueRequest) SetAttributeOpcode()        { r[0] = 0x06 }
func (r FindByTypeValueRequest) StartingHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r FindByTypeValueRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r FindByTypeValueRequest) EndingHandle() uint16       { return binary.LittleEndian.Uint16(r[3:]) }
func (r FindByTypeValueRequest) SetEndingHandle(v uint16)   { binary.LittleEndian.PutUint16(r[3:], v) }
func (r FindByTypeValueRequest) AttributeType() uint16      { return binary.LittleEndian.Uint16(r[5:]) }
func (r FindByTypeValueRequest) SetAttributeType(v uint16)  { binary.LittleEndian.PutUint16(r[5:], v) }
func (r FindByTypeValueRequest) AttributeValue() []byte     { return r[7:] }
func (r FindByTypeValueRequest) SetAttributeValue(v []byte) { copy(r[7:], v) }

const FindByTypeValueResponseCode = 0x07

// FindByTypeValueResponse implements Find By Type Value Response (0x07) [Vol 3, Part E, 3.4.3.4].
type FindByTypeValueResponse []byte

func (r FindByTypeValueResponse) AttributeOpcode() uint8            { return r[0] }
func (r FindByTypeValueResponse) SetAttributeOpcode()               { r[0] = 0x07 }
func (r FindByTypeValueResponse) HandleInformationList() []byte     { return r[1:] }
func (r FindByTypeValueResponse) SetHandleInformationList(v []byte) { copy(r[1:], v) }

const ReadByTypeRequestCode = 0x08

// ReadByTypeRequest implements Read By Type Request (0x08) [Vol 3, Part E, 3.4.4.1].
type ReadByTypeRequest []byte

func (r ReadByTypeRequest) AttributeOpcode() uint8     { return r[0] }
func (r ReadByTypeRequest) SetAttributeOpcode()        { r[0] = 0x08 }
func (r ReadByTypeRequest) StartingHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r ReadByTypeRequest) SetStartingHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r ReadByTypeRequest) EndingHandle() uint16       { return binary.LittleEndian.Uint16(r[3:]) }
func (r ReadByTypeRequest) SetEndingHandle(v uint16)   { binary.LittleEndian.PutUint16(r[3:], v) }
func (r ReadByTypeRequest) AttributeType() []byte      { return r[5:] }
func (r ReadByTypeRequest) SetAttributeType(v []byte)  { copy(r[5:], v) }

const ReadByTypeResponseCode = 0x09

// ReadByTypeResponse implements Read By Type Response (0x09) [Vol 3, Part E, 3.4.4.2].
type ReadByTypeResponse []byte

func (r ReadByTypeResponse) AttributeOpcode() uint8        { return r[0] }
func (r ReadByTypeResponse) SetAttributeOpcode()           { r[0] = 0x09 }
func (r ReadByTypeResponse) Length() uint8                 { return r[1] }
func (r ReadByTypeResponse) SetLength(v uint8)             { r[1] = v }
func (r ReadByTypeResponse) AttributeDataList() []byte     { return r[2:] }
func (r ReadByTypeResponse) SetAttributeDataList(v []byte) { copy(r[2:], v) }

const ReadRequestCode = 0x0A

// ReadRequest implements Read Request (0x0A) [Vol 3, Part E, 3.4.4.3].
type ReadRequest []byte

func (r ReadRequest) AttributeOpcode() uint8      { return r[0] }
func (r ReadRequest) SetAttributeOpcode()         { r[0] = 0x0A }
func (r ReadRequest) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r ReadRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }

const ReadResponseCode = 0x0B

// ReadResponse implements Read Response (0x0B) [Vol 3, Part E, 3.4.4.4].
type ReadResponse []byte

func (r ReadResponse) AttributeOpcode() uint8     { return r[0] }
func (r ReadResponse) SetAttributeOpcode()        { r[0] = 0x0B }
func (r ReadResponse) AttributeValue() []byte     { return r[1:] }
func (r ReadResponse) SetAttributeValue(v []byte) { copy(r[1:], v) }

const ReadBlobRequestCode = 0x0C

// ReadBlobRequest implements Read Blob Request (0x0C) [Vol 3, Part E, 3.4.4.5].
type ReadBlobRequest []byte

func (r ReadBlobRequest) AttributeOpcode() uint8      { return r[0] }
func (r ReadBlobRequest) SetAttributeOpcode()         { r[0] = 0x0C }
func (r ReadBlobRequest) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r ReadBlobRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r ReadBlobRequest) ValueOffset() uint16         { return binary.LittleEndian.Uint16(r[3:]) }
func (r ReadBlobRequest) SetValueOffset(v uint16)     { binary.LittleEndian.PutUint16(r[3:], v) }

const ReadBlobResponseCode = 0x0D

// ReadBlobResponse implements Read Blob Response (0x0D) [Vol 3, Part E, 3.4.4.6].
type ReadBlobResponse []byte

func (r ReadBlobResponse) AttributeOpcode() uint8         { return r[0] }
func (r ReadBlobResponse) SetAttributeOpcode()            { r[0] = 0x0D }
func (r ReadBlobResponse) PartAttributeValue() []byte     { return r[1:] }
func (r ReadBlobResponse) SetPartAttributeValue(v []byte) { copy(r[1:], v) }

const ReadMultipleRequestCode = 0x0E

// ReadMultipleRequest implements Read Multiple Request (0x0E) [Vol 3, Part E, 3.4.4.7].
type ReadMultipleRequest []byte

func (r ReadMultipleRequest) AttributeOpcode() uint8   { return r[0] }
func (r ReadMultipleRequest) SetAttributeOpcode()      { r[0] = 0x0E }
func (r ReadMultipleRequest) SetOfHandles() []byte     { return r[1:] }
func (r ReadMultipleRequest) SetSetOfHandles(v []byte) { copy(r[1:], v) }

const ReadMultipleResponseCode = 0x0F

// ReadMultipleResponse implements Read Multiple Response (0x0F) [Vol 3, Part E, 3.4.4.8].
type ReadMultipleResponse []byte

func (r ReadMultipleResponse) AttributeOpcode() uint8  { return r[0] }
func (r ReadMultipleResponse) SetAttributeOpcode()     { r[0] = 0x0F }
func (r ReadMultipleResponse) SetOfValues() []byte     { return r[1:] }
func (r ReadMultipleResponse) SetSetOfValues(v []byte) { copy(r[1:], v) }

const ReadByGroupTypeRequestCode = 0x10

// ReadByGroupTypeRequest implements Read By Group Type Request (0x10) [Vol 3, Part E, 3.4.4.9].
type ReadByGroupTypeRequest []byte

func (r ReadByGroupTypeRequest) AttributeOpcode() uint8         { return r[0] }
func (r ReadByGroupTypeRequest) SetAttributeOpcode()            { r[0] = 0x10 }
func (r ReadByGroupTypeRequest) StartingHandle() uint16         { return binary.LittleEndian.Uint16(r[1:]) }
func (r ReadByGroupTypeRequest) SetStartingHandle(v uint16)     { binary.LittleEndian.PutUint16(r[1:], v) }
func (r ReadByGroupTypeRequest) EndingHandle() uint16           { return binary.LittleEndian.Uint16(r[3:]) }
func (r ReadByGroupTypeRequest) SetEndingHandle(v uint16)       { binary.LittleEndian.PutUint16(r[3:], v) }
func (r ReadByGroupTypeRequest) AttributeGroupType() []byte     { return r[5:] }
func (r ReadByGroupTypeRequest) SetAttributeGroupType(v []byte) { copy(r[5:], v) }

const ReadByGroupTypeResponseCode = 0x11

// ReadByGroupTypeResponse implements Read By Group Type Response (0x11) [Vol 3, Part E, 3.4.4.10].
type ReadByGroupTypeResponse []byte

func (r ReadByGroupTypeResponse) AttributeOpcode() uint8        { return r[0] }
func (r ReadByGroupTypeResponse) SetAttributeOpcode()           { r[0] = 0x11 }
func (r ReadByGroupTypeResponse) Length() uint8                 { return r[1] }
func (r ReadByGroupTypeResponse) SetLength(v uint8)             { r[1] = v }
func (r ReadByGroupTypeResponse) AttributeDataList() []byte     { return r[2:] }
func (r ReadByGroupTypeResponse) SetAttributeDataList(v []byte) { copy(r[2:], v) }

const WriteRequestCode = 0x12

// WriteRequest implements Write Request (0x12) [Vol 3, Part E, 3.4.5.1].
type WriteRequest []byte

func (r WriteRequest) AttributeOpcode() uint8      { return r[0] }
func (r WriteRequest) SetAttributeOpcode()         { r[0] = 0x12 }
func (r WriteRequest) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r WriteRequest) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r WriteRequest) AttributeValue() []byte      { return r[3:] }
func (r WriteRequest) SetAttributeValue(v []byte)  { copy(r[3:], v) }

const WriteResponseCode = 0x13

// WriteResponse implements Write Response (0x13) [Vol 3, Part E, 3.4.5.2].
type WriteResponse []byte

func (r WriteResponse) AttributeOpcode() uint8 { return r[0] }
func (r WriteResponse) SetAttributeOpcode()    { r[0] = 0x13 }

const WriteCommandCode = 0x52

// WriteCommand implements Write Command (0x52) [Vol 3, Part E, 3.4.5.3].
type WriteCommand []byte

func (r WriteCommand) AttributeOpcode() uint8      { return r[0] }
func (r WriteCommand) SetAttributeOpcode()         { r[0] = 0x52 }
func (r WriteCommand) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r WriteCommand) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r WriteCommand) AttributeValue() []byte      { return r[3:] }
func (r WriteCommand) SetAttributeValue(v []byte)  { copy(r[3:], v) }

const SignedWriteCommandCode = 0xD2

// SignedWriteCommand implements Signed Write Command (0xD2) [Vol 3, Part E, 3.4.5.4].
type SignedWriteCommand []byte

func (r SignedWriteCommand) AttributeOpcode() uint8      { return r[0] }
func (r SignedWriteCommand) SetAttributeOpcode()         { r[0] = 0xD2 }
func (r SignedWriteCommand) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r SignedWriteCommand) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r SignedWriteCommand) AttributeValue() []byte      { return r[3:] }
func (r SignedWriteCommand) SetAttributeValue(v []byte)  { copy(r[3:], v) }
func (r SignedWriteCommand) AuthenticationSignature() [12]byte {
	b := [12]byte{}
	copy(b[:], r[3:])
	return b
}
func (r SignedWriteCommand) SetAuthenticationSignature(v [12]byte) { copy(r[3:3+12], v[:]) }

const PrepareWriteRequestCode = 0x16

// PrepareWriteRequest implements Prepare Write Request (0x16) [Vol 3, Part E, 3.4.6.1].
type PrepareWriteRequest []byte

func (r PrepareWriteRequest) AttributeOpcode() uint8         { return r[0] }
func (r PrepareWriteRequest) SetAttributeOpcode()            { r[0] = 0x16 }
func (r PrepareWriteRequest) AttributeHandle() uint16        { return binary.LittleEndian.Uint16(r[1:]) }
func (r PrepareWriteRequest) SetAttributeHandle(v uint16)    { binary.LittleEndian.PutUint16(r[1:], v) }
func (r PrepareWriteRequest) ValueOffset() uint16            { return binary.LittleEndian.Uint16(r[3:]) }
func (r PrepareWriteRequest) SetValueOffset(v uint16)        { binary.LittleEndian.PutUint16(r[3:], v) }
func (r PrepareWriteRequest) PartAttributeValue() []byte     { return r[5:] }
func (r PrepareWriteRequest) SetPartAttributeValue(v []byte) { copy(r[5:], v) }

const PrepareWriteResponseCode = 0x17

// PrepareWriteResponse implements Prepare Write Response (0x17) [Vol 3, Part E, 3.4.6.2].
type PrepareWriteResponse []byte

func (r PrepareWriteResponse) AttributeOpcode() uint8         { return r[0] }
func (r PrepareWriteResponse) SetAttributeOpcode()            { r[0] = 0x17 }
func (r PrepareWriteResponse) AttributeHandle() uint16        { return binary.LittleEndian.Uint16(r[1:]) }
func (r PrepareWriteResponse) SetAttributeHandle(v uint16)    { binary.LittleEndian.PutUint16(r[1:], v) }
func (r PrepareWriteResponse) ValueOffset() uint16            { return binary.LittleEndian.Uint16(r[3:]) }
func (r PrepareWriteResponse) SetValueOffset(v uint16)        { binary.LittleEndian.PutUint16(r[3:], v) }
func (r PrepareWriteResponse) PartAttributeValue() []byte     { return r[5:] }
func (r PrepareWriteResponse) SetPartAttributeValue(v []byte) { copy(r[5:], v) }

const ExecuteWriteRequestCode = 0x18

// ExecuteWriteRequest implements Execute Write Request (0x18) [Vol 3, Part E, 3.4.6.3].
type ExecuteWriteRequest []byte

func (r ExecuteWriteRequest) AttributeOpcode() uint8 { return r[0] }
func (r ExecuteWriteRequest) SetAttributeOpcode()    { r[0] = 0x18 }
func (r ExecuteWriteRequest) Flags() uint8           { return r[1] }
func (r ExecuteWriteRequest) SetFlags(v uint8)       { r[1] = v }

const ExecuteWriteResponseCode = 0x19

// ExecuteWriteResponse implements Execute Write Response (0x19) [Vol 3, Part E, 3.4.6.4].
type ExecuteWriteResponse []byte

func (r ExecuteWriteResponse) AttributeOpcode() uint8 { return r[0] }
func (r ExecuteWriteResponse) SetAttributeOpcode()    { r[0] = 0x19 }

const HandleValueNotificationCode = 0x1B

// HandleValueNotification implements Handle Value Notification (0x1B) [Vol 3, Part E, 3.4.7.1].
type HandleValueNotification []byte

func (r HandleValueNotification) AttributeOpcode() uint8  { return r[0] }
func (r HandleValueNotification) SetAttributeOpcode()     { r[0] = 0x1B }
func (r HandleValueNotification) AttributeHandle() uint16 { return binary.LittleEndian.Uint16(r[1:]) }
func (r HandleValueNotification) SetAttributeHandle(v uint16) {
	binary.LittleEndian.PutUint16(r[1:], v)
}
func (r HandleValueNotification) AttributeValue() []byte     { return r[3:] }
func (r HandleValueNotification) SetAttributeValue(v []byte) { copy(r[3:], v) }

const HandleValueIndicationCode = 0x1D

// HandleValueIndication implements Handle Value Indication (0x1D) [Vol 3, Part E, 3.4.7.2].
type HandleValueIndication []byte

func (r HandleValueIndication) AttributeOpcode() uint8      { return r[0] }
func (r HandleValueIndication) SetAttributeOpcode()         { r[0] = 0x1D }
func (r HandleValueIndication) AttributeHandle() uint16     { return binary.LittleEndian.Uint16(r[1:]) }
func (r HandleValueIndication) SetAttributeHandle(v uint16) { binary.LittleEndian.PutUint16(r[1:], v) }
func (r HandleValueIndication) AttributeValue() []byte      { return r[3:] }
func (r HandleValueIndication) SetAttributeValue(v []byte)  { copy(r[3:], v) }

const HandleValueConfirmationCode = 0x1E

// HandleValueConfirmation implements Handle Value Confirmation (0x1E) [Vol 3, Part E, 3.4.7.3].
type HandleValueConfirmation []byte

func (r HandleValueConfirmation) AttributeOpcode() uint8 { return r[0] }
func (r HandleValueConfirmation) SetAttributeOpcode()    { r[0] = 0x1E }
