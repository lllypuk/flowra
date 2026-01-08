package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLoggingConfig(t *testing.T) {
	config := middleware.DefaultLoggingConfig()

	assert.NotNil(t, config.Logger)
	assert.Equal(t, []string{"/health", "/ready"}, config.SkipPaths)
	assert.False(t, config.LogRequestBody)
	assert.False(t, config.LogResponseBody)
}

func TestLogging(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectLog      bool
	}{
		{
			name:           "GET request is logged",
			method:         http.MethodGet,
			path:           "/api/users",
			expectedStatus: http.StatusOK,
			expectLog:      true,
		},
		{
			name:           "POST request is logged",
			method:         http.MethodPost,
			path:           "/api/users",
			expectedStatus: http.StatusCreated,
			expectLog:      true,
		},
		{
			name:           "health check is skipped",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectLog:      false,
		},
		{
			name:           "ready check is skipped",
			method:         http.MethodGet,
			path:           "/ready",
			expectedStatus: http.StatusOK,
			expectLog:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

			e := echo.New()
			e.Use(middleware.Logging(middleware.LoggingConfig{
				Logger:    logger,
				SkipPaths: []string{"/health", "/ready"},
			}))

			e.GET("/api/users", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})
			e.POST("/api/users", func(c echo.Context) error {
				return c.String(http.StatusCreated, "created")
			})
			e.GET("/health", func(c echo.Context) error {
				return c.String(http.StatusOK, "healthy")
			})
			e.GET("/ready", func(c echo.Context) error {
				return c.String(http.StatusOK, "ready")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectLog {
				assert.NotEmpty(t, logBuffer.String())
				assert.Contains(t, logBuffer.String(), tt.path)
				assert.Contains(t, logBuffer.String(), tt.method)
			} else {
				assert.Empty(t, logBuffer.String())
			}
		})
	}
}

func TestLoggingRequestID(t *testing.T) {
	tests := []struct {
		name              string
		provideRequestID  bool
		providedRequestID string
	}{
		{
			name:             "generates request ID when not provided",
			provideRequestID: false,
		},
		{
			name:              "uses provided request ID",
			provideRequestID:  true,
			providedRequestID: "custom-request-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

			e := echo.New()
			e.Use(middleware.Logging(middleware.LoggingConfig{
				Logger:    logger,
				SkipPaths: []string{},
			}))

			e.GET("/test", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.provideRequestID {
				req.Header.Set(middleware.RequestIDHeader, tt.providedRequestID)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Check response header has request ID
			responseRequestID := rec.Header().Get(middleware.RequestIDHeader)
			require.NotEmpty(t, responseRequestID)

			if tt.provideRequestID {
				assert.Equal(t, tt.providedRequestID, responseRequestID)
			}

			// Check log contains request ID
			assert.Contains(t, logBuffer.String(), "request_id")
			assert.Contains(t, logBuffer.String(), responseRequestID)
		})
	}
}

func TestLoggingLatency(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Check that latency is logged (it will be in nanoseconds format)
	assert.Contains(t, logBuffer.String(), "latency")
}

func TestLoggingStatusCodeLevels(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{
			name:          "2xx status logs at INFO",
			statusCode:    http.StatusOK,
			expectedLevel: "INFO",
		},
		{
			name:          "3xx status logs at INFO",
			statusCode:    http.StatusMovedPermanently,
			expectedLevel: "INFO",
		},
		{
			name:          "4xx status logs at WARN",
			statusCode:    http.StatusBadRequest,
			expectedLevel: "WARN",
		},
		{
			name:          "404 status logs at WARN",
			statusCode:    http.StatusNotFound,
			expectedLevel: "WARN",
		},
		{
			name:          "5xx status logs at ERROR",
			statusCode:    http.StatusInternalServerError,
			expectedLevel: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

			e := echo.New()
			e.Use(middleware.Logging(middleware.LoggingConfig{
				Logger:    logger,
				SkipPaths: []string{},
			}))

			e.GET("/test", func(c echo.Context) error {
				return c.String(tt.statusCode, "response")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
			assert.Contains(t, logBuffer.String(), tt.expectedLevel)
		})
	}
}

func TestLoggingWithQueryString(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/search", func(c echo.Context) error {
		return c.String(http.StatusOK, "results")
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=test&page=1", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuffer.String(), "query")
	assert.Contains(t, logBuffer.String(), "q=test")
}

func TestLoggingUserAgent(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuffer.String(), "user_agent")
	assert.Contains(t, logBuffer.String(), "TestBrowser/1.0")
}

func TestLoggingContentLength(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.POST("/api/data", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	body := strings.NewReader(`{"name":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/data", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, logBuffer.String(), "content_length")
}

func TestLoggingResponseSize(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	responseBody := `{"id":"123","name":"test user"}`
	e.GET("/api/user", func(c echo.Context) error {
		return c.String(http.StatusOK, responseBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuffer.String(), "response_size")
}

func TestLoggingWithError(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/error", func(_ echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, logBuffer.String(), "error")
	assert.Contains(t, logBuffer.String(), "WARN")
}

func TestLoggingWithDefaults(t *testing.T) {
	e := echo.New()
	e.Use(middleware.LoggingWithDefaults())

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name              string
		setRequestID      bool
		requestID         string
		expectedRequestID string
	}{
		{
			name:              "returns request ID when set",
			setRequestID:      true,
			requestID:         "test-request-id-456",
			expectedRequestID: "test-request-id-456",
		},
		{
			name:              "returns empty string when not set",
			setRequestID:      false,
			expectedRequestID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setRequestID {
				c.Set(middleware.RequestIDKey, tt.requestID)
			}

			result := middleware.GetRequestID(c)
			assert.Equal(t, tt.expectedRequestID, result)
		})
	}
}

func TestLoggingRemoteIP(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuffer.String(), "remote_ip")
}

func TestLoggingNilLogger(t *testing.T) {
	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    nil, // Should use default logger
		SkipPaths: []string{},
	}))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoggingLogFormat(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Logging(middleware.LoggingConfig{
		Logger:    logger,
		SkipPaths: []string{},
	}))

	e.GET("/api/users", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Parse log output as JSON
	var logEntry map[string]any
	err := json.Unmarshal(logBuffer.Bytes(), &logEntry)
	require.NoError(t, err)

	// Check required fields exist
	assert.Contains(t, logEntry, "msg")
	assert.Equal(t, "HTTP request", logEntry["msg"])
	assert.Contains(t, logEntry, "request_id")
	assert.Contains(t, logEntry, "method")
	assert.Contains(t, logEntry, "path")
	assert.Contains(t, logEntry, "status")
	assert.Contains(t, logEntry, "latency")
	assert.Contains(t, logEntry, "remote_ip")
}
