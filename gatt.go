package bt

import "github.com/currantlabs/bt/uuid"

// NotificationHandler ...
type NotificationHandler func(req []byte)

// A Client is a GATT client.
type Client interface {
	// Address is the platform specific unique ID of the remote peripheral, e.g. MAC for Linux, Client UUID for MacOS.
	Address() Addr

	// Name returns the name of the remote peripheral.
	// This can be the advertised name, if exists, or the GAP device name, which takes priority.
	Name() string

	// Services returnns the services of the remote peripheral which has been discovered.
	Services() []Service

	// DiscoverServices discovers all the primary service on a server. [Vol 3, Parg G, 4.4.1]
	// DiscoverServices discover the specified services of the remote peripheral.
	// If the specified services is set to nil, all the available services of the remote peripheral are returned.
	DiscoverServices(filter []uuid.UUID) ([]Service, error)

	// DiscoverIncludedServices discovers the specified included services of a service.
	// If the specified services is set to nil, all the included services of the service are returned.
	DiscoverIncludedServices(ss []uuid.UUID, s Service) ([]Service, error)

	// DiscoverCharacteristics discovers the specified characteristics of a service.
	// If the specified characterstics is set to nil, all the characteristic of the service are returned.
	DiscoverCharacteristics(filter []uuid.UUID, s Service) ([]Characteristic, error)

	// DiscoverDescriptors discovers the descriptors of a characteristic.
	// If the specified descriptors is set to nil, all the descriptors of the characteristic are returned.
	DiscoverDescriptors(filter []uuid.UUID, c Characteristic) ([]Descriptor, error)

	// ReadCharacteristic retrieves the value of a specified characteristic.
	ReadCharacteristic(c Characteristic) ([]byte, error)

	// ReadLongCharacteristic retrieves the value of a specified characteristic that is longer than the MTU.
	ReadLongCharacteristic(c Characteristic) ([]byte, error)

	// WriteCharacteristic writes the value of a characteristic.
	WriteCharacteristic(c Characteristic, value []byte, noRsp bool) error

	// ReadDescriptor retrieves the value of a specified characteristic descriptor.
	ReadDescriptor(d Descriptor) ([]byte, error)

	// WriteDescriptor writes the value of a characteristic descriptor.
	WriteDescriptor(d Descriptor, v []byte) error

	// ReadRSSI retrieves the current RSSI value for the remote peripheral.
	ReadRSSI() int

	// ExchangeMTU exchange local MTU with the remote server.
	// Before the exchange, both client and server shall not send attribute values larger than 20 bytes.
	ExchangeMTU(rxMTU int) (txMTU int, err error)

	// Subscribe subscribes to indication (if ind is set true), or notification of a specified characteristic.
	Subscribe(c Characteristic, ind bool, h NotificationHandler) error

	// ClearSubscriptions clears all subscriptions of notification and indication.
	ClearSubscriptions() error

	// CancelConnection disconnects the connection.
	CancelConnection() error
}

// A Server is a GATT server.
type Server interface {
	// AddService add a service to database.
	AddService(svc Service) Service

	// RemoveAllServices removes all services that are currently in the database.
	RemoveAllServices() error

	// SetServices set the specified service to the database.
	// It removes all currently added services, if any.
	SetServices(svcs []Service) error

	// Start ...
	Start(p Peripheral) error

	// Stop ...
	Stop() error
}
