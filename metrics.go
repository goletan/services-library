// /services/metrics.go
package services

import (
	"github.com/goletan/observability/metrics"
	"github.com/goletan/observability/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type ServicesMetrics struct{}

// Services Metrics: Track services execution durations.
var (
	ServiceExecutionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "goletan",
			Subsystem: "services",
			Name:      "execution_duration_seconds",
			Help:      "Tracks the duration of services execution.",
		},
		[]string{"service", "operation"},
	)
)

// Security Tool: Scrub sensitive data
var (
	scrubber = utils.NewScrubber()
)

func InitMetrics() {
	metrics.NewManager().Register(&ServicesMetrics{})
}

func (em *ServicesMetrics) Register() error {
	if err := prometheus.Register(ServiceExecutionDuration); err != nil {
		return err
	}

	return nil
}

// ObserveServiceExecution records the execution duration of a service operation.
func ObserveServiceExecution(service, operation string, duration float64) {
	scrubbedService := scrubber.Scrub(service)
	scrubbedOperation := scrubber.Scrub(operation)
	ServiceExecutionDuration.WithLabelValues(scrubbedService, scrubbedOperation).Observe(duration)
}
