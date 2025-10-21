# Task 01: Use Case Layer Architecture and Patterns

**Дата:** 2025-10-17
**Статус:** ✅ Completed
**Зависимости:** Domain model (Task aggregate)
**Оценка:** 2-3 часа

## Цель

Спроектировать и документировать архитектуру слоя Use Cases для работы с Task агрегатом. Определить паттерны, интерфейсы и структуру кода, которые будут использоваться во всех use cases.

## Контекст

У нас уже есть:
- ✅ Task агрегат с событиями (`internal/domain/task/`)
- ✅ Event Store инфраструктура
- ❌ Слой application logic отсутствует

Нужно создать слой, который будет:
- Оркестрировать бизнес-логику
- Координировать работу с агрегатами
- Управлять транзакциями
- Обрабатывать ошибки

## Архитектурные решения

### 1. Структура директорий

```
internal/
├── usecase/
│   ├── task/
│   │   ├── create_task.go
│   │   ├── change_status.go
│   │   ├── assign_task.go
│   │   ├── change_priority.go
│   │   ├── set_due_date.go
│   │   ├── commands.go          # Все команды
│   │   ├── results.go           # Результаты выполнения
│   │   └── errors.go            # Специфичные ошибки
│   ├── chat/                    # Будущие use cases для Chat
│   └── shared/
│       ├── interfaces.go        # Общие интерфейсы
│       └── base.go              # Базовая функциональность
```

### 2. Паттерн Command/Result

Каждый use case работает с командой и возвращает результат:

```go
// commands.go
package task

import (
    "time"
    "github.com/google/uuid"
)

// CreateTaskCommand содержит данные для создания задачи
type CreateTaskCommand struct {
    ChatID      uuid.UUID
    Title       string
    Priority    string        // "High", "Medium", "Low"
    AssigneeID  *uuid.UUID    // optional
    DueDate     *time.Time    // optional
    CreatedBy   uuid.UUID
}

// ChangeStatusCommand содержит данные для изменения статуса
type ChangeStatusCommand struct {
    TaskID    uuid.UUID
    NewStatus string        // "To Do", "In Progress", "Done"
    ChangedBy uuid.UUID
}

// AssignTaskCommand содержит данные для назначения исполнителя
type AssignTaskCommand struct {
    TaskID     uuid.UUID
    AssigneeID *uuid.UUID   // nil = снять assignee
    AssignedBy uuid.UUID
}

// ChangePriorityCommand содержит данные для изменения приоритета
type ChangePriorityCommand struct {
    TaskID    uuid.UUID
    Priority  string        // "High", "Medium", "Low"
    ChangedBy uuid.UUID
}

// SetDueDateCommand содержит данные для установки дедлайна
type SetDueDateCommand struct {
    TaskID    uuid.UUID
    DueDate   *time.Time    // nil = снять дедлайн
    SetBy     uuid.UUID
}
```

```go
// results.go
package task

import (
    "github.com/google/uuid"
    "flowra/internal/domain/task"
)

// TaskResult — результат выполнения use case
type TaskResult struct {
    TaskID   uuid.UUID
    Version  int
    Events   []task.Event
    Error    error
}

func (r TaskResult) IsSuccess() bool {
    return r.Error == nil
}

func (r TaskResult) IsFailure() bool {
    return r.Error != nil
}
```

### 3. Интерфейс Use Case

Все use cases реализуют общий интерфейс:

```go
// internal/usecase/shared/interfaces.go
package shared

import "context"

// UseCase — базовый интерфейс для всех use cases
type UseCase[TCommand any, TResult any] interface {
    Execute(ctx context.Context, cmd TCommand) (TResult, error)
}

// Validator — интерфейс для валидации команд
type Validator[T any] interface {
    Validate(cmd T) error
}
```

### 4. Базовая структура Use Case

```go
// internal/usecase/task/create_task.go
package task

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "flowra/internal/domain/task"
    "flowra/internal/infrastructure/eventstore"
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

func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Создание нового агрегата
    taskID := uuid.New()
    aggregate := task.NewTaskAggregate(taskID)

    // 3. Выполнение бизнес-операции
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

    // 4. Сохранение событий
    events := aggregate.UncommittedEvents()
    if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }

    // 5. Возврат результата
    return TaskResult{
        TaskID:  taskID,
        Version: len(events),
        Events:  events,
    }, nil
}

func (uc *CreateTaskUseCase) validate(cmd CreateTaskCommand) error {
    if cmd.ChatID == uuid.Nil {
        return ErrInvalidChatID
    }
    if cmd.Title == "" {
        return ErrEmptyTitle
    }
    if cmd.Priority != "" && !isValidPriority(cmd.Priority) {
        return ErrInvalidPriority
    }
    if cmd.CreatedBy == uuid.Nil {
        return ErrInvalidUserID
    }
    return nil
}

func isValidPriority(priority string) bool {
    return priority == "High" || priority == "Medium" || priority == "Low"
}
```

### 5. Обработка ошибок

```go
// errors.go
package task

import "errors"

var (
    // Validation errors
    ErrInvalidChatID    = errors.New("invalid chat ID")
    ErrEmptyTitle       = errors.New("task title cannot be empty")
    ErrInvalidPriority  = errors.New("invalid priority value")
    ErrInvalidStatus    = errors.New("invalid status value")
    ErrInvalidUserID    = errors.New("invalid user ID")
    ErrInvalidDate      = errors.New("invalid date value")

    // Business logic errors
    ErrTaskNotFound     = errors.New("task not found")
    ErrUnauthorized     = errors.New("user not authorized for this operation")
    ErrConcurrentUpdate = errors.New("concurrent update detected")
)
```

## Принципы проектирования

### 1. Single Responsibility
Каждый use case отвечает за одну операцию:
- `CreateTaskUseCase` — только создание
- `ChangeStatusUseCase` — только изменение статуса
- И т.д.

### 2. Dependency Injection
Use cases зависят от абстракций (интерфейсов), а не конкретных реализаций:
```go
type CreateTaskUseCase struct {
    eventStore eventstore.EventStore  // интерфейс
}
```

### 3. Fail Fast
Валидация выполняется в начале:
```go
if err := uc.validate(cmd); err != nil {
    return TaskResult{}, err
}
```

### 4. Explicit Error Handling
Каждая ошибка обрабатывается явно:
```go
if err := uc.eventStore.SaveEvents(...); err != nil {
    return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
}
```

### 5. Idempotency (V2)
В будущем можно добавить проверку идемпотентности:
```go
// Проверяем, не была ли команда выполнена ранее
if uc.isAlreadyProcessed(ctx, cmd.RequestID) {
    return uc.getPreviousResult(ctx, cmd.RequestID)
}
```

## Тестируемость

Use cases легко тестировать:

```go
func TestCreateTaskUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := NewCreateTaskUseCase(eventStore)

    cmd := CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        Priority:  "High",
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.TaskID)
    assert.Equal(t, 1, result.Version)
    assert.Len(t, result.Events, 1)
}

func TestCreateTaskUseCase_ValidationError(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := NewCreateTaskUseCase(eventStore)

    cmd := CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "", // пустой title
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    assert.Error(t, err)
    assert.ErrorIs(t, err, ErrEmptyTitle)
}
```

## Интеграция с HTTP Handler (будущее)

```go
// internal/handler/task_handler.go
func (h *TaskHandler) CreateTask(c echo.Context) error {
    var req CreateTaskRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, ErrorResponse{Message: "Invalid request"})
    }

    cmd := CreateTaskCommand{
        ChatID:    req.ChatID,
        Title:     req.Title,
        Priority:  req.Priority,
        CreatedBy: getUserIDFromContext(c),
    }

    result, err := h.createTaskUseCase.Execute(c.Request().Context(), cmd)
    if err != nil {
        return c.JSON(500, ErrorResponse{Message: err.Error()})
    }

    return c.JSON(201, CreateTaskResponse{
        TaskID:  result.TaskID,
        Version: result.Version,
    })
}
```

## Checklist

- [x] Создать структуру директорий `internal/usecase/task/`
- [x] Определить команды в `commands.go`
- [x] Определить результаты в `results.go`
- [x] Определить ошибки в `errors.go`
- [x] Создать общие интерфейсы в `internal/usecase/shared/`
- [x] Документировать паттерны в README.md
- [x] Создать EventStore интерфейс и in-memory реализацию

## Следующие шаги

После завершения этой задачи переходим к:
- **Task 02**: Реализация CreateTaskUseCase
- **Task 03**: Реализация ChangeStatusUseCase
- **Task 04**: Реализация AssignTaskUseCase
- **Task 05**: Реализация ChangePriorityUseCase и SetDueDateUseCase
- **Task 06**: Тестирование use cases

## Референсы

- [Domain Model: Task Aggregate](../02-domain-model.md)
- [Event Sourcing Infrastructure](../../internal/infrastructure/eventstore/)
- [Use Case Pattern](https://martinfowler.com/eaaCatalog/applicationFacade.html)
