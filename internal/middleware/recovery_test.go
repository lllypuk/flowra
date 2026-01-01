package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRecoveryConfig(t *testing.T) {
	config := middleware.DefaultRecoveryConfig()

	assert.NotNil(t, config.Logger)
	assert.Equal(t, middleware.DefaultStackSize, config.StackSize)
	assert.True(t, config.DisableStackAll)
	assert.False(t, config.DisablePrintStack)
}

func TestRecovery(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/panic", func(_ echo.Context) error {
		panic("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, false, response["success"])

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INTERNAL_ERROR", errorObj["code"])
	assert.Equal(t, "An internal error occurred", errorObj["message"])

	// Check log was written
	assert.Contains(t, logBuffer.String(), "panic recovered")
	assert.Contains(t, logBuffer.String(), "something went wrong")
}

func TestRecoveryWithErrorPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/error-panic", func(_ echo.Context) error {
		panic(echo.NewHTTPError(http.StatusBadRequest, "bad request"))
	})

	req := httptest.NewRequest(http.MethodGet, "/error-panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "panic recovered")
}

func TestRecoveryWithIntPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/int-panic", func(_ echo.Context) error {
		panic(42)
	})

	req := httptest.NewRequest(http.MethodGet, "/int-panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "panic recovered")
	assert.Contains(t, logBuffer.String(), "42")
}

func TestRecoveryNoPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
	assert.Empty(t, logBuffer.String())
}

func TestRecoveryWithConfig(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	config := middleware.RecoveryConfig{
		Logger:            logger,
		StackSize:         8 << 10, // 8KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}

	e := echo.New()
	e.Use(middleware.RecoveryWithConfig(config))

	e.GET("/panic", func(_ echo.Context) error {
		panic("custom config panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "custom config panic")
	assert.Contains(t, logBuffer.String(), "stack")
}

func TestRecoveryDisablePrintStack(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	config := middleware.RecoveryConfig{
		Logger:            logger,
		StackSize:         middleware.DefaultStackSize,
		DisableStackAll:   true,
		DisablePrintStack: true,
	}

	e := echo.New()
	e.Use(middleware.RecoveryWithConfig(config))

	e.GET("/panic", func(_ echo.Context) error {
		panic("no stack trace")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "panic recovered")
	assert.Contains(t, logBuffer.String(), "no stack trace")
	// Stack should NOT be in the log
	assert.NotContains(t, logBuffer.String(), `"stack"`)
}

func TestRecoveryNilLogger(t *testing.T) {
	e := echo.New()
	e.Use(middleware.RecoveryWithConfig(middleware.RecoveryConfig{
		Logger:    nil, // Should use default logger
		StackSize: middleware.DefaultStackSize,
	}))

	e.GET("/panic", func(_ echo.Context) error {
		panic("nil logger panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRecoveryZeroStackSize(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	config := middleware.RecoveryConfig{
		Logger:    logger,
		StackSize: 0, // Should use default
	}

	e := echo.New()
	e.Use(middleware.RecoveryWithConfig(config))

	e.GET("/panic", func(_ echo.Context) error {
		panic("zero stack size")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "stack")
}

func TestRecoveryLogsRequestInfo(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.POST("/api/users", func(_ echo.Context) error {
		panic("panic in POST")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Check request info is logged
	assert.Contains(t, logBuffer.String(), "method")
	assert.Contains(t, logBuffer.String(), "POST")
	assert.Contains(t, logBuffer.String(), "path")
	assert.Contains(t, logBuffer.String(), "/api/users")
}

func TestRecoveryWithRequestID(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/panic", func(_ echo.Context) error {
		panic("panic with request id")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Request-ID", "test-request-id-789")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "request_id")
	assert.Contains(t, logBuffer.String(), "test-request-id-789")
}

func TestRecoveryRemoteIP(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/panic", func(_ echo.Context) error {
		panic("panic with remote ip")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Real-IP", "10.0.0.50")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "remote_ip")
}

func TestRecoveryResponseFormat(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/panic", func(_ echo.Context) error {
		panic("check response format")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var response struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.Success)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
	assert.Equal(t, "An internal error occurred", response.Error.Message)
}

func TestRecoveryStackTraceContent(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	config := middleware.RecoveryConfig{
		Logger:            logger,
		StackSize:         middleware.DefaultStackSize,
		DisableStackAll:   true,
		DisablePrintStack: false,
	}

	e := echo.New()
	e.Use(middleware.RecoveryWithConfig(config))

	e.GET("/panic", func(_ echo.Context) error {
		panic("stack trace test")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Stack trace should contain goroutine info
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "stack")
	assert.Contains(t, logOutput, "goroutine")
}

func TestRecoveryMiddlewareChain(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()

	// Recovery should be first in chain to catch panics from other middleware
	e.Use(middleware.Recovery(logger))

	// This middleware will panic
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/middleware-panic" {
				panic("middleware panic")
			}
			return next(c)
		}
	})

	e.GET("/middleware-panic", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/middleware-panic", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, logBuffer.String(), "middleware panic")
}

func TestRecoveryWithNilPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	e := echo.New()
	e.Use(middleware.Recovery(logger))

	e.GET("/nil-panic", func(_ echo.Context) error {
		panic(nil) //nolint:govet // testing nil panic handling
	})

	req := httptest.NewRequest(http.MethodGet, "/nil-panic", nil)
	rec := httptest.NewRecorder()

	// Go 1.21+ treats panic(nil) as a real panic
	e.ServeHTTP(rec, req)

	// Should handle gracefully
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
