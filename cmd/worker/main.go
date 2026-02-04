// Package main provides the worker service entry point.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/infrastructure/metrics"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/outbox"
	"github.com/lllypuk/flowra/internal/infrastructure/projector"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/internal/worker"
)

// Timeout constants for worker service.
const redisPingTimeout = 5 * time.Second

//nolint:funlen // Main function handles startup orchestration and is readable as-is
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

	// Setup Redis for EventBus
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Error("failed to close Redis", slog.String("error", closeErr.Error()))
		}
	}()

	// Verify Redis connection
	pingCtx, pingCancel := context.WithTimeout(ctx, redisPingTimeout)
	if pingErr := redisClient.Ping(pingCtx).Err(); pingErr != nil {
		pingCancel()
		logger.Error("failed to connect to Redis", slog.String("error", pingErr.Error()))
		os.Exit(1)
	}
	pingCancel()

	logger.InfoContext(ctx, "connected to Redis", slog.String("addr", cfg.Redis.Addr))

	// Setup EventBus
	eventBus := eventbus.NewRedisEventBus(
		redisClient,
		eventbus.WithLogger(logger),
		eventbus.WithChannelPrefix(cfg.EventBus.RedisChannelPrefix),
	)

	// Setup Outbox
	outboxColl := db.Collection(mongodbinfra.CollectionOutbox)
	mongoOutbox := outbox.NewMongoOutbox(outboxColl, outbox.WithLogger(logger))

	// Setup metrics
	outboxMetrics := metrics.NewOutboxMetrics(prometheus.DefaultRegisterer)

	// Setup workers
	userSyncWorker, syncConfig := setupUserSyncWorker(cfg, userRepo, logger)

	// Create outbox worker configuration
	outboxConfig := worker.OutboxWorkerConfig{
		PollInterval:    cfg.Outbox.PollInterval,
		BatchSize:       cfg.Outbox.BatchSize,
		MaxRetries:      cfg.Outbox.MaxRetries,
		CleanupAge:      cfg.Outbox.CleanupAge,
		CleanupInterval: cfg.Outbox.CleanupInterval,
		Enabled:         cfg.Outbox.Enabled,
	}

	outboxWorker := worker.NewOutboxWorker(
		mongoOutbox,
		eventBus,
		logger,
		outboxConfig,
		outboxMetrics,
	)

	// Setup repair worker
	repairWorker := setupRepairWorker(mongoClient, db, cfg, logger)

	logger.Info("starting workers",
		slog.Bool("user_sync_enabled", syncConfig.Enabled),
		slog.Duration("user_sync_interval", syncConfig.Interval),
		slog.Bool("outbox_enabled", outboxConfig.Enabled),
		slog.Duration("outbox_poll_interval", outboxConfig.PollInterval),
		slog.Bool("repair_enabled", repairWorker != nil),
	)

	// Use WaitGroup to run multiple workers concurrently
	var wg sync.WaitGroup

	// Start user sync worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		if runErr := userSyncWorker.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("user sync worker error", slog.String("error", runErr.Error()))
		}
	}()

	// Start outbox worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		if runErr := outboxWorker.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("outbox worker error", slog.String("error", runErr.Error()))
		}
	}()

	// Start repair worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		if runErr := repairWorker.Start(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("repair worker error", slog.String("error", runErr.Error()))
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

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

// setupUserSyncWorker creates and configures the user sync worker.
func setupUserSyncWorker(
	cfg *config.Config,
	userRepo *mongodb.MongoUserRepository,
	logger *slog.Logger,
) (*worker.UserSyncWorker, worker.UserSyncConfig) {
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

	// Create user sync worker configuration
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

	workerInstance := worker.NewUserSyncWorker(
		userClient,
		userRepo,
		logger,
		syncConfig,
	)

	return workerInstance, syncConfig
}

// setupRepairWorker creates and configures the repair worker.
func setupRepairWorker(
	mongoClient *mongo.Client,
	db *mongo.Database,
	cfg *config.Config,
	logger *slog.Logger,
) *worker.RepairWorker {
	// Create repair worker configuration
	repairConfig := worker.DefaultRepairWorkerConfig()
	if os.Getenv("REPAIR_WORKER_DISABLED") == "true" {
		repairConfig.Enabled = false
	}

	// Setup repair queue
	repairQueueColl := db.Collection(mongodbinfra.CollectionRepairQueue)
	repairQueue := repair.NewMongoQueue(repairQueueColl, logger)

	// Setup projectors for repair worker
	eventStore := eventstore.NewMongoEventStore(
		mongoClient,
		cfg.MongoDB.Database,
		eventstore.WithLogger(logger),
	)

	chatReadModelColl := db.Collection(mongodbinfra.CollectionChatReadModel)
	chatProjector := projector.NewChatProjector(eventStore, chatReadModelColl, logger)

	taskReadModelColl := db.Collection(mongodbinfra.CollectionTaskReadModel)
	taskProjector := projector.NewTaskProjector(eventStore, taskReadModelColl, logger)

	// Create repair worker
	return worker.NewRepairWorker(
		repairQueue,
		chatProjector,
		taskProjector,
		logger,
		repairConfig,
	)
}
