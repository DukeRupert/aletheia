package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// HealthHandler provides comprehensive health check endpoints for monitoring.
type HealthHandler struct {
	db        *pgxpool.Pool
	storage   storage.FileStorage
	queue     queue.Queue
	logger    *slog.Logger
	startTime time.Time
}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler(db *pgxpool.Pool, storage storage.FileStorage, queue queue.Queue, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		db:        db,
		storage:   storage,
		queue:     queue,
		logger:    logger,
		startTime: time.Now(),
	}
}

// HealthCheck provides a basic health check endpoint.
//
// Purpose:
// - Simple endpoint that always returns 200 OK if the application is running
// - Used by load balancers for basic uptime checks
// - Does not check dependencies (fast response)
//
// Response:
//   {"status": "ok"}
//
// Route: GET /health
func (h *HealthHandler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// ReadinessCheck checks if the application is ready to serve traffic.
//
// Purpose:
// - Verify all critical dependencies are available and healthy
// - Check database connectivity (ping)
// - Check storage availability (optional health check if implemented)
// - Check queue connectivity
// - Return 503 Service Unavailable if any dependency is unhealthy
// - Used by Kubernetes/orchestrators to determine if pod should receive traffic
//
// Response (healthy):
//   {
//     "status": "healthy",
//     "checks": {
//       "database": "ok",
//       "storage": "ok",
//       "queue": "ok"
//     }
//   }
//
// Response (unhealthy):
//   {
//     "status": "unhealthy",
//     "checks": {
//       "database": "ok",
//       "storage": "failed: connection timeout",
//       "queue": "ok"
//     }
//   }
//
// Route: GET /health/ready
func (h *HealthHandler) ReadinessCheck(c echo.Context) error {
	ctx := c.Request().Context()
	checks := make(map[string]string)
	healthy := true

	// Check database connectivity
	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "failed: " + err.Error()
		healthy = false
		h.logger.Error("database health check failed", slog.String("err", err.Error()))
	} else {
		checks["database"] = "ok"
	}

	// Check storage - just verify it exists (no health check in interface)
	if h.storage != nil {
		checks["storage"] = "ok"
	} else {
		checks["storage"] = "failed: not initialized"
		healthy = false
	}

	// Check queue - verify it exists
	if h.queue != nil {
		checks["queue"] = "ok"
	} else {
		checks["queue"] = "failed: not initialized"
		healthy = false
	}

	// Determine response status
	status := "healthy"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, map[string]interface{}{
		"status": status,
		"checks": checks,
	})
}

// LivenessCheck checks if the application is alive.
//
// Purpose:
// - Extremely simple check that the application is running
// - Does not check dependencies (unlike readiness)
// - Used by Kubernetes to determine if pod should be restarted
// - Should only fail if the application is deadlocked or in an unrecoverable state
//
// Response: 204 No Content
//
// Route: GET /health/live
func (h *HealthHandler) LivenessCheck(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// DetailedHealthCheck provides detailed system information for debugging.
//
// Purpose:
// - Provide comprehensive health information for operators
// - Include database pool statistics
// - Include queue statistics (pending jobs, workers, etc.)
// - Include application version/build info
// - Should be protected (only accessible to admins)
//
// Response:
//   {
//     "status": "healthy",
//     "version": "1.0.0",
//     "uptime_seconds": 3600,
//     "database": {
//       "status": "ok",
//       "total_connections": 25,
//       "idle_connections": 20,
//       "acquired_connections": 5
//     },
//     "queue": {
//       "status": "ok",
//       "pending_jobs": 42,
//       "active_workers": 3
//     },
//     "storage": {
//       "status": "ok",
//       "provider": "s3"
//     }
//   }
//
// Route: GET /health/detailed
func (h *HealthHandler) DetailedHealthCheck(c echo.Context) error {
	ctx := c.Request().Context()

	// Calculate uptime
	uptime := time.Since(h.startTime)

	// Collect database pool statistics
	dbStats := h.db.Stat()
	databaseInfo := map[string]interface{}{
		"status":               "ok",
		"total_connections":    dbStats.TotalConns(),
		"idle_connections":     dbStats.IdleConns(),
		"acquired_connections": dbStats.AcquiredConns(),
		"max_connections":      dbStats.MaxConns(),
	}

	// Check database connectivity
	if err := h.db.Ping(ctx); err != nil {
		databaseInfo["status"] = "failed: " + err.Error()
		h.logger.Error("database ping failed in detailed health check", slog.String("err", err.Error()))
	}

	// Collect queue statistics
	queueInfo := map[string]interface{}{
		"status": "ok",
	}

	// Try to get stats for common queue (photo_analysis)
	if h.queue != nil {
		stats, err := h.queue.GetQueueStats(ctx, "photo_analysis")
		if err != nil {
			queueInfo["status"] = "failed: " + err.Error()
			h.logger.Error("failed to get queue stats", slog.String("err", err.Error()))
		} else {
			queueInfo["pending_jobs"] = stats.PendingJobs
			queueInfo["processing_jobs"] = stats.ProcessingJobs
			queueInfo["completed_jobs"] = stats.CompletedJobs
			queueInfo["failed_jobs"] = stats.FailedJobs
		}
	}

	// Storage info
	storageInfo := map[string]interface{}{
		"status": "ok",
	}
	if h.storage == nil {
		storageInfo["status"] = "not initialized"
	}

	// Build response
	response := map[string]interface{}{
		"status":         "healthy",
		"uptime_seconds": int(uptime.Seconds()),
		"database":       databaseInfo,
		"queue":          queueInfo,
		"storage":        storageInfo,
	}

	// Overall health status
	if databaseInfo["status"] != "ok" || queueInfo["status"] != "ok" || storageInfo["status"] != "ok" {
		response["status"] = "degraded"
	}

	return c.JSON(http.StatusOK, response)
}
