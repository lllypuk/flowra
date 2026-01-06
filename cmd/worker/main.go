// Package main provides the worker service entry point.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/internal/worker"
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

	logger.Info("starting flowra worker service",
		slog.String("version", "0.1.0"),
		slog.String("environment", getEnvironment(cfg)),
	)

	// Create a context that will be cancelled on shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	go handleShutdown(cancel, logger)

	// Connect to MongoDB
	mongoClient, err := connectMongoDB(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to connect to MongoDB", slog.String("error", err.Error()))
		cancel()
		os.Exit(1) //nolint:gocritic // cancel() called before exit
	}
	defer func() {
		if disconnectErr := mongoClient.Disconnect(context.Background()); disconnectErr != nil {
			logger.Error("failed to disconnect from MongoDB", slog.String("error", disconnectErr.Error()))
		}
	}()

	// Setup repositories
	db := mongoClient.Database(cfg.MongoDB.Database)
	userRepo := mongodb.NewMongoUserRepository(db.Collection("users"))

	// Setup Keycloak clients
	if cfg.Keycloak.URL == "" || cfg.Keycloak.AdminUsername == "" {
		logger.Error("Keycloak configuration is required for user sync worker")
		os.Exit(1)
	}

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: cfg.Keycloak.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    cfg.Keycloak.AdminUsername,
		Password:    cfg.Keycloak.AdminPassword,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: cfg.Keycloak.URL,
		Realm:       cfg.Keycloak.Realm,
	}, tokenManager)

	// Create and start user sync worker
	syncConfig := worker.DefaultUserSyncConfig()
	// Override from environment if needed
	if interval := os.Getenv("USER_SYNC_INTERVAL"); interval != "" {
		if parsed, parseErr := time.ParseDuration(interval); parseErr == nil {
			syncConfig.Interval = parsed
		}
	}
	if os.Getenv("USER_SYNC_DISABLED") == "true" {
		syncConfig.Enabled = false
	}

	userSyncWorker := worker.NewUserSyncWorker(
		userClient,
		userRepo,
		logger,
		syncConfig,
	)

	logger.Info("starting user sync worker",
		slog.Duration("interval", syncConfig.Interval),
		slog.Int("batch_size", syncConfig.BatchSize),
		slog.Bool("enabled", syncConfig.Enabled),
	)

	// Run the worker (blocks until context is cancelled)
	if runErr := userSyncWorker.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error("user sync worker error", slog.String("error", runErr.Error()))
		os.Exit(1)
	}

	logger.Info("worker service shutdown complete")
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
	default:
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

// connectMongoDB establishes a connection to MongoDB.
func connectMongoDB(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*mongo.Client, error) {
	clientOpts := options.Client().
		ApplyURI(cfg.MongoDB.URI).
		SetMaxPoolSize(cfg.MongoDB.MaxPoolSize)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	pingCtx, pingCancel := context.WithTimeout(ctx, cfg.MongoDB.Timeout)
	defer pingCancel()

	if pingErr := client.Ping(pingCtx, nil); pingErr != nil {
		return nil, pingErr
	}

	logger.InfoContext(ctx, "connected to MongoDB",
		slog.String("database", cfg.MongoDB.Database),
	)

	return client, nil
}

// handleShutdown listens for OS signals and cancels the context.
func handleShutdown(cancel context.CancelFunc, logger *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-quit
	logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	cancel()
}
