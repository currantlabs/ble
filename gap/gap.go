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

// Mode ...
type Mode int

// Mode ...
const (
	NonDiscoverable     Mode = iota // [Vol 3, Part C, 9.2.2]
	LimitedDiscoverable             // [Vol 3, Part C, 9.2.3]
	GeneralDiscoverable             // [Vol 3, Part C, 9.2.4]
)
