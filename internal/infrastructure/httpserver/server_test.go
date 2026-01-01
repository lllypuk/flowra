package httpserver_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultServerConfig(t *testing.T) {
	config := httpserver.DefaultServerConfig()

	assert.Equal(t, httpserver.DefaultHost, config.Host)
	assert.Equal(t, httpserver.DefaultPort, config.Port)
	assert.Equal(t, httpserver.DefaultReadTimeout, config.ReadTimeout)
	assert.Equal(t, httpserver.DefaultWriteTimeout, config.WriteTimeout)
	assert.Equal(t, httpserver.DefaultShutdownTimeout, config.ShutdownTimeout)
	assert.Equal(t, httpserver.DefaultBodyLimit, config.BodyLimit)
}

func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		config httpserver.ServerConfig
		logger *slog.Logger
	}{
		{
			name:   "with default config and nil logger",
			config: httpserver.DefaultServerConfig(),
			logger: nil,
		},
		{
			name: "with custom config and logger",
			config: httpserver.ServerConfig{
				Host:            "127.0.0.1",
				Port:            3000,
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				ShutdownTimeout: 5 * time.Second,
				BodyLimit:       "1M",
			},
			logger: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httpserver.NewServer(tt.config, tt.logger)

			require.NotNil(t, server)
			assert.NotNil(t, server.Echo())
		})
	}
}

func TestServerEcho(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	e := server.Echo()

	require.NotNil(t, e)
	assert.True(t, e.HideBanner)
	assert.True(t, e.HidePort)
}

func TestServerUse(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	middlewareCalled := false
	middleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middlewareCalled = true
			return next(c)
		}
	}

	server.Use(middleware)

	// Register a test route
	server.Echo().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Make a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServerRegisterRoutes(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	routeRegistered := false
	server.RegisterRoutes(func(e *echo.Echo) {
		routeRegistered = true
		e.GET("/api/v1/users", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"users": "list"})
		})
	})

	assert.True(t, routeRegistered)

	// Test that the route works
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"users":"list"}`, rec.Body.String())
}

func TestServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   httpserver.ServerConfig
		expected string
	}{
		{
			name:     "default config",
			config:   httpserver.DefaultServerConfig(),
			expected: "0.0.0.0:8080",
		},
		{
			name: "custom config",
			config: httpserver.ServerConfig{
				Host: "localhost",
				Port: 3000,
			},
			expected: "localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httpserver.NewServer(tt.config, nil)
			assert.Equal(t, tt.expected, server.Address())
		})
	}
}

func TestServerHealthCheck(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)
	server.HealthCheck("/health")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"healthy"}`, rec.Body.String())
}

func TestServerHealthCheckCustomPath(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)
	server.HealthCheck("/api/health")

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"healthy"}`, rec.Body.String())
}

func TestServerReady(t *testing.T) {
	tests := []struct {
		name           string
		checkFunc      func() bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "ready when check returns true",
			checkFunc:      func() bool { return true },
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ready"}`,
		},
		{
			name:           "not ready when check returns false",
			checkFunc:      func() bool { return false },
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `{"status":"not ready"}`,
		},
		{
			name:           "ready when check is nil",
			checkFunc:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ready"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)
			server.Ready("/ready", tt.checkFunc)

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.JSONEq(t, tt.expectedBody, rec.Body.String())
		})
	}
}

func TestServerShutdown(t *testing.T) {
	config := httpserver.ServerConfig{
		Host:            "127.0.0.1",
		Port:            0, // Random port
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
	server := httpserver.NewServer(config, nil)

	// Register a simple route
	server.Echo().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Shutdown should complete without error even if server wasn't started
	err := server.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestServerTimeoutConfiguration(t *testing.T) {
	config := httpserver.ServerConfig{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	server := httpserver.NewServer(config, nil)

	assert.Equal(t, 15*time.Second, server.Echo().Server.ReadTimeout)
	assert.Equal(t, 20*time.Second, server.Echo().Server.WriteTimeout)
}

func TestServerMultipleMiddleware(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	order := []string{}

	middleware1 := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			order = append(order, "m1-before")
			err := next(c)
			order = append(order, "m1-after")
			return err
		}
	}

	middleware2 := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			order = append(order, "m2-before")
			err := next(c)
			order = append(order, "m2-after")
			return err
		}
	}

	server.Use(middleware1, middleware2)

	server.Echo().GET("/test", func(c echo.Context) error {
		order = append(order, "handler")
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}, order)
}

func TestServerNotFoundRoute(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServerMethodNotAllowed(t *testing.T) {
	server := httpserver.NewServer(httpserver.DefaultServerConfig(), nil)

	server.Echo().GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Try POST on a GET-only route
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	server.Echo().ServeHTTP(rec, req)

	// Echo returns 405 Method Not Allowed
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
