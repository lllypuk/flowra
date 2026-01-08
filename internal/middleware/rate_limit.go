package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Rate limit defaults.
const (
	DefaultRateLimit       = 100
	DefaultRateLimitWindow = time.Minute
	DefaultBurstSize       = 10
)

// Rate limit errors.
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrRateLimiterFailed = errors.New("rate limiter failed")
)

// RateLimitStore defines the interface for rate limit storage.
type RateLimitStore interface {
	// Increment increments the counter for the given key and returns the new count.
	// It also sets the expiration time if the key is new.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)

	// GetCount returns the current count for the given key.
	GetCount(ctx context.Context, key string) (int64, error)

	// GetTTL returns the remaining TTL for the given key.
	GetTTL(ctx context.Context, key string) (time.Duration, error)
}

// RateLimitConfig holds configuration for the rate limit middleware.
type RateLimitConfig struct {
	// Logger is the structured logger for rate limit events.
	Logger *slog.Logger

	// Store is the rate limit storage backend (Redis).
	Store RateLimitStore

	// Limit is the maximum number of requests allowed per window.
	Limit int

	// Window is the time window for rate limiting.
	Window time.Duration

	// BurstSize is the maximum number of requests that can be made in a burst.
	// This is added to the regular limit.
	BurstSize int

	// KeyFunc is a function that generates a unique key for rate limiting.
	// If nil, defaults to using user ID or IP address.
	KeyFunc func(c echo.Context) string

	// SkipPaths are paths that don't require rate limiting.
	SkipPaths []string

	// SkipSuccessfulAuth skips rate limiting for successfully authenticated requests.
	SkipSuccessfulAuth bool

	// Message is the error message returned when rate limit is exceeded.
	Message string

	// ExceedHandler is a custom handler for rate limit exceeded errors.
	// If nil, the default error response is used.
	ExceedHandler func(c echo.Context, remaining time.Duration) error
}

// DefaultRateLimitConfig returns a RateLimitConfig with sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Logger:    slog.Default(),
		Limit:     DefaultRateLimit,
		Window:    DefaultRateLimitWindow,
		BurstSize: DefaultBurstSize,
		SkipPaths: []string{"/health", "/ready"},
		Message:   "Too many requests. Please try again later.",
	}
}

// RateLimit returns a rate limiting middleware with the given configuration.
//
//nolint:gocognit // Rate limiting middleware requires complex logic for different scenarios.
func RateLimit(config RateLimitConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Limit <= 0 {
		config.Limit = DefaultRateLimit
	}
	if config.Window <= 0 {
		config.Window = DefaultRateLimitWindow
	}
	if config.Message == "" {
		config.Message = "Too many requests. Please try again later."
	}

	skipPaths := make(map[string]struct{}, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip rate limiting for configured paths
			if _, ok := skipPaths[path]; ok {
				return next(c)
			}

			// Skip if store is not configured (disable rate limiting)
			if config.Store == nil {
				return next(c)
			}

			// Generate rate limit key
			key := generateRateLimitKey(c, config.KeyFunc)

			// Increment counter
			count, err := config.Store.Increment(c.Request().Context(), key, config.Window)
			if err != nil {
				config.Logger.Error("failed to increment rate limit counter",
					slog.String("key", key),
					slog.String("error", err.Error()),
				)
				// On error, allow the request to proceed
				return next(c)
			}

			// Calculate limit with burst
			totalLimit := int64(config.Limit + config.BurstSize)

			// Set rate limit headers
			remaining := max(totalLimit-count, 0)

			c.Response().Header().Set("X-Ratelimit-Limit", strconv.FormatInt(totalLimit, 10))
			c.Response().Header().Set("X-Ratelimit-Remaining", strconv.FormatInt(remaining, 10))

			// Get TTL for reset header
			ttl, err := config.Store.GetTTL(c.Request().Context(), key)
			if err == nil && ttl > 0 {
				resetTime := time.Now().Add(ttl).Unix()
				c.Response().Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime, 10))
			}

			// Check if rate limit exceeded
			if count > totalLimit {
				config.Logger.Warn("rate limit exceeded",
					slog.String("key", key),
					slog.Int64("count", count),
					slog.Int64("limit", totalLimit),
					slog.String("path", path),
					slog.String("remote_ip", c.RealIP()),
				)

				// Use custom handler if provided
				if config.ExceedHandler != nil {
					return config.ExceedHandler(c, ttl)
				}

				return respondRateLimitError(c, config.Message, ttl)
			}

			return next(c)
		}
	}
}

// generateRateLimitKey generates a unique key for rate limiting.
func generateRateLimitKey(c echo.Context, keyFunc func(c echo.Context) string) string {
	// Use custom key function if provided
	if keyFunc != nil {
		return keyFunc(c)
	}

	// Try to use user ID first (authenticated user)
	userID := GetUserID(c)
	if !userID.IsZero() {
		return fmt.Sprintf("ratelimit:user:%s", userID.String())
	}

	// Fall back to IP address
	return fmt.Sprintf("ratelimit:ip:%s", c.RealIP())
}

// respondRateLimitError sends a rate limit exceeded error response.
func respondRateLimitError(c echo.Context, message string, retryAfter time.Duration) error {
	if retryAfter > 0 {
		c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
	}

	return c.JSON(http.StatusTooManyRequests, map[string]any{
		"success": false,
		"error": map[string]any{
			"code":        "RATE_LIMIT_EXCEEDED",
			"message":     message,
			"retry_after": int64(retryAfter.Seconds()),
		},
	})
}

// RateLimitByEndpoint returns a rate limiting middleware that limits by endpoint.
func RateLimitByEndpoint(config RateLimitConfig) echo.MiddlewareFunc {
	originalKeyFunc := config.KeyFunc

	config.KeyFunc = func(c echo.Context) string {
		var baseKey string
		if originalKeyFunc != nil {
			baseKey = originalKeyFunc(c)
		} else {
			userID := GetUserID(c)
			if !userID.IsZero() {
				baseKey = fmt.Sprintf("user:%s", userID.String())
			} else {
				baseKey = fmt.Sprintf("ip:%s", c.RealIP())
			}
		}

		return fmt.Sprintf("ratelimit:endpoint:%s:%s:%s", c.Request().Method, c.Path(), baseKey)
	}

	return RateLimit(config)
}

// RateLimitByUser returns a rate limiting middleware that limits by user only.
// Unauthenticated requests are limited by IP.
func RateLimitByUser(config RateLimitConfig) echo.MiddlewareFunc {
	config.KeyFunc = func(c echo.Context) string {
		userID := GetUserID(c)
		if !userID.IsZero() {
			return fmt.Sprintf("ratelimit:user:%s", userID.String())
		}
		return fmt.Sprintf("ratelimit:ip:%s", c.RealIP())
	}

	return RateLimit(config)
}

// RateLimitByIP returns a rate limiting middleware that limits by IP only.
func RateLimitByIP(config RateLimitConfig) echo.MiddlewareFunc {
	config.KeyFunc = func(c echo.Context) string {
		return fmt.Sprintf("ratelimit:ip:%s", c.RealIP())
	}

	return RateLimit(config)
}

// RateLimitByWorkspace returns a rate limiting middleware that limits by workspace.
func RateLimitByWorkspace(config RateLimitConfig) echo.MiddlewareFunc {
	config.KeyFunc = func(c echo.Context) string {
		workspaceID := GetWorkspaceID(c)
		if !workspaceID.IsZero() {
			return fmt.Sprintf("ratelimit:workspace:%s", workspaceID.String())
		}
		// Fall back to user-based limiting
		userID := GetUserID(c)
		if !userID.IsZero() {
			return fmt.Sprintf("ratelimit:user:%s", userID.String())
		}
		return fmt.Sprintf("ratelimit:ip:%s", c.RealIP())
	}

	return RateLimit(config)
}

// MemoryRateLimitStore is an in-memory rate limit store for testing.
type MemoryRateLimitStore struct {
	counts map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count     int64
	expiresAt time.Time
}

// NewMemoryRateLimitStore creates a new in-memory rate limit store.
func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{
		counts: make(map[string]*rateLimitEntry),
	}
}

// Increment increments the counter for the given key.
func (s *MemoryRateLimitStore) Increment(_ context.Context, key string, window time.Duration) (int64, error) {
	entry, exists := s.counts[key]

	// Check if entry exists and is still valid
	if exists && time.Now().Before(entry.expiresAt) {
		entry.count++
		return entry.count, nil
	}

	// Create new entry
	s.counts[key] = &rateLimitEntry{
		count:     1,
		expiresAt: time.Now().Add(window),
	}

	return 1, nil
}

// GetCount returns the current count for the given key.
func (s *MemoryRateLimitStore) GetCount(_ context.Context, key string) (int64, error) {
	entry, exists := s.counts[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return 0, nil
	}
	return entry.count, nil
}

// GetTTL returns the remaining TTL for the given key.
func (s *MemoryRateLimitStore) GetTTL(_ context.Context, key string) (time.Duration, error) {
	entry, exists := s.counts[key]
	if !exists {
		return 0, nil
	}

	ttl := time.Until(entry.expiresAt)
	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}

// Reset clears all rate limit entries (for testing).
func (s *MemoryRateLimitStore) Reset() {
	s.counts = make(map[string]*rateLimitEntry)
}

// RedisRateLimitStore is a Redis-based rate limit store.
type RedisRateLimitStore struct {
	client    RedisClient
	keyPrefix string
}

// RedisClient defines the interface for Redis operations needed by rate limiter.
type RedisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Get(ctx context.Context, key string) (string, error)
}

// NewRedisRateLimitStore creates a new Redis-based rate limit store.
func NewRedisRateLimitStore(client RedisClient, keyPrefix string) *RedisRateLimitStore {
	if keyPrefix == "" {
		keyPrefix = "flowra:ratelimit:"
	}
	return &RedisRateLimitStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Increment increments the counter for the given key.
func (s *RedisRateLimitStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := s.keyPrefix + key

	// Increment the counter
	count, err := s.client.Incr(ctx, fullKey)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration on first request (count == 1)
	if count == 1 {
		if expireErr := s.client.Expire(ctx, fullKey, window); expireErr != nil {
			return count, fmt.Errorf("failed to set expiration: %w", expireErr)
		}
	}

	return count, nil
}

// GetCount returns the current count for the given key.
func (s *RedisRateLimitStore) GetCount(ctx context.Context, key string) (int64, error) {
	fullKey := s.keyPrefix + key

	result, err := s.client.Get(ctx, fullKey)
	if err != nil {
		// Key doesn't exist - return 0 without error
		return 0, nil //nolint:nilerr // Redis returns error for non-existent keys, which is expected
	}

	// Handle empty string (key doesn't exist or was deleted)
	if result == "" {
		return 0, nil
	}

	count, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return count, nil
}

// GetTTL returns the remaining TTL for the given key.
func (s *RedisRateLimitStore) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := s.keyPrefix + key
	return s.client.TTL(ctx, fullKey)
}

// EndpointRateLimits defines rate limits for specific endpoints.
type EndpointRateLimits struct {
	limits map[string]int
}

// NewEndpointRateLimits creates a new endpoint rate limits configuration.
func NewEndpointRateLimits() *EndpointRateLimits {
	return &EndpointRateLimits{
		limits: make(map[string]int),
	}
}

// Set sets the rate limit for a specific endpoint pattern.
func (e *EndpointRateLimits) Set(pattern string, limit int) *EndpointRateLimits {
	e.limits[pattern] = limit
	return e
}

// Get returns the rate limit for a specific endpoint, or the default if not set.
func (e *EndpointRateLimits) Get(method, path string, defaultLimit int) int {
	key := method + ":" + path
	if limit, ok := e.limits[key]; ok {
		return limit
	}

	// Check for pattern match (simple prefix matching)
	for pattern, limit := range e.limits {
		if matchPattern(pattern, key) {
			return limit
		}
	}

	return defaultLimit
}

// matchPattern performs simple pattern matching with wildcard support.
func matchPattern(pattern, key string) bool {
	if pattern == key {
		return true
	}

	// Simple suffix wildcard matching
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return false
}

// WorkspaceRateLimiter provides workspace-specific rate limiting.
type WorkspaceRateLimiter struct {
	store     RateLimitStore
	limits    map[uuid.UUID]int
	defLimit  int
	defWindow time.Duration
}

// NewWorkspaceRateLimiter creates a new workspace-aware rate limiter.
func NewWorkspaceRateLimiter(
	store RateLimitStore,
	defaultLimit int,
	defaultWindow time.Duration,
) *WorkspaceRateLimiter {
	return &WorkspaceRateLimiter{
		store:     store,
		limits:    make(map[uuid.UUID]int),
		defLimit:  defaultLimit,
		defWindow: defaultWindow,
	}
}

// SetWorkspaceLimit sets a custom limit for a specific workspace.
func (w *WorkspaceRateLimiter) SetWorkspaceLimit(workspaceID uuid.UUID, limit int) {
	w.limits[workspaceID] = limit
}

// GetLimit returns the rate limit for a workspace.
func (w *WorkspaceRateLimiter) GetLimit(workspaceID uuid.UUID) int {
	if limit, ok := w.limits[workspaceID]; ok {
		return limit
	}
	return w.defLimit
}

// Middleware returns the rate limiting middleware for workspaces.
func (w *WorkspaceRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			workspaceID := GetWorkspaceID(c)
			if workspaceID.IsZero() {
				return next(c)
			}

			limit := w.GetLimit(workspaceID)
			key := fmt.Sprintf("workspace:%s", workspaceID.String())

			count, err := w.store.Increment(c.Request().Context(), key, w.defWindow)
			if err != nil {
				// On error, allow the request
				return next(c)
			}

			// Set headers
			remaining := max(int64(limit)-count, 0)

			c.Response().Header().Set("X-Ratelimit-Limit", strconv.Itoa(limit))
			c.Response().Header().Set("X-Ratelimit-Remaining", strconv.FormatInt(remaining, 10))

			if count > int64(limit) {
				ttl, _ := w.store.GetTTL(c.Request().Context(), key)
				return respondRateLimitError(c, "Workspace rate limit exceeded", ttl)
			}

			return next(c)
		}
	}
}
