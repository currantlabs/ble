package evt

// DisconnectionCompleteEvent implements Disconnection Complete Event (0x05) [Vol 2, Part E, 7.7.5].
type DisconnectionCompleteEvent struct {
	Status           uint8
	ConnectionHandle uint16
	Reason           uint8
}

// Code returns the event code of the command.
func (e DisconnectionCompleteEvent) Code() int { return 0x05 }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *DisconnectionCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e DisconnectionCompleteEvent) String() string {
	return "Disconnection Complete Event (0x05)"
}

// EncryptionChangeEvent implements Encryption Change Event (0x08) [Vol 2, Part E, 7.7.8].
type EncryptionChangeEvent struct {
	Status            uint8
	ConnectionHandle  uint16
	EncryptionEnabled uint8
}

// Code returns the event code of the command.
func (e EncryptionChangeEvent) Code() int { return 0x08 }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *EncryptionChangeEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e EncryptionChangeEvent) String() string {
	return "Encryption Change Event (0x08)"
}

// ReadRemoteVersionInformationCompleteEvent implements Read Remote Version Information Complete Event (0x0C) [Vol 2, Part E, 7.7.12].
type ReadRemoteVersionInformationCompleteEvent struct {
	Status           uint8
	ConnectionHandle uint16
	Version          uint8
	ManufacturerName uint16
	Subversion       uint16
}

// Code returns the event code of the command.
func (e ReadRemoteVersionInformationCompleteEvent) Code() int { return 0x0C }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *ReadRemoteVersionInformationCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e ReadRemoteVersionInformationCompleteEvent) String() string {
	return "Read Remote Version Information Complete Event (0x0C)"
}

// CommandCompleteEvent implements Command Complete Event (0x0E) [Vol 2, Part E, 7.7.14].
type CommandCompleteEvent struct {
	NumHCICommandPackets uint8
	CommandOpcode        uint16
	ReturnParameters     []byte
}

// Code returns the event code of the command.
func (e CommandCompleteEvent) Code() int { return 0x0E }

func (e CommandCompleteEvent) String() string {
	return "Command Complete Event (0x0E)"
}

// CommandStatusEvent implements Command Status Event (0x0F) [Vol 2, Part E, 7.7.15].
type CommandStatusEvent struct {
	Status               uint8
	NumHCICommandPackets uint8
	CommandOpcode        uint16
}

// Code returns the event code of the command.
func (e CommandStatusEvent) Code() int { return 0x0F }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *CommandStatusEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e CommandStatusEvent) String() string {
	return "Command Status Event (0x0F)"
}

// HardwareErrorEvent implements Hardware Error Event (0x10) [Vol 2, Part E, 7.7.16].
type HardwareErrorEvent struct {
	HardwareCode uint8
}

// Code returns the event code of the command.
func (e HardwareErrorEvent) Code() int { return 0x10 }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *HardwareErrorEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e HardwareErrorEvent) String() string {
	return "Hardware Error Event (0x10)"
}

// NumberOfCompletedPacketsEvent implements Number Of Completed Packets Event (0x13) [Vol 2, Part E, 7.7.19].
type NumberOfCompletedPacketsEvent struct {
	NumberOfHandles         uint8
	ConnectionHandle        []uint16
	HCNumOfCompletedPackets []uint16
}

// Code returns the event code of the command.
func (e NumberOfCompletedPacketsEvent) Code() int { return 0x13 }

func (e NumberOfCompletedPacketsEvent) String() string {
	return "Number Of Completed Packets Event (0x13)"
}

// DataBufferOverflowEvent implements Data Buffer Overflow Event (0x1A) [Vol 2, Part E, 7.7.26].
type DataBufferOverflowEvent struct {
	LinkType uint8
}

// Code returns the event code of the command.
func (e DataBufferOverflowEvent) Code() int { return 0x1A }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *DataBufferOverflowEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e DataBufferOverflowEvent) String() string {
	return "Data Buffer Overflow Event (0x1A)"
}

// EncryptionKeyRefreshCompleteEvent implements Encryption Key Refresh Complete Event (0x30) [Vol 2, Part E, 7.7.39].
type EncryptionKeyRefreshCompleteEvent struct {
	Status           uint8
	ConnectionHandle uint16
}

// Code returns the event code of the command.
func (e EncryptionKeyRefreshCompleteEvent) Code() int { return 0x30 }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *EncryptionKeyRefreshCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e EncryptionKeyRefreshCompleteEvent) String() string {
	return "Encryption Key Refresh Complete Event (0x30)"
}

// LEConnectionCompleteEvent implements LE Connection Complete Event (0x3E:0x01) [Vol 2, Part E, 7.7.65.1].
type LEConnectionCompleteEvent struct {
	SubeventCode        uint8
	Status              uint8
	ConnectionHandle    uint16
	Role                uint8
	PeerAddressType     uint8
	PeerAddress         [6]byte
	ConnInterval        uint16
	ConnLatency         uint16
	SupervisionTimeout  uint16
	MasterClockAccuracy uint8
}

// Code returns the event code of the command.
func (e LEConnectionCompleteEvent) Code() int { return 0x3E }

// SubCode returns the subevent code of the command.
func (e LEConnectionCompleteEvent) SubCode() int {
	return 0x01
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *LEConnectionCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e LEConnectionCompleteEvent) String() string {
	return "LE Connection Complete Event (0x3E:0x01)"
}

// LEAdvertisingReportEvent implements LE Advertising Report Event (0x3E:0x02) [Vol 2, Part E, 7.7.65.2].
type LEAdvertisingReportEvent struct {
	SubeventCode uint8
	NumReports   uint8
	EventType    []uint8
	AddressType  []uint8
	Address      [][6]byte
	Length       []uint8
	Data         [][]byte
	RSSI         []int8
}

// Code returns the event code of the command.
func (e LEAdvertisingReportEvent) Code() int { return 0x3E }

// SubCode returns the subevent code of the command.
func (e LEAdvertisingReportEvent) SubCode() int {
	return 0x02
}

func (e LEAdvertisingReportEvent) String() string {
	return "LE Advertising Report Event (0x3E:0x02)"
}

// LEConnectionUpdateCompleteEvent implements LE Connection Update Complete Event (0x0E:0x03) [Vol 2, Part E, 7.7.65.3].
type LEConnectionUpdateCompleteEvent struct {
	SubeventCode       uint8
	Status             uint8
	ConnectionHandle   uint16
	ConnInterval       uint16
	ConnLatency        uint16
	SupervisionTimeout uint16
}

// Code returns the event code of the command.
func (e LEConnectionUpdateCompleteEvent) Code() int { return 0x0E }

// SubCode returns the subevent code of the command.
func (e LEConnectionUpdateCompleteEvent) SubCode() int {
	return 0x03
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *LEConnectionUpdateCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e LEConnectionUpdateCompleteEvent) String() string {
	return "LE Connection Update Complete Event (0x0E:0x03)"
}

// LEReadRemoteUsedFeaturesCompleteEvent implements LE Read Remote Used Features Complete Event (0x3E:0x04) [Vol 2, Part E, 7.7.65.4].
type LEReadRemoteUsedFeaturesCompleteEvent struct {
	SubeventCode     uint8
	Status           uint8
	ConnectionHandle uint16
	LEFeatures       uint64
}

// Code returns the event code of the command.
func (e LEReadRemoteUsedFeaturesCompleteEvent) Code() int { return 0x3E }

// SubCode returns the subevent code of the command.
func (e LEReadRemoteUsedFeaturesCompleteEvent) SubCode() int {
	return 0x04
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *LEReadRemoteUsedFeaturesCompleteEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e LEReadRemoteUsedFeaturesCompleteEvent) String() string {
	return "LE Read Remote Used Features Complete Event (0x3E:0x04)"
}

// LELongTermKeyRequestEvent implements LE Long Term Key Request Event (0x3E:0x05) [Vol 2, Part E, 7.7.65.5].
type LELongTermKeyRequestEvent struct {
	SubeventCode          uint8
	ConnectionHandle      uint16
	RandomNumber          uint64
	EncryptionDiversifier uint16
}

// Code returns the event code of the command.
func (e LELongTermKeyRequestEvent) Code() int { return 0x3E }

// SubCode returns the subevent code of the command.
func (e LELongTermKeyRequestEvent) SubCode() int {
	return 0x05
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *LELongTermKeyRequestEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e LELongTermKeyRequestEvent) String() string {
	return "LE Long Term Key Request Event (0x3E:0x05)"
}

// LERemoteConnectionParameterRequestEvent implements LE Remote Connection Parameter Request Event (0x3E:0x06) [Vol 2, Part E, 7.7.65.6].
type LERemoteConnectionParameterRequestEvent struct {
	SubeventCode     uint8
	ConnectionHandle uint16
	IntervalMin      uint16
	IntervalMax      uint16
	Latency          uint16
	Timeout          uint16
}

// Code returns the event code of the command.
func (e LERemoteConnectionParameterRequestEvent) Code() int { return 0x3E }

// SubCode returns the subevent code of the command.
func (e LERemoteConnectionParameterRequestEvent) SubCode() int {
	return 0x06
}

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *LERemoteConnectionParameterRequestEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e LERemoteConnectionParameterRequestEvent) String() string {
	return "LE Remote Connection Parameter Request Event (0x3E:0x06)"
}

// AuthenticatedPayloadTimeoutExpiredEvent implements Authenticated Payload Timeout Expired Event (0x57) [Vol 2, Part E, 7.7.75].
type AuthenticatedPayloadTimeoutExpiredEvent struct {
	ConnectionHandle uint16
}

// Code returns the event code of the command.
func (e AuthenticatedPayloadTimeoutExpiredEvent) Code() int { return 0x57 }

// Unmarshal de-serializes the binary data and stores the result in the receiver.
func (e *AuthenticatedPayloadTimeoutExpiredEvent) Unmarshal(b []byte) error {
	return unmarshal(e, b)
}
func (e AuthenticatedPayloadTimeoutExpiredEvent) String() string {
	return "Authenticated Payload Timeout Expired Event (0x57)"
}
