package discovery

import (
	"context"
	"github.com/goletan/services/shared/types"
)

// Strategy defines the interface for service discovery mechanisms.
type Strategy interface {
	Discover(ctx context.Context, namespace string) ([]types.ServiceEndpoint, error)
	Watch(ctx context.Context, namespace string) (<-chan types.ServiceEvent, error)
}
