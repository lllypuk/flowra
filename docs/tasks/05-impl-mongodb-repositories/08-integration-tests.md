# Task 08: Интеграционные тесты с testcontainers

## Цель

Создать полноценную инфраструктуру для интеграционного тестирования всех MongoDB репозиториев с использованием testcontainers-go.

## Контекст

Интеграционные тесты критически важны для проверки:

1. Корректной работы репозиториев с реальной MongoDB
2. Правильности индексов и constraint'ов
3. Транзакций и concurrency
4. Сериализации/десериализации данных

testcontainers-go позволяет запускать MongoDB в Docker контейнере для каждого теста.

## Зависимости

### Требуемые пакеты

```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/mongodb
```

### Требования к окружению

- Docker установлен и запущен
- Доступ к Docker Hub для скачивания образа MongoDB

## Детальное описание

### 1. Test Utilities

Создать `tests/testutil/mongodb.go`:

```go
package testutil

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/mongodb"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    infraMongo "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

// MongoDBContainer представляет запущенный контейнер MongoDB
type MongoDBContainer struct {
    Container testcontainers.Container
    URI       string
    Client    *mongo.Client
    Database  *mongo.Database
}

// SetupMongoDB создает и настраивает MongoDB контейнер для тестов
func SetupMongoDB(t *testing.T) (*MongoDBContainer, func()) {
    t.Helper()

    ctx := context.Background()

    // Создаем контейнер MongoDB
    container, err := mongodb.Run(ctx,
        "mongo:6.0",
        mongodb.WithReplicaSet("rs0"),
    )
    if err != nil {
        t.Fatalf("Failed to start MongoDB container: %v", err)
    }

    // Получаем connection string
    uri, err := container.ConnectionString(ctx)
    if err != nil {
        container.Terminate(ctx)
        t.Fatalf("Failed to get connection string: %v", err)
    }

    // Подключаемся к MongoDB
    clientOpts := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(clientOpts)
    if err != nil {
        container.Terminate(ctx)
        t.Fatalf("Failed to connect to MongoDB: %v", err)
    }

    // Проверяем подключение
    pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    if err := client.Ping(pingCtx, nil); err != nil {
        client.Disconnect(ctx)
        container.Terminate(ctx)
        t.Fatalf("Failed to ping MongoDB: %v", err)
    }

    // Создаем тестовую базу данных
    dbName := fmt.Sprintf("test_db_%d", time.Now().UnixNano())
    db := client.Database(dbName)

    // Создаем индексы
    if err := infraMongo.CreateAllIndexes(ctx, db); err != nil {
        client.Disconnect(ctx)
        container.Terminate(ctx)
        t.Fatalf("Failed to create indexes: %v", err)
    }

    mongoContainer := &MongoDBContainer{
        Container: container,
        URI:       uri,
        Client:    client,
        Database:  db,
    }

    // Функция очистки
    cleanup := func() {
        cleanupCtx := context.Background()
        if client != nil {
            client.Disconnect(cleanupCtx)
        }
        if container != nil {
            container.Terminate(cleanupCtx)
        }
    }

    return mongoContainer, cleanup
}

// SetupMongoDBShared создает shared MongoDB контейнер для группы тестов
// Используется с TestMain для ускорения тестов
func SetupMongoDBShared(m *testing.M) (*MongoDBContainer, func(), error) {
    ctx := context.Background()

    container, err := mongodb.Run(ctx,
        "mongo:6.0",
        mongodb.WithReplicaSet("rs0"),
    )
    if err != nil {
        return nil, nil, fmt.Errorf("failed to start MongoDB container: %w", err)
    }

    uri, err := container.ConnectionString(ctx)
    if err != nil {
        container.Terminate(ctx)
        return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
    }

    clientOpts := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(clientOpts)
    if err != nil {
        container.Terminate(ctx)
        return nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    mongoContainer := &MongoDBContainer{
        Container: container,
        URI:       uri,
        Client:    client,
    }

    cleanup := func() {
        cleanupCtx := context.Background()
        if client != nil {
            client.Disconnect(cleanupCtx)
        }
        if container != nil {
            container.Terminate(cleanupCtx)
        }
    }

    return mongoContainer, cleanup, nil
}

// CreateTestDatabase создает новую тестовую базу данных
func (c *MongoDBContainer) CreateTestDatabase(t *testing.T) *mongo.Database {
    t.Helper()

    dbName := fmt.Sprintf("test_db_%d", time.Now().UnixNano())
    db := c.Client.Database(dbName)

    ctx := context.Background()
    if err := infraMongo.CreateAllIndexes(ctx, db); err != nil {
        t.Fatalf("Failed to create indexes: %v", err)
    }

    c.Database = db
    return db
}

// CleanDatabase очищает все коллекции в базе данных
func (c *MongoDBContainer) CleanDatabase(t *testing.T) {
    t.Helper()

    ctx := context.Background()
    collections, err := c.Database.ListCollectionNames(ctx, map[string]any{})
    if err != nil {
        t.Fatalf("Failed to list collections: %v", err)
    }

    for _, coll := range collections {
        if err := c.Database.Collection(coll).Drop(ctx); err != nil {
            t.Fatalf("Failed to drop collection %s: %v", coll, err)
        }
    }
}
```

### 2. Test Fixtures

Создать `tests/testutil/fixtures.go`:

```go
package testutil

import (
    "time"

    "github.com/lllypuk/flowra/internal/domain/uuid"
    userdomain "github.com/lllypuk/flowra/internal/domain/user"
    workspacedomain "github.com/lllypuk/flowra/internal/domain/workspace"
    messagedomain "github.com/lllypuk/flowra/internal/domain/message"
    notificationdomain "github.com/lllypuk/flowra/internal/domain/notification"
    taskdomain "github.com/lllypuk/flowra/internal/domain/task"
)

// UserFixture создает тестового пользователя
type UserFixture struct {
    ID          uuid.UUID
    Username    string
    Email       string
    DisplayName string
}

// NewUserFixture создает новую фикстуру пользователя
func NewUserFixture() *UserFixture {
    id := uuid.NewUUID()
    return &UserFixture{
        ID:          id,
        Username:    "testuser_" + id.String()[:8],
        Email:       "test_" + id.String()[:8] + "@example.com",
        DisplayName: "Test User",
    }
}

// Build создает domain User из фикстуры
func (f *UserFixture) Build() *userdomain.User {
    return userdomain.NewUser(f.ID, f.Username, f.Email, f.DisplayName)
}

// WithUsername устанавливает username
func (f *UserFixture) WithUsername(username string) *UserFixture {
    f.Username = username
    return f
}

// WithEmail устанавливает email
func (f *UserFixture) WithEmail(email string) *UserFixture {
    f.Email = email
    return f
}

// WorkspaceFixture создает тестовый workspace
type WorkspaceFixture struct {
    ID        uuid.UUID
    Name      string
    CreatedBy uuid.UUID
}

// NewWorkspaceFixture создает новую фикстуру workspace
func NewWorkspaceFixture(createdBy uuid.UUID) *WorkspaceFixture {
    return &WorkspaceFixture{
        ID:        uuid.NewUUID(),
        Name:      "Test Workspace",
        CreatedBy: createdBy,
    }
}

// Build создает domain Workspace из фикстуры
func (f *WorkspaceFixture) Build() *workspacedomain.Workspace {
    return workspacedomain.NewWorkspace(f.ID, f.Name, f.CreatedBy)
}

// WithName устанавливает имя
func (f *WorkspaceFixture) WithName(name string) *WorkspaceFixture {
    f.Name = name
    return f
}

// MessageFixture создает тестовое сообщение
type MessageFixture struct {
    ID              uuid.UUID
    ChatID          uuid.UUID
    AuthorID        uuid.UUID
    Content         string
    ParentMessageID *uuid.UUID
}

// NewMessageFixture создает новую фикстуру сообщения
func NewMessageFixture(chatID, authorID uuid.UUID) *MessageFixture {
    return &MessageFixture{
        ID:       uuid.NewUUID(),
        ChatID:   chatID,
        AuthorID: authorID,
        Content:  "Test message content",
    }
}

// Build создает domain Message из фикстуры
func (f *MessageFixture) Build() *messagedomain.Message {
    if f.ParentMessageID != nil {
        return messagedomain.NewThreadReply(f.ID, f.ChatID, f.AuthorID, f.Content, *f.ParentMessageID)
    }
    return messagedomain.NewMessage(f.ID, f.ChatID, f.AuthorID, f.Content)
}

// WithContent устанавливает контент
func (f *MessageFixture) WithContent(content string) *MessageFixture {
    f.Content = content
    return f
}

// AsReplyTo делает сообщение ответом в треде
func (f *MessageFixture) AsReplyTo(parentID uuid.UUID) *MessageFixture {
    f.ParentMessageID = &parentID
    return f
}

// NotificationFixture создает тестовое уведомление
type NotificationFixture struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    Type         notificationdomain.Type
    Title        string
    Body         string
    ResourceID   *uuid.UUID
    ResourceType *string
}

// NewNotificationFixture создает новую фикстуру уведомления
func NewNotificationFixture(userID uuid.UUID) *NotificationFixture {
    return &NotificationFixture{
        ID:     uuid.NewUUID(),
        UserID: userID,
        Type:   notificationdomain.TypeMention,
        Title:  "Test Notification",
        Body:   "Test notification body",
    }
}

// Build создает domain Notification из фикстуры
func (f *NotificationFixture) Build() *notificationdomain.Notification {
    return notificationdomain.NewNotification(
        f.ID,
        f.UserID,
        f.Type,
        f.Title,
        f.Body,
        f.ResourceID,
        f.ResourceType,
    )
}

// WithType устанавливает тип уведомления
func (f *NotificationFixture) WithType(t notificationdomain.Type) *NotificationFixture {
    f.Type = t
    return f
}

// WithResource устанавливает связанный ресурс
func (f *NotificationFixture) WithResource(id uuid.UUID, resourceType string) *NotificationFixture {
    f.ResourceID = &id
    f.ResourceType = &resourceType
    return f
}

// TaskFixture создает тестовую задачу
type TaskFixture struct {
    ID         uuid.UUID
    ChatID     uuid.UUID
    Title      string
    EntityType taskdomain.EntityType
    Priority   taskdomain.Priority
    AssigneeID *uuid.UUID
    DueDate    *time.Time
    CreatedBy  uuid.UUID
}

// NewTaskFixture создает новую фикстуру задачи
func NewTaskFixture(chatID, createdBy uuid.UUID) *TaskFixture {
    return &TaskFixture{
        ID:         uuid.NewUUID(),
        ChatID:     chatID,
        Title:      "Test Task",
        EntityType: taskdomain.EntityTypeTask,
        Priority:   taskdomain.PriorityMedium,
        CreatedBy:  createdBy,
    }
}

// BuildAggregate создает Task агрегат из фикстуры
func (f *TaskFixture) BuildAggregate() *taskdomain.Aggregate {
    aggregate := taskdomain.NewTaskAggregate(f.ID)
    _ = aggregate.Create(
        f.ChatID,
        f.Title,
        f.EntityType,
        f.Priority,
        f.AssigneeID,
        f.DueDate,
        f.CreatedBy,
    )
    return aggregate
}

// WithTitle устанавливает заголовок
func (f *TaskFixture) WithTitle(title string) *TaskFixture {
    f.Title = title
    return f
}

// WithAssignee устанавливает исполнителя
func (f *TaskFixture) WithAssignee(assigneeID uuid.UUID) *TaskFixture {
    f.AssigneeID = &assigneeID
    return f
}

// WithDueDate устанавливает дедлайн
func (f *TaskFixture) WithDueDate(dueDate time.Time) *TaskFixture {
    f.DueDate = &dueDate
    return f
}

// WithPriority устанавливает приоритет
func (f *TaskFixture) WithPriority(priority taskdomain.Priority) *TaskFixture {
    f.Priority = priority
    return f
}

// WithEntityType устанавливает тип сущности
func (f *TaskFixture) WithEntityType(entityType taskdomain.EntityType) *TaskFixture {
    f.EntityType = entityType
    return f
}
```

### 3. Assertions

Создать `tests/testutil/assertions.go`:

```go
package testutil

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/domain/uuid"
)

// AssertUUIDEqual проверяет равенство UUID с понятным сообщением
func AssertUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...interface{}) {
    t.Helper()
    assert.Equal(t, expected.String(), actual.String(), msgAndArgs...)
}

// RequireUUIDEqual требует равенство UUID
func RequireUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...interface{}) {
    t.Helper()
    require.Equal(t, expected.String(), actual.String(), msgAndArgs...)
}

// AssertTimeApproximatelyEqual проверяет что времена близки (с точностью до секунды)
func AssertTimeApproximatelyEqual(t *testing.T, expected, actual time.Time, msgAndArgs ...interface{}) {
    t.Helper()
    diff := expected.Sub(actual)
    if diff < 0 {
        diff = -diff
    }
    assert.Less(t, diff, time.Second, msgAndArgs...)
}

// AssertNotZeroUUID проверяет что UUID не нулевой
func AssertNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...interface{}) {
    t.Helper()
    assert.False(t, id.IsZero(), msgAndArgs...)
}

// RequireNotZeroUUID требует что UUID не нулевой
func RequireNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...interface{}) {
    t.Helper()
    require.False(t, id.IsZero(), msgAndArgs...)
}
```

### 4. Интеграционные тесты User Repository

Создать `tests/integration/repository/user_repository_test.go`:

```go
//go:build integration

package repository_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/domain/errs"
    "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
    "github.com/lllypuk/flowra/tests/testutil"
)

func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    mongoContainer, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    coll := mongoContainer.Database.Collection("users")
    repo := mongodb.NewMongoUserRepository(coll)

    t.Run("Save and FindByID", func(t *testing.T) {
        ctx := context.Background()

        fixture := testutil.NewUserFixture()
        user := fixture.Build()

        // Save
        err := repo.Save(ctx, user)
        require.NoError(t, err)

        // FindByID
        loaded, err := repo.FindByID(ctx, fixture.ID)
        require.NoError(t, err)

        testutil.AssertUUIDEqual(t, fixture.ID, loaded.ID())
        assert.Equal(t, fixture.Username, loaded.Username())
        assert.Equal(t, fixture.Email, loaded.Email())
    })

    t.Run("FindByUsername", func(t *testing.T) {
        ctx := context.Background()

        fixture := testutil.NewUserFixture().WithUsername("unique_username_123")
        user := fixture.Build()
        _ = repo.Save(ctx, user)

        loaded, err := repo.FindByUsername(ctx, "unique_username_123")
        require.NoError(t, err)

        testutil.AssertUUIDEqual(t, fixture.ID, loaded.ID())
    })

    t.Run("FindByEmail", func(t *testing.T) {
        ctx := context.Background()

        fixture := testutil.NewUserFixture().WithEmail("unique@example.com")
        user := fixture.Build()
        _ = repo.Save(ctx, user)

        loaded, err := repo.FindByEmail(ctx, "unique@example.com")
        require.NoError(t, err)

        testutil.AssertUUIDEqual(t, fixture.ID, loaded.ID())
    })

    t.Run("Exists", func(t *testing.T) {
        ctx := context.Background()

        fixture := testutil.NewUserFixture()
        user := fixture.Build()
        _ = repo.Save(ctx, user)

        exists, err := repo.Exists(ctx, fixture.ID)
        require.NoError(t, err)
        assert.True(t, exists)

        exists, err = repo.Exists(ctx, uuid.NewUUID())
        require.NoError(t, err)
        assert.False(t, exists)
    })

    t.Run("Delete", func(t *testing.T) {
        ctx := context.Background()

        fixture := testutil.NewUserFixture()
        user := fixture.Build()
        _ = repo.Save(ctx, user)

        err := repo.Delete(ctx, fixture.ID)
        require.NoError(t, err)

        _, err = repo.FindByID(ctx, fixture.ID)
        assert.ErrorIs(t, err, errs.ErrNotFound)
    })

    t.Run("Duplicate username should fail", func(t *testing.T) {
        ctx := context.Background()

        fixture1 := testutil.NewUserFixture().WithUsername("duplicate_user")
        fixture2 := testutil.NewUserFixture().WithUsername("duplicate_user")

        _ = repo.Save(ctx, fixture1.Build())
        err := repo.Save(ctx, fixture2.Build())

        assert.ErrorIs(t, err, errs.ErrAlreadyExists)
    })

    t.Run("List with pagination", func(t *testing.T) {
        ctx := context.Background()

        // Create 5 users
        for i := 0; i < 5; i++ {
            fixture := testutil.NewUserFixture()
            _ = repo.Save(ctx, fixture.Build())
        }

        // Get first page
        users, err := repo.List(ctx, 0, 3)
        require.NoError(t, err)
        assert.Len(t, users, 3)

        // Get second page
        users, err = repo.List(ctx, 3, 3)
        require.NoError(t, err)
        assert.GreaterOrEqual(t, len(users), 2)
    })
}
```

### 5. Интеграционные тесты Task Repository

Создать `tests/integration/repository/task_repository_test.go`:

```go
//go:build integration

package repository_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/task"
    "github.com/lllypuk/flowra/internal/domain/errs"
    taskdomain "github.com/lllypuk/flowra/internal/domain/task"
    "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/internal/infrastructure/eventstore"
    "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
    "github.com/lllypuk/flowra/tests/testutil"
)

func TestTaskRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    mongoContainer, cleanup := testutil.SetupMongoDB(t)
    defer cleanup()

    es := eventstore.NewMongoEventStore(mongoContainer.Client, mongoContainer.Database.Name())
    readModelColl := mongoContainer.Database.Collection("task_read_model")
    repo := mongodb.NewMongoTaskFullRepository(es, readModelColl)

    t.Run("Save and Load", func(t *testing.T) {
        ctx := context.Background()

        chatID := uuid.NewUUID()
        createdBy := uuid.NewUUID()
        fixture := testutil.NewTaskFixture(chatID, createdBy)
        aggregate := fixture.BuildAggregate()

        // Save
        err := repo.Save(ctx, aggregate)
        require.NoError(t, err)

        // Load
        loaded, err := repo.Load(ctx, fixture.ID)
        require.NoError(t, err)

        testutil.AssertUUIDEqual(t, fixture.ID, loaded.ID())
        testutil.AssertUUIDEqual(t, chatID, loaded.ChatID())
        assert.Equal(t, fixture.Title, loaded.Title())
        assert.Equal(t, taskdomain.StatusToDo, loaded.Status())
    })

    t.Run("Concurrency conflict", func(t *testing.T) {
        ctx := context.Background()

        chatID := uuid.NewUUID()
        createdBy := uuid.NewUUID()
        fixture := testutil.NewTaskFixture(chatID, createdBy)
        aggregate := fixture.BuildAggregate()
        _ = repo.Save(ctx, aggregate)

        // Load two instances
        instance1, _ := repo.Load(ctx, fixture.ID)
        instance2, _ := repo.Load(ctx, fixture.ID)

        // Modify first instance
        _ = instance1.ChangeStatus(taskdomain.StatusInProgress, createdBy)
        err := repo.Save(ctx, instance1)
        require.NoError(t, err)

        // Try to modify second instance
        _ = instance2.ChangePriority(taskdomain.PriorityHigh, createdBy)
        err = repo.Save(ctx, instance2)

        assert.ErrorIs(t, err, errs.ErrConcurrentModification)
    })

    t.Run("FindByID (read model)", func(t *testing.T) {
        ctx := context.Background()

        chatID := uuid.NewUUID()
        createdBy := uuid.NewUUID()
        fixture := testutil.NewTaskFixture(chatID, createdBy)
        aggregate := fixture.BuildAggregate()
        _ = repo.Save(ctx, aggregate)

        readModel, err := repo.FindByID(ctx, fixture.ID)
        require.NoError(t, err)

        testutil.AssertUUIDEqual(t, fixture.ID, readModel.ID)
        assert.Equal(t, fixture.Title, readModel.Title)
    })

    t.Run("FindByAssignee", func(t *testing.T) {
        ctx := context.Background()

        assignee := uuid.NewUUID()
        createdBy := uuid.NewUUID()

        // Create 3 tasks assigned to the same user
        for i := 0; i < 3; i++ {
            chatID := uuid.NewUUID()
            fixture := testutil.NewTaskFixture(chatID, createdBy).WithAssignee(assignee)
            aggregate := fixture.BuildAggregate()
            _ = repo.Save(ctx, aggregate)
        }

        results, err := repo.FindByAssignee(ctx, assignee, task.Filters{Limit: 10})
        require.NoError(t, err)

        assert.Len(t, results, 3)
    })

    t.Run("FindByStatus", func(t *testing.T) {
        ctx := context.Background()

        createdBy := uuid.NewUUID()

        // Create task and change its status
        chatID := uuid.NewUUID()
        fixture := testutil.NewTaskFixture(chatID, createdBy)
        aggregate := fixture.BuildAggregate()
        _ = aggregate.ChangeStatus(taskdomain.StatusInProgress, createdBy)
        _ = repo.Save(ctx, aggregate)

        results, err := repo.FindByStatus(ctx, taskdomain.StatusInProgress, task.Filters{Limit: 10})
        require.NoError(t, err)

        assert.GreaterOrEqual(t, len(results), 1)
    })

    t.Run("GetEvents", func(t *testing.T) {
        ctx := context.Background()

        chatID := uuid.NewUUID()
        createdBy := uuid.NewUUID()
        fixture := testutil.NewTaskFixture(chatID, createdBy)
        aggregate := fixture.BuildAggregate()
        _ = repo.Save(ctx, aggregate)

        // Reload, change status, save again
        reloaded, _ := repo.Load(ctx, fixture.ID)
        _ = reloaded.ChangeStatus(taskdomain.StatusInProgress, createdBy)
        _ = repo.Save(ctx, reloaded)

        events, err := repo.GetEvents(ctx, fixture.ID)
        require.NoError(t, err)

        assert.Len(t, events, 2) // TaskCreated + StatusChanged
    })
}
```

### 6. Запуск интеграционных тестов

Добавить в `Makefile`:

```makefile
.PHONY: test-integration
test-integration:
	go test ./tests/integration/... -tags=integration -v

.PHONY: test-integration-short
test-integration-short:
	go test ./tests/integration/... -tags=integration -short -v

.PHONY: test-all
test-all:
	go test ./...
	go test ./tests/integration/... -tags=integration -v
```

### 7. CI/CD Configuration

Добавить в `.github/workflows/test.yml`:

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test ./...

  integration-tests:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
        options: --privileged
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test ./tests/integration/... -tags=integration -v
```

## Структура файлов

```
tests/
├── testutil/
│   ├── mongodb.go          # MongoDB container setup
│   ├── fixtures.go         # Test data builders
│   └── assertions.go       # Custom assertions
├── integration/
│   └── repository/
│       ├── user_repository_test.go
│       ├── workspace_repository_test.go
│       ├── task_repository_test.go
│       ├── message_repository_test.go
│       └── notification_repository_test.go
└── e2e/
    └── ... (future)
```

## Checklist

### Phase 1: Test Utilities

- [ ] Создать `tests/testutil/mongodb.go`
- [ ] Реализовать `SetupMongoDB`
- [ ] Реализовать `SetupMongoDBShared`
- [ ] Реализовать `CreateTestDatabase`
- [ ] Реализовать `CleanDatabase`

### Phase 2: Fixtures

- [ ] Создать `tests/testutil/fixtures.go`
- [ ] Реализовать `UserFixture`
- [ ] Реализовать `WorkspaceFixture`
- [ ] Реализовать `MessageFixture`
- [ ] Реализовать `NotificationFixture`
- [ ] Реализовать `TaskFixture`

### Phase 3: Assertions

- [ ] Создать `tests/testutil/assertions.go`
- [ ] Реализовать UUID assertions
- [ ] Реализовать time assertions

### Phase 4: User Repository Tests

- [ ] Save and FindByID
- [ ] FindByUsername
- [ ] FindByEmail
- [ ] Exists
- [ ] Delete
- [ ] Duplicate handling
- [ ] List with pagination

### Phase 5: Task Repository Tests

- [ ] Save and Load
- [ ] Concurrency conflict
- [ ] FindByID (read model)
- [ ] FindByAssignee
- [ ] FindByStatus
- [ ] GetEvents

### Phase 6: Other Repository Tests

- [ ] Workspace Repository tests
- [ ] Message Repository tests
- [ ] Notification Repository tests

### Phase 7: CI/CD

- [ ] Обновить Makefile
- [ ] Создать GitHub Actions workflow
- [ ] Протестировать в CI

## Советы по отладке

### Просмотр логов контейнера

```go
logs, err := container.Logs(ctx)
if err == nil {
    io.Copy(os.Stdout, logs)
}
```

### Сохранение контейнера после теста

```go
// Не вызывать cleanup для отладки
// defer cleanup()
t.Logf("MongoDB URI: %s", mongoContainer.URI)
```

### Подключение к тестовому MongoDB

```bash
# После того как тест застрял
mongosh "mongodb://localhost:ПОРТ/?replicaSet=rs0"
```

## Следующие шаги

После завершения всех задач в этой директории:

1. **HTTP Handlers** — использование репозиториев в handlers
2. **Dependency Injection** — настройка DI
3. **Load Testing** — нагрузочное тестирование

## Референсы

- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [testcontainers-go MongoDB Module](https://golang.testcontainers.org/modules/mongodb/)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [MongoDB Testing Strategies](https://www.mongodb.com/docs/manual/reference/configuration-options/#test-commands)
