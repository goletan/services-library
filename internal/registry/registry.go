package registry

import (
	"context"
	"fmt"
	"github.com/goletan/observability-library/pkg"
	"github.com/goletan/services-library/internal/metrics"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	"time"
)

// ServiceConstructor defines a function that creates a new Service from an endpoint.
type ServiceConstructor func(endpoint types.ServiceEndpoint) (types.Service, error)

// Registry manages the lifecycle of services-library.
type Registry struct {
	obs     *observability.Observability
	metrics *metrics.ServicesMetrics
	cache   *ServiceCache
}

// NewRegistry creates a new Registry instance with observability-library and metrics.
func NewRegistry(obs *observability.Observability, met *metrics.ServicesMetrics) *Registry {
	serviceCache := NewCache(obs.Logger)
	return &Registry{
		obs:     obs,
		metrics: met,
		cache:   serviceCache,
	}
}

// Register creates and registers a service.
func (r *Registry) Register(endpoint types.ServiceEndpoint) (types.Service, error) {
	// TODO: Improve check by Ports/Address/Tags
	if r.cache.exists(endpoint.Name) {
		r.obs.Logger.Warn("Service already registered", zap.String("service", endpoint.Name))
		return nil, fmt.Errorf("service already registered: %s", endpoint.Name)
	}

	service := NewService(endpoint)
	r.cache.store(endpoint.Name, service)
	r.obs.Logger.Info("Service registered", zap.String("service", endpoint.Name))
	return service, nil
}

func (r *Registry) Unregister(name string) error {
	// Check if the service exists
	service, exists := r.cache.get(name)
	if !exists {
		r.obs.Logger.Warn("Attempted to unregister a non-existent service", zap.String("service", name))
		return fmt.Errorf("service not found: %s", name)
	}

	// Stop the service before removal
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Stop(ctx); err != nil {
		r.obs.Logger.Error("Failed to stop service before unregistering", zap.String("service", name), zap.Error(err))
		return fmt.Errorf("failed to stop service '%s' before unregistering: %w", name, err)
	}

	// Remove from cache
	r.cache.delete(name)
	r.obs.Logger.Info("Service unregistered successfully", zap.String("service", name))
	return nil
}

// GetService retrieves a registered service.
func (r *Registry) GetService(name string) (types.Service, bool) {
	service, exists := r.cache.get(name)
	if !exists {
		return nil, false
	}

	return service.(types.Service), true
}

// InitializeAll initializes all registered services-library with observability-library.
func (r *Registry) InitializeAll(ctx context.Context) error {
	return r.processAllServices(ctx, "initialize", func(ctx context.Context, service types.Service) error {
		return service.Initialize()
	})
}

// StartAll starts all registered services-library with observability-library.
func (r *Registry) StartAll(ctx context.Context) error {
	return r.processAllServices(ctx, "start", func(ctx context.Context, service types.Service) error {
		return service.Start(ctx)
	})
}

// StopAll stops all registered services-library with observability-library.
func (r *Registry) StopAll(ctx context.Context) error {
	return r.processAllServices(ctx, "stop", func(ctx context.Context, service types.Service) error {
		return service.Stop(ctx)
	})
}

func (r *Registry) List() []types.Service {
	var servicesList []types.Service
	r.cache.rangeAll(func(name string, service types.Service) {
		servicesList = append(servicesList, service)
	})

	return servicesList
}

// processAllServices applies an operation (initialize/start/stop) to all registered services-library.
func (r *Registry) processAllServices(ctx context.Context, action string, operation func(context.Context, types.Service) error) error {
	var operationErrors []error

	r.cache.rangeAll(func(name string, service types.Service) {
		_, span := r.obs.Tracer.Start(ctx, fmt.Sprintf("%s-service-%s", action, name))
		err := operation(ctx, service)
		span.End()

		if err != nil {
			operationErrors = append(operationErrors, err)
			r.obs.Logger.Error(fmt.Sprintf("Failed to %s service", action), zap.String("service", name), zap.Error(err))
		}
	})

	if len(operationErrors) > 0 {
		return fmt.Errorf("failed to %s one or more services-library: %v", action, operationErrors)
	}

	return nil
}
