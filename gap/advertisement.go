package gap

// type State int
//
// type StateUpdateHandler interface {
// 	HandleUpdate(s State)
// }
//
// type StateUpdater interface {
// 	StateUpdate(h StateUpdateHandler)
// }

// Advertisement ...
type Advertisement interface {
	AdvertisingData() []byte
	ScanResponse() []byte
}

// AdvertisementFilter ...
type AdvertisementFilter interface {
	Filter(a Advertisement) bool
}

// AdvertisementHandler ...
type AdvertisementHandler interface {
	HandleAdvertisement(a Advertisement)
}
