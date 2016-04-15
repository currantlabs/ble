package adv

// MaxEIRPacketLength is the maximum allowed AdvertisingPacket
// and ScanResponsePacket length.
const MaxEIRPacketLength = 31

// Advertising data field s
const (
	Flags             = 0x01 // Flags
	SomeUUID16        = 0x02 // Incomplete List of 16-bit Service Class UUIDs
	AllUUID16         = 0x03 // Complete List of 16-bit Service Class UUIDs
	SomeUUID32        = 0x04 // Incomplete List of 32-bit Service Class UUIDs
	AllUUID32         = 0x05 // Complete List of 32-bit Service Class UUIDs
	SomeUUID128       = 0x06 // Incomplete List of 128-bit Service Class UUIDs
	AllUUID128        = 0x07 // Complete List of 128-bit Service Class UUIDs
	ShortName         = 0x08 // Shortened Local Name
	CompleteName      = 0x09 // Complete Local Name
	TxPower           = 0x0A // Tx Power Level
	ClassOfDevice     = 0x0D // Class of Device
	SimplePairingC192 = 0x0E // Simple Pairing Hash C-192
	SimplePairingR192 = 0x0F // Simple Pairing Randomizer R-192
	SecManagerTK      = 0x10 // Security Manager TK Value
	SecManagerOOB     = 0x11 // Security Manager Out of Band Flags
	SlaveConnInt      = 0x12 // Slave Connection Interval Range
	ServiceSol16      = 0x14 // List of 16-bit Service Solicitation UUIDs
	ServiceSol128     = 0x15 // List of 128-bit Service Solicitation UUIDs
	ServiceData16     = 0x16 // Service Data - 16-bit UUID
	PubTargetAddr     = 0x17 // Public Target Address
	RandTargetAddr    = 0x18 // Random Target Address
	Appearance        = 0x19 // Appearance
	AdvInterval       = 0x1A // Advertising Interval
	LEDeviceAddr      = 0x1B // LE Bluetooth Device Address
	LERole            = 0x1C // LE Role
	ServiceSol32      = 0x1F // List of 32-bit Service Solicitation UUIDs
	ServiceData32     = 0x20 // Service Data - 32-bit UUID
	ServiceData128    = 0x21 // Service Data - 128-bit UUID
	LESecConfirm      = 0x22 // LE Secure Connections Confirmation Value
	LESecRandom       = 0x23 // LE Secure Connections Random Value
	ManufacturerData  = 0xFF // Manufacturer Specific Data
)

// Advertising flags
const (
	FlagLimitedDiscoverable = 0x01 // LE Limited Discoverable Mode
	FlagGeneralDiscoverable = 0x02 // LE General Discoverable Mode
	FlagLEOnly              = 0x04 // BR/EDR Not Supported. Bit 37 of LMP Feature Mask Definitions (Page 0)
	FlagBothController      = 0x08 // Simultaneous LE and BR/EDR to Same Device Capable (Controller).
	FlagBothHost            = 0x10 // Simultaneous LE and BR/EDR to Same Device Capable (Host).
)
