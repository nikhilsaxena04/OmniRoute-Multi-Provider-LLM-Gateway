package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by endpoint, provider and status.",
		},
		[]string{"endpoint", "provider", "status"},
	)

	LatencyHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Histogram of request latencies.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 15},
		},
		[]string{"endpoint", "provider"},
	)

	CircuitState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_circuit_state",
			Help: "Current state of the circuit breaker (0=Closed, 1=Open, 2=HalfOpen).",
		},
		[]string{"provider"},
	)
)
