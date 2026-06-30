// Package observability provides reusable Prometheus instrumentation.
package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP request metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by service, method, route, and status",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method", "route"},
	)

	httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Active concurrent HTTP requests",
		},
		[]string{"service"},
	)

	registry = prometheus.NewRegistry()
)

func init() {
	// Register metrics with custom registry
	registry.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestsInFlight,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
}

// PrometheusHandler returns HTTP handler for /metrics endpoint.
func PrometheusHandler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Middleware instruments HTTP handlers with Prometheus metrics.
func Middleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip /metrics to avoid self-instrumentation loop
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			httpRequestsInFlight.WithLabelValues(serviceName).Inc()
			defer httpRequestsInFlight.WithLabelValues(serviceName).Dec()

			// Capture response status
			rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)

			duration := time.Since(start).Seconds()
			route := r.URL.Path
			method := r.Method
			status := strconv.Itoa(rec.statusCode)

			httpRequestsTotal.WithLabelValues(serviceName, method, route, status).Inc()
			httpRequestDuration.WithLabelValues(serviceName, method, route).Observe(duration)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// NewCounter creates a custom Counter and registers it.
func NewCounter(name, help string, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: name, Help: help},
		labels,
	)
	registry.MustRegister(c)
	return c
}

// NewGauge creates a custom Gauge and registers it.
func NewGauge(name, help string, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: name, Help: help},
		labels,
	)
	registry.MustRegister(g)
	return g
}

// NewHistogram creates a custom Histogram and registers it.
func NewHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
	registry.MustRegister(h)
	return h
}
