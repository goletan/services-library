package discovery

import (
	"context"
	"fmt"
	logger "github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	"sync"
)

type CompositeDiscovery struct {
	strategies []types.Strategy
	logger     *logger.ZapLogger
}

func NewCompositeDiscovery(log *logger.ZapLogger, strategies ...types.Strategy) *CompositeDiscovery {
	return &CompositeDiscovery{
		strategies: strategies,
		logger:     log,
	}
}

// AddStrategy support for dynamic updates of strategies.
func (cd *CompositeDiscovery) AddStrategy(strategy types.Strategy) {
	cd.logger.Info("Adding discovery strategy", zap.String("strategy", strategy.Name()))
	cd.strategies = append(cd.strategies, strategy)
}

func (cd *CompositeDiscovery) RemoveStrategy(name string) error {
	for i, strategy := range cd.strategies {
		if strategy.Name() == name {
			cd.logger.Info("Removing discovery strategy", zap.String("strategy", name))
			cd.strategies = append(cd.strategies[:i], cd.strategies[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("strategy not found: %s", name)
}

func (cd *CompositeDiscovery) Discover(ctx context.Context, namespace string, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	var discovered []types.ServiceEndpoint
	var strategyErrors []error

	for _, strategy := range cd.strategies {
		cd.logger.Info("Attempting service discovery using strategy", zap.String("strategy", strategy.Name()))
		endpoints, err := strategy.Discover(ctx, namespace, filter)
		if err != nil {
			cd.logger.Warn("Discovery strategy failed", zap.String("strategy", strategy.Name()), zap.Error(err))
			strategyErrors = append(strategyErrors, err)
		} else {
			discovered = append(discovered, endpoints...)
		}
	}

	if len(discovered) == 0 {
		return nil, fmt.Errorf("no services discovered in namespace: %s, errors: %v", namespace, strategyErrors)
	}

	return discovered, nil
}

func (cd *CompositeDiscovery) Watch(ctx context.Context, namespace string, filter *types.Filter) (<-chan types.ServiceEvent, error) {
	// Single aggregated channel for service events-service
	aggregatedEvents := make(chan types.ServiceEvent)

	// WaitGroup to synchronize the goroutines
	var wg sync.WaitGroup

	// Context to handle cancellations for individual strategies
	watchCtx, cancel := context.WithCancel(ctx)

	// Start a goroutine to collect events-service from each strategy
	for _, strategy := range cd.strategies {
		wg.Add(1)
		go func(strategy types.Strategy) {
			defer wg.Done()
			eventCh, err := strategy.Watch(watchCtx, namespace, filter)
			if err != nil {
				cd.logger.Warn("Failed to start watcher for strategy",
					zap.String("strategy", fmt.Sprintf("%T", strategy)),
					zap.Error(err))
				return
			}

			// Forward events-service to the aggregated channel
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
