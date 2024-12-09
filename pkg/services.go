package services

import (
	"context"

	observability "github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"github.com/goletan/services/internal/registry"
	"github.com/goletan/services/shared/types"
)

// Service interface that all services must implement.
type Service interface {
	Name() string
	Initialize() error
	Start() error
	Stop() error
}

// Services struct encapsulates the service registry and metrics.
type Services struct {
	Registry *registry.Registry
	Metrics  *metrics.ServicesMetrics
}

// NewServices creates and returns a new Services instance with observability.
func NewServices(obs *observability.Observability) *Services {
	met := metrics.InitMetrics(obs)
	reg := registry.NewRegistry(obs, met)
	return &Services{
		Registry: reg,
		Metrics:  met,
	}
}

// Register a service in the Registry
func (s *Services) Register(service types.Service) error {
	return s.Registry.Register(service)
}

// InitializeAll initializes all services via registry.
func (s *Services) InitializeAll(ctx context.Context) error {
	return s.Registry.InitializeAll(ctx)
}

// StartAll starts all services via registry.
func (s *Services) StartAll(ctx context.Context) error {
	return s.Registry.StartAll(ctx)
}

// StopAll stops all services via registry.
func (s *Services) StopAll(ctx context.Context) error {
	return s.Registry.StopAll(ctx)
}

// Discover retrieves a list of service endpoints within a specified namespace.
// Returns an array of ServiceEndpoint and an error if the discovery process fails.
func (s *Services) Discover(namespace string) ([]types.ServiceEndpoint, error) {
	return s.Registry.Discover(namespace)
}

// DiscoverByTag retrieves a list of service endpoints filtered by tag within a specific namespace.
func (s *Services) DiscoverByTag(namespace, tag string) ([]types.ServiceEndpoint, error) {
	return s.Registry.DiscoverByTag(namespace, tag)
}

// Watch subscribes to events for services in the specified namespace and tag.
// Returns a channel to receive service events and an error if the operation fails.
func (s *Services) Watch(ctx context.Context, namespace, tag string) (<-chan types.ServiceEvent, error) {
	return s.Registry.Watch(ctx, namespace, tag)
}

// StopWatch stops the service event watcher by cancelling the provided context, ensuring no further events are processed.
func (s *Services) StopWatch(ctx context.Context) error {
	return s.Registry.StopWatch(ctx)
}
