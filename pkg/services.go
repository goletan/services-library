// /services/pkg/services.go
package services

import (
	"context"

	observability "github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"github.com/goletan/services/internal/registry"
)

type Services struct {
	Registry *registry.Registry
	Metrics  *metrics.ServicesMetrics
}

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
