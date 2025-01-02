package services

import (
	"context"
	"fmt"
	observability "github.com/goletan/observability-library/pkg"
	"github.com/goletan/services-library/internal/discovery"
	"github.com/goletan/services-library/internal/discovery/strategies"
	"github.com/goletan/services-library/internal/metrics"
	"github.com/goletan/services-library/internal/registry"
	"github.com/goletan/services-library/shared/types"
	"sync"
)

// Services encapsulates service discovery, registration, and lifecycle management.
type Services struct {
	discovery       *discovery.CompositeDiscovery
	registry        *registry.Registry
	metrics         *metrics.ServicesMetrics
	factoryRegistry sync.Map
}

// NewServices initializes a new Services instance with strategy-based discovery mechanisms.
func NewServices(obs *observability.Observability) (*Services, error) {

	discoveryStrategies := []types.Strategy{
		strategies.NewKubernetesStrategy(obs.Logger),
		strategies.NewDockerSwarmStrategy(obs.Logger),
		strategies.NewDNSStrategy(obs.Logger),
	}

	compositeDiscovery := discovery.NewCompositeDiscovery(
		obs.Logger,
		discoveryStrategies...,
	)

	// Initialize registry and metrics
	newMetrics := metrics.InitMetrics(obs)
	newRegistry := registry.NewRegistry(obs, newMetrics)

	return &Services{
		discovery: compositeDiscovery,
		registry:  newRegistry,
		metrics:   newMetrics,
	}, nil
}

// RegisterFactory registers a factory for dynamically creating services-library.
func (s *Services) RegisterFactory(name string, factory types.ServiceFactory) {
	s.factoryRegistry.Store(name, factory)
}

// CreateService dynamically creates a Service using a registered factory.
func (s *Services) CreateService(endpoint types.ServiceEndpoint) (types.Service, error) {
	factoryInterface, ok := s.factoryRegistry.Load(endpoint.Name)
	if !ok {
		return nil, fmt.Errorf("no factory registered for service: %s", endpoint.Name)
	}
	factory, ok := factoryInterface.(types.ServiceFactory)
	if !ok {
		return nil, fmt.Errorf("invalid factory for service: %s", endpoint.Name)
	}
	return factory(endpoint), nil
}

// Discover discovers all services-library in a namespace.
func (s *Services) Discover(ctx context.Context, namespace string, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	return s.discovery.Discover(ctx, namespace, filter)
}

// Watch discovers all services-library in a namespace.
func (s *Services) Watch(ctx context.Context, namespace string, filter *types.Filter) (<-chan types.ServiceEvent, error) {
	return s.discovery.Watch(ctx, namespace, filter)
}

// Register registers a service in the registry.
func (s *Services) Register(service types.Service) error {
	return s.registry.Register(service)
}

// InitializeAll initializes all registered services-library in the registry.
func (s *Services) InitializeAll(ctx context.Context) error {
	return s.registry.InitializeAll(ctx)
}

// StartAll starts all registered services-library in the registry.
func (s *Services) StartAll(ctx context.Context) error {
	return s.registry.StartAll(ctx)
}

// StopAll stops all registered services-library in the registry.
func (s *Services) StopAll(ctx context.Context) error {
	return s.registry.StopAll(ctx)
}
