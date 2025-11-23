package middleware

import (
	"strconv"
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

	// Queue metrics
	queueJobsTotal *prometheus.CounterVec
	queueJobDuration *prometheus.HistogramVec
	queueDepth *prometheus.GaugeVec
	queueWorkersActive *prometheus.GaugeVec
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
			status := strconv.Itoa(c.Response().Status)

			// Record metrics
			httpRequestsTotal.WithLabelValues(method, path, status).Inc()
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
// - Should be called once during application startup before MetricsMiddleware
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

	// Queue job counter
	queueJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_jobs_total",
			Help: "Total number of queue jobs processed",
		},
		[]string{"job_type", "status"},
	)

	// Queue job duration histogram
	queueJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "queue_job_duration_seconds",
			Help:    "Queue job processing duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0},
		},
		[]string{"job_type"},
	)

	// Queue depth gauge
	queueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Number of jobs pending in queue",
		},
		[]string{"queue_name"},
	)

	// Active workers gauge
	queueWorkersActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_workers_active",
			Help: "Number of workers currently processing jobs",
		},
		[]string{"queue_name"},
	)
}

// RecordQueueJobMetrics records metrics for background queue jobs.
//
// Purpose:
// - Track job processing time
// - Count jobs by type and status (success/failure)
// - Monitor queue depth
// - Separate from HTTP metrics since jobs are async
//
// Metrics:
// - queue_jobs_total (counter) - labels: job_type, status
// - queue_job_duration_seconds (histogram) - labels: job_type
// - queue_depth (gauge) - labels: queue_name
// - queue_workers_active (gauge) - labels: queue_name
//
// Usage in queue/postgres.go:
//   defer middleware.RecordQueueJobMetrics(jobType, startTime, err)
func RecordQueueJobMetrics(jobType string, duration float64, err error) {
	// Determine status based on error
	status := "success"
	if err != nil {
		status = "failed"
	}

	// Increment job counter with status label
	queueJobsTotal.WithLabelValues(jobType, status).Inc()

	// Record job duration in histogram
	queueJobDuration.WithLabelValues(jobType).Observe(duration)
}

// UpdateQueueDepth updates the queue depth gauge.
//
// Purpose:
// - Provide visibility into queue backlog
// - Alert when queue depth exceeds thresholds
// - Should be called periodically or on enqueue/dequeue
//
// Usage in queue/postgres.go:
//   middleware.UpdateQueueDepth(queueName, depth)
func UpdateQueueDepth(queueName string, depth int64) {
	// Set gauge value for queue_depth with queue_name label
	queueDepth.WithLabelValues(queueName).Set(float64(depth))
}

// UpdateActiveWorkers updates the active workers gauge.
//
// Purpose:
// - Track how many workers are currently processing jobs
// - Detect worker pool issues (all workers stuck, no workers running)
//
// Usage in queue/worker.go:
//   middleware.UpdateActiveWorkers(queueName, activeCount)
func UpdateActiveWorkers(queueName string, count int) {
	// Set gauge value for queue_workers_active with queue_name label
	queueWorkersActive.WithLabelValues(queueName).Set(float64(count))
}
