# Task 02: Task Repository с Event Sourcing

## Цель

Создать MongoDB репозиторий для Task агрегата с использованием Event Sourcing, аналогично существующему ChatRepository.

## Контекст

Task — это event-sourced агрегат, уже реализованный в `internal/domain/task/task.go`. Он поддерживает:

- Создание задачи (`Create`)
- Изменение статуса (`ChangeStatus`)
- Назначение исполнителя (`Assign`)
- Изменение приоритета (`ChangePriority`)
- Установка дедлайна (`SetDueDate`)

Все изменения генерируют события, которые нужно сохранять в EventStore.

## Зависимости

### Уже реализовано

- `internal/domain/task/task.go` — агрегат Task
- `internal/domain/task/events.go` — события Task
- `internal/infrastructure/eventstore/mongodb_store.go` — EventStore
- `internal/application/shared/eventstore.go` — интерфейс EventStore

### Требуется создать

1. Интерфейс репозитория в application layer
2. MongoDB реализацию репозитория
3. Read Model для быстрых запросов
4. Тесты

## Детальное описание

### 1. Интерфейс репозитория

Создать `internal/application/task/repository.go`:

```go
package task

import (
    "context"

    "github.com/lllypuk/flowra/internal/domain/event"
    taskdomain "github.com/lllypuk/flowra/internal/domain/task"
    "github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository предоставляет методы для работы с агрегатом Task
// через Event Sourcing (запись)
type CommandRepository interface {
    // Load загружает Task из event store путем восстановления состояния из событий
    Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error)

    // Save сохраняет новые события Task в event store
    Save(ctx context.Context, task *taskdomain.Aggregate) error

    // GetEvents возвращает все события задачи
    GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository предоставляет методы для чтения данных Task
// из read model (денормализованное представление)
type QueryRepository interface {
    // FindByID находит задачу по ID (из read model)
    FindByID(ctx context.Context, taskID uuid.UUID) (*ReadModel, error)

    // FindByChatID находит задачу по ID чата
    FindByChatID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

    // FindByAssignee находит задачи назначенные пользователю
    FindByAssignee(ctx context.Context, assigneeID uuid.UUID, filters Filters) ([]*ReadModel, error)

    // FindByStatus находит задачи с определенным статусом
    FindByStatus(ctx context.Context, status taskdomain.Status, filters Filters) ([]*ReadModel, error)

    // List возвращает список задач с фильтрами
    List(ctx context.Context, filters Filters) ([]*ReadModel, error)

    // Count возвращает количество задач с фильтрами
    Count(ctx context.Context, filters Filters) (int, error)
}

// Repository объединяет Command и Query репозитории
type Repository interface {
    CommandRepository
    QueryRepository
}

// Filters содержит параметры фильтрации для запросов
type Filters struct {
    ChatID     *uuid.UUID
    AssigneeID *uuid.UUID
    Status     *taskdomain.Status
    Priority   *taskdomain.Priority
    EntityType *taskdomain.EntityType
    CreatedBy  *uuid.UUID
    Offset     int
    Limit      int
}

// ReadModel представляет денормализованное представление Task для запросов
type ReadModel struct {
    ID         uuid.UUID
    ChatID     uuid.UUID
    Title      string
    EntityType taskdomain.EntityType
    Status     taskdomain.Status
    Priority   taskdomain.Priority
    AssignedTo *uuid.UUID
    DueDate    *time.Time
    CreatedBy  uuid.UUID
    CreatedAt  time.Time
    Version    int
}
```

### 2. MongoDB реализация

Создать `internal/infrastructure/repository/mongodb/task_repository.go`:

```go
package mongodb

import (
    "context"
    "errors"
    "fmt"
    "time"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    "github.com/lllypuk/flowra/internal/application/shared"
    taskapp "github.com/lllypuk/flowra/internal/application/task"
    "github.com/lllypuk/flowra/internal/domain/errs"
    "github.com/lllypuk/flowra/internal/domain/event"
    taskdomain "github.com/lllypuk/flowra/internal/domain/task"
    "github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoTaskRepository реализует taskapp.CommandRepository
type MongoTaskRepository struct {
    eventStore    shared.EventStore
    readModelColl *mongo.Collection
}

// NewMongoTaskRepository создает новый MongoDB Task Repository
func NewMongoTaskRepository(
    eventStore shared.EventStore,
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
        if errors.Is(err, shared.ErrAggregateNotFound) {
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
        if errors.Is(err, shared.ErrConcurrencyConflict) {
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
        if errors.Is(err, shared.ErrAggregateNotFound) {
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
```

### 3. Query Repository реализация

Добавить в тот же файл или создать отдельный `task_query_repository.go`:

```go
// MongoTaskQueryRepository реализует taskapp.QueryRepository
type MongoTaskQueryRepository struct {
    collection *mongo.Collection
    eventStore shared.EventStore
}

// NewMongoTaskQueryRepository создает новый MongoDB Task Query Repository
func NewMongoTaskQueryRepository(
    collection *mongo.Collection,
    eventStore shared.EventStore,
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
    opts := options.Find().
        SetSort(bson.D{{Key: "created_at", Value: -1}}).
        SetLimit(int64(filters.Limit)).
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
```

### 4. Полный репозиторий (Command + Query)

```go
// MongoTaskFullRepository объединяет Command и Query репозитории
type MongoTaskFullRepository struct {
    *MongoTaskRepository
    *MongoTaskQueryRepository
}

// NewMongoTaskFullRepository создает полный репозиторий
func NewMongoTaskFullRepository(
    eventStore shared.EventStore,
    readModelColl *mongo.Collection,
) *MongoTaskFullRepository {
    return &MongoTaskFullRepository{
        MongoTaskRepository: NewMongoTaskRepository(eventStore, readModelColl),
        MongoTaskQueryRepository: NewMongoTaskQueryRepository(readModelColl, eventStore),
    }
}
```

## Тестирование

### Unit тесты

Создать `internal/infrastructure/repository/mongodb/task_repository_test.go`:

```go
package mongodb_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/shared"
    taskdomain "github.com/lllypuk/flowra/internal/domain/task"
    "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/internal/infrastructure/eventstore"
    "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
    "github.com/lllypuk/flowra/tests/testutil"
)

func TestMongoTaskRepository_Save_And_Load(t *testing.T) {
    // Setup
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    es := eventstore.NewMongoEventStore(client, "test_db")
    readModelColl := db.Collection("task_read_model")

    repo := mongodb.NewMongoTaskRepository(es, readModelColl)

    // Create task aggregate
    taskID := uuid.NewUUID()
    chatID := uuid.NewUUID()
    createdBy := uuid.NewUUID()

    aggregate := taskdomain.NewTaskAggregate(taskID)
    err := aggregate.Create(
        chatID,
        "Test Task",
        taskdomain.EntityTypeTask,
        taskdomain.PriorityMedium,
        nil,
        nil,
        createdBy,
    )
    require.NoError(t, err)

    // Save
    err = repo.Save(ctx, aggregate)
    require.NoError(t, err)

    // Load
    loaded, err := repo.Load(ctx, taskID)
    require.NoError(t, err)

    // Assert
    assert.Equal(t, taskID, loaded.ID())
    assert.Equal(t, chatID, loaded.ChatID())
    assert.Equal(t, "Test Task", loaded.Title())
    assert.Equal(t, taskdomain.EntityTypeTask, loaded.EntityType())
    assert.Equal(t, taskdomain.StatusToDo, loaded.Status())
    assert.Equal(t, taskdomain.PriorityMedium, loaded.Priority())
}

func TestMongoTaskRepository_ConcurrentModification(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    es := eventstore.NewMongoEventStore(client, "test_db")
    readModelColl := db.Collection("task_read_model")

    repo := mongodb.NewMongoTaskRepository(es, readModelColl)

    // Create and save initial task
    taskID := uuid.NewUUID()
    chatID := uuid.NewUUID()
    createdBy := uuid.NewUUID()

    aggregate := taskdomain.NewTaskAggregate(taskID)
    _ = aggregate.Create(chatID, "Test Task", taskdomain.EntityTypeTask, taskdomain.PriorityMedium, nil, nil, createdBy)
    _ = repo.Save(ctx, aggregate)

    // Load two instances
    instance1, _ := repo.Load(ctx, taskID)
    instance2, _ := repo.Load(ctx, taskID)

    // Modify and save first instance
    _ = instance1.ChangeStatus(taskdomain.StatusInProgress, createdBy)
    err1 := repo.Save(ctx, instance1)
    require.NoError(t, err1)

    // Modify and try to save second instance - should fail
    _ = instance2.ChangePriority(taskdomain.PriorityHigh, createdBy)
    err2 := repo.Save(ctx, instance2)

    assert.Error(t, err2)
    assert.ErrorIs(t, err2, errs.ErrConcurrentModification)
}

func TestMongoTaskQueryRepository_FindByAssignee(t *testing.T) {
    ctx := context.Background()
    client, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    db := client.Database("test_db")
    es := eventstore.NewMongoEventStore(client, "test_db")
    readModelColl := db.Collection("task_read_model")

    repo := mongodb.NewMongoTaskFullRepository(es, readModelColl)

    // Create tasks
    assignee := uuid.NewUUID()
    createdBy := uuid.NewUUID()

    for i := 0; i < 3; i++ {
        taskID := uuid.NewUUID()
        chatID := uuid.NewUUID()

        aggregate := taskdomain.NewTaskAggregate(taskID)
        _ = aggregate.Create(chatID, fmt.Sprintf("Task %d", i), taskdomain.EntityTypeTask, taskdomain.PriorityMedium, &assignee, nil, createdBy)
        _ = repo.Save(ctx, aggregate)
    }

    // Query
    results, err := repo.FindByAssignee(ctx, assignee, taskapp.Filters{Limit: 10})
    require.NoError(t, err)

    assert.Len(t, results, 3)
}
```

## Индексы для Read Model

Добавить в `07-mongodb-indexes.md`:

```javascript
// Task Read Model Collection
db.task_read_model.createIndex({ "task_id": 1 }, { unique: true })
db.task_read_model.createIndex({ "chat_id": 1 }, { unique: true })
db.task_read_model.createIndex({ "assigned_to": 1 })
db.task_read_model.createIndex({ "status": 1 })
db.task_read_model.createIndex({ "priority": 1 })
db.task_read_model.createIndex({ "entity_type": 1 })
db.task_read_model.createIndex({ "created_by": 1 })
db.task_read_model.createIndex({ "created_at": -1 })

// Compound indexes for common queries
db.task_read_model.createIndex({ "assigned_to": 1, "status": 1 })
db.task_read_model.createIndex({ "status": 1, "priority": 1 })
```

## Checklist

### Phase 1: Интерфейсы

- [ ] Создать `internal/application/task/repository.go`
- [ ] Определить `CommandRepository` интерфейс
- [ ] Определить `QueryRepository` интерфейс
- [ ] Определить `Filters` структуру
- [ ] Определить `ReadModel` структуру

### Phase 2: Command Repository

- [ ] Создать `MongoTaskRepository` структуру
- [ ] Реализовать `NewMongoTaskRepository`
- [ ] Реализовать `Load` метод
- [ ] Реализовать `Save` метод
- [ ] Реализовать `GetEvents` метод
- [ ] Реализовать `updateReadModel` helper

### Phase 3: Query Repository

- [ ] Создать `MongoTaskQueryRepository` структуру
- [ ] Реализовать `FindByID`
- [ ] Реализовать `FindByChatID`
- [ ] Реализовать `FindByAssignee`
- [ ] Реализовать `FindByStatus`
- [ ] Реализовать `List`
- [ ] Реализовать `Count`
- [ ] Создать `taskReadModelDocument` структуру
- [ ] Реализовать `documentToReadModel`

### Phase 4: Объединенный репозиторий

- [ ] Создать `MongoTaskFullRepository`
- [ ] Проверить, что реализует `taskapp.Repository`

### Phase 5: Тестирование

- [ ] Написать тест `Save_And_Load`
- [ ] Написать тест `ConcurrentModification`
- [ ] Написать тест `FindByAssignee`
- [ ] Написать тест `FindByStatus`
- [ ] Написать тест `List` с фильтрами
- [ ] Достичь coverage > 80%

## Следующие шаги

После завершения этой задачи:

1. **Task 03** — завершение UserRepository
2. **Task 07** — добавление индексов для Task Read Model

## Референсы

- Существующая реализация: `chat_repository.go`
- Task агрегат: `internal/domain/task/task.go`
- Event Store: `internal/infrastructure/eventstore/mongodb_store.go`
