package ble

// AdvHandler handles advertisement.
type AdvHandler func(a Advertisement)

// AdvFilter returns true if the advertisement matches specified condition.
type AdvFilter func(a Advertisement) bool

// Advertisement ...
type Advertisement interface {
	LocalName() string
	ManufacturerData() []byte
	ServiceData() []ServiceData
	Services() []UUID
	OverflowService() []UUID
	TxPowerLevel() int
	Connectable() bool
	SolicitedService() []UUID

	RSSI() int
	Address() Addr
}

// ServiceData ...
type ServiceData struct {
	UUID UUID
	Data []byte
}
