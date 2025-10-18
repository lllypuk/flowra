# Task 03: ChangeStatus Use Case Implementation

**Дата:** 2025-10-17
**Статус:** Pending
**Зависимости:** Task 02 (CreateTask UseCase)
**Оценка:** 2-3 часа

## Цель

Реализовать use case для изменения статуса существующей задачи с учетом Event Sourcing и валидации допустимых переходов.

## Контекст

Изменение статуса — одна из самых частых операций в системе. Она происходит:
- При перетаскивании карточки на канбане (drag-n-drop)
- При использовании тега `#status <value>` в сообщении
- При использовании кнопки "Change Status" в UI

### Бизнес-требования

1. **Допустимые статусы для Task**:
   - "To Do" (начальный)
   - "In Progress"
   - "Done"

2. **Валидация**:
   - Task должна существовать
   - Новый статус должен быть валидным
   - Новый статус должен отличаться от текущего (идемпотентность)

3. **Событие**:
   - Генерируется `TaskStatusChangedEvent`
   - Содержит старый и новый статус

4. **Особенности**:
   - Нужно загружать существующий агрегат из Event Store
   - Применять все прошлые события для восстановления состояния
   - Добавить новое событие

## Реализация

### 1. Use Case

```go
// change_status.go
package task

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "teams-up/internal/domain/task"
    "teams-up/internal/infrastructure/eventstore"
)

// ChangeStatusUseCase обрабатывает изменение статуса задачи
type ChangeStatusUseCase struct {
    eventStore eventstore.EventStore
}

func NewChangeStatusUseCase(eventStore eventstore.EventStore) *ChangeStatusUseCase {
    return &ChangeStatusUseCase{
        eventStore: eventStore,
    }
}

// Execute изменяет статус задачи
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Загрузка событий из Event Store
    events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return TaskResult{}, ErrTaskNotFound
    }

    // 3. Восстановление агрегата из событий
    aggregate := task.NewTaskAggregate(cmd.TaskID)
    if err := aggregate.LoadFromHistory(events); err != nil {
        return TaskResult{}, fmt.Errorf("failed to load aggregate: %w", err)
    }

    // 4. Выполнение бизнес-операции
    err = aggregate.ChangeStatus(cmd.NewStatus, cmd.ChangedBy)
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to change status: %w", err)
    }

    // 5. Получение только новых событий
    newEvents := aggregate.UncommittedEvents()

    // Если новых событий нет (идемпотентность), возвращаем успех
    if len(newEvents) == 0 {
        return TaskResult{
            TaskID:  cmd.TaskID,
            Version: len(events),
            Events:  []task.Event{},
        }, nil
    }

    // 6. Сохранение новых событий
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
func (uc *ChangeStatusUseCase) validate(cmd ChangeStatusCommand) error {
    if cmd.TaskID == uuid.Nil {
        return ErrInvalidTaskID
    }

    if cmd.NewStatus == "" {
        return ErrEmptyStatus
    }

    if !isValidTaskStatus(cmd.NewStatus) {
        return fmt.Errorf("%w: must be 'To Do', 'In Progress', or 'Done'", ErrInvalidStatus)
    }

    if cmd.ChangedBy == uuid.Nil {
        return ErrInvalidUserID
    }

    return nil
}

// isValidTaskStatus проверяет валидность статуса для Task
func isValidTaskStatus(status string) bool {
    return status == "To Do" || status == "In Progress" || status == "Done"
}
```

### 2. Дополнительные ошибки

```go
// errors.go (добавить к существующим)
var (
    // ... существующие ошибки ...

    ErrInvalidTaskID = errors.New("invalid task ID")
    ErrEmptyStatus   = errors.New("status cannot be empty")
)
```

## Unit тесты

```go
// change_status_test.go
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

func TestChangeStatusUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    changeStatusUseCase := taskusecase.NewChangeStatusUseCase(eventStore)

    // Создаем задачу
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Меняем статус
    userID := uuid.New()
    changeCmd := taskusecase.ChangeStatusCommand{
        TaskID:    createResult.TaskID,
        NewStatus: "In Progress",
        ChangedBy: userID,
    }

    // Act
    result, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, createResult.TaskID, result.TaskID)
    assert.Equal(t, 2, result.Version) // 1 событие создания + 1 событие изменения статуса
    require.Len(t, result.Events, 1)

    // Проверяем событие
    event, ok := result.Events[0].(task.TaskStatusChangedEvent)
    require.True(t, ok, "Expected TaskStatusChangedEvent")
    assert.Equal(t, createResult.TaskID, event.TaskID)
    assert.Equal(t, "To Do", event.OldStatus)
    assert.Equal(t, "In Progress", event.NewStatus)
    assert.Equal(t, userID, event.ChangedBy)

    // Проверяем, что события сохранены
    storedEvents, err := eventStore.LoadEvents(context.Background(), result.TaskID.String())
    require.NoError(t, err)
    assert.Len(t, storedEvents, 2)
}

func TestChangeStatusUseCase_MultipleTransitions(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    changeStatusUseCase := taskusecase.NewChangeStatusUseCase(eventStore)

    // Создаем задачу
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    userID := uuid.New()

    // Act & Assert
    // To Do → In Progress
    result1, err := changeStatusUseCase.Execute(context.Background(), taskusecase.ChangeStatusCommand{
        TaskID:    createResult.TaskID,
        NewStatus: "In Progress",
        ChangedBy: userID,
    })
    require.NoError(t, err)
    assert.Equal(t, 2, result1.Version)

    // In Progress → Done
    result2, err := changeStatusUseCase.Execute(context.Background(), taskusecase.ChangeStatusCommand{
        TaskID:    createResult.TaskID,
        NewStatus: "Done",
        ChangedBy: userID,
    })
    require.NoError(t, err)
    assert.Equal(t, 3, result2.Version)

    // Проверяем полную историю
    storedEvents, err := eventStore.LoadEvents(context.Background(), createResult.TaskID.String())
    require.NoError(t, err)
    assert.Len(t, storedEvents, 3) // Create + 2x StatusChanged
}

func TestChangeStatusUseCase_Idempotent(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    changeStatusUseCase := taskusecase.NewChangeStatusUseCase(eventStore)

    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Первое изменение статуса
    changeCmd := taskusecase.ChangeStatusCommand{
        TaskID:    createResult.TaskID,
        NewStatus: "In Progress",
        ChangedBy: uuid.New(),
    }
    result1, err := changeStatusUseCase.Execute(context.Background(), changeCmd)
    require.NoError(t, err)
    assert.Len(t, result1.Events, 1)

    // Act: Повторное изменение на тот же статус
    result2, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

    // Assert: Должно быть успешно, но без новых событий
    require.NoError(t, err)
    assert.Empty(t, result2.Events, "No new events should be generated for idempotent operation")
    assert.Equal(t, result1.Version, result2.Version, "Version should not change")
}

func TestChangeStatusUseCase_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        cmd         taskusecase.ChangeStatusCommand
        expectedErr error
    }{
        {
            name: "Empty TaskID",
            cmd: taskusecase.ChangeStatusCommand{
                TaskID:    uuid.Nil,
                NewStatus: "Done",
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidTaskID,
        },
        {
            name: "Empty Status",
            cmd: taskusecase.ChangeStatusCommand{
                TaskID:    uuid.New(),
                NewStatus: "",
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrEmptyStatus,
        },
        {
            name: "Invalid Status",
            cmd: taskusecase.ChangeStatusCommand{
                TaskID:    uuid.New(),
                NewStatus: "Completed", // не существует для Task
                ChangedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidStatus,
        },
        {
            name: "Empty ChangedBy",
            cmd: taskusecase.ChangeStatusCommand{
                TaskID:    uuid.New(),
                NewStatus: "Done",
                ChangedBy: uuid.Nil,
            },
            expectedErr: taskusecase.ErrInvalidUserID,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            eventStore := eventstore.NewInMemoryEventStore()
            useCase := taskusecase.NewChangeStatusUseCase(eventStore)

            // Act
            result, err := useCase.Execute(context.Background(), tt.cmd)

            // Assert
            require.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedErr)
            assert.Empty(t, result.Events)
        })
    }
}

func TestChangeStatusUseCase_TaskNotFound(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := taskusecase.NewChangeStatusUseCase(eventStore)

    cmd := taskusecase.ChangeStatusCommand{
        TaskID:    uuid.New(), // не существует
        NewStatus: "Done",
        ChangedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.ErrorIs(t, err, taskusecase.ErrTaskNotFound)
    assert.Empty(t, result.Events)
}

func TestChangeStatusUseCase_ConcurrentUpdate(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    changeStatusUseCase := taskusecase.NewChangeStatusUseCase(eventStore)

    // Создаем задачу
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Первое изменение
    _, err = changeStatusUseCase.Execute(context.Background(), taskusecase.ChangeStatusCommand{
        TaskID:    createResult.TaskID,
        NewStatus: "In Progress",
        ChangedBy: uuid.New(),
    })
    require.NoError(t, err)

    // Act: Пытаемся изменить с устаревшей версией
    // (Event Store должен отклонить, если expectedVersion не совпадает)
    // Это будет протестировано на уровне интеграционных тестов с реальной БД
}
```

## Сценарии использования

### 1. Drag-n-drop на канбане

```go
// В handler'е
func (h *BoardHandler) MoveCard(c echo.Context) error {
    var req MoveCardRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, ErrorResponse{Message: "Invalid request"})
    }

    // Преобразуем колонку в статус
    newStatus := columnToStatus(req.TargetColumn) // "todo" → "To Do"

    cmd := taskusecase.ChangeStatusCommand{
        TaskID:    req.TaskID,
        NewStatus: newStatus,
        ChangedBy: getUserIDFromContext(c),
    }

    result, err := h.changeStatusUseCase.Execute(c.Request().Context(), cmd)
    if err != nil {
        if errors.Is(err, taskusecase.ErrTaskNotFound) {
            return c.JSON(404, ErrorResponse{Message: "Task not found"})
        }
        return c.JSON(500, ErrorResponse{Message: "Failed to move card"})
    }

    // Отправляем обновление через WebSocket всем участникам
    h.wsHub.BroadcastToChat(req.ChatID, UpdateMessage{
        Type:    "task_status_changed",
        TaskID:  result.TaskID,
        Version: result.Version,
    })

    return c.JSON(200, MoveCardResponse{Success: true})
}
```

### 2. Тег в сообщении

```go
// В tag processor
func (p *TagProcessor) ProcessStatusTag(msg Message, tag ParsedTag) error {
    cmd := taskusecase.ChangeStatusCommand{
        TaskID:    msg.RelatedTaskID, // из контекста чата
        NewStatus: tag.Value,          // "Done"
        ChangedBy: msg.AuthorID,
    }

    result, err := p.changeStatusUseCase.Execute(context.Background(), cmd)
    if err != nil {
        // Отправляем ошибку в чат
        p.sendErrorMessage(msg.ChatID, fmt.Sprintf("❌ %s", err.Error()))
        return err
    }

    // Отправляем подтверждение
    p.sendConfirmation(msg.ChatID, fmt.Sprintf("✅ Status changed to %s", cmd.NewStatus))
    return nil
}
```

## Optimistic Locking

Event Store должен проверять expectedVersion при сохранении:

```go
// В EventStore.SaveEvents
func (es *MongoDBEventStore) SaveEvents(ctx context.Context, aggregateID string, events []Event, expectedVersion int) error {
    // Начинаем транзакцию
    tx, err := es.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Проверяем текущую версию
    var currentVersion int
    err = tx.QueryRow(`
        SELECT COALESCE(MAX(version), 0)
        FROM events
        WHERE aggregate_id = $1
    `, aggregateID).Scan(&currentVersion)
    if err != nil {
        return err
    }

    // Optimistic locking check
    if currentVersion != expectedVersion {
        return ErrConcurrentUpdate
    }

    // Сохраняем события
    for i, event := range events {
        version := expectedVersion + i + 1
        // INSERT INTO events ...
    }

    return tx.Commit()
}
```

## Checklist

- [ ] Реализовать `ChangeStatusUseCase` в `change_status.go`
- [ ] Добавить валидацию статуса для Task
- [ ] Реализовать загрузку агрегата из Event Store
- [ ] Обработать идемпотентность (не создавать событие, если статус не изменился)
- [ ] Написать unit тесты для успешного случая
- [ ] Написать тест для множественных переходов
- [ ] Написать тест для идемпотентности
- [ ] Написать тесты для validation errors
- [ ] Написать тест для TaskNotFound
- [ ] Проверить покрытие тестами (>80%)

## Критерии приемки

- ✅ Use case изменяет статус существующей задачи
- ✅ Агрегат корректно восстанавливается из событий
- ✅ Валидируются допустимые статусы
- ✅ Идемпотентность работает (повторный вызов не создает событие)
- ✅ Обрабатывается случай "задача не найдена"
- ✅ Optimistic locking предотвращает конфликты
- ✅ Покрытие тестами >80%

## Следующие шаги

После завершения переходим к:
- **Task 04**: AssignTaskUseCase
- **Task 05**: ChangePriorityUseCase и SetDueDateUseCase

## Референсы

- [Task 02: CreateTask UseCase](02-create-task-usecase.md)
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Optimistic Locking](https://en.wikipedia.org/wiki/Optimistic_concurrency_control)
