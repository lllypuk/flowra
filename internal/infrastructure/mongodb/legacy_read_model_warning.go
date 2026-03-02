package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	legacyChatReadModelCollection = "chat_read_model"
	legacyTaskReadModelCollection = "task_read_model"
)

// WarnIfLegacyReadModelCollectionsContainData logs warnings when legacy read-model
// collections still contain data from pre-Chat=SoT schema names.
func WarnIfLegacyReadModelCollectionsContainData(
	ctx context.Context,
	db *mongo.Database,
	logger *slog.Logger,
) error {
	if db == nil {
		return errors.New("database is nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	legacyCollections := []string{
		legacyChatReadModelCollection,
		legacyTaskReadModelCollection,
	}

	for _, collectionName := range legacyCollections {
		count, err := db.Collection(collectionName).CountDocuments(ctx, bson.M{})
		if err != nil {
			return fmt.Errorf("failed to inspect collection %s: %w", collectionName, err)
		}

		if count > 0 {
			logger.WarnContext(ctx, "legacy read model collection contains data",
				slog.String("collection", collectionName),
				slog.Int64("documents", count),
				slog.String("guidance", "run make reset-data"),
			)
		}
	}

	return nil
}
