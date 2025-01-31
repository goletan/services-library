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
	sc.logger.Info("Storing service in cache", zap.String("name", name))
	sc.cache.Store(name, service)
	sc.logger.Info("Added to cache", zap.String("name", name))
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

// delete removes a service from the cache.
func (sc *ServiceCache) delete(name string) {
	sc.cache.Delete(name)
	sc.logger.Info("Removed from cache", zap.String("name", name))
}

// exists checks if a key exists in the cache.
func (sc *ServiceCache) exists(name string) bool {
	_, exists := sc.cache.Load(name)
	return exists
}

// rangeAll iterates over all services in the cache and applies a handler function.
func (sc *ServiceCache) rangeAll(handler func(name string, service types.Service)) {
	sc.logger.Info("Iterating over cache...")
	sc.cache.Range(func(key, value interface{}) bool {
		service, ok := value.(types.Service)
		if ok {
			sc.logger.Info("Handling service", zap.String("name", key.(string)))
			handler(key.(string), service)
		}
		return true
	})
}
