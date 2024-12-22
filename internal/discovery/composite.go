package discovery

import (
	"context"
	"fmt"
	logger "github.com/goletan/logger/pkg"
	"github.com/goletan/services/shared/types"
	"go.uber.org/zap"
	"sync"
)

type CompositeDiscovery struct {
	strategies []Strategy
	logger     *logger.ZapLogger
}

func NewCompositeDiscovery(log *logger.ZapLogger, strategies ...Strategy) *CompositeDiscovery {
	return &CompositeDiscovery{
		strategies: strategies,
		logger:     log,
	}
}

func (cd *CompositeDiscovery) Discover(ctx context.Context, namespace string) ([]types.ServiceEndpoint, error) {
	var (
		aggregatedEndpoints []types.ServiceEndpoint
		aggregatedErrors    []error
	)

	for _, strategy := range cd.strategies {
		endpoints, err := strategy.Discover(ctx, namespace)
		if err != nil {
			cd.logger.Warn("Discovery strategy failed", zap.Error(err))
			aggregatedErrors = append(aggregatedErrors, err)
			continue
		}
		aggregatedEndpoints = append(aggregatedEndpoints, endpoints...)
	}

	if len(aggregatedErrors) > 0 {
		return aggregatedEndpoints, fmt.Errorf("some strategies failed: %v", aggregatedErrors)
	}

	return aggregatedEndpoints, nil
}

func (cd *CompositeDiscovery) Watch(ctx context.Context, namespace string) (<-chan types.ServiceEvent, error) {
	// Single aggregated channel for service events
	aggregatedEvents := make(chan types.ServiceEvent)

	// WaitGroup to synchronize the goroutines
	var wg sync.WaitGroup

	// Context to handle cancellations for individual strategies
	watchCtx, cancel := context.WithCancel(ctx)

	// Start a goroutine to collect events from each strategy
	for _, strategy := range cd.strategies {
		wg.Add(1)
		go func(strategy Strategy) {
			defer wg.Done()
			eventCh, err := strategy.Watch(watchCtx, namespace)
			if err != nil {
				cd.logger.Warn("Failed to start watcher for strategy",
					zap.String("strategy", fmt.Sprintf("%T", strategy)),
					zap.Error(err))
				return
			}

			// Forward events to the aggregated channel
			for event := range eventCh {
				select {
				case aggregatedEvents <- event:
				case <-watchCtx.Done():
					return
				}
			}
		}(strategy)
	}

	// Start a goroutine to close the aggregated channel when all watchers are done
	go func() {
		wg.Wait()
		cancel() // Ensure all resources are released
		close(aggregatedEvents)
	}()

	return aggregatedEvents, nil
}
