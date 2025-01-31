package services

import (
	"context"
	observability "github.com/goletan/observability-library/pkg"
	"github.com/goletan/services-library/internal/config"
	"github.com/goletan/services-library/internal/discovery"
	"github.com/goletan/services-library/internal/metrics"
	"github.com/goletan/services-library/internal/registry"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
)

// Services encapsulates service discovery, registration, and lifecycle management.
type Services struct {
	cfg       *types.ServicesConfig
	discovery *discovery.CompositeDiscovery
	registry  *registry.Registry
	metrics   *metrics.ServicesMetrics
}

// NewServices initializes a new Services instance with strategy-based discovery mechanisms.
func NewServices(obs *observability.Observability) (*Services, error) {
	cfg, err := config.LoadServicesConfig(obs.Logger)
	if err != nil {
		obs.Logger.Fatal("Failed to load services-library configuration", zap.Error(err))
	}

	compositeDiscovery := discovery.NewCompositeDiscovery(obs.Logger, cfg)

	// Initialize registry and metrics
	newMetrics := metrics.InitMetrics(obs)
	newRegistry := registry.NewRegistry(obs, newMetrics)

	return &Services{
		cfg:       cfg,
		discovery: compositeDiscovery,
		registry:  newRegistry,
		metrics:   newMetrics,
	}, nil
}

// Discover discovers all services-library in a namespace.
func (s *Services) Discover(ctx context.Context, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	return s.discovery.Discover(ctx, filter)
}

// Watch discovers all services-library in a namespace.
func (s *Services) Watch(ctx context.Context, filter *types.Filter) (<-chan types.ServiceEvent, error) {
	return s.discovery.Watch(ctx, filter)
}

// Register registers a service in the registry.
func (s *Services) Register(endpoint types.ServiceEndpoint) (types.Service, error) {
	return s.registry.Register(endpoint)
}

// Unregister unregisters a service in the registry
func (s *Services) Unregister(name string) error {
	return s.registry.Unregister(name)
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

// GetService retrieves a service by name from the registry. Returns the service and a boolean indicating if it was found.
func (s *Services) GetService(name string) (types.Service, bool) {
	return s.registry.GetService(name)
}

// List retrieves all registered Services
func (s *Services) List() []types.Service {
	return s.registry.List()
}
