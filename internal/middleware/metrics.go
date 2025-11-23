package middleware

import (
	"github.com/labstack/echo/v4"
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
			// TODO: Record start time
			// TODO: Increment in-flight requests gauge
			// TODO: Defer: decrement in-flight requests gauge
			// TODO: Call next(c)
			// TODO: Record request duration
			// TODO: Increment request counter with labels (method, path, status)
			// TODO: Record request/response sizes
			return nil
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
	// TODO: Create prometheus.CounterVec for http_requests_total
	// TODO: Create prometheus.HistogramVec for http_request_duration_seconds
	// TODO: Define histogram buckets: [0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0]
	// TODO: Create prometheus.Gauge for http_requests_in_flight
	// TODO: Create prometheus.HistogramVec for request/response sizes
	// TODO: Register all metrics with prometheus.MustRegister()
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
	// TODO: Increment job counter with status label (success/failed)
	// TODO: Record job duration in histogram
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
	// TODO: Set gauge value for queue_depth with queue_name label
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
	// TODO: Set gauge value for queue_workers_active with queue_name label
}
