package middleware

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HTTP status code thresholds for log levels.
const (
	statusClientError = 400
	statusServerError = 500
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the context key for request ID.
	RequestIDKey = "request_id"
)

// LoggingConfig holds configuration for the logging middleware.
type LoggingConfig struct {
	Logger          *slog.Logger
	SkipPaths       []string
	LogRequestBody  bool
	LogResponseBody bool
}

// DefaultLoggingConfig returns a LoggingConfig with sensible defaults.
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:          slog.Default(),
		SkipPaths:       []string{"/health", "/ready"},
		LogRequestBody:  false,
		LogResponseBody: false,
	}
}

// Logging returns a middleware that logs HTTP requests with request ID tracking.
//
//nolint:gocognit // Middleware functions are inherently complex due to request lifecycle handling.
func Logging(config LoggingConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	skipPaths := make(map[string]struct{}, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			path := req.URL.Path

			// Skip logging for configured paths
			if _, ok := skipPaths[path]; ok {
				return next(c)
			}

			// Get or generate request ID
			requestID := req.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Set request ID in response header and context
			res.Header().Set(RequestIDHeader, requestID)
			c.Set(RequestIDKey, requestID)

			// Record start time
			start := time.Now()

			// Process request
			err := next(c)

			// Calculate latency
			latency := time.Since(start)

			// Determine status code
			status := res.Status
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					status = he.Code
				}
			}

			// Build log attributes
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.Duration("latency", latency),
				slog.String("remote_ip", c.RealIP()),
				slog.String("user_agent", req.UserAgent()),
			}

			// Add query string if present
			if query := req.URL.RawQuery; query != "" {
				attrs = append(attrs, slog.String("query", query))
			}

			// Add content length if present
			if req.ContentLength > 0 {
				attrs = append(attrs, slog.Int64("content_length", req.ContentLength))
			}

			// Add response size
			attrs = append(attrs, slog.Int64("response_size", res.Size))

			// Log based on status code
			msg := "HTTP request"
			level := slog.LevelInfo

			switch {
			case status >= statusServerError:
				level = slog.LevelError
				if err != nil {
					attrs = append(attrs, slog.String("error", err.Error()))
				}
			case status >= statusClientError:
				level = slog.LevelWarn
				if err != nil {
					attrs = append(attrs, slog.String("error", err.Error()))
				}
			}

			config.Logger.LogAttrs(c.Request().Context(), level, msg, attrs...)

			return err
		}
	}
}

// LoggingWithDefaults returns a logging middleware with default configuration.
func LoggingWithDefaults() echo.MiddlewareFunc {
	return Logging(DefaultLoggingConfig())
}

// GetRequestID retrieves the request ID from the echo context.
func GetRequestID(c echo.Context) string {
	if id, ok := c.Get(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
