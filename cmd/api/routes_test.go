package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusConstants(t *testing.T) {
	// Test that constants are defined in httpserver package
	assert.Equal(t, "healthy", httpserver.StatusHealthy)
	assert.Equal(t, "unhealthy", httpserver.StatusUnhealthy)
	assert.Equal(t, "ready", httpserver.StatusReady)
	assert.Equal(t, "not_ready", httpserver.StatusNotReady)
	assert.Equal(t, "degraded", httpserver.StatusDegraded)
}

func TestSetupRoutes_ReturnsRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	// Create minimal container with mock components
	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)

	require.NotNil(t, router)
	require.NotNil(t, router.Echo())
}

func TestSetupRoutes_HealthEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Test /health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), httpserver.StatusHealthy)
}

func TestSetupRoutes_ReadyEndpoint_NotReady(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	// Container without initialized resources should not be ready
	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Test /ready endpoint - should not be ready since no MongoDB/Redis
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), httpserver.StatusNotReady)
}

func TestSetupRoutes_HealthDetailsEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Test /health/details endpoint
	req := httptest.NewRequest(http.MethodGet, "/health/details", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return unhealthy status since no resources are initialized
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), httpserver.StatusUnhealthy)
	assert.Contains(t, rec.Body.String(), "components")
}

func TestSetupRoutes_RegistersAuthRoutes(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Find registered routes
	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Auth routes should be registered
	assert.True(t, routePaths["POST:/api/v1/auth/login"], "login route should be registered")
}

func TestSetupRoutes_RegistersHealthEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Find registered routes
	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Health endpoints should be registered
	assert.True(t, routePaths["GET:/health"], "health route should be registered")
	assert.True(t, routePaths["GET:/ready"], "ready route should be registered")
	assert.True(t, routePaths["GET:/health/details"], "health details route should be registered")
}

func TestSetupRoutes_RegistersWorkspaceRoutes(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Workspace routes should be registered
	assert.True(t, routePaths["POST:/api/v1/workspaces"], "create workspace route should be registered")
	assert.True(t, routePaths["GET:/api/v1/workspaces"], "list workspaces route should be registered")
}

func TestSetupRoutes_RegistersChatRoutes(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Chat routes should be registered
	assert.True(t, routePaths["POST:/api/v1/workspaces/:workspace_id/chats"], "create chat route should be registered")
	assert.True(t, routePaths["GET:/api/v1/workspaces/:workspace_id/chats"], "list chats route should be registered")
}

func TestSetupRoutes_RegistersWebSocketRoute(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// WebSocket route should be registered
	assert.True(t, routePaths["GET:/api/v1/ws"], "websocket route should be registered")
}

func TestSetupRoutes_PlaceholderEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	// Container without message, task, notification, user handlers
	c := &Container{
		Config:              cfg,
		Logger:              logger,
		TokenValidator:      middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:       middleware.NewMockWorkspaceAccessChecker(),
		Hub:                 websocket.NewHub(),
		MessageHandler:      nil,
		TaskHandler:         nil,
		NotificationHandler: nil,
		UserHandler:         nil,
	}

	router := SetupRoutes(c)
	e := router.Echo()

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	// Placeholder routes should still be registered
	assert.True(t, routePaths["GET:/api/v1/notifications"], "notification placeholder should be registered")
	assert.True(t, routePaths["GET:/api/v1/users/me"], "user placeholder should be registered")
}

func TestContainer_IsReady_Context(t *testing.T) {
	cfg := config.DefaultConfig()
	c := &Container{
		Config: cfg,
		Logger: slog.Default(),
	}

	ctx := context.Background()
	ready := c.IsReady(ctx)

	// Should not be ready since no resources are initialized
	assert.False(t, ready)
}

func TestHealthEndpoint_DirectCall(t *testing.T) {
	e := echo.New()

	// Register a simple health handler
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": httpserver.StatusHealthy,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
}

func TestRouteGroups_Created(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	c := &Container{
		Config:         cfg,
		Logger:         logger,
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)

	// Verify groups are created
	assert.NotNil(t, router.Public())
	assert.NotNil(t, router.Auth())
	assert.NotNil(t, router.Workspace())
}

func TestSetupRoutes_EchoConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Level = "debug" // Enable development mode

	c := &Container{
		Config:         cfg,
		Logger:         slog.Default(),
		TokenValidator: middleware.NewStaticTokenValidator(cfg.Auth.JWTSecret),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
		Hub:            websocket.NewHub(),
	}

	router := SetupRoutes(c)
	e := router.Echo()

	// Echo should be configured to hide banner
	assert.True(t, e.HideBanner)
	assert.True(t, e.HidePort)
}

func TestCreatePlaceholderHandler(t *testing.T) {
	handler := createPlaceholderHandler("Test")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotImplemented, rec.Code)
	assert.Contains(t, rec.Body.String(), "NOT_IMPLEMENTED")
	assert.Contains(t, rec.Body.String(), "Test service not available")
}
