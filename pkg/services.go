// /services/pkg/services.go
package services

import (
	"context"

	observability "github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"github.com/goletan/services/internal/registry"
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
