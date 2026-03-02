package projector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	aggregateTypeChat = "chat"
	aggregateTypeTask = "task"
	chatEventPrefix   = "chat."
)

func isAggregateType(value, expected string) bool {
	return strings.EqualFold(strings.TrimSpace(value), expected)
}

func getAllAggregateIDsByType(
	ctx context.Context,
	readModelColl *mongo.Collection,
	aggregateTypeLower, aggregateTypeTitle string,
	logger *slog.Logger,
) ([]uuid.UUID, error) {
	eventsColl := readModelColl.Database().Collection("events")
	filter := bson.M{"aggregate_type": bson.M{"$in": []string{aggregateTypeLower, aggregateTypeTitle}}}
	result := eventsColl.Distinct(ctx, "aggregate_id", filter)

	var stringIDs []string
	if err := result.Decode(&stringIDs); err != nil {
		return nil, fmt.Errorf("failed to decode aggregate IDs: %w", err)
	}

	aggregateIDs := make([]uuid.UUID, 0, len(stringIDs))
	for _, idStr := range stringIDs {
		id, err := uuid.ParseUUID(idStr)
		if err != nil {
			logger.WarnContext(ctx, "skipping invalid aggregate ID",
				slog.String("aggregate_id", idStr),
				slog.String("error", err.Error()),
			)
			continue
		}
		aggregateIDs = append(aggregateIDs, id)
	}

	return aggregateIDs, nil
}
