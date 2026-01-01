package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Default server configuration values.
const (
	DefaultHost            = "0.0.0.0"
	DefaultPort            = 8080
	DefaultReadTimeout     = 30 * time.Second
	DefaultWriteTimeout    = 30 * time.Second
	DefaultShutdownTimeout = 10 * time.Second
	DefaultBodyLimit       = "2M"
	DefaultMaxHeaderBytes  = 1 << 20 // 1MB
)

// ServerConfig holds configuration for the HTTP server.
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	BodyLimit       string
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            DefaultHost,
		Port:            DefaultPort,
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		ShutdownTimeout: DefaultShutdownTimeout,
		BodyLimit:       DefaultBodyLimit,
	}
}

// Server represents the HTTP server.
type Server struct {
	echo            *echo.Echo
	config          ServerConfig
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer creates a new HTTP server with the given configuration.
func NewServer(config ServerConfig, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Configure server timeouts
	e.Server.ReadTimeout = config.ReadTimeout
	e.Server.WriteTimeout = config.WriteTimeout

	// Configure body limit
	if config.BodyLimit != "" {
		e.Server.MaxHeaderBytes = DefaultMaxHeaderBytes
	}

	return &Server{
		echo:            e,
		config:          config,
		logger:          logger,
		shutdownTimeout: config.ShutdownTimeout,
	}
}

// Echo returns the underlying Echo instance for middleware and route registration.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Use adds middleware to the server.
func (s *Server) Use(middleware ...echo.MiddlewareFunc) {
	s.echo.Use(middleware...)
}

// RegisterRoutes allows external route registration via a callback function.
func (s *Server) RegisterRoutes(register func(e *echo.Echo)) {
	register(s.echo)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("starting HTTP server",
		slog.String("address", addr),
		slog.Duration("read_timeout", s.config.ReadTimeout),
		slog.Duration("write_timeout", s.config.WriteTimeout),
	)

	if err := s.echo.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "shutting down HTTP server",
		slog.Duration("timeout", s.shutdownTimeout),
	)

	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	if err := s.echo.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.InfoContext(ctx, "HTTP server stopped")
	return nil
}

// Address returns the server address.
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// HealthCheck registers a basic health check endpoint.
func (s *Server) HealthCheck(path string) {
	s.echo.GET(path, func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})
}

// Ready registers a readiness check endpoint with a custom check function.
func (s *Server) Ready(path string, check func() bool) {
	s.echo.GET(path, func(c echo.Context) error {
		if check == nil || check() {
			return c.JSON(http.StatusOK, map[string]string{
				"status": "ready",
			})
		}
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
		})
	})
}
