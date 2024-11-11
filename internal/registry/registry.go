// /services/internal/registry/registry.go
package registry

import (
	"context"
	"sync"
	"time"

	observability "github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"go.uber.org/zap"
)

// Service interface that all services must implement.
type Service interface {
	Name() string
	Initialize() error
	Start() error
	Stop() error
}

// Registry manages the lifecycle of services.
type Registry struct {
	services      map[string]Service
	mu            sync.RWMutex
	observability *observability.Observability
	metrics       *metrics.ServicesMetrics
}

// NewRegistry creates a new Registry instance with observability and metrics.
func NewRegistry(obs *observability.Observability, met *metrics.ServicesMetrics) *Registry {
	return &Registry{
		services:      make(map[string]Service),
		observability: obs,
		metrics:       met,
	}
}

// RegisterService adds a new service to the registry.
func (r *Registry) RegisterService(service Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		r.observability.Logger.Error("service %s already registered", zap.String("service", name))
		return nil
	}

	r.services[name] = service
	r.observability.Logger.Info("Service registered", zap.String("service", name))
	return nil
}

// InitializeAll initializes all registered services with observability.
func (r *Registry) InitializeAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var initErrors []error
	for name, service := range r.services {
		_, span := r.observability.Tracer.Start(ctx, "InitializeService")
		defer span.End()

		startTime := time.Now()
		err := service.Initialize()
		duration := time.Since(startTime).Seconds()

		r.metrics.ObserveExecution(name, "initialize", duration)

		if err != nil {
			initErrors = append(initErrors, err)
			r.observability.Logger.Error("Failed to initialize service", zap.String("service", name), zap.Error(err))
		}
	}

	if len(initErrors) > 0 {
		r.observability.Logger.Error("One or more services failed to initialize", zap.Errors("errors", initErrors))
		return initErrors[0] // Return the first error
	}
	return nil
}

// StartAll starts all registered services with observability.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var startErrors []error
	for name, service := range r.services {
		_, span := r.observability.Tracer.Start(ctx, "StartService")
		defer span.End()

		startTime := time.Now()
		err := service.Start()
		duration := time.Since(startTime).Seconds()

		r.metrics.ObserveExecution(name, "start", duration)

		if err != nil {
			startErrors = append(startErrors, err)
			r.observability.Logger.Error("Failed to start service", zap.String("service", name), zap.Error(err))
		}
	}

	if len(startErrors) > 0 {
		r.observability.Logger.Error("One or more services failed to start", zap.Errors("errors", startErrors))
		return startErrors[0] // Return the first error
	}
	return nil
}

// StopAll stops all registered services with observability.
func (r *Registry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var stopErrors []error
	for name, service := range r.services {
		_, span := r.observability.Tracer.Start(ctx, "StopService")
		defer span.End()

		startTime := time.Now()
		err := service.Stop()
		duration := time.Since(startTime).Seconds()

		r.metrics.ObserveExecution(name, "stop", duration)

		if err != nil {
			stopErrors = append(stopErrors, err)
			r.observability.Logger.Error("Failed to stop service", zap.String("service", name), zap.Error(err))
		}
	}

	if len(stopErrors) > 0 {
		r.observability.Logger.Error("One or more services failed to stop", zap.Errors("errors", stopErrors))
		return stopErrors[0] // Return the first error
	}
	return nil
}
