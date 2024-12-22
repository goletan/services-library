package strategies

import (
	"context"
	"fmt"
	"github.com/goletan/logger/pkg"
	"github.com/goletan/services/shared/types"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type KubernetesDiscovery struct {
	client kubernetes.Interface
	logger *logger.ZapLogger
}

func NewKubernetesDiscovery(log *logger.ZapLogger) (*KubernetesDiscovery, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error("Failed to create kubernetes client", zap.Error(err))
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Failed to create kubernetes client", zap.Error(err))
		return nil, err
	}

	return &KubernetesDiscovery{
		client: clientSet,
		logger: log,
	}, nil
}

func (kd *KubernetesDiscovery) Discover(ctx context.Context, namespace string) ([]types.ServiceEndpoint, error) {
	if deadline, ok := ctx.Deadline(); !ok || deadline.IsZero() {
		return nil, fmt.Errorf("context must have a timeout or deadline")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: kd.getLabelSelector(),
	}
	services, err := clientSet.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	var aggregatedEndpoints []types.ServiceEndpoint
	for _, svc := range services.Items {
		endpoint := types.ServiceEndpoint{
			Name:    svc.Name,
			Address: normalizeClusterIP(svc.Spec.ClusterIP),
			Ports:   convertPorts(svc.Spec.Ports),
		}
		if err := validateEndpoint(endpoint); err != nil {
			kd.logger.Warn("Invalid service metadata", zap.Error(err))
			continue
		}
		aggregatedEndpoints = append(aggregatedEndpoints, endpoint)
	}

	return aggregatedEndpoints, nil
}

func (kd *KubernetesDiscovery) Watch(ctx context.Context, namespace string) (<-chan types.ServiceEvent, error) {
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
		0, // No re-sync period
		informers.WithNamespace(namespace),
	)

	serviceInformer := informerFactory.Core().V1().Services().Informer()

	// Add event handlers for added, updated, and deleted events
	_, err = serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			eventsChan <- types.ServiceEvent{
				Type: "ADDED",
				Service: types.ServiceEndpoint{
					Name:    svc.Name,
					Address: svc.Spec.ClusterIP,
					Ports:   convertPorts(svc.Spec.Ports),
				},
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			svc := newObj.(*v1.Service)
			eventsChan <- types.ServiceEvent{
				Type: "MODIFIED",
				Service: types.ServiceEndpoint{
					Name:    svc.Name,
					Address: svc.Spec.ClusterIP,
					Ports:   convertPorts(svc.Spec.Ports),
				},
			}
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			eventsChan <- types.ServiceEvent{
				Type: "DELETED",
				Service: types.ServiceEndpoint{
					Name:    svc.Name,
					Address: svc.Spec.ClusterIP,
					Ports:   convertPorts(svc.Spec.Ports),
				},
			}
		},
	})

	if err != nil {
		kd.logger.Error("Failed to add event handlers for service informer", zap.Error(err))
		return nil, err
	}

	go func() {
		serviceInformer.Run(ctx.Done())
		defer close(eventsChan)
	}()

	return eventsChan, nil
}

func normalizeClusterIP(ip string) string {
	if ip == "None" {
		return ""
	}
	return ip
}

func (kd *KubernetesDiscovery) getLabelSelector() string {
	return "app=discovery"
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

// validateEndpoint checks that all required fields are populated.
func validateEndpoint(endpoint types.ServiceEndpoint) error {
	if endpoint.Name == "" {
		return fmt.Errorf("missing name")
	}
	if endpoint.Address == "" {
		return fmt.Errorf("missing address")
	}
	if len(endpoint.Ports) == 0 {
		return fmt.Errorf("missing ports")
	}
	return nil
}
