package registry

import (
	logger "github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	"sync"
)

// ServiceCache wraps a sync.Map for thread-safe service storage.
type ServiceCache struct {
	logger *logger.ZapLogger
	cache  sync.Map
}

func NewCache(logger *logger.ZapLogger) *ServiceCache {
	return &ServiceCache{
		logger: logger,
	}
}

// store stores a key-value pair in the cache.
func (sc *ServiceCache) store(name string, service types.Service) {
	sc.cache.Store(name, service)
	sc.logger.Debug("Added to cache", zap.String("name", name))
}

// get retrieves a value by key from the cache, returning the value and a bool indicating success.
func (sc *ServiceCache) get(name string) (types.Service, bool) {
	value, exists := sc.cache.Load(name)
	if !exists {
		sc.logger.Warn("Service not found in cache", zap.String("name", name))
		return nil, false
	}

	service, ok := value.(types.Service)
	if !ok {
		sc.logger.Error("Cache contains invalid service type", zap.String("name", name))
		return nil, false
	}

	return service, true
}

// exists checks if a key exists in the cache.
func (sc *ServiceCache) exists(name string) bool {
	_, exists := sc.cache.Load(name)
	return exists
}

// rangeAll iterates over all items in the cache and applies the handler function to each.
func (sc *ServiceCache) rangeAll(handler func(name string, service types.Service)) {
	sc.cache.Range(func(key, value interface{}) bool {
		handler(key.(string), value.(types.Service))
		return true
	})
}
