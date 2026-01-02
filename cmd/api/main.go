// Package main provides the API server entry point.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/config"
)

// Shutdown constants.
const (
	gracefulShutdownSleep = 100 * time.Millisecond
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		//nolint:sloglint // No context available before logger setup
		slog.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg)

	logger.Info("starting flowra API server",
		slog.String("version", "0.1.0"),
		slog.String("environment", getEnvironment(cfg)),
	)

	// Build DI container
	container, err := NewContainer(cfg, WithLogger(logger))
	if err != nil {
		logger.Error("failed to build container", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create a context that will be cancelled on shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Event Bus
	if startErr := container.StartEventBus(ctx); startErr != nil {
		logger.Error("failed to start event bus", slog.String("error", startErr.Error()))
		_ = container.Close()
		os.Exit(1) //nolint:gocritic // Intentional exit after cleanup
	}

	// Start WebSocket Hub
	container.StartHub(ctx)

	// Setup routes
	router := SetupRoutes(container)

	// Get the Echo instance from the router
	e := router.Echo()

	// Configure Echo server timeouts
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout

	// Start graceful shutdown handler
	go gracefulShutdown(ctx, cancel, e, container, cfg.Server.ShutdownTimeout, logger)

	// Start server
	logger.Info("server listening",
		slog.String("address", cfg.Server.Address()),
		slog.Duration("read_timeout", cfg.Server.ReadTimeout),
		slog.Duration("write_timeout", cfg.Server.WriteTimeout),
	)

	if serverErr := e.Start(cfg.Server.Address()); serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		logger.Error("server error", slog.String("error", serverErr.Error()))
		cancel()
		_ = container.Close()
		os.Exit(1)
	}
}

// setupLogger creates and configures the structured logger based on configuration.
func setupLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	level := parseLogLevel(cfg.Log.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.IsDevelopment(),
	}

	switch cfg.Log.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default: // "json" or any other value defaults to JSON
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// parseLogLevel converts a string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getEnvironment returns the environment name based on configuration.
func getEnvironment(cfg *config.Config) string {
	if cfg.IsDevelopment() {
		return "development"
	}
	if cfg.IsProduction() {
		return "production"
	}
	return "unknown"
}

// gracefulShutdown handles graceful shutdown on OS signals.
func gracefulShutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	e *echo.Echo,
	container *Container,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) {
	// Listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Create a background context for shutdown logging
	shutdownLogCtx := context.Background()

	select {
	case sig := <-quit:
		logger.InfoContext(shutdownLogCtx, "received shutdown signal", slog.String("signal", sig.String()))
	case <-ctx.Done():
		logger.InfoContext(shutdownLogCtx, "context cancelled, initiating shutdown")
	}

	logger.InfoContext(shutdownLogCtx, "shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer shutdownCancel()

	// 1. Stop accepting new connections
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "server shutdown error", slog.String("error", err.Error()))
	} else {
		logger.InfoContext(shutdownCtx, "HTTP server stopped")
	}

	// 2. Cancel the main context to stop background services
	cancel()

	// Give background services a moment to clean up
	time.Sleep(gracefulShutdownSleep)

	// 3. Close container resources
	if err := container.Close(); err != nil {
		logger.ErrorContext(shutdownCtx, "container close error", slog.String("error", err.Error()))
	}

	logger.InfoContext(shutdownCtx, "server shutdown complete")
}
