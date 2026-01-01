package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
)

// Recovery configuration constants.
const (
	// DefaultStackSize is the default stack trace size (4KB).
	DefaultStackSize = 4 << 10
)

// RecoveryConfig holds configuration for the recovery middleware.
type RecoveryConfig struct {
	// Logger is the structured logger to use for panic logging.
	Logger *slog.Logger

	// StackSize is the maximum size of the stack trace to capture.
	// Default is 4KB.
	StackSize int

	// DisableStackAll disables capturing all goroutines stack traces.
	// When false, only the current goroutine's stack is captured.
	DisableStackAll bool

	// DisablePrintStack disables printing the stack trace to the logger.
	DisablePrintStack bool
}

// DefaultRecoveryConfig returns a RecoveryConfig with sensible defaults.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		Logger:            slog.Default(),
		StackSize:         DefaultStackSize,
		DisableStackAll:   true,
		DisablePrintStack: false,
	}
}

// Recovery returns a middleware that recovers from panics and logs the error.
func Recovery(logger *slog.Logger) echo.MiddlewareFunc {
	config := DefaultRecoveryConfig()
	config.Logger = logger
	return RecoveryWithConfig(config)
}

// RecoveryWithConfig returns a recovery middleware with custom configuration.
//
//nolint:gocognit,nestif // Recovery middleware requires complex nested error handling for panic recovery.
func RecoveryWithConfig(config RecoveryConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.StackSize == 0 {
		config.StackSize = DefaultStackSize
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}

					// Capture stack trace
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, !config.DisableStackAll)
					stack = stack[:length]

					// Get request context for logging
					req := c.Request()
					requestID := c.Response().Header().Get(echo.HeaderXRequestID)
					if requestID == "" {
						requestID = req.Header.Get(echo.HeaderXRequestID)
					}

					// Log the panic
					logAttrs := []any{
						slog.String("error", err.Error()),
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
						slog.String("remote_ip", c.RealIP()),
					}

					if requestID != "" {
						logAttrs = append(logAttrs, slog.String("request_id", requestID))
					}

					if !config.DisablePrintStack {
						logAttrs = append(logAttrs, slog.String("stack", string(stack)))
					}

					config.Logger.Error("panic recovered", logAttrs...)

					// Send error response
					if !c.Response().Committed {
						_ = c.JSON(http.StatusInternalServerError, map[string]any{
							"success": false,
							"error": map[string]string{
								"code":    "INTERNAL_ERROR",
								"message": "An internal error occurred",
							},
						})
					}
				}
			}()

			return next(c)
		}
	}
}
