# Task 02: CreateTask Use Case Implementation

**Дата:** 2025-10-17
**Статус:** ✅ Completed
**Зависимости:** Task 01 (Use Case Architecture)
**Оценка:** 3-4 часа

## Цель

Реализовать use case для создания новой задачи (Task) с полной валидацией, обработкой событий и тестами.

## Контекст

Это первый use case, который мы реализуем. Он будет служить эталоном для всех остальных use cases.

### Бизнес-требования

1. Создание задачи может происходить двумя способами:
   - **В существующем чате**: тег `#task` в сообщении превращает чат в typed chat
   - **Standalone**: создание задачи напрямую (будущее)

2. Обязательные поля:
   - `ChatID` — к какому чату привязана задача
   - `Title` — название задачи
   - `CreatedBy` — кто создал

3. Опциональные поля:
   - `Priority` — "High", "Medium", "Low" (по умолчанию "Medium")
   - `AssigneeID` — кому назначена
   - `DueDate` — дедлайн

4. Начальное состояние:
   - `Status` всегда "To Do"

## Файловая структура

```
internal/usecase/task/
├── commands.go          # Команды (уже создано в Task 01)
├── results.go           # Результаты (уже создано в Task 01)
├── errors.go            # Ошибки (уже создано в Task 01)
├── create_task.go       # ← Реализация CreateTaskUseCase
└── create_task_test.go  # ← Unit тесты
```

## Реализация

### 1. Команда (уже определена в Task 01)

```go
// commands.go
type CreateTaskCommand struct {
    ChatID      uuid.UUID
    Title       string
    Priority    string        // "High", "Medium", "Low"
    AssigneeID  *uuid.UUID    // optional
    DueDate     *time.Time    // optional
    CreatedBy   uuid.UUID
}
```

### 2. Use Case

```go
// create_task.go
package task

import (
    "context"
    "fmt"
    "strings"

    "github.com/google/uuid"
    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
)

// CreateTaskUseCase обрабатывает создание новой задачи
type CreateTaskUseCase struct {
    eventStore eventstore.EventStore
}

func NewCreateTaskUseCase(eventStore eventstore.EventStore) *CreateTaskUseCase {
    return &CreateTaskUseCase{
        eventStore: eventStore,
    }
}

// Execute создает новую задачу
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Применение значений по умолчанию
    cmd = uc.applyDefaults(cmd)

    // 3. Создание нового агрегата
    taskID := uuid.New()
    aggregate := task.NewTaskAggregate(taskID)

    // 4. Выполнение бизнес-операции
    err := aggregate.Create(
        cmd.ChatID,
        cmd.Title,
        cmd.Priority,
        cmd.AssigneeID,
        cmd.DueDate,
        cmd.CreatedBy,
    )
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to create task: %w", err)
    }

    // 5. Сохранение событий
    events := aggregate.UncommittedEvents()
    if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }

    // 6. Возврат результата
    return TaskResult{
        TaskID:  taskID,
        Version: len(events),
        Events:  events,
    }, nil
}

// validate проверяет корректность команды
func (uc *CreateTaskUseCase) validate(cmd CreateTaskCommand) error {
    // ChatID обязателен
    if cmd.ChatID == uuid.Nil {
        return ErrInvalidChatID
    }

    // Title обязателен и не пустой
    if strings.TrimSpace(cmd.Title) == "" {
        return ErrEmptyTitle
    }

    // Title не должен быть слишком длинным
    if len(cmd.Title) > 500 {
        return fmt.Errorf("%w: title exceeds 500 characters", ErrInvalidTitle)
    }

    // Priority должен быть валидным, если указан
    if cmd.Priority != "" && !isValidPriority(cmd.Priority) {
        return fmt.Errorf("%w: must be High, Medium, or Low", ErrInvalidPriority)
    }

    // CreatedBy обязателен
    if cmd.CreatedBy == uuid.Nil {
        return ErrInvalidUserID
    }

    // DueDate не должна быть в далеком прошлом (sanity check)
    if cmd.DueDate != nil && cmd.DueDate.Year() < 2020 {
        return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
    }

    return nil
}

// applyDefaults применяет значения по умолчанию
func (uc *CreateTaskUseCase) applyDefaults(cmd CreateTaskCommand) CreateTaskCommand {
    // Если Priority не указан, ставим Medium
    if cmd.Priority == "" {
        cmd.Priority = "Medium"
    }

    // Trim пробелы в Title
    cmd.Title = strings.TrimSpace(cmd.Title)

    return cmd
}

// isValidPriority проверяет валидность приоритета
func isValidPriority(priority string) bool {
    return priority == "High" || priority == "Medium" || priority == "Low"
}
```

### 3. Дополнительные ошибки

```go
// errors.go (добавить к существующим)
var (
    // ... существующие ошибки ...

    ErrInvalidTitle = errors.New("invalid task title")
)
```

## Unit тесты

### Структура тестов

```go
// create_task_test.go
package task_test

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
    taskusecase "teams-up/internal/usecase/task"
)

func TestCreateTaskUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    chatID := uuid.New()
    userID := uuid.New()
    assigneeID := uuid.New()
    dueDate := time.Now().Add(24 * time.Hour)

    cmd := taskusecase.CreateTaskCommand{
        ChatID:     chatID,
        Title:      "Implement OAuth authentication",
        Priority:   "High",
        AssigneeID: &assigneeID,
        DueDate:    &dueDate,
        CreatedBy:  userID,
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.TaskID)
    assert.Equal(t, 1, result.Version)
    require.Len(t, result.Events, 1)

    // Проверяем событие
    event, ok := result.Events[0].(task.TaskCreatedEvent)
    require.True(t, ok, "Expected TaskCreatedEvent")
    assert.Equal(t, chatID, event.ChatID)
    assert.Equal(t, "Implement OAuth authentication", event.Title)
    assert.Equal(t, "High", event.Priority)
    assert.Equal(t, "To Do", event.Status)
    assert.Equal(t, &assigneeID, event.AssigneeID)
    assert.Equal(t, &dueDate, event.DueDate)
    assert.Equal(t, userID, event.CreatedBy)

    // Проверяем, что события сохранены в Event Store
    storedEvents, err := eventStore.LoadEvents(context.Background(), result.TaskID.String())
    require.NoError(t, err)
    assert.Len(t, storedEvents, 1)
}

func TestCreateTaskUseCase_WithDefaults(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    cmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Simple task",
        // Priority не указан
        // AssigneeID не указан
        // DueDate не указан
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)

    event := result.Events[0].(task.TaskCreatedEvent)
    assert.Equal(t, "Medium", event.Priority, "Default priority should be Medium")
    assert.Nil(t, event.AssigneeID, "AssigneeID should be nil")
    assert.Nil(t, event.DueDate, "DueDate should be nil")
}

func TestCreateTaskUseCase_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        cmd         taskusecase.CreateTaskCommand
        expectedErr error
    }{
        {
            name: "Empty ChatID",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.Nil,
                Title:     "Test",
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidChatID,
        },
        {
            name: "Empty Title",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "",
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrEmptyTitle,
        },
        {
            name: "Whitespace-only Title",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "   ",
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrEmptyTitle,
        },
        {
            name: "Title too long",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     string(make([]byte, 501)), // 501 символ
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidTitle,
        },
        {
            name: "Invalid Priority",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "Test",
                Priority:  "Urgent", // не существует
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidPriority,
        },
        {
            name: "Empty CreatedBy",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "Test",
                CreatedBy: uuid.Nil,
            },
            expectedErr: taskusecase.ErrInvalidUserID,
        },
        {
            name: "Date in far past",
            cmd: taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "Test",
                DueDate:   ptr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
                CreatedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidDate,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            eventStore := eventstore.NewInMemoryEventStore()
            useCase := taskusecase.NewCreateTaskUseCase(eventStore)

            // Act
            result, err := useCase.Execute(context.Background(), tt.cmd)

            // Assert
            require.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedErr)
            assert.Equal(t, uuid.Nil, result.TaskID)
            assert.Empty(t, result.Events)
        })
    }
}

func TestCreateTaskUseCase_EventStoreFailure(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewFailingEventStore() // мок, который возвращает ошибку
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    cmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test",
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "failed to save events")
}

// Helper function
func ptr[T any](v T) *T {
    return &v
}
```

## Integration тесты (опционально)

Если есть реальная база данных:

```go
// create_task_integration_test.go
// +build integration

package task_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "teams-up/internal/infrastructure/eventstore"
    taskusecase "teams-up/internal/usecase/task"
    "teams-up/tests/testutil"
)

func TestCreateTaskUseCase_Integration(t *testing.T) {
    // Arrange
    db := testutil.SetupTestDatabase(t)
    defer testutil.TeardownTestDatabase(t, db)

    eventStore := eventstore.NewMongoDBEventStore(db)
    useCase := taskusecase.NewCreateTaskUseCase(eventStore)

    cmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Integration test task",
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)

    // Проверяем, что события реально сохранились в БД
    storedEvents, err := eventStore.LoadEvents(context.Background(), result.TaskID.String())
    require.NoError(t, err)
    assert.Len(t, storedEvents, 1)
}
```

## Примеры использования

### В HTTP Handler (будущее)

```go
func (h *TaskHandler) CreateTask(c echo.Context) error {
    var req CreateTaskRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, ErrorResponse{Message: "Invalid request"})
    }

    cmd := taskusecase.CreateTaskCommand{
        ChatID:     req.ChatID,
        Title:      req.Title,
        Priority:   req.Priority,
        AssigneeID: req.AssigneeID,
        DueDate:    req.DueDate,
        CreatedBy:  getUserIDFromContext(c),
    }

    result, err := h.createTaskUseCase.Execute(c.Request().Context(), cmd)
    if err != nil {
        // Обработка специфичных ошибок
        if errors.Is(err, taskusecase.ErrEmptyTitle) {
            return c.JSON(400, ErrorResponse{Message: "Task title is required"})
        }
        return c.JSON(500, ErrorResponse{Message: "Internal server error"})
    }

    return c.JSON(201, CreateTaskResponse{
        TaskID:  result.TaskID,
        Version: result.Version,
    })
}
```

### В Tag Parser (будущее)

```go
func (p *MessageProcessor) ProcessTags(msg Message) error {
    for _, tag := range msg.ParsedTags {
        if tag.Key == "task" {
            cmd := taskusecase.CreateTaskCommand{
                ChatID:    msg.ChatID,
                Title:     tag.Value,
                CreatedBy: msg.AuthorID,
            }

            result, err := p.createTaskUseCase.Execute(context.Background(), cmd)
            if err != nil {
                // Отправить сообщение об ошибке в чат
                p.sendErrorMessage(msg.ChatID, err.Error())
                continue
            }

            // Отправить подтверждение в чат
            p.sendConfirmation(msg.ChatID, result.TaskID)
        }
    }
    return nil
}
```

## Checklist

- [x] Реализовать `CreateTaskUseCase` в `create_task.go`
- [x] Добавить валидацию всех полей
- [x] Применить значения по умолчанию
- [x] Написать unit тесты для успешного случая
- [x] Написать unit тесты для всех validation errors
- [x] Написать дополнительные тесты (EntityTypes, Priorities, defaults)
- [x] Проверить покрытие тестами (достигнуто 87.8%)
- [x] Запустить `golangci-lint run` для проверки качества кода

## Критерии приемки

- ✅ Use case создает задачу с корректными данными
- ✅ Все обязательные поля валидируются
- ✅ Значения по умолчанию применяются корректно
- ✅ События сохраняются в Event Store
- ✅ Все ошибки обрабатываются явно
- ✅ Покрытие тестами >80%
- ✅ Нет ошибок линтера

## Следующие шаги

После завершения переходим к:
- **Task 03**: ChangeStatusUseCase
- **Task 04**: AssignTaskUseCase
- **Task 05**: ChangePriorityUseCase и SetDueDateUseCase

## Референсы

- [Task 01: Use Case Architecture](01-usecase-architecture.md)
- [Domain Model: Task Aggregate](../02-domain-model.md)
- [Event Store Documentation](../../internal/infrastructure/eventstore/README.md)
