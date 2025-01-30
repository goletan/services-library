package registry

import (
	"context"
	"github.com/goletan/services-library/shared/types"
)

// Service is a fallback implementation for unknown services.
type Service struct {
	ServiceName    string
	ServiceAddress string
	ServiceType    string
	Ports          []types.ServicePort
	Version        string
	Tags           map[string]string
}

func NewService(endpoint types.ServiceEndpoint) types.Service {
	return &Service{
		ServiceName:    endpoint.Name,
		ServiceAddress: endpoint.Address,
		ServiceType:    endpoint.Type,
		Ports:          endpoint.Ports,
		Version:        endpoint.Version,
		Tags:           endpoint.Tags,
	}
}

func (g *Service) Name() string {
	return g.ServiceName
}

func (g *Service) Type() string {
	return g.ServiceType // Fixed to return actual service type
}

func (g *Service) Address() string {
	return g.ServiceAddress
}

func (g *Service) Metadata() map[string]string {
	return g.Tags
}

func (g *Service) Initialize() error {
	return nil
}

func (g *Service) Start(ctx context.Context) error {
	return nil
}

func (g *Service) Stop(ctx context.Context) error {
	return nil
}
