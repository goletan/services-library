package strategies

import (
	"context"
	"github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type KubernetesDiscovery struct {
	logger    *logger.ZapLogger
	namespace string
}

func NewKubernetesStrategy(log *logger.ZapLogger, namespace string) *KubernetesDiscovery {
	return &KubernetesDiscovery{
		logger:    log,
		namespace: namespace,
	}
}

func (kd *KubernetesDiscovery) Name() string {
	return "kubernetes"
}

func (kd *KubernetesDiscovery) Discover(ctx context.Context, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	kd.logger.Info("Discovering services in Kubernetes", zap.String("namespace", kd.namespace))

	config, err := rest.InClusterConfig()
	if err != nil {
		kd.logger.Error("Failed to load in-cluster config", zap.Error(err))
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		kd.logger.Error("Failed to load new config", zap.Error(err))
		return nil, err
	}

	services, err := clientSet.CoreV1().Services(kd.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		kd.logger.Error("Failed to list services", zap.Error(err))
		return nil, err
	}

	var endpoints []types.ServiceEndpoint
	for _, svc := range services.Items {
		endpoint := types.ServiceEndpoint{
			Name:    svc.Name,
			Address: svc.Spec.ClusterIP,
			Ports:   ConvertPorts(svc.Spec.Ports),
			Tags:    svc.Labels,
		}

		if isDiscoverable(endpoint.Tags, filter) {
			endpoints = append(endpoints, endpoint)
			kd.logger.Info("Added service to the list of discovered services", zap.String("name", svc.Name))
		}
	}

	return endpoints, nil
}

func isDiscoverable(endpointTags map[string]string, filter *types.Filter) bool {
	return MatchTags(endpointTags, filter.Tags) || MatchLabels(endpointTags, filter.Labels)
}

func (kd *KubernetesDiscovery) Watch(ctx context.Context, filter *types.Filter) (<-chan types.ServiceEvent, error) {
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
		0,
		informers.WithNamespace(kd.namespace),
	)

	serviceInformer := informerFactory.Core().V1().Services().Informer()

	_, err = serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			endpoint := types.ServiceEndpoint{
				Name:    svc.Name,
				Address: svc.Spec.ClusterIP,
				Ports:   ConvertPorts(svc.Spec.Ports),
				Tags:    svc.Labels,
			}

			if MatchTags(endpoint.Tags, filter.Tags) {
				eventsChan <- types.ServiceEvent{Type: "ADDED", Service: endpoint}
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			svc := newObj.(*v1.Service)
			endpoint := types.ServiceEndpoint{
				Name:    svc.Name,
				Address: svc.Spec.ClusterIP,
				Ports:   ConvertPorts(svc.Spec.Ports),
				Tags:    svc.Labels,
			}

			if MatchTags(endpoint.Tags, filter.Tags) {
				eventsChan <- types.ServiceEvent{Type: "MODIFIED", Service: endpoint}
			}
		},
		DeleteFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			endpoint := types.ServiceEndpoint{
				Name:    svc.Name,
				Address: svc.Spec.ClusterIP,
				Ports:   ConvertPorts(svc.Spec.Ports),
				Tags:    svc.Labels,
			}

			eventsChan <- types.ServiceEvent{Type: "DELETED", Service: endpoint}
		},
	})

	if err != nil {
		return nil, err
	}

	go func() {
		serviceInformer.Run(ctx.Done())
		close(eventsChan)
	}()

	return eventsChan, nil
}
