package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/projector"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	// Setup logger first
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Define flags
	aggregateType := flag.String("type", "", "Aggregate type (chat or task)")
	aggregateID := flag.String("id", "", "Aggregate ID (optional, omit for --all)")
	all := flag.Bool("all", false, "Rebuild all aggregates of the type")
	verify := flag.Bool("verify", false, "Verify consistency instead of rebuilding")
	reportFile := flag.String("report", "", "File to write verification report (only with --verify --all)")

	flag.Parse()

	// Validate flags
	if *aggregateType == "" {
		logger.Error("type is required", slog.String("valid_values", "chat or task"))
		flag.Usage()
		os.Exit(1)
	}

	if *aggregateType != "chat" && *aggregateType != "task" {
		logger.Error("invalid type", slog.String("type", *aggregateType), slog.String("valid_values", "chat or task"))
		os.Exit(1)
	}

	if !*all && *aggregateID == "" && !*verify {
		logger.Error("either --id or --all must be specified")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()

	// Connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		logger.Error("failed to connect to MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("failed to disconnect from MongoDB", slog.String("error", disconnectErr.Error()))
		}
	}()

	db := client.Database(cfg.MongoDB.Database)

	// Create event store
	eventStore := eventstore.NewMongoEventStore(
		client,
		cfg.MongoDB.Database,
		eventstore.WithLogger(logger),
	)

	// Create projector based on type
	var proj appcore.ReadModelProjector
	switch *aggregateType {
	case "chat":
		readModelColl := db.Collection(mongodb.CollectionChatReadModel)
		proj = projector.NewChatProjector(eventStore, readModelColl, logger)
	case "task":
		readModelColl := db.Collection(mongodb.CollectionTaskReadModel)
		proj = projector.NewTaskProjector(eventStore, readModelColl, logger)
	}

	// Execute operation
	switch {
	case *verify && *all:
		runVerifyAll(ctx, proj, *aggregateType, *reportFile, logger)
	case *verify:
		runVerifyOne(ctx, proj, *aggregateID, logger)
	case *all:
		runRebuildAll(ctx, proj, *aggregateType, logger)
	default:
		runRebuildOne(ctx, proj, *aggregateID, logger)
	}
}

func runRebuildOne(ctx context.Context, proj appcore.ReadModelProjector, idStr string, logger *slog.Logger) {
	id, parseErr := uuid.ParseUUID(idStr)
	if parseErr != nil {
		logger.ErrorContext(ctx, "invalid aggregate ID",
			slog.String("id", idStr),
			slog.String("error", parseErr.Error()))
		os.Exit(1)
	}

	logger.InfoContext(ctx, "rebuilding read model", slog.String("aggregate_id", id.String()))

	if rebuildErr := proj.RebuildOne(ctx, id); rebuildErr != nil {
		logger.ErrorContext(ctx, "rebuild failed", slog.String("error", rebuildErr.Error()))
		os.Exit(1)
	}

	logger.InfoContext(ctx, "rebuild completed successfully")
}

func runRebuildAll(ctx context.Context, proj appcore.ReadModelProjector, aggregateType string, logger *slog.Logger) {
	logger.InfoContext(ctx, "rebuilding all read models", slog.String("type", aggregateType))

	if rebuildErr := proj.RebuildAll(ctx); rebuildErr != nil {
		logger.ErrorContext(ctx, "rebuild all failed", slog.String("error", rebuildErr.Error()))
		os.Exit(1)
	}

	logger.InfoContext(ctx, "rebuild all completed successfully")
}

func runVerifyOne(ctx context.Context, proj appcore.ReadModelProjector, idStr string, logger *slog.Logger) {
	id, parseErr := uuid.ParseUUID(idStr)
	if parseErr != nil {
		logger.ErrorContext(ctx, "invalid aggregate ID",
			slog.String("id", idStr),
			slog.String("error", parseErr.Error()))
		os.Exit(1)
	}

	logger.InfoContext(ctx, "verifying consistency", slog.String("aggregate_id", id.String()))

	consistent, verifyErr := proj.VerifyConsistency(ctx, id)
	if verifyErr != nil {
		logger.ErrorContext(ctx, "verification failed", slog.String("error", verifyErr.Error()))
		os.Exit(1)
	}

	if consistent {
		logger.InfoContext(ctx, "read model is consistent")
	} else {
		logger.WarnContext(ctx, "read model is INCONSISTENT - rebuild recommended")
		os.Exit(1)
	}
}

func runVerifyAll(
	ctx context.Context,
	_ appcore.ReadModelProjector,
	aggregateType, reportFile string,
	logger *slog.Logger,
) {
	logger.InfoContext(ctx, "verifying all read models", slog.String("type", aggregateType))

	// This is a simplified implementation - in production you'd want to iterate through all aggregates
	// For now, we'll just log that this feature needs full implementation
	logger.WarnContext(ctx, "verify-all is not fully implemented yet - use RebuildAll to fix inconsistencies")

	if reportFile != "" {
		logger.InfoContext(ctx, "report file generation not yet implemented", slog.String("file", reportFile))
	}
}
