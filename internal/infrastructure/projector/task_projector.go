package projector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TaskProjector rebuilds task read models from event store.
type TaskProjector struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
	logger        *slog.Logger
}

// NewTaskProjector creates a new task projector.
func NewTaskProjector(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	logger *slog.Logger,
) *TaskProjector {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskProjector{
		eventStore:    eventStore,
		readModelColl: readModelColl,
		logger:        logger,
	}
}

// RebuildOne rebuilds read model for a single task from its events.
func (p *TaskProjector) RebuildOne(ctx context.Context, taskID uuid.UUID) error {
	p.logger.InfoContext(ctx, "rebuilding task read model",
		slog.String("task_id", taskID.String()),
	)

	// Load all events from event store
	events, err := p.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		return fmt.Errorf("failed to load events for task %s: %w", taskID, err)
	}

	if len(events) == 0 {
		return appcore.ErrAggregateNotFound
	}

	// Reconstruct aggregate from events
	task := taskdomain.NewTaskAggregate(taskID)
	task.ReplayEvents(events)

	// Update read model with reconstructed state
	if updateErr := p.updateReadModel(ctx, task); updateErr != nil {
		return fmt.Errorf("failed to update read model: %w", updateErr)
	}

	p.logger.InfoContext(ctx, "successfully rebuilt task read model",
		slog.String("task_id", taskID.String()),
		slog.Int("events_applied", len(events)),
		slog.Int("version", task.Version()),
	)

	return nil
}

// RebuildAll rebuilds read models for all tasks.
func (p *TaskProjector) RebuildAll(ctx context.Context) error {
	p.logger.InfoContext(ctx, "starting rebuild of all task read models")

	// Get all unique task IDs from events collection
	aggregateIDs, err := p.getAllAggregateIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get aggregate IDs: %w", err)
	}

	successCount := 0
	failCount := 0

	for _, id := range aggregateIDs {
		if rebuildErr := p.RebuildOne(ctx, id); rebuildErr != nil {
			p.logger.ErrorContext(ctx, "failed to rebuild task",
				slog.String("task_id", id.String()),
				slog.String("error", rebuildErr.Error()),
			)
			failCount++
			continue
		}
		successCount++
	}

	p.logger.InfoContext(ctx, "completed rebuild of all task read models",
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
func (p *TaskProjector) ProcessEvent(ctx context.Context, evt event.DomainEvent) error {
	// Check if this is a task event
	if evt.AggregateType() != "task" {
		return fmt.Errorf("invalid aggregate type: expected 'task', got '%s'", evt.AggregateType())
	}

	taskID, err := uuid.ParseUUID(evt.AggregateID())
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Rebuild the entire read model from events
	// This ensures consistency even if some events were missed
	return p.RebuildOne(ctx, taskID)
}

// VerifyConsistency checks if read model matches the state derived from events.
func (p *TaskProjector) VerifyConsistency(ctx context.Context, taskID uuid.UUID) (bool, error) {
	p.logger.InfoContext(ctx, "verifying task read model consistency",
		slog.String("task_id", taskID.String()),
	)

	// Load all events and reconstruct aggregate
	events, err := p.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		return false, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		// Check if read model exists
		filter := bson.M{"task_id": taskID.String()}
		count, countErr := p.readModelColl.CountDocuments(ctx, filter)
		if countErr != nil {
			return false, fmt.Errorf("failed to count read model documents: %w", countErr)
		}
		// Both should not exist - consistent
		return count == 0, nil
	}

	// Reconstruct expected state
	expectedTask := taskdomain.NewTaskAggregate(taskID)
	expectedTask.ReplayEvents(events)

	// Load actual read model
	filter := bson.M{"task_id": taskID.String()}
	var actualDoc bson.M
	err = p.readModelColl.FindOne(ctx, filter).Decode(&actualDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			p.logger.WarnContext(ctx, "read model missing for task with events",
				slog.String("task_id", taskID.String()),
				slog.Int("events_count", len(events)),
			)
			return false, nil
		}
		return false, fmt.Errorf("failed to load read model: %w", err)
	}

	// Compare key fields
	consistent := actualDoc["task_id"] == taskID.String()

	if actualDoc["chat_id"] != expectedTask.ChatID().String() {
		consistent = false
	}
	if actualDoc["title"] != expectedTask.Title() {
		consistent = false
	}
	if actualDoc["status"] != string(expectedTask.Status()) {
		consistent = false
	}
	if actualDoc["priority"] != string(expectedTask.Priority()) {
		consistent = false
	}

	if !consistent {
		p.logger.WarnContext(ctx, "read model inconsistency detected",
			slog.String("task_id", taskID.String()),
			slog.String("expected_status", string(expectedTask.Status())),
			slog.Any("actual_status", actualDoc["status"]),
		)
	}

	return consistent, nil
}

// updateReadModel updates the read model for a task.
func (p *TaskProjector) updateReadModel(ctx context.Context, task *taskdomain.Aggregate) error {
	if task.ID().IsZero() {
		return errors.New("invalid task ID")
	}

	doc := bson.M{
		"task_id":     task.ID().String(),
		"chat_id":     task.ChatID().String(),
		"title":       task.Title(),
		"entity_type": string(task.EntityType()),
		"status":      string(task.Status()),
		"priority":    string(task.Priority()),
		"created_by":  task.CreatedBy().String(),
		"created_at":  task.CreatedAt(),
		"version":     task.Version(),
	}

	if task.AssignedTo() != nil {
		doc["assigned_to"] = task.AssignedTo().String()
	} else {
		doc["assigned_to"] = nil
	}

	if task.DueDate() != nil {
		doc["due_date"] = *task.DueDate()
	} else {
		doc["due_date"] = nil
	}

	filter := bson.M{"task_id": task.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := p.readModelColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert read model: %w", err)
	}

	return nil
}

// getAllAggregateIDs retrieves all unique task IDs from the events collection.
func (p *TaskProjector) getAllAggregateIDs(ctx context.Context) ([]uuid.UUID, error) {
	// Get the events collection from the database
	eventsDB := p.readModelColl.Database()
	eventsColl := eventsDB.Collection("events")

	// Use distinct to get unique aggregate IDs for task type
	filter := bson.M{"aggregate_type": "task"}
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
