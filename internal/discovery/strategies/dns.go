package strategies

import (
	"context"
	logger "github.com/goletan/logger/pkg"
	"github.com/goletan/services/shared/types"
	"go.uber.org/zap"
	"net"
	"time"
)

type DNSDiscovery struct {
	logger *logger.ZapLogger
}

func NewDNSDiscovery(logger *logger.ZapLogger) (*DNSDiscovery, error) {
	return &DNSDiscovery{logger: logger}, nil
}

func (d *DNSDiscovery) Discover(ctx context.Context, namespace string) ([]types.ServiceEndpoint, error) {
	records, err := net.LookupTXT(namespace + ".services.local")
	if err != nil {
		d.logger.Warn("DNS lookup failed", zap.Error(err))
		return nil, err
	}

	var endpoints []types.ServiceEndpoint
	for _, record := range records {
		endpoints = append(endpoints, parseTXTRecord(record))
	}

	return endpoints, nil
}

func (d *DNSDiscovery) Watch(ctx context.Context, namespace string) (<-chan types.ServiceEvent, error) {
	// Create a channel to send updates of service events
	serviceEventCh := make(chan types.ServiceEvent)

	// Start a goroutine to watch for DNS changes
	go func() {
		defer close(serviceEventCh)
		ticker := time.NewTicker(10 * time.Second) // Polling interval
		defer ticker.Stop()

		// Map to track previously seen records
		prevRecords := make(map[string]types.ServiceEndpoint)

		for {
			select {
			case <-ctx.Done():
				// Exit the goroutine when the context is canceled
				d.logger.Info("Stopping DNS discovery watcher...")
				return
			case <-ticker.C:
				// Perform DNS lookup
				records, err := net.LookupTXT(namespace + ".services.local")
				if err != nil {
					d.logger.Warn("DNS lookup failed", zap.Error(err))
					continue
				}

				// Map to store current records
				currentRecords := make(map[string]types.ServiceEndpoint)

				for _, record := range records {
					endpoint := parseTXTRecord(record)

					// Validate endpoint
					if err := validateEndpoint(endpoint); err != nil {
						d.logger.Warn("Invalid service endpoint", zap.Error(err))
						continue
					}

					currentRecords[record] = endpoint

					if _, seen := prevRecords[record]; !seen {
						// Send an "added" event for new records
						serviceEventCh <- types.ServiceEvent{
							Type:    "ADDED",
							Service: endpoint,
						}
					}
				}

				// Check for removed records
				for record, endpoint := range prevRecords {
					if _, stillPresent := currentRecords[record]; !stillPresent {
						// Send a "deleted" event for removed records
						serviceEventCh <- types.ServiceEvent{
							Type:    "DELETED",
							Service: endpoint,
						}
					}
				}

				// Update previously seen records
				prevRecords = currentRecords
			}
		}
	}()

	return serviceEventCh, nil
}

func parseTXTRecord(record string) types.ServiceEndpoint {
	return types.ServiceEndpoint{
		Name:    "example-service",
		Address: "10.0.0.1",
		Ports:   []types.ServicePort{{Name: "http", Port: 8080, Protocol: "TCP"}},
		Tags:    []string{"dns", "example"},
		Version: "1.0.0",
	}
}
