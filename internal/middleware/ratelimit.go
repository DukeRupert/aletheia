package middleware

import (
	"log/slog"
	"sync"

	"github.com/labstack/echo/v4"
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
	limiters sync.Map // IP address -> *rate.Limiter
	logger   *slog.Logger
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
func NewRateLimiter(logger *slog.Logger) *RateLimiter {
	// TODO: Create RateLimiter instance
	// TODO: Start cleanup goroutine to remove old limiters
	// TODO: Return limiter
	return &RateLimiter{
		logger: logger,
	}
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
			// TODO: Get client IP from c.RealIP()
			// TODO: Get or create rate limiter for this IP
			// TODO: Check if request is allowed (limiter.Allow())
			// TODO: If not allowed, return 429 with retry-after header
			// TODO: Add rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)
			// TODO: Call next(c)
			return nil
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
func (rl *RateLimiter) GetLimiter(ip string) interface{} {
	// TODO: Check if limiter exists in map (sync.Map.Load)
	// TODO: If exists, return it
	// TODO: If not, create new rate.Limiter with rate.Limit(100) and burst 200
	// TODO: Store in map (sync.Map.LoadOrStore for race-free creation)
	// TODO: Return limiter
	return nil
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
func (rl *RateLimiter) CleanupOldLimiters() {
	// TODO: Create ticker for periodic cleanup (1 hour interval)
	// TODO: Range over sync.Map
	// TODO: Check last access time for each limiter
	// TODO: Delete entries older than threshold
	// TODO: Log cleanup stats (number of limiters removed)
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
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Similar to Middleware() but with stricter limits
			// TODO: Use rate of 5 req/min (rate.Limit(5.0/60.0))
			// TODO: Burst of 10
			// TODO: Consider tracking by IP + endpoint for finer control
			return nil
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
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Extract user ID from context (set by session middleware)
			// TODO: If not authenticated, skip (rely on IP-based limiting)
			// TODO: Get or create limiter for user ID
			// TODO: Check if allowed (100 req/min per user)
			// TODO: Return 429 if exceeded
			// TODO: Add user-specific rate limit headers
			// TODO: Call next(c)
			return nil
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
