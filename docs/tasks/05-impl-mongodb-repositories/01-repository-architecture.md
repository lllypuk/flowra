# Task 01: Repository Architecture

## Цель

Определить архитектуру MongoDB репозиториев, паттерны работы и общие компоненты для переиспользования.

## Контекст

Проект использует два типа репозиториев:

1. **Event-Sourced** — для агрегатов Chat и Task, где состояние восстанавливается из событий
2. **CRUD** — для простых сущностей (User, Workspace, Message, Notification)

Все репозитории следуют принципу "Accept Interfaces, Return Structs" — интерфейсы объявляются на стороне потребителя (application layer).

## Архитектурные решения

### 1. Структура директорий

```
internal/
├── application/                          # Потребители (объявляют интерфейсы)
│   ├── shared/
│   │   └── eventstore.go                 # EventStore interface
│   ├── chat/
│   │   └── repository.go                 # Chat repository interfaces
│   ├── task/
│   │   └── repository.go                 # Task repository interfaces (создать!)
│   ├── user/
│   │   └── repository.go                 # User repository interfaces
│   ├── workspace/
│   │   └── repository.go                 # Workspace repository interfaces
│   ├── message/
│   │   └── repository.go                 # Message repository interfaces
│   └── notification/
│       └── repository.go                 # Notification repository interfaces
│
└── infrastructure/
    ├── eventstore/
    │   ├── mongodb_store.go              # EventStore implementation
    │   └── serializer.go                 # Event serialization
    │
    └── repository/
        └── mongodb/
            ├── common.go                 # Общие функции
            ├── list_helper.go            # Пагинация helper
            ├── chat_repository.go        # Event-sourced
            ├── task_repository.go        # Event-sourced (создать!)
            ├── user_repository.go        # CRUD
            ├── workspace_repository.go   # CRUD
            ├── message_repository.go     # CRUD
            └── notification_repository.go # CRUD
```

### 2. Общие компоненты (common.go)

Файл `common.go` уже содержит базовые функции. Нужно расширить:

```go
// internal/infrastructure/repository/mongodb/common.go
package mongodb

import (
    "context"
    "errors"
    "fmt"
    "time"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    "github.com/lllypuk/flowra/internal/domain/errs"
)

// HandleMongoError преобразует ошибку MongoDB в доменную ошибку
func HandleMongoError(err error, resourceType string) error {
    if err == nil {
        return nil
    }

    if errors.Is(err, mongo.ErrNoDocuments) {
        return errs.ErrNotFound
    }

    if mongo.IsDuplicateKeyError(err) {
        return errs.ErrAlreadyExists
    }

    return fmt.Errorf("failed to operate on %s: %w", resourceType, err)
}

// BaseDocument содержит общие поля для всех документов
type BaseDocument struct {
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}

// TouchUpdatedAt обновляет поле updated_at
func (d *BaseDocument) TouchUpdatedAt() {
    d.UpdatedAt = time.Now().UTC()
}

// SetCreatedAt устанавливает created_at если не установлен
func (d *BaseDocument) SetCreatedAt() {
    if d.CreatedAt.IsZero() {
        d.CreatedAt = time.Now().UTC()
    }
}

// UpsertOptions возвращает стандартные опции для upsert операции
func UpsertOptions() *options.UpdateOneOptionsBuilder {
    return options.UpdateOne().SetUpsert(true)
}

// FindWithPagination возвращает опции для find с пагинацией
func FindWithPagination(offset, limit int, sortField string, sortOrder int) *options.FindOptionsBuilder {
    return options.Find().
        SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
        SetLimit(int64(limit)).
        SetSkip(int64(offset))
}

// CountFilter выполняет подсчет документов с фильтром
func CountFilter(ctx context.Context, coll *mongo.Collection, filter bson.M) (int, error) {
    count, err := coll.CountDocuments(ctx, filter)
    if err != nil {
        return 0, err
    }
    return int(count), nil
}
```

### 3. Паттерн Event-Sourced Repository

Для Chat и Task используется следующий паттерн:

```go
// Структура репозитория
type MongoTaskRepository struct {
    eventStore    shared.EventStore      // Для хранения событий
    readModelColl *mongo.Collection      // Для read model (денормализованные данные)
}

// Load — восстановление агрегата из событий
func (r *MongoTaskRepository) Load(ctx context.Context, taskID uuid.UUID) (*task.Aggregate, error) {
    // 1. Загрузить события из EventStore
    events, err := r.eventStore.LoadEvents(ctx, taskID.String())
    if err != nil {
        if errors.Is(err, shared.ErrAggregateNotFound) {
            return nil, errs.ErrNotFound
        }
        return nil, err
    }

    // 2. Создать пустой агрегат
    aggregate := task.NewTaskAggregate(taskID)

    // 3. Воспроизвести события
    aggregate.ReplayEvents(events)

    // 4. Пометить события как committed
    aggregate.MarkEventsAsCommitted()

    return aggregate, nil
}

// Save — сохранение новых событий и обновление read model
func (r *MongoTaskRepository) Save(ctx context.Context, aggregate *task.Aggregate) error {
    uncommittedEvents := aggregate.UncommittedEvents()
    if len(uncommittedEvents) == 0 {
        return nil
    }

    // 1. Сохранить события в EventStore
    expectedVersion := aggregate.Version() - len(uncommittedEvents)
    err := r.eventStore.SaveEvents(ctx, aggregate.ID().String(), uncommittedEvents, expectedVersion)
    if err != nil {
        if errors.Is(err, shared.ErrConcurrencyConflict) {
            return errs.ErrConcurrentModification
        }
        return err
    }

    // 2. Обновить read model (денормализованное представление)
    _ = r.updateReadModel(ctx, aggregate)

    // 3. Пометить события как committed
    aggregate.MarkEventsAsCommitted()

    return nil
}
```

### 4. Паттерн CRUD Repository

Для простых сущностей:

```go
type MongoUserRepository struct {
    collection *mongo.Collection
}

// Save — сохранение через upsert
func (r *MongoUserRepository) Save(ctx context.Context, user *userdomain.User) error {
    if user == nil || user.ID().IsZero() {
        return errs.ErrInvalidInput
    }

    doc := r.userToDocument(user)
    filter := bson.M{"user_id": user.ID().String()}
    update := bson.M{"$set": doc}

    _, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
    return HandleMongoError(err, "user")
}

// FindByID — поиск по ID
func (r *MongoUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
    if id.IsZero() {
        return nil, errs.ErrInvalidInput
    }

    filter := bson.M{"user_id": id.String()}
    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        return nil, HandleMongoError(err, "user")
    }

    return r.documentToUser(&doc)
}
```

### 5. Структуры документов

Каждый репозиторий определяет свою структуру документа:

```go
// userDocument — структура MongoDB документа
type userDocument struct {
    UserID        string    `bson:"user_id"`        // Primary key
    KeycloakID    *string   `bson:"keycloak_id,omitempty"`
    Username      string    `bson:"username"`
    Email         string    `bson:"email"`
    DisplayName   string    `bson:"display_name"`
    IsSystemAdmin bool      `bson:"is_system_admin"`
    CreatedAt     time.Time `bson:"created_at"`
    UpdatedAt     time.Time `bson:"updated_at"`
}
```

**Важные правила:**

1. UUID хранится как `string` (не binary) для удобства отладки
2. Nullable поля используют указатели (`*string`, `*time.Time`)
3. Временные метки всегда в UTC
4. `bson` теги обязательны для всех полей

### 6. Преобразование Document ↔ Domain

```go
// userToDocument — Domain → Document
func (r *MongoUserRepository) userToDocument(user *userdomain.User) userDocument {
    doc := userDocument{
        UserID:        user.ID().String(),
        Username:      user.Username(),
        Email:         user.Email(),
        DisplayName:   user.DisplayName(),
        IsSystemAdmin: user.IsSystemAdmin(),
        CreatedAt:     user.CreatedAt(),
        UpdatedAt:     user.UpdatedAt(),
    }

    if externalID := user.ExternalID(); externalID != "" {
        doc.KeycloakID = &externalID
    }

    return doc
}

// documentToUser — Document → Domain
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
    if doc == nil {
        return nil, errs.ErrInvalidInput
    }

    // Используем Restore функцию из domain (нужно создать!)
    return userdomain.Restore(
        uuid.UUID(doc.UserID),
        doc.KeycloakID,
        doc.Username,
        doc.Email,
        doc.DisplayName,
        doc.IsSystemAdmin,
        doc.CreatedAt,
        doc.UpdatedAt,
    )
}
```

### 7. Read Model для Event-Sourced агрегатов

Read Model — денормализованное представление для быстрых запросов:

```go
// taskReadModelDocument — структура read model для Task
type taskReadModelDocument struct {
    TaskID      string     `bson:"task_id"`
    ChatID      string     `bson:"chat_id"`
    Title       string     `bson:"title"`
    EntityType  string     `bson:"entity_type"`
    Status      string     `bson:"status"`
    Priority    string     `bson:"priority"`
    AssignedTo  *string    `bson:"assigned_to,omitempty"`
    DueDate     *time.Time `bson:"due_date,omitempty"`
    CreatedBy   string     `bson:"created_by"`
    CreatedAt   time.Time  `bson:"created_at"`
    Version     int        `bson:"version"`
}
```

Read Model обновляется при каждом Save:

```go
func (r *MongoTaskRepository) updateReadModel(ctx context.Context, agg *task.Aggregate) error {
    doc := bson.M{
        "task_id":     agg.ID().String(),
        "chat_id":     agg.ChatID().String(),
        "title":       agg.Title(),
        "entity_type": string(agg.EntityType()),
        "status":      string(agg.Status()),
        "priority":    string(agg.Priority()),
        "created_by":  agg.CreatedBy().String(),
        "created_at":  agg.CreatedAt(),
        "version":     agg.Version(),
    }

    if agg.AssignedTo() != nil {
        doc["assigned_to"] = agg.AssignedTo().String()
    }

    if agg.DueDate() != nil {
        doc["due_date"] = *agg.DueDate()
    }

    filter := bson.M{"task_id": agg.ID().String()}
    update := bson.M{"$set": doc}

    _, err := r.readModelColl.UpdateOne(ctx, filter, update, UpsertOptions())
    return err
}
```

## Обработка ошибок

### Маппинг ошибок MongoDB → Domain

| MongoDB Error | Domain Error |
|---------------|--------------|
| `mongo.ErrNoDocuments` | `errs.ErrNotFound` |
| `DuplicateKeyError` | `errs.ErrAlreadyExists` |
| `shared.ErrConcurrencyConflict` | `errs.ErrConcurrentModification` |
| `shared.ErrAggregateNotFound` | `errs.ErrNotFound` |

### Обработка в коде

```go
err := r.eventStore.SaveEvents(ctx, id, events, version)
if err != nil {
    if errors.Is(err, shared.ErrConcurrencyConflict) {
        return errs.ErrConcurrentModification
    }
    return fmt.Errorf("failed to save events: %w", err)
}
```

## Транзакции

### Когда использовать

1. **Удаление с каскадом** — workspace + members
2. **Event Store + Read Model** — уже обрабатывается в EventStore
3. **Bulk операции** — если требуется атомарность

### Пример

```go
func (r *MongoWorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
    session, err := r.collection.Database().Client().StartSession()
    if err != nil {
        return fmt.Errorf("failed to start session: %w", err)
    }
    defer session.EndSession(ctx)

    _, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
        // 1. Удалить workspace
        _, err := r.collection.DeleteOne(txCtx, bson.M{"workspace_id": id.String()})
        if err != nil {
            return nil, err
        }

        // 2. Удалить members
        _, err = r.membersCollection.DeleteMany(txCtx, bson.M{"workspace_id": id.String()})
        if err != nil {
            return nil, err
        }

        return nil, nil
    })

    return err
}
```

## Checklist

### Общие компоненты

- [ ] Расширить `common.go` новыми helper-функциями
- [ ] Добавить `BaseDocument` структуру
- [ ] Добавить `UpsertOptions()` helper
- [ ] Добавить `FindWithPagination()` helper

### Domain restore функции

Для корректной десериализации нужны `Restore` функции в domain:

- [ ] `user.Restore()` — восстановление User из полей
- [ ] `workspace.Restore()` — восстановление Workspace из полей
- [ ] `message.Restore()` — восстановление Message из полей
- [ ] `notification.Restore()` — восстановление Notification из полей

### Документация

- [ ] Добавить комментарии ко всем публичным функциям
- [ ] Документировать правила маппинга ошибок

## Следующие шаги

После выполнения этой задачи:

1. **Task 02** — реализация TaskRepository (Event-Sourced)
2. **Task 03** — завершение UserRepository
3. **Task 04** — завершение WorkspaceRepository

## Референсы

- [MongoDB Go Driver v2](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2)
- [BSON Specification](https://bsonspec.org/)
- Существующая реализация: `chat_repository.go`
- Event Store: `internal/infrastructure/eventstore/mongodb_store.go`
