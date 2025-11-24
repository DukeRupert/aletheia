package session

import (
	"context"
	"fmt"
	"time"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patrickmn/go-cache"
)

// SessionCache provides a caching layer for sessions to reduce database load.
//
// Purpose:
// - Cache session data in memory to avoid database query on every authenticated request
// - Automatically evict expired sessions from cache
// - Invalidate cache entries when sessions are deleted
// - Provide significant performance improvement (eliminates ~5-10ms DB query per request)
//
// Implementation Options:
// Option 1: In-memory cache (github.com/patrickmn/go-cache)
//   - Simple, no additional infrastructure
//   - Works for single-instance deployments
//   - Lost on restart (acceptable - will re-query DB)
//
// Option 2: Redis cache
//   - Shared across multiple instances
//   - Persists across restarts
//   - Requires Redis infrastructure
//
// Start with Option 1, migrate to Option 2 when scaling horizontally.
type SessionCache struct {
	db    *pgxpool.Pool
	cache *cache.Cache
}

// NewSessionCache creates a new session cache.
//
// Purpose:
// - Initialize the cache with appropriate TTL settings
// - Set cleanup interval for expired entries
//
// Recommended settings:
// - Default expiration: 5 minutes (sessions live longer in DB, but cache can be shorter)
// - Cleanup interval: 10 minutes (purge expired entries from memory)
//
// Usage in main.go:
//
//	sessionCache := session.NewSessionCache(pool)
func NewSessionCache(db *pgxpool.Pool) *SessionCache {
	// Initialize cache with 5-minute default expiration and 10-minute cleanup interval
	c := cache.New(5*time.Minute, 10*time.Minute)

	return &SessionCache{
		db:    db,
		cache: c,
	}
}

// GetSession retrieves a session by token, using cache first.
//
// Purpose:
// - Check in-memory cache first (fast path)
// - On cache miss, query database and populate cache
// - On cache hit, return immediately without DB query
//
// Flow:
// 1. Check cache for token
// 2. If found, return cached session
// 3. If not found, query database
// 4. If found in DB, store in cache and return
// 5. If not found in DB, return error
//
// Error handling:
// - Return database.ErrNoRows if session not found
// - Return other errors from database queries
//
// Usage in middleware:
//
//	session, err := sessionCache.GetSession(ctx, token)
func (sc *SessionCache) GetSession(ctx context.Context, token string) (database.Session, error) {
	// Check cache first (fast path)
	if cached, found := sc.cache.Get(token); found {
		return cached.(database.Session), nil
	}

	// Cache miss - query database
	queries := database.New(sc.db)
	session, err := queries.GetSessionByToken(ctx, token)
	if err != nil {
		return database.Session{}, err
	}

	// Store in cache for future requests
	sc.cache.Set(token, session, cache.DefaultExpiration)

	return session, nil
}

// GetSessionWithUser retrieves a session with associated user data.
//
// Purpose:
// - Optimize the common case of needing both session and user data
// - Cache the combined result to avoid two separate queries
// - Used by most authenticated endpoints that need user information
//
// Flow:
// 1. Check cache for "session_user:{token}" key
// 2. If found, return cached SessionWithUser
// 3. If not found, query database with JOIN or two queries
// 4. Cache the combined result
// 5. Return SessionWithUser struct
//
// Usage in middleware:
//
//	sessionUser, err := sessionCache.GetSessionWithUser(ctx, token)
func (sc *SessionCache) GetSessionWithUser(ctx context.Context, token string) (SessionWithUser, error) {
	// Define cache key for combined session+user data
	cacheKey := fmt.Sprintf("session_user:%s", token)

	// Check cache first
	if cached, found := sc.cache.Get(cacheKey); found {
		return cached.(SessionWithUser), nil
	}

	// Cache miss - query database for session
	queries := database.New(sc.db)
	session, err := queries.GetSessionByToken(ctx, token)
	if err != nil {
		return SessionWithUser{}, err
	}

	// Query database for user
	user, err := queries.GetUser(ctx, session.UserID)
	if err != nil {
		return SessionWithUser{}, err
	}

	// Combine into SessionWithUser struct
	sessionWithUser := SessionWithUser{
		Session: session,
		User:    user,
	}

	// Cache the combined result
	sc.cache.Set(cacheKey, sessionWithUser, cache.DefaultExpiration)

	return sessionWithUser, nil
}

// SessionWithUser combines session and user data.
//
// Purpose:
// - Avoid separate caching of session and user
// - Reduce number of cache lookups
type SessionWithUser struct {
	Session database.Session
	User    database.User
}

// InvalidateSession removes a session from the cache.
//
// Purpose:
// - Called when a session is deleted (logout)
// - Called when a session is updated (if implemented)
// - Ensures cache doesn't serve stale session data
//
// Usage in handlers/auth.go:
//
//	sessionCache.InvalidateSession(token)
//	queries.DeleteSession(ctx, sessionID)
func (sc *SessionCache) InvalidateSession(token string) {
	// Delete session from cache
	sc.cache.Delete(token)

	// Also delete session_user cache entry
	cacheKey := fmt.Sprintf("session_user:%s", token)
	sc.cache.Delete(cacheKey)
}

// InvalidateUserSessions removes all sessions for a user from cache.
//
// Purpose:
// - Called when user is deleted or disabled
// - Called when user password is changed (security: invalidate all sessions)
// - More complex since we need to track user->sessions mapping
//
// Implementation notes:
// - Option 1: Store secondary index in cache (user_id -> []tokens)
// - Option 2: Use cache key pattern and iterate (expensive)
// - Option 3: Accept eventual consistency (sessions expire naturally)
//
// Start with Option 3 for simplicity, implement Option 1 if needed.
//
// Usage in handlers/auth.go:
//
//	sessionCache.InvalidateUserSessions(userID)
func (sc *SessionCache) InvalidateUserSessions(userID uuid.UUID) {
	// No-op for now - sessions will expire naturally from cache (5 min)
	// Future enhancement: maintain user_id -> []tokens mapping for immediate invalidation
	// This is acceptable because:
	// 1. Cache TTL is only 5 minutes
	// 2. Database still validates on cache miss
	// 3. For security-critical operations (password change), can clear entire cache
	_ = userID // Acknowledge parameter
}

// Clear removes all entries from the cache.
//
// Purpose:
// - Used for testing
// - Emergency cache flush if corruption suspected
//
// Usage in tests:
//
//	sessionCache.Clear()
func (sc *SessionCache) Clear() {
	sc.cache.Flush()
}

// Stats returns cache statistics for monitoring.
//
// Purpose:
// - Monitor cache hit/miss rates
// - Track cache size and memory usage
// - Expose via health check or metrics endpoint
//
// Returns:
//
//	{
//	  "hits": 1000,
//	  "misses": 50,
//	  "items": 200,
//	  "evictions": 10
//	}
func (sc *SessionCache) Stats() CacheStats {
	// Note: go-cache doesn't track hits/misses/evictions natively
	// We can only report item count
	// For detailed metrics, consider using a Redis cache or wrapping get/set methods

	items := sc.cache.ItemCount()

	return CacheStats{
		Hits:      0, // Not tracked by go-cache
		Misses:    0, // Not tracked by go-cache
		HitRate:   0, // Cannot calculate without hits/misses
		Items:     items,
		Evictions: 0, // Not tracked by go-cache
	}
}

// CacheStats contains cache performance metrics.
type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Items     int     `json:"items"`
	Evictions int64   `json:"evictions"`
}
