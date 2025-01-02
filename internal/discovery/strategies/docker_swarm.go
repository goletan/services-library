package strategies

import (
	"context"
	"fmt"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	logger "github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	"strings"
)

type DockerSwarmStrategy struct {
	logger *logger.ZapLogger
}

// NewDockerSwarmStrategy creates a new Docker Swarm discovery strategy.
func NewDockerSwarmStrategy(logger *logger.ZapLogger) *DockerSwarmStrategy {
	return &DockerSwarmStrategy{logger: logger}
}

// Name returns the name of the strategy.
func (d *DockerSwarmStrategy) Name() string {
	return "docker_swarm"
}

func (d *DockerSwarmStrategy) Discover(ctx context.Context, namespace string, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	d.logger.Info("Using Docker Swarm strategy for service discovery", zap.String("namespace", namespace))

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		d.logger.Error("Failed to create Docker client", zap.Error(err))
		return nil, err
	}

	// Find the target network by namespace
	var targetNetworkID string
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		d.logger.Error("Failed to list Docker Swarm networks", zap.Error(err))
		return nil, err
	}

	for _, net := range networks {
		if net.Name == namespace {
			targetNetworkID = net.ID
			break
		}
	}

	if targetNetworkID == "" {
		d.logger.Warn("No matching network found for namespace", zap.String("namespace", namespace))
		return nil, fmt.Errorf("network not found: %s", namespace)
	}

	// List services in Swarm
	services, err := cli.ServiceList(ctx, dockerTypes.ServiceListOptions{})
	if err != nil {
		d.logger.Error("Failed to list Docker Swarm services", zap.Error(err))
		return nil, err
	}

	// Extract endpoints and apply filtering
	var endpoints []types.ServiceEndpoint
	for _, service := range services {
		for _, vip := range service.Endpoint.VirtualIPs {
			if vip.NetworkID == targetNetworkID {
				endpoint := types.ServiceEndpoint{
					Name:    service.Spec.Name,
					Address: strings.Split(vip.Addr, "/")[0],
				}

				// Apply filters (if provided)
				if filter != nil {
					if !MatchLabels(service.Spec.Labels, filter.Labels) {
						continue
					}
					if !MatchTags(service.Spec.Annotations.Labels, filter.Tags) {
						continue
					}
				}

				endpoints = append(endpoints, endpoint)
			}
		}
	}

	if len(endpoints) == 0 {
		d.logger.Warn("No services discovered in the target network", zap.String("namespace", namespace))
		return nil, fmt.Errorf("no services discovered in namespace: %s", namespace)
	}

	// Log discovered endpoints
	for _, endpoint := range endpoints {
		d.logger.Info("Discovered Service Endpoint:",
			zap.String("name", endpoint.Name),
			zap.String("address", endpoint.Address))
	}

	return endpoints, nil
}

// Watch watches for service changes in the Docker Swarm namespace.
func (d *DockerSwarmStrategy) Watch(ctx context.Context, namespace string, filter *types.Filter) (<-chan types.ServiceEvent, error) {
	// TODO: Implement watch functionality for Docker Swarm services
	d.logger.Info("Watching Docker Swarm services is not implemented yet", zap.String("namespace", namespace))
	return nil, fmt.Errorf("watch functionality not implemented")
}
