package projector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ChatProjector rebuilds chat read models from event store.
type ChatProjector struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
	logger        *slog.Logger
}

// NewChatProjector creates a new chat projector.
func NewChatProjector(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	logger *slog.Logger,
) *ChatProjector {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChatProjector{
		eventStore:    eventStore,
		readModelColl: readModelColl,
		logger:        logger,
	}
}

// RebuildOne rebuilds read model for a single chat from its events.
func (p *ChatProjector) RebuildOne(ctx context.Context, chatID uuid.UUID) error {
	p.logger.InfoContext(ctx, "rebuilding chat read model",
		slog.String("chat_id", chatID.String()),
	)

	// Load all events from event store
	events, err := p.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return fmt.Errorf("failed to load events for chat %s: %w", chatID, err)
	}

	if len(events) == 0 {
		return appcore.ErrAggregateNotFound
	}

	// Reconstruct aggregate from events
	chat := chatdomain.NewEmptyChat()
	for _, evt := range events {
		if applyErr := chat.Apply(evt); applyErr != nil {
			return fmt.Errorf("failed to apply event %s: %w", evt.EventType(), applyErr)
		}
	}

	// Update read model with reconstructed state
	if updateErr := p.updateReadModel(ctx, chat); updateErr != nil {
		return fmt.Errorf("failed to update read model: %w", updateErr)
	}

	p.logger.InfoContext(ctx, "successfully rebuilt chat read model",
		slog.String("chat_id", chatID.String()),
		slog.Int("events_applied", len(events)),
		slog.Int("version", chat.Version()),
	)

	return nil
}

// RebuildAll rebuilds read models for all chats.
func (p *ChatProjector) RebuildAll(ctx context.Context) error {
	p.logger.InfoContext(ctx, "starting rebuild of all chat read models")

	// Get all unique chat IDs from events collection
	aggregateIDs, err := p.getAllAggregateIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get aggregate IDs: %w", err)
	}

	successCount := 0
	failCount := 0

	for _, id := range aggregateIDs {
		if rebuildErr := p.RebuildOne(ctx, id); rebuildErr != nil {
			p.logger.ErrorContext(ctx, "failed to rebuild chat",
				slog.String("chat_id", id.String()),
				slog.String("error", rebuildErr.Error()),
			)
			failCount++
			continue
		}
		successCount++
	}

	p.logger.InfoContext(ctx, "completed rebuild of all chat read models",
		slog.Int("total", len(aggregateIDs)),
		slog.Int("success", successCount),
		slog.Int("failed", failCount),
	)

	if failCount > 0 {
		return fmt.Errorf("rebuild completed with %d failures out of %d total", failCount, len(aggregateIDs))
	}

	return nil
}

// ProcessEvent applies a single event to the read model.
func (p *ChatProjector) ProcessEvent(ctx context.Context, evt event.DomainEvent) error {
	// Check if this is a chat event
	if evt.AggregateType() != "chat" {
		return fmt.Errorf("invalid aggregate type: expected 'chat', got '%s'", evt.AggregateType())
	}

	chatID, err := uuid.ParseUUID(evt.AggregateID())
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Rebuild the entire read model from events
	// This ensures consistency even if some events were missed
	return p.RebuildOne(ctx, chatID)
}

// VerifyConsistency checks if read model matches the state derived from events.
func (p *ChatProjector) VerifyConsistency(ctx context.Context, chatID uuid.UUID) (bool, error) {
	p.logger.InfoContext(ctx, "verifying chat read model consistency",
		slog.String("chat_id", chatID.String()),
	)

	// Load all events and reconstruct aggregate
	events, err := p.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return false, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		// Check if read model exists
		filter := bson.M{"chat_id": chatID.String()}
		count, countErr := p.readModelColl.CountDocuments(ctx, filter)
		if countErr != nil {
			return false, fmt.Errorf("failed to count read model documents: %w", countErr)
		}
		// Both should not exist - consistent
		return count == 0, nil
	}

	// Reconstruct expected state
	expectedChat := chatdomain.NewEmptyChat()
	for _, evt := range events {
		if applyErr := expectedChat.Apply(evt); applyErr != nil {
			return false, fmt.Errorf("failed to apply event: %w", applyErr)
		}
	}

	// Load actual read model
	filter := bson.M{"chat_id": chatID.String()}
	var actualDoc bson.M
	err = p.readModelColl.FindOne(ctx, filter).Decode(&actualDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			p.logger.WarnContext(ctx, "read model missing for chat with events",
				slog.String("chat_id", chatID.String()),
				slog.Int("events_count", len(events)),
			)
			return false, nil
		}
		return false, fmt.Errorf("failed to load read model: %w", err)
	}

	// Compare key fields
	consistent := actualDoc["chat_id"] == chatID.String()

	if actualDoc["workspace_id"] != expectedChat.WorkspaceID().String() {
		consistent = false
	}
	if actualDoc["type"] != string(expectedChat.Type()) {
		consistent = false
	}
	if actualDoc["title"] != expectedChat.Title() {
		consistent = false
	}
	if actualDoc["is_public"] != expectedChat.IsPublic() {
		consistent = false
	}

	if !consistent {
		p.logger.WarnContext(ctx, "read model inconsistency detected",
			slog.String("chat_id", chatID.String()),
			slog.String("expected_type", string(expectedChat.Type())),
			slog.Any("actual_type", actualDoc["type"]),
		)
	}

	return consistent, nil
}

// updateReadModel updates the read model for a chat.
func (p *ChatProjector) updateReadModel(ctx context.Context, chat *chatdomain.Chat) error {
	if chat.ID().IsZero() {
		return errors.New("invalid chat ID")
	}

	// Convert participants to strings
	participantStrs := make([]string, len(chat.Participants()))
	for i, p := range chat.Participants() {
		participantStrs[i] = p.UserID().String()
	}

	// Build read model document
	doc := bson.M{
		"chat_id":      chat.ID().String(),
		"workspace_id": chat.WorkspaceID().String(),
		"type":         string(chat.Type()),
		"title":        chat.Title(),
		"is_public":    chat.IsPublic(),
		"created_by":   chat.CreatedBy().String(),
		"created_at":   chat.CreatedAt(),
		"participants": participantStrs,
	}

	// Add additional fields for typed chats (task/bug/epic)
	if chat.Type() != chatdomain.TypeDiscussion {
		doc["status"] = chat.Status()
		doc["priority"] = chat.Priority()

		if chat.AssigneeID() != nil {
			doc["assigned_to"] = chat.AssigneeID().String()
		}

		if chat.DueDate() != nil {
			doc["due_date"] = *chat.DueDate()
		}

		if chat.Type() == chatdomain.TypeBug {
			doc["severity"] = chat.Severity()
		}
	}

	// Upsert the document
	filter := bson.M{"chat_id": chat.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := p.readModelColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert read model: %w", err)
	}

	return nil
}

// getAllAggregateIDs retrieves all unique chat IDs from the events collection.
func (p *ChatProjector) getAllAggregateIDs(ctx context.Context) ([]uuid.UUID, error) {
	// Get the events collection from the database
	eventsDB := p.readModelColl.Database()
	eventsColl := eventsDB.Collection("events")

	// Use distinct to get unique aggregate IDs for chat type
	filter := bson.M{"aggregate_type": "chat"}
	result := eventsColl.Distinct(ctx, "aggregate_id", filter)

	var aggregateIDs []uuid.UUID
	var stringIDs []string
	if err := result.Decode(&stringIDs); err != nil {
		return nil, fmt.Errorf("failed to decode aggregate IDs: %w", err)
	}

	for _, idStr := range stringIDs {
		id, err := uuid.ParseUUID(idStr)
		if err != nil {
			p.logger.WarnContext(ctx, "skipping invalid aggregate ID",
				slog.String("aggregate_id", idStr),
				slog.String("error", err.Error()),
			)
			continue
		}
		aggregateIDs = append(aggregateIDs, id)
	}

	return aggregateIDs, nil
}
