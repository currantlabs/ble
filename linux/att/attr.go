package att

import "github.com/currantlabs/bt"

// attr is a BLE attribute.
type attr struct {
	h    uint16
	endh uint16
	typ  bt.UUID

	v  []byte
	rh bt.ReadHandler
	wh bt.WriteHandler
}
