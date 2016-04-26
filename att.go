package bt

// AttServer ...
type AttServer interface {
	// Notify sends notification to remote central.
	Notify(h uint16, data []byte) (int, error)

	// Indicate sends indication to remote central.
	Indicate(h uint16, data []byte) (int, error)
}
