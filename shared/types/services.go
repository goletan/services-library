package types

// Service interface that all services must implement.
type Service interface {
	Name() string
	Initialize() error
	Start() error
	Stop() error
	Discover() ([]ServiceEndpoint, error)
}

// ServiceEndpoint represents the metadata and connection details for a service.
type ServiceEndpoint struct {
	Name    string        // The name of the service (e.g., "auth-service").
	Address string        // The IP or hostname of the service.
	Ports   []ServicePort // List of exposed ports and their purposes.
	Version string        // Optional: version of the service for future use (e.g., "v1.0").
	Tags    []string      // Optional: tags for categorization or discovery filters (e.g., ["grpc", "core"]).
}

// ServicePort represents the details of a single port.
type ServicePort struct {
	Name     string // The name of the port (e.g., "grpc", "http").
	Port     int    // The port number.
	Protocol string // The protocol used (e.g., "TCP", "UDP").
}

// ServiceEvent
type ServiceEvent struct {
	Type    string // The name of the service (e.g., "auth-service").
	Service ServiceEndpoint
}
