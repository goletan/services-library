package types

import observability "github.com/goletan/observability-library/pkg"

type ServiceFactory func(endpoint ServiceEndpoint) Service

// Service interface that all services-library must implement.
type Service interface {
	Build(address string, ports []ServicePort) *ServiceEndpoint
	Name() string
	Initialize() error
	Start() error
	Stop() error
	Discover(obs *observability.Observability) ([]ServiceEndpoint, error)
}

// ServiceEvent represents an event related to a service, such as its addition,
// modification, or deletion. It contains metadata about the event type and the
// associated service endpoint details.
//
// Fields:
//   - Type: Describes the nature of the event (e.g., "ADDED", "MODIFIED", "DELETED").
//   - Service: Provides information about the service endpoint involved in the event,
//     including its name, address, ports, and optional metadata such as version and tags.
type ServiceEvent struct {
	Type    string
	Service ServiceEndpoint
}

// ServiceEndpoint represents the metadata and connection details for a service.
type ServiceEndpoint struct {
	Name    string        // The name of the service (e.g., "auth-service").
	Address string        // The IP or hostname of the service.
	Ports   []ServicePort // List of exposed ports and their purposes.
	Version string        // Optional: version of the service for future use (e.g., "v1.0").
	Tags    []string      // Optional: tags for categorization or discovery filters (e.g., ["grpc", "core-service"]).
}

// ServicePort represents the details of a single port.
type ServicePort struct {
	Name     string // The name of the port (e.g., "grpc", "http").
	Port     int    // The port number.
	Protocol string // The protocol used (e.g., "TCP", "UDP").
}
