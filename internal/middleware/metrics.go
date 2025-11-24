package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	httpRequestsTotal *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsInFlight prometheus.Gauge
	httpRequestSizeBytes *prometheus.HistogramVec
	httpResponseSizeBytes *prometheus.HistogramVec

	// metricsInitOnce ensures metrics are initialized exactly once
	metricsInitOnce sync.Once
)

// MetricsMiddleware collects application metrics for monitoring and observability.
//
// Purpose:
// - Track HTTP request count by method, path, and status code
// - Measure request duration/latency
// - Track error rates
// - Expose metrics in Prometheus format at /metrics endpoint
//
// Metrics to collect:
// - http_requests_total (counter) - labels: method, path, status
// - http_request_duration_seconds (histogram) - labels: method, path
// - http_requests_in_flight (gauge) - currently processing requests
// - http_request_size_bytes (histogram) - request body size
// - http_response_size_bytes (histogram) - response body size
//
// Usage in main.go:
//   e.Use(middleware.MetricsMiddleware())
//   e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip metrics endpoint itself to avoid recursion
			if c.Path() == "/metrics" {
				return next(c)
			}

			// Record start time
			start := time.Now()

			// Increment in-flight requests gauge
			httpRequestsInFlight.Inc()

			// Defer: decrement in-flight requests gauge
			defer httpRequestsInFlight.Dec()

			// Get request size
			requestSize := float64(c.Request().ContentLength)
			if requestSize < 0 {
				requestSize = 0
			}

			// Call next handler
			err := next(c)

			// Record request duration
			duration := time.Since(start).Seconds()

			// Get method and path
			method := c.Request().Method
			path := c.Path()

			// Get status code, handling edge cases
			status := c.Response().Status

			// Handle errors that might not have set a status code
			if status == 0 && err != nil {
				// Default to 500 for errors without explicit status
				status = http.StatusInternalServerError
				// Check if error is an Echo HTTPError with specific code
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				}
			}

			// Check if request was cancelled
			if c.Request().Context().Err() != nil {
				// Request was cancelled - track separately for accurate monitoring
				// Use 499 (Client Closed Request) as nginx convention
				if status == 0 {
					status = 499
				}
			}

			statusStr := strconv.Itoa(status)

			// Record metrics
			httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
			httpRequestDuration.WithLabelValues(method, path).Observe(duration)
			httpRequestSizeBytes.WithLabelValues(method, path).Observe(requestSize)
			httpResponseSizeBytes.WithLabelValues(method, path).Observe(float64(c.Response().Size))

			return err
		}
	}
}

// InitMetrics initializes Prometheus metrics collectors.
//
// Purpose:
// - Create and register all Prometheus metric collectors
// - Define metric names, help text, and labels
// - Safe to call multiple times (only initializes once using sync.Once)
//
// Thread Safety:
// - Uses sync.Once to prevent Prometheus registration panics
// - Can be safely called from tests or multiple initialization paths
//
// Metrics to initialize:
// - Counter: http_requests_total
// - Histogram: http_request_duration_seconds (with buckets)
// - Gauge: http_requests_in_flight
// - Histogram: http_request_size_bytes
// - Histogram: http_response_size_bytes
//
// Usage in main.go:
//   middleware.InitMetrics()
func InitMetrics() {
	// Ensure metrics are initialized exactly once, even if called multiple times
	metricsInitOnce.Do(func() {
		// HTTP request counter
		httpRequestsTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		)

		// HTTP request duration histogram
		httpRequestDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "path"},
		)

		// HTTP requests currently in flight
		httpRequestsInFlight = promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		)

		// HTTP request size histogram
		httpRequestSizeBytes = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to 10MB
			},
			[]string{"method", "path"},
		)

		// HTTP response size histogram
		httpResponseSizeBytes = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to 10MB
			},
			[]string{"method", "path"},
		)
	})
}

// NOTE: Queue metrics have been removed as they were not being used.
// If you need queue metrics in the future, you can add them to the InitMetrics()
// function above and create corresponding recording functions to call from
// queue/worker.go and queue/postgres.go code.
