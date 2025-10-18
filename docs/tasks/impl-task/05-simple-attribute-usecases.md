# Task 05: Simple Attribute Use Cases (Priority & DueDate)

**Дата:** 2025-10-17
**Статус:** Pending
**Зависимости:** Task 04 (AssignTask UseCase)
**Оценка:** 2-3 часа

## Цель

Реализовать use cases для изменения простых атрибутов задачи: приоритета и дедлайна. Эти use cases похожи друг на друга и не требуют внешних зависимостей.

## Контекст

ChangePriority и SetDueDate — простые операции, которые:
- Не требуют внешних зависимостей (в отличие от AssignTask)
- Имеют схожую структуру
- Валидируют только формат данных

### Бизнес-требования

#### ChangePriority

1. **Допустимые значения**: "High", "Medium", "Low"
2. **Валидация**: Case-sensitive, только из списка
3. **Идемпотентность**: Повторная установка того же приоритета не создает событие
4. **Событие**: `TaskPriorityChangedEvent`

#### SetDueDate

1. **Формат**: `time.Time` или `nil` (снять дедлайн)
2. **Валидация**: Дата может быть в прошлом (просроченная задача)
3. **Идемпотентность**: Повторная установка той же даты не создает событие
4. **Событие**: `TaskDueDateChangedEvent`

## Реализация

### 1. ChangePriorityUseCase

```go
// change_priority.go
package task

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
)

// ChangePriorityUseCase обрабатывает изменение приоритета задачи
type ChangePriorityUseCase struct {
    eventStore eventstore.EventStore
}

func NewChangePriorityUseCase(eventStore eventstore.EventStore) *ChangePriorityUseCase {
    return &ChangePriorityUseCase{
        eventStore: eventStore,
    }
}

// Execute изменяет приоритет задачи
func (uc *ChangePriorityUseCase) Execute(ctx context.Context, cmd ChangePriorityCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Загрузка событий
    events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return TaskResult{}, ErrTaskNotFound
    }

    // 3. Восстановление агрегата
    aggregate := task.NewTaskAggregate(cmd.TaskID)
    if err := aggregate.LoadFromHistory(events); err != nil {
        return TaskResult{}, fmt.Errorf("failed to load aggregate: %w", err)
    }

    // 4. Выполнение бизнес-операции
    err = aggregate.ChangePriority(cmd.Priority, cmd.ChangedBy)
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to change priority: %w", err)
    }

    // 5. Получение новых событий
    newEvents := aggregate.UncommittedEvents()

    // Идемпотентность
    if len(newEvents) == 0 {
        return TaskResult{
            TaskID:  cmd.TaskID,
            Version: len(events),
            Events:  []task.Event{},
        }, nil
    }

    // 6. Сохранение
    expectedVersion := len(events)
    if err := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }

    // 7. Возврат результата
    return TaskResult{
        TaskID:  cmd.TaskID,
        Version: expectedVersion + len(newEvents),
        Events:  newEvents,
    }, nil
}

// validate проверяет корректность команды
func (uc *ChangePriorityUseCase) validate(cmd ChangePriorityCommand) error {
    if cmd.TaskID == uuid.Nil {
        return ErrInvalidTaskID
    }

    if cmd.Priority == "" {
        return ErrEmptyPriority
    }

    if !isValidPriority(cmd.Priority) {
        return fmt.Errorf("%w: must be High, Medium, or Low", ErrInvalidPriority)
    }

    if cmd.ChangedBy == uuid.Nil {
        return ErrInvalidUserID
    }

    return nil
}
```

### 2. SetDueDateUseCase

```go
// set_due_date.go
package task

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
)

// SetDueDateUseCase обрабатывает установку дедлайна задачи
type SetDueDateUseCase struct {
    eventStore eventstore.EventStore
}

func NewSetDueDateUseCase(eventStore eventstore.EventStore) *SetDueDateUseCase {
    return &SetDueDateUseCase{
        eventStore: eventStore,
    }
}

// Execute устанавливает дедлайн задачи
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Загрузка событий
    events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return TaskResult{}, ErrTaskNotFound
    }

    // 3. Восстановление агрегата
    aggregate := task.NewTaskAggregate(cmd.TaskID)
    if err := aggregate.LoadFromHistory(events); err != nil {
        return TaskResult{}, fmt.Errorf("failed to load aggregate: %w", err)
    }

    // 4. Выполнение бизнес-операции
    err = aggregate.SetDueDate(cmd.DueDate, cmd.SetBy)
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to set due date: %w", err)
    }

    // 5. Получение новых событий
    newEvents := aggregate.UncommittedEvents()

    // Идемпотентность
    if len(newEvents) == 0 {
        return TaskResult{
            TaskID:  cmd.TaskID,
            Version: len(events),
            Events:  []task.Event{},
        }, nil
    }

    // 6. Сохранение
    expectedVersion := len(events)
    if err := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }

    // 7. Возврат результата
    return TaskResult{
        TaskID:  cmd.TaskID,
        Version: expectedVersion + len(newEvents),
        Events:  newEvents,
    }, nil
}

// validate проверяет корректность команды
func (uc *SetDueDateUseCase) validate(cmd SetDueDateCommand) error {
    if cmd.TaskID == uuid.Nil {
        return ErrInvalidTaskID
    }

    // DueDate может быть nil (снятие дедлайна) — это валидно

    // Sanity check: дата не должна быть слишком далеко в прошлом
    if cmd.DueDate != nil && cmd.DueDate.Year() < 2020 {
        return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
    }

    if cmd.SetBy == uuid.Nil {
        return ErrInvalidUserID
    }

    return nil
}
```

### 3. Дополнительные ошибки

```go
// errors.go (добавить к существующим)
var (
    // ... существующие ошибки ...

    ErrEmptyPriority = errors.New("priority cannot be empty")
)
```

## Unit тесты

### ChangePriorityUseCase Tests

```go
// change_priority_test.go
package task_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
    taskusecase "teams-up/internal/usecase/task"
)

func TestChangePriorityUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    priorityUseCase := taskusecase.NewChangePriorityUseCase(eventStore)

    // Создаем задачу с Medium priority (default)
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Меняем приоритет
    userID := uuid.New()
    priorityCmd := taskusecase.ChangePriorityCommand{
        TaskID:    createResult.TaskID,
        Priority:  "High",
        ChangedBy: userID,
    }

    // Act
    result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, 2, result.Version)
    require.Len(t, result.Events, 1)

    event, ok := result.Events[0].(task.TaskPriorityChangedEvent)
    require.True(t, ok)
    assert.Equal(t, "Medium", event.OldPriority)
    assert.Equal(t, "High", event.NewPriority)
    assert.Equal(t, userID, event.ChangedBy)
}

func TestChangePriorityUseCase_AllPriorities(t *testing.T) {
    priorities := []string{"High", "Medium", "Low"}

    for _, priority := range priorities {
        t.Run(priority, func(t *testing.T) {
            // Arrange
            eventStore := eventstore.NewInMemoryEventStore()
            createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
            priorityUseCase := taskusecase.NewChangePriorityUseCase(eventStore)

            createCmd := taskusecase.CreateTaskCommand{
                ChatID:    uuid.New(),
                Title:     "Test Task",
                Priority:  "Low",
                CreatedBy: uuid.New(),
            }
            createResult, err := createUseCase.Execute(context.Background(), createCmd)
            require.NoError(t, err)

            // Act
            priorityCmd := taskusecase.ChangePriorityCommand{
                TaskID:    createResult.TaskID,
                Priority:  priority,
                ChangedBy: uuid.New(),
            }
            result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

            // Assert
            require.NoError(t, err)
            event := result.Events[0].(task.TaskPriorityChangedEvent)
            assert.Equal(t, priority, event.NewPriority)
        })
    }
}

func TestChangePriorityUseCase_Idempotent(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    priorityUseCase := taskusecase.NewChangePriorityUseCase(eventStore)

    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        Priority:  "High",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Повторная установка того же приоритета
    priorityCmd := taskusecase.ChangePriorityCommand{
        TaskID:    createResult.TaskID,
        Priority:  "High",
        ChangedBy: uuid.New(),
    }
    result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

    // Assert
    require.NoError(t, err)
    assert.Empty(t, result.Events)
    assert.Equal(t, 1, result.Version)
}

func TestChangePriorityUseCase_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        cmd         taskusecase.ChangePriorityCommand
        expectedErr error
    }{
        {
            name: "Empty TaskID",
            cmd: taskusecase.ChangePriorityCommand{
                TaskID:    uuid.Nil,
                Priority:  "High",
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidTaskID,
        },
        {
            name: "Empty Priority",
            cmd: taskusecase.ChangePriorityCommand{
                TaskID:    uuid.New(),
                Priority:  "",
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrEmptyPriority,
        },
        {
            name: "Invalid Priority",
            cmd: taskusecase.ChangePriorityCommand{
                TaskID:    uuid.New(),
                Priority:  "Urgent",
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidPriority,
        },
        {
            name: "Case Sensitive",
            cmd: taskusecase.ChangePriorityCommand{
                TaskID:    uuid.New(),
                Priority:  "high", // lowercase
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidPriority,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            eventStore := eventstore.NewInMemoryEventStore()
            useCase := taskusecase.NewChangePriorityUseCase(eventStore)

            result, err := useCase.Execute(context.Background(), tt.cmd)

            require.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedErr)
            assert.Empty(t, result.Events)
        })
    }
}
```

### SetDueDateUseCase Tests

```go
// set_due_date_test.go
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

func TestSetDueDateUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    dueDateUseCase := taskusecase.NewSetDueDateUseCase(eventStore)

    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Устанавливаем дедлайн
    dueDate := time.Now().Add(7 * 24 * time.Hour) // через неделю
    userID := uuid.New()
    dueDateCmd := taskusecase.SetDueDateCommand{
        TaskID:  createResult.TaskID,
        DueDate: &dueDate,
        SetBy:   userID,
    }

    // Act
    result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, 2, result.Version)
    require.Len(t, result.Events, 1)

    event, ok := result.Events[0].(task.TaskDueDateChangedEvent)
    require.True(t, ok)
    assert.Nil(t, event.OldDueDate)
    assert.NotNil(t, event.NewDueDate)
    assert.Equal(t, dueDate.Unix(), event.NewDueDate.Unix())
    assert.Equal(t, userID, event.ChangedBy)
}

func TestSetDueDateUseCase_RemoveDueDate(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    dueDateUseCase := taskusecase.NewSetDueDateUseCase(eventStore)

    // Создаем задачу с дедлайном
    dueDate := time.Now().Add(7 * 24 * time.Hour)
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        DueDate:   &dueDate,
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Снимаем дедлайн (nil)
    dueDateCmd := taskusecase.SetDueDateCommand{
        TaskID:  createResult.TaskID,
        DueDate: nil,
        SetBy:   uuid.New(),
    }
    result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

    // Assert
    require.NoError(t, err)
    require.Len(t, result.Events, 1)

    event := result.Events[0].(task.TaskDueDateChangedEvent)
    assert.NotNil(t, event.OldDueDate)
    assert.Nil(t, event.NewDueDate)
}

func TestSetDueDateUseCase_Idempotent(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    dueDateUseCase := taskusecase.NewSetDueDateUseCase(eventStore)

    dueDate := time.Now().Add(7 * 24 * time.Hour)
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        DueDate:   &dueDate,
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Повторная установка той же даты
    dueDateCmd := taskusecase.SetDueDateCommand{
        TaskID:  createResult.TaskID,
        DueDate: &dueDate,
        SetBy:   uuid.New(),
    }
    result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

    // Assert
    require.NoError(t, err)
    assert.Empty(t, result.Events)
    assert.Equal(t, 1, result.Version)
}

func TestSetDueDateUseCase_PastDate(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    dueDateUseCase := taskusecase.NewSetDueDateUseCase(eventStore)

    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Устанавливаем дату в прошлом (просроченная задача)
    pastDate := time.Now().Add(-7 * 24 * time.Hour)
    dueDateCmd := taskusecase.SetDueDateCommand{
        TaskID:  createResult.TaskID,
        DueDate: &pastDate,
        SetBy:   uuid.New(),
    }
    result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

    // Assert: Должно быть успешно (дата в прошлом допустима)
    require.NoError(t, err)
    assert.Len(t, result.Events, 1)
}

func TestSetDueDateUseCase_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        cmd         taskusecase.SetDueDateCommand
        expectedErr error
    }{
        {
            name: "Empty TaskID",
            cmd: taskusecase.SetDueDateCommand{
                TaskID:  uuid.Nil,
                DueDate: ptr(time.Now()),
                SetBy:   uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidTaskID,
        },
        {
            name: "Date Too Far in Past",
            cmd: taskusecase.SetDueDateCommand{
                TaskID:  uuid.New(),
                DueDate: ptr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
                SetBy:   uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidDate,
        },
        {
            name: "Empty SetBy",
            cmd: taskusecase.SetDueDateCommand{
                TaskID:  uuid.New(),
                DueDate: ptr(time.Now()),
                SetBy:   uuid.Nil,
            },
            expectedErr: taskusecase.ErrInvalidUserID,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            eventStore := eventstore.NewInMemoryEventStore()
            useCase := taskusecase.NewSetDueDateUseCase(eventStore)

            result, err := useCase.Execute(context.Background(), tt.cmd)

            require.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedErr)
            assert.Empty(t, result.Events)
        })
    }
}

func ptr[T any](v T) *T {
    return &v
}
```

## Паттерн для будущих Use Cases

Эти два use case демонстрируют общий паттерн:

```go
func (uc *SomeUseCase) Execute(ctx context.Context, cmd SomeCommand) (TaskResult, error) {
    // 1. Validate
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Load events
    events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
    }
    if len(events) == 0 {
        return TaskResult{}, ErrTaskNotFound
    }

    // 3. Rebuild aggregate
    aggregate := task.NewTaskAggregate(cmd.TaskID)
    if err := aggregate.LoadFromHistory(events); err != nil {
        return TaskResult{}, fmt.Errorf("failed to load aggregate: %w", err)
    }

    // 4. Execute business operation
    err = aggregate.SomeOperation(cmd.Params...)
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to execute: %w", err)
    }

    // 5. Get new events (idempotency check)
    newEvents := aggregate.UncommittedEvents()
    if len(newEvents) == 0 {
        return TaskResult{TaskID: cmd.TaskID, Version: len(events), Events: []task.Event{}}, nil
    }

    // 6. Save events
    expectedVersion := len(events)
    if err := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }

    // 7. Return result
    return TaskResult{
        TaskID:  cmd.TaskID,
        Version: expectedVersion + len(newEvents),
        Events:  newEvents,
    }, nil
}
```

## Checklist

- [ ] Реализовать `ChangePriorityUseCase` в `change_priority.go`
- [ ] Реализовать `SetDueDateUseCase` в `set_due_date.go`
- [ ] Добавить валидацию приоритетов (case-sensitive)
- [ ] Добавить валидацию дат (sanity check)
- [ ] Написать тесты для ChangePriority (success, all priorities, idempotent, validation)
- [ ] Написать тесты для SetDueDate (set, remove, idempotent, past date, validation)
- [ ] Проверить покрытие тестами (>80%)
- [ ] Запустить `golangci-lint run`

## Критерии приемки

- ✅ ChangePriorityUseCase работает для всех приоритетов
- ✅ SetDueDateUseCase устанавливает и снимает дедлайн
- ✅ Валидация работает корректно
- ✅ Идемпотентность реализована
- ✅ Даты в прошлом допустимы
- ✅ Покрытие тестами >80%

## Следующие шаги

После завершения переходим к:
- **Task 06**: Use Case Testing Strategy — общая стратегия тестирования

## Референсы

- [Task 04: AssignTask UseCase](04-assign-task-usecase.md)
- [Use Case Architecture](01-usecase-architecture.md)
