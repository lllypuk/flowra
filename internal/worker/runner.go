package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/infrastructure/metrics"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/outbox"
	"github.com/lllypuk/flowra/internal/infrastructure/projector"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
)

const masterRealm = "master"

// Run starts all worker loops and blocks until they are stopped.
func Run(ctx context.Context, cfg *config.Config, mongoDB *mongo.Database, redisCli *redis.Client) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if mongoDB == nil {
		return errors.New("mongodb database is nil")
	}
	if redisCli == nil {
		return errors.New("redis client is nil")
	}

	logger := slog.Default()

	legacyCtx, legacyCancel := context.WithTimeout(ctx, cfg.MongoDB.Timeout)
	if warnErr := mongodbinfra.WarnIfLegacyReadModelCollectionsContainData(legacyCtx, mongoDB, logger); warnErr != nil {
		logger.WarnContext(legacyCtx, "failed to inspect legacy read model collections",
			slog.String("error", warnErr.Error()),
		)
	}
	legacyCancel()

	userRepo := mongorepo.NewMongoUserRepository(mongoDB.Collection("users"))

	eventBusInstance := eventbus.NewRedisEventBus(
		redisCli,
		eventbus.WithLogger(logger),
		eventbus.WithChannelPrefix(cfg.EventBus.RedisChannelPrefix),
	)

	outboxColl := mongoDB.Collection(mongodbinfra.CollectionOutbox)
	mongoOutbox := outbox.NewMongoOutbox(outboxColl, outbox.WithLogger(logger))
	outboxMetrics := metrics.NewOutboxMetrics(prometheus.DefaultRegisterer)

	userSyncWorker, syncConfig, err := setupUserSyncWorker(cfg, userRepo, logger)
	if err != nil {
		return fmt.Errorf("setup user sync worker: %w", err)
	}

	outboxConfig := OutboxWorkerConfig{
		PollInterval:    cfg.Outbox.PollInterval,
		BatchSize:       cfg.Outbox.BatchSize,
		MaxRetries:      cfg.Outbox.MaxRetries,
		CleanupAge:      cfg.Outbox.CleanupAge,
		CleanupInterval: cfg.Outbox.CleanupInterval,
		Enabled:         cfg.Outbox.Enabled,
	}

	outboxWorker := NewOutboxWorker(
		mongoOutbox,
		eventBusInstance,
		logger,
		outboxConfig,
		outboxMetrics,
	)
	repairWorker := setupRepairWorker(mongoDB, logger)

	logger.InfoContext(ctx, "starting workers",
		slog.Bool("user_sync_enabled", syncConfig.Enabled),
		slog.Duration("user_sync_interval", syncConfig.Interval),
		slog.Bool("outbox_enabled", outboxConfig.Enabled),
		slog.Duration("outbox_poll_interval", outboxConfig.PollInterval),
		slog.Bool("repair_enabled", repairWorker != nil),
	)

	var wg sync.WaitGroup

	wg.Go(func() {
		if runErr := userSyncWorker.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("user sync worker error", slog.String("error", runErr.Error()))
		}
	})

	wg.Go(func() {
		if runErr := outboxWorker.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("outbox worker error", slog.String("error", runErr.Error()))
		}
	})

	wg.Go(func() {
		if runErr := repairWorker.Start(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Error("repair worker error", slog.String("error", runErr.Error()))
		}
	})

	wg.Wait()

	logger.InfoContext(ctx, "worker service shutdown complete")

	return nil
}

func setupUserSyncWorker(
	cfg *config.Config,
	userRepo *mongorepo.MongoUserRepository,
	logger *slog.Logger,
) (*UserSyncWorker, UserSyncConfig, error) {
	syncConfig := DefaultUserSyncConfig()

	if interval := os.Getenv("USER_SYNC_INTERVAL"); interval != "" {
		parsed, parseErr := time.ParseDuration(interval)
		if parseErr != nil {
			logger.Warn("invalid USER_SYNC_INTERVAL, using default interval",
				slog.String("value", interval),
				slog.String("error", parseErr.Error()),
			)
		} else {
			syncConfig.Interval = parsed
		}
	}

	if isEnvBoolTrue("USER_SYNC_DISABLED") {
		syncConfig.Enabled = false

		workerInstance := NewUserSyncWorker(
			nil,
			userRepo,
			logger,
			syncConfig,
		)

		return workerInstance, syncConfig, nil
	}

	if cfg.Keycloak.URL == "" || cfg.Keycloak.AdminUsername == "" {
		return nil, UserSyncConfig{}, errors.New("keycloak configuration is required for user sync worker")
	}

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: cfg.Keycloak.URL,
		Realm:       masterRealm,
		ClientID:    "admin-cli",
		Username:    cfg.Keycloak.AdminUsername,
		Password:    cfg.Keycloak.AdminPassword,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: cfg.Keycloak.URL,
		Realm:       cfg.Keycloak.Realm,
	}, tokenManager)

	workerInstance := NewUserSyncWorker(
		userClient,
		userRepo,
		logger,
		syncConfig,
	)

	return workerInstance, syncConfig, nil
}

func setupRepairWorker(mongoDB *mongo.Database, logger *slog.Logger) *RepairWorker {
	repairConfig := DefaultRepairWorkerConfig()
	if isEnvBoolTrue("REPAIR_WORKER_DISABLED") {
		repairConfig.Enabled = false
	}

	repairQueueColl := mongoDB.Collection(mongodbinfra.CollectionRepairQueue)
	repairQueue := repair.NewMongoQueue(repairQueueColl, logger)

	eventStore := eventstore.NewMongoEventStore(
		mongoDB.Client(),
		mongoDB.Name(),
		eventstore.WithLogger(logger),
	)

	chatReadModelColl := mongoDB.Collection(mongodbinfra.CollectionChatReadModel)
	chatProjector := projector.NewChatProjector(eventStore, chatReadModelColl, logger)

	taskReadModelColl := mongoDB.Collection(mongodbinfra.CollectionTaskReadModel)
	taskProjector := projector.NewChatToTaskReadModelProjector(eventStore, taskReadModelColl, logger)

	return NewRepairWorker(
		repairQueue,
		chatProjector,
		taskProjector,
		logger,
		repairConfig,
	)
}

func isEnvBoolTrue(key string) bool {
	value := os.Getenv(key)
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}

	return enabled
}
