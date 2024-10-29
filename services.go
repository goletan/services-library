// /services/registry.go
package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/goletan/observability/errors"
	"github.com/goletan/observability/logger"
	"github.com/goletan/observability/metrics"
	"github.com/goletan/observability/tracing"
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
	services map[string]Service
	mu       sync.RWMutex
}

// NewRegistry creates a new Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]Service),
	}
}

// RegisterService adds a new service to the registry.
func (r *Registry) RegisterService(service Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		return errors.NewError("service already registered", 400, map[string]interface{}{"service": name})
	}

	r.services[name] = service
	logger.Info("Service registered", zap.String("service", name))
	return nil
}

// InitializeAll initializes all registered services with observability.
func (r *Registry) InitializeAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var initErrors []error
	for name, service := range r.services {
		_, span := tracing.StartSpan(ctx, "InitializeService", map[string]interface{}{"service": name})
		log.Printf("Initializing service: %s", name)

		startTime := time.Now()
		err := service.Initialize()
		metrics.ObserveServiceExecution(name, "initialize", time.Since(startTime).Seconds())
		tracing.EndSpan(span, err)

		if err != nil {
			initErrors = append(initErrors, errors.WrapError(err, "failed to initialize service", 500, map[string]interface{}{"service": name}))
		}
	}

	if len(initErrors) > 0 {
		return errors.NewError("initialization errors", 500, map[string]interface{}{"errors": initErrors})
	}
	return nil
}

// StartAll starts all registered services with observability.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var startErrors []error
	for name, service := range r.services {
		_, span := tracing.StartSpan(ctx, "StartService", map[string]interface{}{"service": name})
		log.Printf("Starting service: %s", name)

		startTime := time.Now()
		err := service.Start()
		metrics.ObserveServiceExecution(name, "start", time.Since(startTime).Seconds())
		tracing.EndSpan(span, err)

		if err != nil {
			startErrors = append(startErrors, errors.WrapError(err, "failed to start service", 500, map[string]interface{}{"service": name}))
		}
	}

	if len(startErrors) > 0 {
		return errors.NewError("start errors", 500, map[string]interface{}{"errors": startErrors})
	}
	return nil
}

// StopAll stops all registered services with observability.
func (r *Registry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var stopErrors []error
	for name, service := range r.services {
		_, span := tracing.StartSpan(ctx, "StopService", map[string]interface{}{"service": name})
		log.Printf("Stopping service: %s", name)

		startTime := time.Now()
		err := service.Stop()
		metrics.ObserveServiceExecution(name, "stop", time.Since(startTime).Seconds())
		tracing.EndSpan(span, err)

		if err != nil {
			stopErrors = append(stopErrors, errors.WrapError(err, "failed to stop service", 500, map[string]interface{}{"service": name}))
		}
	}

	if len(stopErrors) > 0 {
		return errors.NewError("stop errors", 500, map[string]interface{}{"errors": stopErrors})
	}
	return nil
}
