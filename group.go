package bt

import "github.com/currantlabs/bt/uuid"

// Property ...
type Property int

// Characteristic property flags (spec 3.3.3.1)
const (
	CharBroadcast   Property = 0x01 // may be brocasted
	CharRead        Property = 0x02 // may be read
	CharWriteNR     Property = 0x04 // may be written to, with no reply
	CharWrite       Property = 0x08 // may be written to, with a reply
	CharNotify      Property = 0x10 // supports notifications
	CharIndicate    Property = 0x20 // supports Indications
	CharSignedWrite Property = 0x40 // supports signed write
	CharExtended    Property = 0x80 // supports extended properties
)

// A Service is a GATT service.
type Service interface {
	// UUID returns the UUID of the service.
	UUID() uuid.UUID

	// Characteristics returns the contained characteristic of this service.
	Characteristics() []Characteristic

	// AddCharacteristic adds a characteristic to a service.
	// AddCharacteristic panics if the service already contains another characteristic with the same UUID.
	AddCharacteristic(c Characteristic) Characteristic

	// NewCharacteristic adds a characteristic to a service.
	// NewCharacteristic panics if the service already contains another characteristic with the same UUID.
	NewCharacteristic(u uuid.UUID) Characteristic
}

// A Characteristic is a GATT characteristic.
type Characteristic interface {
	// UUID returns the UUID of the characteristic.
	UUID() uuid.UUID

	// Properties returns the properties of this characteristic.
	Properties() Property

	// Descriptors returns the contained descriptors of this characteristic.
	Descriptors() []Descriptor

	// SetValue makes the characteristic support read requests, and returns a static value.
	// SetValue panics if the characteristic has been configured with a ReadHandler.
	// SetValue must be called before the containing service is added to a server.
	SetValue(b []byte) Characteristic

	// HandleRead makes the characteristic support read requests, and routes the requests to h.
	// HandleRead panics if the characteristic has been configured with a static attr.
	// HandleRead must be called before the containing service is added to a server.
	HandleRead(h ReadHandler) Characteristic

	// HandleWrite makes the characteristic support write and write-no-response requests, and routes the requests to h.
	// The WriteHandler does not differentiate between write and write-no-response requests; it is handled automatically.
	// HandleWrite must be called before the containing service is added to a server.
	HandleWrite(h WriteHandler) Characteristic

	// HandleNotify makes the characteristic support notify requests, and routes the requests to h.
	// HandleNotify must be called before the containing service is added to a server.
	HandleNotify(ind bool, h NotifyHandler) Characteristic

	// AddDescriptor adds a descriptor to a characteristic.
	// AddDescriptor panics if the characteristic already contains another descriptor with the same UUID.
	AddDescriptor(d Descriptor) Descriptor

	// NewDescriptor adds a descriptor to a characteristic.
	// NewDescriptor panics if the characteristic already contains another descriptor with the same UUID.
	NewDescriptor(u uuid.UUID) Descriptor
}

// Descriptor is a GATT descriptor
type Descriptor interface {
	// UUID returns the UUID of the descriptor.
	UUID() uuid.UUID

	// SetValue makes the descriptor support read requests, and returns a static attr.
	// SetValue must be called before the containing service is added to a server.
	// SetValue panics if the descriptor has already configured with a ReadHandler.
	SetValue(b []byte) Descriptor

	// HandleRead makes the descriptor support write requests, and routes the requests to h.
	// HandleRead must be called before the containing service is added to a server.
	// HandleRead panics if the descriptor has been configured with a static attr.
	HandleRead(h ReadHandler) Descriptor

	// HandleWrite makes the descriptor support write and write-no-response requests, and routes the requests to h.
	// The WriteHandler does not differentiate between write and write-no-response requests; it is handled automatically.
	// HandleWrite must be called before the containing service is added to a server.
	HandleWrite(h WriteHandler) Descriptor
}
