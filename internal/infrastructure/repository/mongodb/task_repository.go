package mongodb

import (
	"context"
	"errors"
	"fmt"
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

// MongoTaskRepository реализует taskapp.CommandRepository
type MongoTaskRepository struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
}

// NewMongoTaskRepository создает новый MongoDB Task Repository
func NewMongoTaskRepository(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
) *MongoTaskRepository {
	return &MongoTaskRepository{
		eventStore:    eventStore,
		readModelColl: readModelColl,
	}
}

// Load загружает Task из event store путем восстановления состояния из событий
func (r *MongoTaskRepository) Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error) {
	if taskID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Загружаем события из event store
	events, err := r.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("failed to load events for task %s: %w", taskID, err)
	}

	if len(events) == 0 {
		return nil, errs.ErrNotFound
	}

	// Создаем агрегат и применяем события
	aggregate := taskdomain.NewTaskAggregate(taskID)
	aggregate.ReplayEvents(events)

	// Помечаем события как committed
	aggregate.MarkEventsAsCommitted()

	return aggregate, nil
}

// Save сохраняет новые события Task в event store и обновляет read model
func (r *MongoTaskRepository) Save(ctx context.Context, task *taskdomain.Aggregate) error {
	if task == nil {
		return errs.ErrInvalidInput
	}

	uncommittedEvents := task.UncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil // Нечего сохранять
	}

	// 1. Сохраняем события в event store
	expectedVersion := task.Version() - len(uncommittedEvents)
	err := r.eventStore.SaveEvents(ctx, task.ID().String(), uncommittedEvents, expectedVersion)
	if err != nil {
		if errors.Is(err, appcore.ErrConcurrencyConflict) {
			return errs.ErrConcurrentModification
		}
		return fmt.Errorf("failed to save events: %w", err)
	}

	// 2. Обновляем read model
	if updateErr := r.updateReadModel(ctx, task); updateErr != nil {
		// Логируем ошибку, но не падаем (read model можно пересчитать)
		// TODO: добавить proper logging
		_ = updateErr
	}

	// 3. Помечаем события как committed
	task.MarkEventsAsCommitted()

	return nil
}

// GetEvents возвращает все события задачи
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

// updateReadModel обновляет денормализованное представление в read model
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

// MongoTaskQueryRepository реализует taskapp.QueryRepository
type MongoTaskQueryRepository struct {
	collection *mongo.Collection
	eventStore appcore.EventStore
}

// NewMongoTaskQueryRepository создает новый MongoDB Task Query Repository
func NewMongoTaskQueryRepository(
	collection *mongo.Collection,
	eventStore appcore.EventStore,
) *MongoTaskQueryRepository {
	return &MongoTaskQueryRepository{
		collection: collection,
		eventStore: eventStore,
	}
}

// FindByID находит задачу по ID из read model
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

// FindByChatID находит задачу по ID чата
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

// FindByAssignee находит задачи назначенные пользователю
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

// FindByStatus находит задачи с определенным статусом
func (r *MongoTaskQueryRepository) FindByStatus(
	ctx context.Context,
	status taskdomain.Status,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	filter := bson.M{"status": string(status)}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// List возвращает список задач с фильтрами
func (r *MongoTaskQueryRepository) List(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// Count возвращает количество задач с фильтрами
func (r *MongoTaskQueryRepository) Count(ctx context.Context, filters taskapp.Filters) (int, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "tasks")
	}

	return int(count), nil
}

// applyFilters применяет фильтры к MongoDB запросу
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

// findMany выполняет поиск с пагинацией
func (r *MongoTaskQueryRepository) findMany(
	ctx context.Context,
	filter bson.M,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	// Применяем дефолтный лимит если не указан
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

// taskReadModelDocument структура документа read model
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

// documentToReadModel преобразует документ в ReadModel
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

// MongoTaskFullRepository объединяет Command и Query репозитории
type MongoTaskFullRepository struct {
	*MongoTaskRepository
	*MongoTaskQueryRepository
}

// NewMongoTaskFullRepository создает полный репозиторий
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
