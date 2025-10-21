# Task 06: Use Case Testing Strategy

**Дата:** 2025-10-17
**Статус:** Completed
**Зависимости:** Tasks 02-05 (All Use Cases)
**Оценка:** 2 часа (документация + setup)
**Завершено:** 2025-10-18

## Цель

Определить единую стратегию тестирования для всех use cases, настроить инструменты и создать шаблоны для будущих тестов.

## Типы тестов

### 1. Unit Tests

**Цель**: Тестировать use case изолированно с моками

**Характеристики**:
- Быстрые (< 1 секунды на тест)
- Без внешних зависимостей (БД, сеть)
- In-memory Event Store
- Mock repositories

**Структура**:
```
internal/usecase/task/
├── create_task.go
├── create_task_test.go       # unit tests
├── change_status.go
├── change_status_test.go     # unit tests
└── ...
```

**Пример**:
```go
func TestCreateTaskUseCase_Success(t *testing.T) {
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    // Test...
}
```

### 2. Integration Tests

**Цель**: Тестировать use case с реальной БД

**Характеристики**:
- Медленнее (1-5 секунд на тест)
- Реальная MongoDB (тестовая БД)
- Реальный Event Store
- Build tag: `// +build integration`

**Структура**:
```
tests/integration/
├── usecase/
│   ├── task_create_test.go
│   ├── task_status_test.go
│   └── ...
└── testutil/
    ├── db.go              # Helpers для работы с БД
    └── fixtures.go        # Тестовые данные
```

**Пример**:
```go
// +build integration

package integration_test

import (
    "context"
    "testing"

    "flowra/tests/testutil"
)

func TestCreateTaskUseCase_Integration(t *testing.T) {
    db := testutil.SetupTestDatabase(t)
    defer testutil.TeardownTestDatabase(t, db)

    eventStore := eventstore.NewMongoDBEventStore(db)
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    // Test...
}
```

**Запуск**:
```bash
# Только unit tests
go test ./internal/usecase/task/

# Только integration tests
go test -tags=integration ./tests/integration/...

# Все тесты
go test -tags=integration ./...
```

### 3. E2E Tests (будущее)

**Цель**: Тестировать полный флоу через HTTP API

**Характеристики**:
- Самые медленные
- Запускается реальное приложение
- HTTP запросы
- WebSocket тесты

**Структура**:
```
tests/e2e/
├── task_lifecycle_test.go
├── chat_integration_test.go
└── ...
```

## Test Coverage Goals

### Минимальные требования

```
internal/usecase/task/          > 80%
internal/domain/task/           > 90%
internal/infrastructure/        > 70%
```

### Проверка покрытия

```bash
# Генерация отчета
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Проверка порога
go test -coverprofile=coverage.out ./internal/usecase/task/
go tool cover -func=coverage.out | grep total
```

### CI Integration (будущее)

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Unit Tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Check Coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage $coverage% is below 80%"
            exit 1
          fi

      - name: Run Integration Tests
        run: |
          docker-compose up -d mongodb
          go test -tags=integration -v ./tests/integration/...
        env:
          TEST_MONGODB_URI: mongodb://admin:admin123@localhost:27017/test_db
```

## Test Utilities

### 1. In-Memory Event Store

```go
// internal/infrastructure/eventstore/inmemory.go
package eventstore

import (
    "context"
    "sync"
    "flowra/internal/domain/task"
)

type InMemoryEventStore struct {
    mu     sync.RWMutex
    events map[string][]task.Event
}

func NewInMemoryEventStore() *InMemoryEventStore {
    return &InMemoryEventStore{
        events: make(map[string][]task.Event),
    }
}

func (s *InMemoryEventStore) SaveEvents(ctx context.Context, aggregateID string, events []task.Event, expectedVersion int) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    storedEvents := s.events[aggregateID]
    if len(storedEvents) != expectedVersion {
        return ErrConcurrentUpdate
    }

    s.events[aggregateID] = append(storedEvents, events...)
    return nil
}

func (s *InMemoryEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]task.Event, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    events, exists := s.events[aggregateID]
    if !exists {
        return []task.Event{}, nil
    }

    // Return a copy to prevent external modifications
    result := make([]task.Event, len(events))
    copy(result, events)
    return result, nil
}

func (s *InMemoryEventStore) Reset() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.events = make(map[string][]task.Event)
}
```

### 2. Mock User Repository

```go
// tests/mocks/user_repository.go
package mocks

import (
    "context"
    "github.com/google/uuid"
    "flowra/internal/usecase/shared"
)

type MockUserRepository struct {
    users map[uuid.UUID]*shared.User
}

func NewMockUserRepository() *MockUserRepository {
    return &MockUserRepository{
        users: make(map[uuid.UUID]*shared.User),
    }
}

func (m *MockUserRepository) AddUser(id uuid.UUID, username, fullName string) {
    m.users[id] = &shared.User{
        ID:       id,
        Username: username,
        FullName: fullName,
    }
}

func (m *MockUserRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
    _, exists := m.users[userID]
    return exists, nil
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*shared.User, error) {
    for _, user := range m.users {
        if user.Username == username {
            return user, nil
        }
    }
    return nil, nil
}

func (m *MockUserRepository) Reset() {
    m.users = make(map[uuid.UUID]*shared.User)
}
```

### 3. Test Database Helpers

```go
// tests/testutil/db.go
package testutil

import (
    "database/sql"
    "fmt"
    "os"
    "testing"

    _ "github.com/lib/pq"
)

// SetupTestDatabase создает тестовую БД и применяет миграции
func SetupTestDatabase(t *testing.T) *sql.DB {
    dbURL := os.Getenv("TEST_DATABASE_URL")
    if dbURL == "" {
        t.Skip("TEST_DATABASE_URL not set, skipping integration test")
    }

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }

    // Создаем уникальное имя схемы для изоляции тестов
    schema := fmt.Sprintf("test_%s", t.Name())

    _, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
    if err != nil {
        t.Fatalf("Failed to create test schema: %v", err)
    }

    _, err = db.Exec(fmt.Sprintf("SET search_path TO %s", schema))
    if err != nil {
        t.Fatalf("Failed to set search path: %v", err)
    }

    // Применяем миграции
    if err := applyMigrations(db, schema); err != nil {
        t.Fatalf("Failed to apply migrations: %v", err)
    }

    return db
}

// TeardownTestDatabase удаляет тестовую схему
func TeardownTestDatabase(t *testing.T, db *sql.DB) {
    schema := fmt.Sprintf("test_%s", t.Name())
    _, err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
    if err != nil {
        t.Logf("Warning: Failed to drop test schema: %v", err)
    }
    db.Close()
}

func applyMigrations(db *sql.DB, schema string) error {
    // TODO: Реализовать применение миграций
    // Можно использовать golang-migrate или выполнять SQL файлы
    return nil
}
```

### 4. Test Fixtures

```go
// tests/testutil/fixtures.go
package testutil

import (
    "time"
    "github.com/google/uuid"
    taskusecase "flowra/internal/usecase/task"
)

// CreateTaskCommandFixture возвращает валидную команду создания задачи
func CreateTaskCommandFixture() taskusecase.CreateTaskCommand {
    return taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        Priority:  "Medium",
        CreatedBy: uuid.New(),
    }
}

// WithTitle модифицирует title
func WithTitle(title string) func(*taskusecase.CreateTaskCommand) {
    return func(cmd *taskusecase.CreateTaskCommand) {
        cmd.Title = title
    }
}

// WithPriority модифицирует priority
func WithPriority(priority string) func(*taskusecase.CreateTaskCommand) {
    return func(cmd *taskusecase.CreateTaskCommand) {
        cmd.Priority = priority
    }
}

// WithAssignee добавляет assignee
func WithAssignee(assigneeID uuid.UUID) func(*taskusecase.CreateTaskCommand) {
    return func(cmd *taskusecase.CreateTaskCommand) {
        cmd.AssigneeID = &assigneeID
    }
}

// WithDueDate добавляет дедлайн
func WithDueDate(dueDate time.Time) func(*taskusecase.CreateTaskCommand) {
    return func(cmd *taskusecase.CreateTaskCommand) {
        cmd.DueDate = &dueDate
    }
}

// BuildCreateTaskCommand создает команду с модификаторами
func BuildCreateTaskCommand(modifiers ...func(*taskusecase.CreateTaskCommand)) taskusecase.CreateTaskCommand {
    cmd := CreateTaskCommandFixture()
    for _, modifier := range modifiers {
        modifier(&cmd)
    }
    return cmd
}

// Пример использования:
// cmd := testutil.BuildCreateTaskCommand(
//     testutil.WithTitle("Custom Title"),
//     testutil.WithPriority("High"),
// )
```

## Table-Driven Tests

Используем table-driven подход для тестирования множества сценариев:

```go
func TestValidation_TableDriven(t *testing.T) {
    tests := []struct {
        name        string
        cmd         taskusecase.CreateTaskCommand
        expectedErr error
    }{
        {
            name: "Valid command",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "Test",
                CreatedBy: uuid.New(),
            },
            expectedErr: nil,
        },
        {
            name: "Empty title",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "",
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrEmptyTitle,
        },
        // ... больше случаев
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            useCase := taskusecase.NewCreateTaskUseCase(eventstore.NewInMemoryEventStore())
            _, err := useCase.Execute(context.Background(), tt.cmd)

            if tt.expectedErr == nil {
                assert.NoError(t, err)
            } else {
                assert.ErrorIs(t, err, tt.expectedErr)
            }
        })
    }
}
```

## Test Organization

### По файлам

```
internal/usecase/task/
├── create_task.go
├── create_task_test.go          # Только CreateTask тесты
├── change_status.go
├── change_status_test.go        # Только ChangeStatus тесты
└── ...
```

### По сценариям в тесте

```go
// create_task_test.go

// Успешные сценарии
func TestCreateTaskUseCase_Success(t *testing.T) { ... }
func TestCreateTaskUseCase_WithDefaults(t *testing.T) { ... }
func TestCreateTaskUseCase_WithAllFields(t *testing.T) { ... }

// Ошибки валидации
func TestCreateTaskUseCase_ValidationErrors(t *testing.T) { ... }

// Ошибки инфраструктуры
func TestCreateTaskUseCase_EventStoreFailure(t *testing.T) { ... }
```

## Benchmarks (опционально)

Для критичных операций:

```go
func BenchmarkCreateTaskUseCase(b *testing.B) {
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    cmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Benchmark Task",
        CreatedBy: uuid.New(),
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := useCase.Execute(context.Background(), cmd)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

Запуск:
```bash
go test -bench=. -benchmem ./internal/usecase/task/
```

## Test Naming Convention

```go
// Паттерн: Test<UseCaseName>_<Scenario>

// Успешные случаи
TestCreateTaskUseCase_Success
TestCreateTaskUseCase_WithDefaults

// Ошибки
TestCreateTaskUseCase_ValidationErrors
TestCreateTaskUseCase_EmptyTitle
TestCreateTaskUseCase_TaskNotFound

// Edge cases
TestCreateTaskUseCase_Idempotent
TestCreateTaskUseCase_ConcurrentUpdate
```

## Assertion Libraries

Используем `testify/assert` и `testify/require`:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require - останавливает тест при ошибке
require.NoError(t, err)
require.NotNil(t, result)

// assert - продолжает выполнение
assert.Equal(t, expected, actual)
assert.Len(t, events, 1)
```

## Checklist

- [x] Создать `InMemoryEventStore` в `internal/infrastructure/eventstore/inmemory.go`
- [x] Создать `MockUserRepository` в `tests/mocks/user_repository.go`
- [x] Создать test helpers в `tests/testutil/`
- [x] Настроить coverage checking в Makefile
- [x] Документировать test conventions в README
- [x] Создать примеры table-driven tests
- [x] Настроить integration tests структуру

## Makefile Targets

```makefile
# Makefile

.PHONY: test
test:
	go test -v -race ./...

.PHONY: test-unit
test-unit:
	go test -v -race ./internal/...

.PHONY: test-integration
test-integration:
	docker-compose up -d mongodb
	go test -tags=integration -v ./tests/integration/...
	docker-compose down

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-coverage-check
test-coverage-check:
	@go test -coverprofile=coverage.out ./... > /dev/null
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$coverage < 80" | bc -l)" -eq 1 ]; then \
		echo "❌ Coverage $$coverage% is below 80%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets threshold"; \
	fi
```

## Критерии приемки

- ✅ InMemoryEventStore реализован и работает (internal/infrastructure/eventstore/inmemory.go)
- ✅ Mock repositories созданы (tests/mocks/user_repository.go)
- ✅ Test helpers и fixtures готовы (tests/testutil/db.go, tests/testutil/fixtures.go)
- ✅ Coverage checking настроен (Makefile: test-coverage, test-coverage-check)
- ✅ Integration tests структура создана (tests/testutil/db.go with build tag)
- ✅ Makefile targets для тестов работают (test, test-unit, test-integration, test-coverage, test-coverage-check)
- ✅ Документация обновлена (Task 06 marked as Completed)

## Следующие шаги

После завершения всех задач (01-06):
- Реализация кода use cases согласно документации
- Написание тестов для каждого use case
- Достижение coverage >80%
- Интеграция с HTTP handlers

## Референсы

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify Documentation](https://github.com/stretchr/testify)
- [golang-migrate](https://github.com/golang-migrate/migrate)
