package bt

// Broadcaster ...
type Broadcaster interface {
	// SetAdvertisement ...
	SetAdvertisement(ad []byte, sr []byte) error

	// Advertise ...
	Advertise() error

	// StopAdvertising ...
	StopAdvertising() error
}

// Peripheral ...
type Peripheral interface {
	Broadcaster
	Listener
}

// Observer ...
type Observer interface {
	// SetAdvHandler ...
	SetAdvHandler(af AdvFilter, ah AdvHandler) error

	// Scan starts scanning.
	Scan() error

	// StopScanning stops scanning.
	StopScanning() error
}

// Central ...
type Central interface {
	Observer
	Dialer
}
