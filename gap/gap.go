package gap

// [Vol 3, Part C, 2.2 Roles ]
// Connection Update procedure
// Channel Map Update procedure
// Encryption procedure
// Master-initiated Feature Exchange procedure
// Slave-initiated Feature Exchange procedure
// Connection Parameters Request procedure
// Version Exchange procedure
// Termination procedure

// State ...
type State string

// State ...
const (
	StateUnknown      = "Unknown"
	StateResetting    = "Resetting"
	StateUnsupported  = "Unsupported"
	StateUnauthorized = "Unauthorized"
	StatePoweredOff   = "PoweredOff"
	StatePoweredOn    = "PoweredOn"
)
