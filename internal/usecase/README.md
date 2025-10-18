# Use Case Layer

Слой application logic (use cases) для управления бизнес-операциями с агрегатами.

## Архитектура

### Структура директорий

```
internal/usecase/
├── shared/
│   ├── interfaces.go      # Общие интерфейсы (UseCase, Validator, CommandHandler)
│   └── base.go            # Базовая функциональность для всех use cases
└── task/
    ├── commands.go        # Команды для операций с задачами
    ├── results.go         # Результаты выполнения операций
    ├── errors.go          # Специфичные ошибки use cases
    ├── create_task.go     # Use case: создание задачи
    ├── change_status.go   # Use case: изменение статуса
    ├── assign_task.go     # Use case: назначение исполнителя
    ├── change_priority.go # Use case: изменение приоритета
    └── set_due_date.go    # Use case: установка дедлайна
```

## Паттерны и принципы

### 1. Command/Result Pattern

Каждый use case работает с командой (входные данные) и возвращает результат (выходные данные):

```go
type CreateTaskCommand struct {
    ChatID     uuid.UUID
    Title      string
    EntityType task.EntityType
    Priority   task.Priority
    CreatedBy  uuid.UUID
}

type TaskResult struct {
    TaskID  uuid.UUID
    Version int
    Events  []event.DomainEvent
    Success bool
}
```

### 2. UseCase Interface

Все use cases реализуют общий интерфейс:

```go
type UseCase[TCommand any, TResult any] interface {
    Execute(ctx context.Context, cmd TCommand) (TResult, error)
}
```

Это позволяет:
- Единообразно работать с разными use cases
- Легко тестировать через моки
- Применять декораторы (логирование, метрики, и т.д.)

### 3. Dependency Injection

Use cases зависят от абстракций (интерфейсов):

```go
type CreateTaskUseCase struct {
    eventStore eventstore.EventStore  // интерфейс, а не конкретная реализация
}

func NewCreateTaskUseCase(eventStore eventstore.EventStore) *CreateTaskUseCase {
    return &CreateTaskUseCase{
        eventStore: eventStore,
    }
}
```

### 4. Fail Fast

Валидация выполняется в начале метода Execute:

```go
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
    // 1. Валидация
    if err := uc.validate(cmd); err != nil {
        return TaskResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Бизнес-логика
    // ...
}
```

### 5. Explicit Error Handling

Каждая ошибка обрабатывается явно с использованием `fmt.Errorf` и `%w`:

```go
if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
    return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
}
```

### 6. Single Responsibility

Каждый use case отвечает за одну операцию:
- `CreateTaskUseCase` — только создание задачи
- `ChangeStatusUseCase` — только изменение статуса
- `AssignTaskUseCase` — только назначение исполнителя

## Работа с Event Sourcing

Use cases работают с агрегатами через Event Store:

```go
// 1. Создание нового агрегата
taskID := uuid.New()
aggregate := task.NewTaskAggregate(taskID)

// 2. Выполнение бизнес-операции
err := aggregate.Create(cmd.ChatID, cmd.Title, cmd.Priority, cmd.CreatedBy)
if err != nil {
    return TaskResult{}, err
}

// 3. Получение событий
events := aggregate.UncommittedEvents()

// 4. Сохранение в Event Store
if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
    return TaskResult{}, err
}
```

### Optimistic Locking

При изменении существующих агрегатов используется optimistic locking:

```go
// Загрузка агрегата
events, err := uc.eventStore.LoadEvents(ctx, taskID.String())
if err != nil {
    return TaskResult{}, err
}

// Восстановление состояния
aggregate := task.NewTaskAggregate(taskID)
aggregate.ReplayEvents(events)

// Получаем текущую версию
expectedVersion := aggregate.Version()

// Выполняем операцию
aggregate.ChangeStatus(cmd.NewStatus, cmd.ChangedBy)

// Сохраняем с проверкой версии
newEvents := aggregate.UncommittedEvents()
if err := uc.eventStore.SaveEvents(ctx, taskID.String(), newEvents, expectedVersion); err != nil {
    if errors.Is(err, eventstore.ErrConcurrencyConflict) {
        return TaskResult{}, ErrConcurrentUpdate
    }
    return TaskResult{}, err
}
```

## Тестирование

Use cases легко тестировать благодаря dependency injection:

```go
func TestCreateTaskUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    useCase := NewCreateTaskUseCase(eventStore)

    cmd := CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        Priority:  task.PriorityHigh,
        CreatedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    assert.NoError(t, err)
    assert.True(t, result.IsSuccess())
    assert.NotEqual(t, uuid.Nil, result.TaskID)
}
```

## Обработка ошибок

Use cases определяют собственные ошибки в `errors.go`:

- **Validation errors** - ошибки валидации входных данных
  - `ErrInvalidChatID`
  - `ErrEmptyTitle`
  - `ErrInvalidPriority`

- **Business logic errors** - ошибки бизнес-логики
  - `ErrTaskNotFound`
  - `ErrUnauthorized`
  - `ErrConcurrentUpdate`

Все ошибки создаются через `errors.New()` и могут быть проверены через `errors.Is()`.

## Интеграция с другими слоями

### HTTP Handlers

```go
func (h *TaskHandler) CreateTask(c echo.Context) error {
    var req CreateTaskRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, ErrorResponse{Message: "Invalid request"})
    }

    cmd := CreateTaskCommand{
        ChatID:    req.ChatID,
        Title:     req.Title,
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

### Tag Parser

```go
// Парсинг тега из сообщения
tag := parser.ParseTag(message)

// Конвертация в команду
cmd := ConvertTagToCommand(tag)

// Выполнение use case
result, err := createTaskUseCase.Execute(ctx, cmd)
```

## Следующие шаги

После завершения архитектуры переходим к реализации конкретных use cases:

1. **CreateTaskUseCase** - создание новой задачи
2. **ChangeStatusUseCase** - изменение статуса существующей задачи
3. **AssignTaskUseCase** - назначение исполнителя
4. **ChangePriorityUseCase** - изменение приоритета
5. **SetDueDateUseCase** - установка/снятие дедлайна

## Референсы

- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Use Case Pattern](https://martinfowler.com/eaaCatalog/applicationFacade.html)
- [Command Pattern](https://refactoring.guru/design-patterns/command)
- [CQRS](https://martinfowler.com/bliki/CQRS.html)
