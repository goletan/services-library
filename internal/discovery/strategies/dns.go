package strategies

import (
	"context"
	"github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
	"net"
	"strings"
	"time"
)

type DNSDiscovery struct {
	logger    *logger.ZapLogger
	namespace string
}

func NewDNSStrategy(logger *logger.ZapLogger, namespace string) *DNSDiscovery {
	return &DNSDiscovery{
		logger:    logger,
		namespace: namespace,
	}
}

func (d *DNSDiscovery) Name() string {
	return "dns"
}

func (d *DNSDiscovery) Discover(ctx context.Context, filter *types.Filter) ([]types.ServiceEndpoint, error) {
	records, err := net.LookupTXT(d.namespace)
	if err != nil {
		d.logger.Warn("DNS lookup failed", zap.Error(err))
		return nil, err
	}

	var endpoints []types.ServiceEndpoint
	for _, record := range records {
		endpoint := parseTXTRecord(record)

		// Apply filters
		if !MatchTags(endpoint.Tags, filter.Tags) {
			continue
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func (d *DNSDiscovery) Watch(ctx context.Context, filter *types.Filter) (<-chan types.ServiceEvent, error) {
	serviceEventCh := make(chan types.ServiceEvent)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	prevRecords := make(map[string]types.ServiceEndpoint)

	go func() {
		defer close(serviceEventCh)
		for {
			select {
			case <-ctx.Done():
				d.logger.Info("Stopping DNS discovery watcher...")
				return
			case <-ticker.C:
				records, err := net.LookupTXT(d.namespace)
				if err != nil {
					d.logger.Warn("DNS lookup failed", zap.Error(err))
					continue
				}

				currentRecords := make(map[string]types.ServiceEndpoint)
				for _, record := range records {
					endpoint := parseTXTRecord(record)

					if !MatchTags(endpoint.Tags, filter.Tags) {
						continue
					}

					currentRecords[record] = endpoint
					if _, seen := prevRecords[record]; !seen {
						serviceEventCh <- types.ServiceEvent{Type: "ADDED", Service: endpoint}
					}
				}

				for record, endpoint := range prevRecords {
					if _, stillPresent := currentRecords[record]; !stillPresent {
						serviceEventCh <- types.ServiceEvent{Type: "DELETED", Service: endpoint}
					}
				}

				prevRecords = currentRecords
			}
		}
	}()

	return serviceEventCh, nil
}

func parseTXTRecord(record string) types.ServiceEndpoint {
	tags := make(map[string]string)
	parts := strings.Split(record, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			tags[kv[0]] = kv[1]
		}
	}

	return types.ServiceEndpoint{
		Name:    "http-service",
		Address: "10.0.0.1",
		Ports:   []types.ServicePort{{Name: "http", Port: 8080, Protocol: "TCP"}},
		Tags:    tags,
		Version: "1.0.0",
	}
}
