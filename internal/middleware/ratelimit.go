package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dukerupert/aletheia/internal/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimiter provides HTTP-level rate limiting to prevent abuse.
//
// Purpose:
// - Protect against brute force attacks (login, password reset)
// - Prevent API abuse and DoS attacks
// - Enforce fair usage across clients
// - Separate from queue-level rate limiting (which is per-organization)
//
// Implementation approach:
// - Use token bucket algorithm (golang.org/x/time/rate)
// - Track limits per IP address
// - Store limiters in memory (sync.Map for thread safety)
// - Periodically clean up old limiters to prevent memory leaks
//
// Rate limit strategy:
// - Global: 100 requests/second per IP, burst of 200
// - Stricter for auth endpoints: 5 requests/minute per IP
type RateLimiter struct {
	limiters  sync.Map // IP address -> *limiterEntry
	logger    *slog.Logger
	config    RateLimitConfig
	ctx       context.Context
	cancel    context.CancelFunc
}

// limiterEntry wraps a rate limiter with metadata for cleanup.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter.
//
// Purpose:
// - Initialize the rate limiter with default settings
// - Start background cleanup goroutine
//
// Usage in main.go:
//   rateLimiter := middleware.NewRateLimiter(logger)
//   e.Use(rateLimiter.Middleware())
//   defer rateLimiter.Shutdown() // Important: call Shutdown() during graceful shutdown
func NewRateLimiter(logger *slog.Logger) *RateLimiter {
	// Create context for managing goroutine lifecycle
	ctx, cancel := context.WithCancel(context.Background())

	// Create RateLimiter instance with default config
	rl := &RateLimiter{
		logger: logger,
		config: DefaultRateLimitConfig(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine to remove old limiters
	go rl.CleanupOldLimiters()

	return rl
}

// Middleware returns the rate limiting middleware.
//
// Purpose:
// - Apply rate limiting to all HTTP requests
// - Extract client IP from request
// - Get or create rate limiter for that IP
// - Check if request is allowed
// - Return 429 Too Many Requests if rate limit exceeded
// - Add rate limit headers to response (X-RateLimit-Limit, X-RateLimit-Remaining)
//
// Flow:
// 1. Extract IP from c.RealIP()
// 2. Get rate limiter for IP (create if doesn't exist)
// 3. Check limiter.Allow()
// 4. If not allowed, return 429 with Retry-After header
// 5. If allowed, add rate limit headers and continue
//
// Usage:
//   e.Use(rateLimiter.Middleware())
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get client IP from request
			ip := c.RealIP()

			// Get or create rate limiter for this IP
			limiter := rl.GetLimiter(ip)

			// Check if request is allowed
			if !limiter.Allow() {
				// Rate limit exceeded
				rl.logger.Warn("rate limit exceeded",
					slog.String("ip", ip),
					slog.String("path", c.Path()),
					slog.String("method", c.Request().Method))

				// Set Retry-After header (1 second)
				c.Response().Header().Set("Retry-After", "1")
				c.Response().Header().Set("X-RateLimit-Limit", "100")
				c.Response().Header().Set("X-RateLimit-Remaining", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			// Add rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit", "100")

			return next(c)
		}
	}
}

// GetLimiter returns the rate limiter for a given IP address.
//
// Purpose:
// - Get existing limiter from map, or create new one
// - Use sync.Map for thread-safe access
// - Configure appropriate rate (100 req/sec, burst 200)
//
// Parameters:
//   ip - Client IP address
//
// Returns rate limiter instance for this IP.
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	// Check if limiter exists in map
	if entry, exists := rl.limiters.Load(ip); exists {
		limEntry := entry.(*limiterEntry)
		// Update last access time
		limEntry.lastAccess = time.Now()
		return limEntry.limiter
	}

	// Create new rate limiter with configured rate and burst
	limiter := rate.NewLimiter(rate.Limit(rl.config.GlobalRate), rl.config.GlobalBurst)

	// Store in map (use LoadOrStore for race-free creation)
	entry := &limiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	actual, _ := rl.limiters.LoadOrStore(ip, entry)
	return actual.(*limiterEntry).limiter
}

// CleanupOldLimiters periodically removes unused limiters.
//
// Purpose:
// - Prevent memory leaks from storing limiters for many IPs
// - Run periodically (e.g., every hour)
// - Remove limiters that haven't been used recently
//
// Strategy:
// - Store last access time with each limiter
// - Remove limiters unused for > 1 hour
// - Run in background goroutine
// - Respects context cancellation for graceful shutdown
func (rl *RateLimiter) CleanupOldLimiters() {
	// Parse cleanup interval from config
	interval, err := time.ParseDuration(rl.config.CleanupInterval)
	if err != nil {
		rl.logger.Error("invalid cleanup interval, using default 1h", slog.String("error", err.Error()))
		interval = time.Hour
	}

	// Create ticker for periodic cleanup
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Threshold for removing old limiters (1 hour of inactivity)
	inactivityThreshold := time.Hour

	for {
		select {
		case <-ticker.C:
			var removed int

			// Range over all limiters in the map
			rl.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*limiterEntry)

				// Check if limiter hasn't been accessed recently
				if time.Since(entry.lastAccess) > inactivityThreshold {
					// Delete old limiter
					rl.limiters.Delete(key)
					removed++
				}

				return true // continue iteration
			})

			// Log cleanup stats
			if removed > 0 {
				rl.logger.Info("cleaned up old rate limiters",
					slog.Int("removed", removed))
			}
		case <-rl.ctx.Done():
			// Context cancelled, stop cleanup goroutine
			rl.logger.Debug("rate limiter cleanup goroutine stopping")
			return
		}
	}
}

// Shutdown gracefully stops the rate limiter's background cleanup goroutine.
//
// Purpose:
// - Stop the cleanup goroutine during application shutdown
// - Prevent goroutine leaks in tests and production
//
// Usage in main.go:
//   rateLimiter := middleware.NewRateLimiter(logger)
//   defer rateLimiter.Shutdown()
func (rl *RateLimiter) Shutdown() {
	if rl.cancel != nil {
		rl.cancel()
	}
}

// StrictRateLimitMiddleware provides stricter rate limiting for sensitive endpoints.
//
// Purpose:
// - Apply tighter limits to authentication endpoints (login, register, password reset)
// - Prevent brute force attacks
// - Use much lower rate: 5 requests/minute per IP
//
// Usage in main.go (apply to specific routes):
//   auth := e.Group("/auth")
//   auth.Use(middleware.StrictRateLimitMiddleware(logger))
//   auth.POST("/login", authHandler.Login)
//   auth.POST("/register", authHandler.Register)
func StrictRateLimitMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	// Create a separate limiter map for strict rate limiting
	limiters := &sync.Map{}
	config := DefaultRateLimitConfig()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get client IP
			ip := c.RealIP()

			// Get or create strict rate limiter for this IP
			var limiter *rate.Limiter
			if entry, exists := limiters.Load(ip); exists {
				limiter = entry.(*rate.Limiter)
			} else {
				// Create new strict limiter: 5 requests per minute
				limiter = rate.NewLimiter(rate.Limit(config.StrictRate), config.StrictBurst)
				limiters.Store(ip, limiter)
			}

			// Check if request is allowed
			if !limiter.Allow() {
				logger.Warn("strict rate limit exceeded",
					slog.String("ip", ip),
					slog.String("path", c.Path()),
					slog.String("method", c.Request().Method))

				// Set Retry-After header (60 seconds for stricter limit)
				c.Response().Header().Set("Retry-After", "60")
				c.Response().Header().Set("X-RateLimit-Limit", "5")
				c.Response().Header().Set("X-RateLimit-Remaining", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "too many authentication attempts, please try again later")
			}

			// Add rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit", "5")

			return next(c)
		}
	}
}

// PerUserRateLimitMiddleware provides rate limiting per authenticated user.
//
// Purpose:
// - Limit requests per user account (in addition to IP-based limiting)
// - Prevent single user from overwhelming system
// - Useful for API endpoints after authentication
//
// Implementation:
// - Extract user ID from session/context
// - Maintain separate limiters per user ID
// - Apply in addition to (not instead of) IP-based limiting
//
// Usage:
//   api := e.Group("/api")
//   api.Use(session.SessionMiddleware(pool))
//   api.Use(middleware.PerUserRateLimitMiddleware(logger))
func PerUserRateLimitMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	// Create a separate limiter map for per-user rate limiting
	limiters := &sync.Map{}
	config := DefaultRateLimitConfig()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user ID from context (set by session middleware)
			userID, ok := session.GetUserID(c)
			if !ok {
				// Not authenticated, skip user-based limiting (rely on IP-based limiting)
				return next(c)
			}

			// Convert user ID to string for use as map key
			userKey := userID.String()

			// Get or create rate limiter for this user
			var limiter *rate.Limiter
			if entry, exists := limiters.Load(userKey); exists {
				limiter = entry.(*rate.Limiter)
			} else {
				// Create new user limiter: 100 requests per minute
				limiter = rate.NewLimiter(rate.Limit(config.UserRate), config.UserBurst)
				limiters.Store(userKey, limiter)
			}

			// Check if request is allowed
			if !limiter.Allow() {
				logger.Warn("per-user rate limit exceeded",
					slog.String("user_id", userKey),
					slog.String("path", c.Path()),
					slog.String("method", c.Request().Method))

				// Set Retry-After header (60 seconds)
				c.Response().Header().Set("Retry-After", "60")
				c.Response().Header().Set("X-RateLimit-Limit-User", "100")
				c.Response().Header().Set("X-RateLimit-Remaining-User", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "user rate limit exceeded")
			}

			// Add user-specific rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit-User", "100")

			return next(c)
		}
	}
}

// RateLimitConfig holds configuration for rate limiting.
//
// Purpose:
// - Make rate limits configurable via environment variables
// - Different limits for different environments (stricter in production)
type RateLimitConfig struct {
	// GlobalRate is requests per second for general endpoints
	GlobalRate float64

	// GlobalBurst is burst size for general endpoints
	GlobalBurst int

	// StrictRate is requests per minute for auth endpoints
	StrictRate float64

	// StrictBurst is burst size for auth endpoints
	StrictBurst int

	// UserRate is requests per minute per authenticated user
	UserRate float64

	// UserBurst is burst size per user
	UserBurst int

	// CleanupInterval is how often to clean up old limiters
	CleanupInterval string // e.g., "1h"
}

// DefaultRateLimitConfig returns default rate limit settings.
//
// Purpose:
// - Provide sensible defaults
// - Can be overridden via environment variables
//
// Defaults:
// - Global: 100 req/sec, burst 200
// - Strict (auth): 5 req/min, burst 10
// - Per-user: 100 req/min, burst 150
func DefaultRateLimitConfig() RateLimitConfig {
	// TODO: Return RateLimitConfig with defaults
	return RateLimitConfig{
		GlobalRate:      100.0,
		GlobalBurst:     200,
		StrictRate:      5.0 / 60.0, // 5 per minute
		StrictBurst:     10,
		UserRate:        100.0 / 60.0, // 100 per minute
		UserBurst:       150,
		CleanupInterval: "1h",
	}
}
