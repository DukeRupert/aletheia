package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
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
//
// SECURITY IMPORTANT: IP-based rate limiting configuration
//
// This rate limiter uses c.RealIP() to extract the client's IP address. To prevent
// IP spoofing attacks (where attackers bypass rate limits by forging X-Forwarded-For
// headers), you MUST properly configure Echo's IPExtractor in production:
//
// Production configuration (main.go):
//   e.IPExtractor = echo.ExtractIPFromXFFHeader(
//       echo.TrustLoopback(true),   // Trust localhost
//       echo.TrustLinkLocal(false), // Don't trust link-local
//       echo.TrustPrivateNet(true), // Trust private networks (adjust based on your setup)
//   )
//
// Alternative for environments behind a known proxy:
//   e.IPExtractor = echo.ExtractIPDirect() // Only if not behind proxy
//
// If misconfigured, attackers can:
// - Bypass rate limits by spoofing X-Forwarded-For headers
// - Perform unlimited requests by rotating fake IPs
// - Execute brute force attacks without being throttled
//
// See: https://echo.labstack.com/docs/ip-address
type RateLimiter struct {
	limiters  sync.Map // IP address -> *limiterEntry
	logger    *slog.Logger
	config    RateLimitConfig
	ctx       context.Context
	cancel    context.CancelFunc
}

// limiterEntry wraps a rate limiter with metadata for cleanup.
// lastAccess is stored as Unix timestamp (int64) for thread-safe atomic access.
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess atomic.Int64 // Unix timestamp in seconds
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
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", rl.config.GlobalRate))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			// Add rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", rl.config.GlobalRate))

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
// - Thread-safe lastAccess tracking using atomic operations
//
// Parameters:
//   ip - Client IP address
//
// Returns rate limiter instance for this IP.
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	// Check if limiter exists in map
	if entry, exists := rl.limiters.Load(ip); exists {
		limEntry := entry.(*limiterEntry)
		// Update last access time atomically (thread-safe)
		limEntry.lastAccess.Store(time.Now().Unix())
		return limEntry.limiter
	}

	// Create new rate limiter with configured rate and burst
	limiter := rate.NewLimiter(rate.Limit(rl.config.GlobalRate), rl.config.GlobalBurst)

	// Store in map (use LoadOrStore for race-free creation)
	entry := &limiterEntry{
		limiter: limiter,
	}
	entry.lastAccess.Store(time.Now().Unix())
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
			currentTime := time.Now().Unix()

			// Range over all limiters in the map
			rl.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*limiterEntry)

				// Check if limiter hasn't been accessed recently (atomically read lastAccess)
				lastAccess := entry.lastAccess.Load()
				if currentTime-lastAccess > int64(inactivityThreshold.Seconds()) {
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

// StrictRateLimiter provides stricter rate limiting for sensitive endpoints.
// Similar to RateLimiter but with stricter limits and cleanup support.
type StrictRateLimiter struct {
	limiters sync.Map
	logger   *slog.Logger
	config   RateLimitConfig
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStrictRateLimiter creates a new strict rate limiter for auth endpoints.
//
// Purpose:
// - Apply tighter limits to authentication endpoints (login, register, password reset)
// - Prevent brute force attacks
// - Use much lower rate: 5 requests/minute per IP
//
// Usage in main.go (apply to specific routes):
//   strictLimiter := middleware.NewStrictRateLimiter(logger)
//   defer strictLimiter.Shutdown()
//   auth := e.Group("/auth")
//   auth.Use(strictLimiter.Middleware())
//   auth.POST("/login", authHandler.Login)
func NewStrictRateLimiter(logger *slog.Logger) *StrictRateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &StrictRateLimiter{
		logger: logger,
		config: DefaultRateLimitConfig(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine
	go rl.cleanupOldLimiters()

	return rl
}

// Middleware returns the strict rate limiting middleware.
func (rl *StrictRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			// Get or create limiter for this IP
			limiter := rl.getLimiter(ip)

			// Check if request is allowed
			if !limiter.Allow() {
				rl.logger.Warn("strict rate limit exceeded",
					slog.String("ip", ip),
					slog.String("path", c.Path()),
					slog.String("method", c.Request().Method))

				c.Response().Header().Set("Retry-After", "60")
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", rl.config.StrictRate*60))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "too many authentication attempts, please try again later")
			}

			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", rl.config.StrictRate*60))
			return next(c)
		}
	}
}

// getLimiter gets or creates a rate limiter for the given key.
func (rl *StrictRateLimiter) getLimiter(key string) *rate.Limiter {
	if entry, exists := rl.limiters.Load(key); exists {
		limEntry := entry.(*limiterEntry)
		limEntry.lastAccess.Store(time.Now().Unix())
		return limEntry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.config.StrictRate), rl.config.StrictBurst)
	entry := &limiterEntry{
		limiter: limiter,
	}
	entry.lastAccess.Store(time.Now().Unix())
	actual, _ := rl.limiters.LoadOrStore(key, entry)
	return actual.(*limiterEntry).limiter
}

// cleanupOldLimiters removes inactive limiters periodically.
func (rl *StrictRateLimiter) cleanupOldLimiters() {
	interval, err := time.ParseDuration(rl.config.CleanupInterval)
	if err != nil {
		interval = time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	inactivityThreshold := time.Hour

	for {
		select {
		case <-ticker.C:
			var removed int
			currentTime := time.Now().Unix()

			rl.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*limiterEntry)
				lastAccess := entry.lastAccess.Load()
				if currentTime-lastAccess > int64(inactivityThreshold.Seconds()) {
					rl.limiters.Delete(key)
					removed++
				}
				return true
			})

			if removed > 0 {
				rl.logger.Info("cleaned up old strict rate limiters",
					slog.Int("removed", removed))
			}
		case <-rl.ctx.Done():
			rl.logger.Debug("strict rate limiter cleanup goroutine stopping")
			return
		}
	}
}

// Shutdown stops the cleanup goroutine.
func (rl *StrictRateLimiter) Shutdown() {
	if rl.cancel != nil {
		rl.cancel()
	}
}

// PerUserRateLimiter provides rate limiting per authenticated user.
// Similar to RateLimiter but tracks by user ID instead of IP.
type PerUserRateLimiter struct {
	limiters sync.Map
	logger   *slog.Logger
	config   RateLimitConfig
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPerUserRateLimiter creates a new per-user rate limiter.
//
// Purpose:
// - Limit requests per user account (in addition to IP-based limiting)
// - Prevent single user from overwhelming system
// - Useful for API endpoints after authentication
//
// Usage:
//   userLimiter := middleware.NewPerUserRateLimiter(logger)
//   defer userLimiter.Shutdown()
//   api := e.Group("/api")
//   api.Use(session.SessionMiddleware(pool))
//   api.Use(userLimiter.Middleware())
func NewPerUserRateLimiter(logger *slog.Logger) *PerUserRateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &PerUserRateLimiter{
		logger: logger,
		config: DefaultRateLimitConfig(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine
	go rl.cleanupOldLimiters()

	return rl
}

// Middleware returns the per-user rate limiting middleware.
func (rl *PerUserRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user ID from context (set by session middleware)
			userID, ok := session.GetUserID(c)
			if !ok {
				// Not authenticated, skip user-based limiting
				return next(c)
			}

			userKey := userID.String()
			limiter := rl.getLimiter(userKey)

			// Check if request is allowed
			if !limiter.Allow() {
				rl.logger.Warn("per-user rate limit exceeded",
					slog.String("user_id", userKey),
					slog.String("path", c.Path()),
					slog.String("method", c.Request().Method))

				c.Response().Header().Set("Retry-After", "60")
				c.Response().Header().Set("X-RateLimit-Limit-User", fmt.Sprintf("%.0f", rl.config.UserRate*60))
				c.Response().Header().Set("X-RateLimit-Remaining-User", "0")

				return echo.NewHTTPError(http.StatusTooManyRequests, "user rate limit exceeded")
			}

			c.Response().Header().Set("X-RateLimit-Limit-User", fmt.Sprintf("%.0f", rl.config.UserRate*60))
			return next(c)
		}
	}
}

// getLimiter gets or creates a rate limiter for the given user.
func (rl *PerUserRateLimiter) getLimiter(userKey string) *rate.Limiter {
	if entry, exists := rl.limiters.Load(userKey); exists {
		limEntry := entry.(*limiterEntry)
		limEntry.lastAccess.Store(time.Now().Unix())
		return limEntry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.config.UserRate), rl.config.UserBurst)
	entry := &limiterEntry{
		limiter: limiter,
	}
	entry.lastAccess.Store(time.Now().Unix())
	actual, _ := rl.limiters.LoadOrStore(userKey, entry)
	return actual.(*limiterEntry).limiter
}

// cleanupOldLimiters removes inactive limiters periodically.
func (rl *PerUserRateLimiter) cleanupOldLimiters() {
	interval, err := time.ParseDuration(rl.config.CleanupInterval)
	if err != nil {
		interval = time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	inactivityThreshold := time.Hour

	for {
		select {
		case <-ticker.C:
			var removed int
			currentTime := time.Now().Unix()

			rl.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*limiterEntry)
				lastAccess := entry.lastAccess.Load()
				if currentTime-lastAccess > int64(inactivityThreshold.Seconds()) {
					rl.limiters.Delete(key)
					removed++
				}
				return true
			})

			if removed > 0 {
				rl.logger.Info("cleaned up old per-user rate limiters",
					slog.Int("removed", removed))
			}
		case <-rl.ctx.Done():
			rl.logger.Debug("per-user rate limiter cleanup goroutine stopping")
			return
		}
	}
}

// Shutdown stops the cleanup goroutine.
func (rl *PerUserRateLimiter) Shutdown() {
	if rl.cancel != nil {
		rl.cancel()
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
