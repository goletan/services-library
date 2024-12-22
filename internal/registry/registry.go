package registry

import (
	"context"
	"fmt"
	"github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"github.com/goletan/services/shared/types"
	"go.uber.org/zap"
	"sync"
)

// Registry manages the lifecycle of services.
type Registry struct {
	servicesCache *serviceCache
	obs           *observability.Observability
	metrics       *metrics.ServicesMetrics
}

// NewRegistry creates a new Registry instance with observability and metrics.
func NewRegistry(obs *observability.Observability, met *metrics.ServicesMetrics) *Registry {
	return &Registry{
		servicesCache: newServiceCache(),
		obs:           obs,
		metrics:       met,
	}
}

// Register adds a new service to the registry.
func (r *Registry) Register(service types.Service) error {
	name := service.Name()
	if r.servicesCache.exists(name) {
		r.obs.Logger.Error("Service already registered", zap.String("service", name))
		return fmt.Errorf("service already registered: %s", name)
	}

	r.servicesCache.store(name, service)
	r.obs.Logger.Info("Service registered", zap.String("service", name))
	return nil
}

// InitializeAll initializes all registered services with observability.
func (r *Registry) InitializeAll(ctx context.Context) error {
	return r.processAllServices(ctx, "initialize", func(ctx context.Context, svc types.Service) error {
		return svc.Initialize()
	})
}

// StartAll starts all registered services with observability.
func (r *Registry) StartAll(ctx context.Context) error {
	return r.processAllServices(ctx, "start", func(ctx context.Context, svc types.Service) error {
		return svc.Start()
	})
}

// StopAll stops all registered services with observability.
func (r *Registry) StopAll(ctx context.Context) error {
	return r.processAllServices(ctx, "stop", func(ctx context.Context, svc types.Service) error {
		return svc.Stop()
	})
}

// processAllServices applies an operation (initialize/start/stop) to all registered services.
func (r *Registry) processAllServices(ctx context.Context, action string, operation func(context.Context, types.Service) error) error {
	var operationErrors []error

	r.servicesCache.rangeAll(func(name string, service types.Service) {
		_, span := r.obs.Tracer.Start(ctx, fmt.Sprintf("%s-service-%s", action, name))
		err := operation(ctx, service)
		span.End()

		if err != nil {
			operationErrors = append(operationErrors, err)
			r.obs.Logger.Error(fmt.Sprintf("Failed to %s service", action), zap.String("service", name), zap.Error(err))
		}
	})

	if len(operationErrors) > 0 {
		return fmt.Errorf("failed to %s one or more services: %v", action, operationErrors)
	}

	return nil
}

// serviceCache wraps a sync.Map for thread-safe service storage.
type serviceCache struct {
	cache sync.Map
}

func newServiceCache() *serviceCache {
	return &serviceCache{}
}

func (sc *serviceCache) store(name string, service types.Service) {
	sc.cache.Store(name, service)
}

func (sc *serviceCache) exists(name string) bool {
	_, exists := sc.cache.Load(name)
	return exists
}

func (sc *serviceCache) rangeAll(handler func(name string, service types.Service)) {
	sc.cache.Range(func(key, value interface{}) bool {
		handler(key.(string), value.(types.Service))
		return true
	})
}
