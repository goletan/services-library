package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goletan/observability/pkg"
	"github.com/goletan/services/internal/metrics"
	"github.com/goletan/services/shared/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"go.uber.org/zap"
)

// Registry manages the lifecycle of services.
type Registry struct {
	services      map[string]types.Service
	mu            sync.RWMutex
	observability *observability.Observability
	metrics       *metrics.ServicesMetrics
}

// NewRegistry creates a new Registry instance with observability and metrics.
func NewRegistry(obs *observability.Observability, met *metrics.ServicesMetrics) *Registry {
	return &Registry{
		services:      make(map[string]types.Service),
		observability: obs,
		metrics:       met,
	}
}

// Register adds a new service to the registry.
func (r *Registry) Register(service types.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		r.observability.Logger.Error("Service already registered", zap.String("service", name))
		return errors.New("service already registered: " + name)
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
		startTime := time.Now()

		err := service.Initialize()
		duration := time.Since(startTime).Seconds()
		r.metrics.ObserveExecution(name, "initialize", duration)
		span.End()

		if err != nil {
			initErrors = append(initErrors, err)
			r.observability.Logger.Error("Failed to initialize service", zap.String("service", name), zap.Error(err))
		}
	}

	if len(initErrors) > 0 {
		errMsg := aggregateErrors(initErrors)
		r.observability.Logger.Error("Failed to initialize services", zap.String("errors", errMsg))
		return fmt.Errorf("initialization errors: %s", errMsg)
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
		startTime := time.Now()

		err := service.Start()
		duration := time.Since(startTime).Seconds()
		r.metrics.ObserveExecution(name, "start", duration)
		span.End()

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
		startTime := time.Now()

		err := service.Stop()
		duration := time.Since(startTime).Seconds()
		r.metrics.ObserveExecution(name, "stop", duration)
		span.End()

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

func (r *Registry) Discover(namespace string) ([]types.ServiceEndpoint, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	services, err := clientSet.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var endpoints []types.ServiceEndpoint
	for _, svc := range services.Items {
		endpoints = append(endpoints, types.ServiceEndpoint{
			Name:    svc.Name,
			Address: svc.Spec.ClusterIP,
		})
	}

	return endpoints, nil
}

func (r *Registry) DiscoverByTag(namespace, tag string) ([]types.ServiceEndpoint, error) {
	endpoints, err := r.Discover(namespace)
	if err != nil {
		return nil, err
	}

	var filtered []types.ServiceEndpoint
	for _, endpoint := range endpoints {
		for _, endpointTag := range endpoint.Tags {
			if endpointTag == tag {
				filtered = append(filtered, endpoint)
				break
			}
		}
	}
	return filtered, nil
}

func (r *Registry) Watch(ctx context.Context, namespace, tag string) (<-chan types.ServiceEvent, error) {
	eventsChan := make(chan types.ServiceEvent)
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		clientSet,
		0, // No resync period
		informers.WithNamespace(namespace),
	)

	serviceInformer := informerFactory.Core().V1().Services().Informer()

	// Add event handlers for added, updated, and deleted events
	_, err = serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			if hasTag(svc, tag) {
				eventsChan <- types.ServiceEvent{
					Type: "ADDED",
					Service: types.ServiceEndpoint{
						Name:    svc.Name,
						Address: svc.Spec.ClusterIP,
						Ports:   convertPorts(svc.Spec.Ports),
					},
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			svc := newObj.(*v1.Service)
			if hasTag(svc, tag) {
				eventsChan <- types.ServiceEvent{
					Type: "MODIFIED",
					Service: types.ServiceEndpoint{
						Name:    svc.Name,
						Address: svc.Spec.ClusterIP,
						Ports:   convertPorts(svc.Spec.Ports),
					},
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			if hasTag(svc, tag) {
				eventsChan <- types.ServiceEvent{
					Type: "DELETED",
					Service: types.ServiceEndpoint{
						Name:    svc.Name,
						Address: svc.Spec.ClusterIP,
						Ports:   convertPorts(svc.Spec.Ports),
					},
				}
			}
		},
	})

	if err != nil {
		r.observability.Logger.Error("Failed to add event handlers for service informer", zap.Error(err))
		return nil, err
	}

	go serviceInformer.Run(ctx.Done())

	return eventsChan, nil
}

func (r *Registry) StopWatch(ctx context.Context) error {
	r.observability.Logger.Info("Stopping service watcher...")
	// Simply cancel the context to stop watching
	if cancelFunc, ok := ctx.Value("cancelFunc").(context.CancelFunc); ok {
		cancelFunc()
	}
	return nil
}

// Helper function to check if a service has the desired tag
func hasTag(svc *v1.Service, tag string) bool {
	// Example: Tags could be stored in annotations
	for k, v := range svc.Annotations {
		if k == "tag" && v == tag {
			return true
		}
	}
	return false
}

func convertPorts(k8sPorts []v1.ServicePort) []types.ServicePort {
	var servicePorts []types.ServicePort
	for _, k8sPort := range k8sPorts {
		tsPort := types.ServicePort{
			Name:     k8sPort.Name,
			Port:     int(k8sPort.Port),
			Protocol: string(k8sPort.Protocol),
		}
		servicePorts = append(servicePorts, tsPort)
	}
	return servicePorts
}

func aggregateErrors(errors []error) string {
	var errMsg string
	for _, err := range errors {
		errMsg += err.Error() + "; "
	}

	return errMsg
}
