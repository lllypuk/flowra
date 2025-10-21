# Testing Guide

Эта директория содержит все тестовые утилиты, моки и integration тесты для проекта flowra.

## Структура

```
tests/
├── README.md              # Этот файл
├── mocks/                 # Mock реализации для тестирования
│   ├── user_repository.go
│   └── ...
├── testutil/              # Утилиты для тестирования
│   ├── db.go             # Helpers для работы с БД
│   └── fixtures.go       # Тестовые данные (builders)
└── integration/           # Integration тесты
    └── usecase/
        └── ...
```

## Типы тестов

### 1. Unit Tests

**Расположение**: Рядом с тестируемым кодом (например `create_task_test.go` рядом с `create_task.go`)

**Характеристики**:
- Быстрые (< 1 секунды на тест)
- Используют in-memory Event Store
- Используют mock repositories
- Без внешних зависимостей

**Запуск**:
```bash
# Все unit тесты
make test-unit

# Конкретный пакет
go test ./internal/usecase/task/
```

### 2. Integration Tests

**Расположение**: `tests/integration/`

**Характеристики**:
- Медленнее (1-5 секунд на тест)
- Используют реальную PostgreSQL
- Требуют build tag `integration`
- Каждый тест создает свою изолированную схему

**Запуск**:
```bash
# Все integration тесты
make test-integration

# Вручную
TEST_DATABASE_URL="postgresql://postgres:postgres123@localhost:5432/test_db?sslmode=disable" \
  go test -tags=integration ./tests/integration/...
```

### 3. Coverage

**Проверка coverage**:
```bash
# Генерация HTML отчета
make test-coverage

# Проверка порога (80%)
make test-coverage-check

# Просмотр coverage в терминале
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Test Utilities

### InMemoryEventStore

In-memory реализация EventStore для быстрых unit тестов.

```go
import "github.com/flowra/flowra/internal/infrastructure/eventstore"

eventStore := eventstore.NewInMemoryEventStore()
// Используйте в тестах
```

### MockUserRepository

Mock реализация UserRepository для тестирования.

```go
import "github.com/flowra/flowra/tests/mocks"

repo := mocks.NewMockUserRepository()
repo.AddUser(userID, "testuser", "Test User")
exists, _ := repo.Exists(ctx, userID) // true
```

### Test Fixtures (Builders)

Builders для создания тестовых команд с fluent API.

```go
import "github.com/flowra/flowra/tests/testutil"

// Базовая команда с дефолтными значениями
cmd := testutil.CreateTaskCommandFixture()

// Команда с кастомными значениями
cmd := testutil.BuildCreateTaskCommand(
    testutil.WithTitle("Custom Task"),
    testutil.WithPriority(task.PriorityHigh),
    testutil.WithAssignee(assigneeID),
    testutil.WithDueDate(tomorrow),
)
```

Доступные builders:
- `CreateTaskCommandFixture()` / `BuildCreateTaskCommand(...)`
- `ChangeStatusCommandFixture(taskID)` / `BuildChangeStatusCommand(taskID, ...)`
- `AssignTaskCommandFixture(taskID, assigneeID)` / `BuildAssignTaskCommand(...)`
- `ChangePriorityCommandFixture(taskID)` / `BuildChangePriorityCommand(taskID, ...)`
- `SetDueDateCommandFixture(taskID)` / `BuildSetDueDateCommand(taskID, ...)`

### Database Helpers (Integration Tests)

Helpers для работы с тестовой БД в integration тестах.

```go
//go:build integration

import "github.com/flowra/flowra/tests/testutil"

func TestSomething_Integration(t *testing.T) {
    db := testutil.SetupTestDatabase(t)
    defer testutil.TeardownTestDatabase(t, db)
    
    // Используйте db для создания eventStore и т.д.
}
```

## Best Practices

### Table-Driven Tests

Используйте table-driven подход для множества сценариев:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectedErr error
    }{
        {name: "valid input", input: "test", expectedErr: nil},
        {name: "empty input", input: "", expectedErr: ErrEmptyInput},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.expectedErr == nil {
                assert.NoError(t, err)
            } else {
                assert.ErrorIs(t, err, tt.expectedErr)
            }
        })
    }
}
```

### Test Naming Convention

```go
// Pattern: Test<FunctionName>_<Scenario>

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

### Assertions

Используйте `testify/assert` и `testify/require`:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require - останавливает тест при ошибке (для критичных проверок)
require.NoError(t, err)
require.NotNil(t, result)

// assert - продолжает выполнение (для множественных проверок)
assert.Equal(t, expected, actual)
assert.Len(t, events, 1)
assert.True(t, condition)
```

## Makefile Targets

```bash
make test                  # Все тесты
make test-unit            # Только unit тесты
make test-integration     # Только integration тесты
make test-coverage        # Генерация HTML отчета
make test-coverage-check  # Проверка порога (80%)
make test-verbose         # Тесты с verbose output
make test-clean           # Очистка cache и coverage файлов
```

## Environment Variables

### Integration Tests

- `TEST_DATABASE_URL` - URL тестовой PostgreSQL БД
  - Example: `postgresql://postgres:postgres123@localhost:5432/test_db?sslmode=disable`
  - Если не установлена, integration тесты будут пропущены

## CI/CD (Future)

При настройке CI будут автоматически запускаться:
1. Unit тесты при каждом push
2. Coverage check (минимум 80%)
3. Integration тесты при PR
4. Linting

## Coverage Goals

```
internal/usecase/task/          > 80%
internal/domain/task/           > 90%
internal/infrastructure/        > 70%
```

## Troubleshooting

### Integration тесты не запускаются

Проверьте:
1. PostgreSQL запущен: `docker-compose ps postgres`
2. Переменная `TEST_DATABASE_URL` установлена
3. Build tag `integration` указан: `-tags=integration`

### Coverage низкий

1. Проверьте какие пакеты не покрыты: `go tool cover -func=coverage.out`
2. Добавьте тесты для непокрытого кода
3. Используйте HTML отчет для визуализации: `make test-coverage` → `coverage.html`

### Тесты медленные

1. Убедитесь что используете unit тесты, а не integration
2. Используйте `InMemoryEventStore` вместо БД
3. Избегайте `time.Sleep()` в тестах
4. Используйте моки для внешних зависимостей

## References

- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify Documentation](https://github.com/stretchr/testify)
