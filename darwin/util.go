package darwin

import "github.com/currantlabs/x/io/bt"

func uuidSlice(uu []bt.UUID) [][]byte {
	us := [][]byte{}
	for _, u := range uu {
		us = append(us, bt.Reverse(u))
	}
	return us
}
