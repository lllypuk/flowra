package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := middleware.DefaultRateLimitConfig()

	assert.NotNil(t, config.Logger)
	assert.Equal(t, middleware.DefaultRateLimit, config.Limit)
	assert.Equal(t, middleware.DefaultRateLimitWindow, config.Window)
	assert.Equal(t, middleware.DefaultBurstSize, config.BurstSize)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.NotEmpty(t, config.Message)
}

func TestRateLimit_NoStore(t *testing.T) {
	e := echo.New()

	config := middleware.RateLimitConfig{
		Store: nil, // No store configured
		Limit: 10,
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Should pass through without rate limiting
	for range 20 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_SkipPaths(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     1,
		Window:    time.Minute,
		BurstSize: 0,
		SkipPaths: []string{"/health", "/ready"},
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})

	// Should skip rate limiting for /health
	for range 10 {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     3,
		Window:    time.Minute,
		BurstSize: 0,
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First 3 requests should succeed
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "RATE_LIMIT_EXCEEDED")
}

func TestRateLimit_WithBurst(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     3,
		Window:    time.Minute,
		BurstSize: 2, // Total allowed: 3 + 2 = 5
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First 5 requests should succeed (limit + burst)
	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_Headers(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     10,
		Window:    time.Minute,
		BurstSize: 5,
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check rate limit headers
	limitHeader := rec.Header().Get("X-Ratelimit-Limit")
	assert.Equal(t, "15", limitHeader) // 10 + 5 burst

	remainingHeader := rec.Header().Get("X-Ratelimit-Remaining")
	remaining, err := strconv.Atoi(remainingHeader)
	require.NoError(t, err)
	assert.Equal(t, 14, remaining) // 15 - 1

	resetHeader := rec.Header().Get("X-Ratelimit-Reset")
	assert.NotEmpty(t, resetHeader)
}

func TestRateLimit_RemainingDecrements(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     5,
		Window:    time.Minute,
		BurstSize: 0,
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	expectedRemaining := []int{4, 3, 2, 1, 0}

	for i, expected := range expectedRemaining {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		remainingHeader := rec.Header().Get("X-Ratelimit-Remaining")
		remaining, err := strconv.Atoi(remainingHeader)
		require.NoError(t, err)
		assert.Equal(t, expected, remaining, "request %d", i+1)
	}
}

func TestRateLimit_ByUser(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     2,
		Window:    time.Minute,
		BurstSize: 0,
	}

	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()

	// Middleware to set user ID
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Request().Header.Get("X-User-Id")
			if userID != "" {
				id, _ := uuid.ParseUUID(userID)
				c.Set(string(middleware.ContextKeyUserID), id)
			}
			return next(c)
		}
	})

	e.Use(middleware.RateLimitByUser(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// User 1: 2 requests should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-User-Id", user1ID.String())
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// User 1: 3rd request should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-Id", user1ID.String())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// User 2: should still have quota
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-Id", user2ID.String())
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_ByIP(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     2,
		Window:    time.Minute,
		BurstSize: 0,
	}

	e.Use(middleware.RateLimitByIP(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// 2 requests from same IP should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request from same IP should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Request from different IP should succeed
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_ByEndpoint(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     2,
		Window:    time.Minute,
		BurstSize: 0,
	}

	e.Use(middleware.RateLimitByEndpoint(config))
	e.GET("/endpoint1", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok1")
	})
	e.GET("/endpoint2", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok2")
	})

	// 2 requests to endpoint1 should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/endpoint1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request to endpoint1 should fail
	req := httptest.NewRequest(http.MethodGet, "/endpoint1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Request to endpoint2 should succeed (separate limit)
	req = httptest.NewRequest(http.MethodGet, "/endpoint2", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_ByWorkspace(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     2,
		Window:    time.Minute,
		BurstSize: 0,
	}

	workspace1ID := uuid.NewUUID()
	workspace2ID := uuid.NewUUID()

	// Middleware to set workspace ID
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			wsID := c.Request().Header.Get("X-Workspace-Id")
			if wsID != "" {
				id, _ := uuid.ParseUUID(wsID)
				c.Set(string(middleware.ContextKeyWorkspaceID), id)
			}
			return next(c)
		}
	})

	e.Use(middleware.RateLimitByWorkspace(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Workspace 1: 2 requests should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Workspace-Id", workspace1ID.String())
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Workspace 1: 3rd request should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Workspace-Id", workspace1ID.String())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Workspace 2: should still have quota
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Workspace-Id", workspace2ID.String())
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_CustomKeyFunc(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     2,
		Window:    time.Minute,
		BurstSize: 0,
		KeyFunc: func(c echo.Context) string {
			// Custom key based on API key header
			return "apikey:" + c.Request().Header.Get("X-Api-Key")
		},
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// 2 requests with same API key should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Api-Key", "key123")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request with same API key should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Api-Key", "key123")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Request with different API key should succeed
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Api-Key", "key456")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_CustomExceedHandler(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     1,
		Window:    time.Minute,
		BurstSize: 0,
		ExceedHandler: func(c echo.Context, _ time.Duration) error {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"custom": "response",
			})
		},
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request succeeds
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request uses custom handler
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "custom")
}

func TestRateLimit_RetryAfterHeader(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	config := middleware.RateLimitConfig{
		Store:     store,
		Limit:     1,
		Window:    time.Minute,
		BurstSize: 0,
	}

	e.Use(middleware.RateLimit(config))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request succeeds
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request fails with Retry-After header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	retryAfter := rec.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter)
	retrySeconds, err := strconv.Atoi(retryAfter)
	require.NoError(t, err)
	assert.Positive(t, retrySeconds)
	assert.LessOrEqual(t, retrySeconds, 60)
}

// MemoryRateLimitStore tests

func TestMemoryRateLimitStore_Increment(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	ctx := context.Background()

	// First increment
	count, err := store.Increment(ctx, "key1", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Second increment
	count, err = store.Increment(ctx, "key1", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Different key
	count, err = store.Increment(ctx, "key2", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMemoryRateLimitStore_GetCount(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	ctx := context.Background()

	// Non-existent key
	count, err := store.GetCount(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// After increment
	_, _ = store.Increment(ctx, "key1", time.Minute)
	_, _ = store.Increment(ctx, "key1", time.Minute)
	count, err = store.GetCount(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestMemoryRateLimitStore_GetTTL(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	ctx := context.Background()

	// Non-existent key
	ttl, err := store.GetTTL(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), ttl)

	// After increment
	_, _ = store.Increment(ctx, "key1", time.Minute)
	ttl, err = store.GetTTL(ctx, "key1")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, time.Minute)
}

func TestMemoryRateLimitStore_Reset(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	ctx := context.Background()

	_, _ = store.Increment(ctx, "key1", time.Minute)
	_, _ = store.Increment(ctx, "key2", time.Minute)

	store.Reset()

	count, _ := store.GetCount(ctx, "key1")
	assert.Equal(t, int64(0), count)

	count, _ = store.GetCount(ctx, "key2")
	assert.Equal(t, int64(0), count)
}

func TestMemoryRateLimitStore_Expiration(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	ctx := context.Background()

	// Use very short window for testing
	window := 50 * time.Millisecond

	_, _ = store.Increment(ctx, "key1", window)
	count, _ := store.GetCount(ctx, "key1")
	assert.Equal(t, int64(1), count)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be reset
	count, _ = store.GetCount(ctx, "key1")
	assert.Equal(t, int64(0), count)

	// New increment should start fresh
	count, _ = store.Increment(ctx, "key1", window)
	assert.Equal(t, int64(1), count)
}

// RedisRateLimitStore tests

type mockRedisClient struct {
	values map[string]int64
	ttls   map[string]time.Duration
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		values: make(map[string]int64),
		ttls:   make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) Incr(_ context.Context, key string) (int64, error) {
	m.values[key]++
	return m.values[key], nil
}

func (m *mockRedisClient) Expire(_ context.Context, key string, expiration time.Duration) error {
	m.ttls[key] = expiration
	return nil
}

func (m *mockRedisClient) TTL(_ context.Context, key string) (time.Duration, error) {
	if ttl, ok := m.ttls[key]; ok {
		return ttl, nil
	}
	return 0, nil
}

func (m *mockRedisClient) Get(_ context.Context, key string) (string, error) {
	if val, ok := m.values[key]; ok {
		return strconv.FormatInt(val, 10), nil
	}
	return "", nil
}

func TestRedisRateLimitStore_Increment(t *testing.T) {
	client := newMockRedisClient()
	store := middleware.NewRedisRateLimitStore(client, "test:")
	ctx := context.Background()

	// First increment
	count, err := store.Increment(ctx, "key1", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Check that expiration was set
	assert.Equal(t, time.Minute, client.ttls["test:key1"])

	// Second increment
	count, err = store.Increment(ctx, "key1", time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRedisRateLimitStore_GetCount(t *testing.T) {
	client := newMockRedisClient()
	store := middleware.NewRedisRateLimitStore(client, "test:")
	ctx := context.Background()

	// Non-existent key
	count, err := store.GetCount(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// After increment
	_, _ = store.Increment(ctx, "key1", time.Minute)
	_, _ = store.Increment(ctx, "key1", time.Minute)
	count, err = store.GetCount(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRedisRateLimitStore_GetTTL(t *testing.T) {
	client := newMockRedisClient()
	store := middleware.NewRedisRateLimitStore(client, "test:")
	ctx := context.Background()

	_, _ = store.Increment(ctx, "key1", time.Minute)

	ttl, err := store.GetTTL(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)
}

func TestRedisRateLimitStore_DefaultPrefix(t *testing.T) {
	client := newMockRedisClient()
	store := middleware.NewRedisRateLimitStore(client, "")
	ctx := context.Background()

	_, _ = store.Increment(ctx, "key1", time.Minute)

	// Should use default prefix
	_, exists := client.values["flowra:ratelimit:key1"]
	assert.True(t, exists)
}

// EndpointRateLimits tests

func TestEndpointRateLimits_Set(t *testing.T) {
	limits := middleware.NewEndpointRateLimits()

	limits.Set("GET:/api/users", 100)
	limits.Set("POST:/api/users", 10)

	assert.Equal(t, 100, limits.Get("GET", "/api/users", 50))
	assert.Equal(t, 10, limits.Get("POST", "/api/users", 50))
	assert.Equal(t, 50, limits.Get("GET", "/api/other", 50)) // Default
}

func TestEndpointRateLimits_PatternMatching(t *testing.T) {
	limits := middleware.NewEndpointRateLimits()

	limits.Set("GET:/api/*", 100)

	assert.Equal(t, 100, limits.Get("GET", "/api/users", 50))
	assert.Equal(t, 100, limits.Get("GET", "/api/posts", 50))
	assert.Equal(t, 50, limits.Get("POST", "/api/users", 50)) // Different method
}

// WorkspaceRateLimiter tests

func TestWorkspaceRateLimiter_CustomLimits(t *testing.T) {
	store := middleware.NewMemoryRateLimitStore()
	limiter := middleware.NewWorkspaceRateLimiter(store, 100, time.Minute)

	workspace1 := uuid.NewUUID()
	workspace2 := uuid.NewUUID()

	// Set custom limit for workspace1
	limiter.SetWorkspaceLimit(workspace1, 50)

	assert.Equal(t, 50, limiter.GetLimit(workspace1))
	assert.Equal(t, 100, limiter.GetLimit(workspace2)) // Default
}

func TestWorkspaceRateLimiter_Middleware(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	limiter := middleware.NewWorkspaceRateLimiter(store, 2, time.Minute)

	workspaceID := uuid.NewUUID()

	// Set up workspace context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(string(middleware.ContextKeyWorkspaceID), workspaceID)
			return next(c)
		}
	})

	e.Use(limiter.Middleware())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First 2 requests should succeed
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "Workspace rate limit exceeded")
}

func TestWorkspaceRateLimiter_NoWorkspaceID(t *testing.T) {
	e := echo.New()

	store := middleware.NewMemoryRateLimitStore()
	limiter := middleware.NewWorkspaceRateLimiter(store, 1, time.Minute)

	// No workspace context set
	e.Use(limiter.Middleware())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Should pass through without limiting
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
