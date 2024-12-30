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

// Discover discovers services in the Docker Swarm namespace.
func (d *DockerSwarmStrategy) Discover(ctx context.Context, namespace string) ([]types.ServiceEndpoint, error) {
	d.logger.Info("Using Docker Swarm strategy for service discovery", zap.String("namespace", namespace))

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		d.logger.Error("Failed to create Docker client", zap.Error(err))
		return nil, err
	}

	var targetNetworkID string
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		d.logger.Error("Failed to list Docker Swarm networks", zap.Error(err))
		return nil, err
	}

	for _, net := range networks {
		d.logger.Info("Network:", zap.String("name", net.Name), zap.String("id", net.ID))
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

	// Extract endpoints from services
	var endpoints []types.ServiceEndpoint
	for _, service := range services {
		for _, vip := range service.Endpoint.VirtualIPs {
			address := strings.Split(vip.Addr, "/")[0]
			if vip.NetworkID == targetNetworkID {
				endpoints = append(endpoints, types.ServiceEndpoint{
					Name:    service.Spec.Name,
					Address: address,
				})
			}
		}
	}

	// Return results or error if no services found
	if len(endpoints) == 0 {
		d.logger.Warn("No services discovered in the target network", zap.String("namespace", namespace))
		return nil, fmt.Errorf("no services discovered in namespace: %s", namespace)
	}

	for _, endpoint := range endpoints {
		d.logger.Info("Discovered Service Endpoint:",
			zap.String("name", endpoint.Name),
			zap.String("address", endpoint.Address))
	}

	return endpoints, nil
}

// Name returns the name of the strategy.
func (d *DockerSwarmStrategy) Name() string {
	return "docker_swarm"
}

// Watch watches for service changes in the Docker Swarm namespace.
func (d *DockerSwarmStrategy) Watch(ctx context.Context, namespace string) (<-chan types.ServiceEvent, error) {
	// TODO: Implement watch functionality for Docker Swarm services
	d.logger.Info("Watching Docker Swarm services is not implemented yet", zap.String("namespace", namespace))
	return nil, fmt.Errorf("watch functionality not implemented")
}
