package metrics

import (
	"github.com/goletan/observability/pkg"
	"github.com/prometheus/client_golang/prometheus"
)

type ServicesMetrics struct {
	obs *observability.Observability
}

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

func InitMetrics(obs *observability.Observability) *ServicesMetrics {
	metrics := &ServicesMetrics{obs: obs}
	metrics.Register()
	return metrics
}

func (em *ServicesMetrics) Register() {
	prometheus.MustRegister(ServiceExecutionDuration)
}

// ObserveExecution records the execution duration of a service operation.
func (em *ServicesMetrics) ObserveExecution(service, operation string, duration float64) {
	ServiceExecutionDuration.WithLabelValues(service, operation).Observe(duration)
}
