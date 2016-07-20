package gatt

import (
	"fmt"

	"github.com/currantlabs/ble"
)

// Matcher matches returns true if advertisement matches specific confition.
type Matcher interface {
	Match(a ble.Advertisement) bool
}

// MatcherFunc is an adapter to allow the use of ordinary functions as Matchers.
type MatcherFunc func(a ble.Advertisement) bool

// Match returns true if the adversisement matches specific condition.
func (m MatcherFunc) Match(a ble.Advertisement) bool {
	return m(a)
}

// Discover searches for and connects to a Peripheral which matches specified condition.
func Discover(m Matcher) (ble.Client, error) {
	ch := make(chan ble.Advertisement)
	fn := func(a ble.Advertisement) {
		if !m.Match(a) {
			return
		}
		StopScanning()
		ch <- a
	}
	if err := SetAdvHandler(ble.AdvHandlerFunc(fn)); err != nil {
		return nil, fmt.Errorf("can't set adv handler: %s", err)
	}
	if err := Scan(false); err != nil {
		return nil, fmt.Errorf("can't scan: %s", err)
	}

	a := <-ch

	cln, err := Dial(a.Address())
	if err != nil {
		return nil, fmt.Errorf("can't dial: %s", err)
	}
	return cln, nil
}
