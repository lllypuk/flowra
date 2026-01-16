package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoTaskRepository realizuet taskapp.CommandRepository
type MongoTaskRepository struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
	outbox        appcore.Outbox
	eventBus      event.Bus // deprecated: use outbox for reliable event delivery
	logger        *slog.Logger
}

// TaskRepoOption configures MongoTaskRepository.
type TaskRepoOption func(*MongoTaskRepository)

// WithTaskRepoLogger sets the logger for task repository.
func WithTaskRepoLogger(logger *slog.Logger) TaskRepoOption {
	return func(r *MongoTaskRepository) {
		r.logger = logger
	}
}

// WithTaskRepoEventBus sets the event bus for task repository.
//
// Deprecated: Use WithTaskRepoOutbox for reliable event delivery via outbox pattern.
func WithTaskRepoEventBus(eventBus event.Bus) TaskRepoOption {
	return func(r *MongoTaskRepository) {
		r.eventBus = eventBus
	}
}

// WithTaskRepoOutbox sets the outbox for reliable event delivery.
func WithTaskRepoOutbox(outbox appcore.Outbox) TaskRepoOption {
	return func(r *MongoTaskRepository) {
		r.outbox = outbox
	}
}

// NewMongoTaskRepository creates New MongoDB Task Repository
func NewMongoTaskRepository(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	opts ...TaskRepoOption,
) *MongoTaskRepository {
	r := &MongoTaskRepository{
		eventStore:    eventStore,
		readModelColl: readModelColl,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Load loads Task from event store putem reconstruction state from events
func (r *MongoTaskRepository) Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error) {
	if taskID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Loading event from event store
	events, err := r.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "failed to load task events from event store",
			slog.String("task_id", taskID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to load events for task %s: %w", taskID, err)
	}

	if len(events) == 0 {
		return nil, errs.ErrNotFound
	}

	// Creating aggregate and primenyaem event
	aggregate := taskdomain.NewTaskAggregate(taskID)
	aggregate.ReplayEvents(events)

	// pomechaem event as committed
	aggregate.MarkEventsAsCommitted()

	return aggregate, nil
}

// Save saves novye event Task in event store and obnovlyaet read model
func (r *MongoTaskRepository) Save(ctx context.Context, task *taskdomain.Aggregate) error {
	if task == nil {
		return errs.ErrInvalidInput
	}

	uncommittedEvents := task.UncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil // nechego sav
	}

	// 1. Saving event in event store
	expectedVersion := task.Version() - len(uncommittedEvents)
	err := r.eventStore.SaveEvents(ctx, task.ID().String(), uncommittedEvents, expectedVersion)
	if err != nil {
		if errors.Is(err, appcore.ErrConcurrencyConflict) {
			r.logger.WarnContext(ctx, "concurrency conflict while saving task events",
				slog.String("task_id", task.ID().String()),
				slog.Int("expected_version", expectedVersion),
				slog.Int("events_count", len(uncommittedEvents)),
			)
			return errs.ErrConcurrentModification
		}
		r.logger.ErrorContext(ctx, "failed to save task events to event store",
			slog.String("task_id", task.ID().String()),
			slog.Int("events_count", len(uncommittedEvents)),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to save events: %w", err)
	}

	// 2. Updating read model
	if updateErr := r.updateReadModel(ctx, task); updateErr != nil {
		r.logger.ErrorContext(ctx, "failed to update task read model",
			slog.String("task_id", task.ID().String()),
			slog.String("error", updateErr.Error()),
		)
		// Don't fail - read model can be recalculated
	}

	// 3. Write events to outbox for reliable delivery (preferred)
	if r.outbox != nil {
		if outboxErr := r.outbox.AddBatch(ctx, uncommittedEvents); outboxErr != nil {
			r.logger.ErrorContext(ctx, "failed to add events to outbox",
				slog.String("task_id", task.ID().String()),
				slog.Int("events_count", len(uncommittedEvents)),
				slog.String("error", outboxErr.Error()),
			)
			// Don't fail - events are saved in event store
		}
	} else if r.eventBus != nil {
		// Fallback: direct publish to EventBus (deprecated, less reliable)
		for _, evt := range uncommittedEvents {
			if pubErr := r.eventBus.Publish(ctx, evt); pubErr != nil {
				r.logger.WarnContext(ctx, "failed to publish task event to bus",
					slog.String("task_id", task.ID().String()),
					slog.String("event_type", evt.EventType()),
					slog.String("error", pubErr.Error()),
				)
				// Don't fail - event is already persisted
			}
		}
	}

	// 4. Mark events as committed
	task.MarkEventsAsCommitted()

	return nil
}

// GetEvents returns all event tasks
func (r *MongoTaskRepository) GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error) {
	if taskID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	events, err := r.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	return events, nil
}

// updateReadModel obnovlyaet denormalizovannoe view in read model
func (r *MongoTaskRepository) updateReadModel(ctx context.Context, task *taskdomain.Aggregate) error {
	if task.ID().IsZero() {
		return errs.ErrInvalidInput
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

	_, err := r.readModelColl.UpdateOne(ctx, filter, update, opts)
	return HandleMongoError(err, "task_read_model")
}

// MongoTaskQueryRepository realizuet taskapp.QueryRepository
type MongoTaskQueryRepository struct {
	collection *mongo.Collection
	eventStore appcore.EventStore
}

// NewMongoTaskQueryRepository creates New MongoDB Task Query Repository
func NewMongoTaskQueryRepository(
	collection *mongo.Collection,
	eventStore appcore.EventStore,
) *MongoTaskQueryRepository {
	return &MongoTaskQueryRepository{
		collection: collection,
		eventStore: eventStore,
	}
}

// FindByID finds zadachu po ID from read model
func (r *MongoTaskQueryRepository) FindByID(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error) {
	if taskID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"task_id": taskID.String()}
	var doc taskReadModelDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "task")
	}

	return r.documentToReadModel(&doc)
}

// FindByChatID finds zadachu po ID chat
func (r *MongoTaskQueryRepository) FindByChatID(ctx context.Context, chatID uuid.UUID) (*taskapp.ReadModel, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"chat_id": chatID.String()}
	var doc taskReadModelDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "task")
	}

	return r.documentToReadModel(&doc)
}

// FindByAssignee finds tasks value user
func (r *MongoTaskQueryRepository) FindByAssignee(
	ctx context.Context,
	assigneeID uuid.UUID,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	if assigneeID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"assigned_to": assigneeID.String()}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// FindByStatus finds tasks s opredelennym statusom
func (r *MongoTaskQueryRepository) FindByStatus(
	ctx context.Context,
	status taskdomain.Status,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	filter := bson.M{"status": string(status)}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// List returns list zadach s filtrami
func (r *MongoTaskQueryRepository) List(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// Count returns count zadach s filtrami
func (r *MongoTaskQueryRepository) Count(ctx context.Context, filters taskapp.Filters) (int, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "tasks")
	}

	return int(count), nil
}

// applyFilters primenyaet filters to MongoDB query
func (r *MongoTaskQueryRepository) applyFilters(filter bson.M, filters taskapp.Filters) {
	if filters.ChatID != nil {
		filter["chat_id"] = filters.ChatID.String()
	}
	if filters.AssigneeID != nil {
		filter["assigned_to"] = filters.AssigneeID.String()
	}
	if filters.Status != nil {
		filter["status"] = string(*filters.Status)
	}
	if filters.Priority != nil {
		filter["priority"] = string(*filters.Priority)
	}
	if filters.EntityType != nil {
		filter["entity_type"] = string(*filters.EntityType)
	}
	if filters.CreatedBy != nil {
		filter["created_by"] = filters.CreatedBy.String()
	}
}

// findMany performs search s paginatsiey
func (r *MongoTaskQueryRepository) findMany(
	ctx context.Context,
	filter bson.M,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	// primenyaem defoltnyy limit if not ukazan
	limit := DefaultLimitWithMax(filters.Limit, DefaultPaginationLimit, MaxPaginationLimit)

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(filters.Offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "tasks")
	}
	defer cursor.Close(ctx)

	var results []*taskapp.ReadModel
	for cursor.Next(ctx) {
		var doc taskReadModelDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		rm, docErr := r.documentToReadModel(&doc)
		if docErr != nil {
			continue
		}

		results = append(results, rm)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if results == nil {
		results = make([]*taskapp.ReadModel, 0)
	}

	return results, nil
}

// taskReadModelDocument struct dokumenta read model
type taskReadModelDocument struct {
	TaskID     string     `bson:"task_id"`
	ChatID     string     `bson:"chat_id"`
	Title      string     `bson:"title"`
	EntityType string     `bson:"entity_type"`
	Status     string     `bson:"status"`
	Priority   string     `bson:"priority"`
	AssignedTo *string    `bson:"assigned_to,omitempty"`
	DueDate    *time.Time `bson:"due_date,omitempty"`
	CreatedBy  string     `bson:"created_by"`
	CreatedAt  time.Time  `bson:"created_at"`
	Version    int        `bson:"version"`
}

// documentToReadModel preobrazuet dokument in ReadModel
func (r *MongoTaskQueryRepository) documentToReadModel(doc *taskReadModelDocument) (*taskapp.ReadModel, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	rm := &taskapp.ReadModel{
		ID:         uuid.UUID(doc.TaskID),
		ChatID:     uuid.UUID(doc.ChatID),
		Title:      doc.Title,
		EntityType: taskdomain.EntityType(doc.EntityType),
		Status:     taskdomain.Status(doc.Status),
		Priority:   taskdomain.Priority(doc.Priority),
		CreatedBy:  uuid.UUID(doc.CreatedBy),
		CreatedAt:  doc.CreatedAt,
		Version:    doc.Version,
	}

	if doc.AssignedTo != nil {
		assignee := uuid.UUID(*doc.AssignedTo)
		rm.AssignedTo = &assignee
	}

	if doc.DueDate != nil {
		rm.DueDate = doc.DueDate
	}

	return rm, nil
}

// MongoTaskFullRepository combines Command and Query repozitorii
type MongoTaskFullRepository struct {
	*MongoTaskRepository
	*MongoTaskQueryRepository
}

// NewMongoTaskFullRepository creates full repozitoriy
func NewMongoTaskFullRepository(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
) *MongoTaskFullRepository {
	return &MongoTaskFullRepository{
		MongoTaskRepository:      NewMongoTaskRepository(eventStore, readModelColl),
		MongoTaskQueryRepository: NewMongoTaskQueryRepository(readModelColl, eventStore),
	}
}

// Compile-time interface checks
var (
	_ taskapp.CommandRepository = (*MongoTaskRepository)(nil)
	_ taskapp.QueryRepository   = (*MongoTaskQueryRepository)(nil)
	_ taskapp.Repository        = (*MongoTaskFullRepository)(nil)
)
