# Task 04: AssignTask Use Case Implementation

**Дата:** 2025-10-17
**Статус:** Pending
**Зависимости:** Task 03 (ChangeStatus UseCase)
**Оценка:** 2-3 часа

## Цель

Реализовать use case для назначения исполнителя задачи с валидацией пользователя и поддержкой снятия назначения.

## Контекст

Назначение исполнителя — критичная операция для task management. Она происходит:
- При использовании тега `#assignee @username` в сообщении
- При выборе assignee через UI dropdown
- При drag-n-drop пользователя на карточку (будущее)

### Бизнес-требования

1. **Назначение исполнителя**:
   - Можно назначить любого пользователя системы
   - AssigneeID хранится в состоянии задачи
   - Генерируется `TaskAssigneeChangedEvent`

2. **Снятие назначения**:
   - `AssigneeID = nil` означает "снять assignee"
   - Генерируется то же событие с `NewAssigneeID = nil`

3. **Валидация**:
   - Task должна существовать
   - Пользователь должен существовать (если назначается)
   - Идемпотентность: повторное назначение того же пользователя не создает событие

4. **Зависимости**:
   - Нужен `UserRepository` для проверки существования пользователя
   - Это первый use case с внешней зависимостью

## Архитектурное решение

### Dependency Injection

```go
type AssignTaskUseCase struct {
    eventStore     eventstore.EventStore
    userRepository UserRepository  // ← новая зависимость
}
```

### User Repository Interface

```go
// internal/usecase/shared/interfaces.go
package shared

import (
    "context"
    "github.com/google/uuid"
)

// UserRepository предоставляет доступ к информации о пользователях
type UserRepository interface {
    // Exists проверяет, существует ли пользователь
    Exists(ctx context.Context, userID uuid.UUID) (bool, error)

    // GetByUsername ищет пользователя по username (для будущего парсинга @mentions)
    GetByUsername(ctx context.Context, username string) (*User, error)
}

// User — минимальная информация о пользователе
type User struct {
    ID       uuid.UUID
    Username string
    FullName string
}
```

## Реализация

### 1. Use Case

```go
// assign_task.go
package task

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "flowra/internal/domain/task"
    "flowra/internal/infrastructure/eventstore"
    "flowra/internal/usecase/shared"
)

// AssignTaskUseCase обрабатывает назначение исполнителя задачи
type AssignTaskUseCase struct {
    eventStore     eventstore.EventStore
    userRepository shared.UserRepository
}

func NewAssignTaskUseCase(
    eventStore eventstore.EventStore,
    userRepository shared.UserRepository,
) *AssignTaskUseCase {
    return &AssignTaskUseCase{
        eventStore:     eventStore,
        userRepository: userRepository,
    }
}

// Execute назначает исполнителя задаче
func (uc *AssignTaskUseCase) Execute(ctx context.Context, cmd AssignTaskCommand) (TaskResult, error) {
    // 1. Валидация команды
    if err := uc.validate(ctx, cmd); err != nil {
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
    err = aggregate.Assign(cmd.AssigneeID, cmd.AssignedBy)
    if err != nil {
        return TaskResult{}, fmt.Errorf("failed to assign task: %w", err)
    }

    // 5. Получение новых событий
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
func (uc *AssignTaskUseCase) validate(ctx context.Context, cmd AssignTaskCommand) error {
    if cmd.TaskID == uuid.Nil {
        return ErrInvalidTaskID
    }

    if cmd.AssignedBy == uuid.Nil {
        return ErrInvalidUserID
    }

    // Если AssigneeID указан (не снятие assignee), проверяем существование пользователя
    if cmd.AssigneeID != nil && *cmd.AssigneeID != uuid.Nil {
        exists, err := uc.userRepository.Exists(ctx, *cmd.AssigneeID)
        if err != nil {
            return fmt.Errorf("failed to check user existence: %w", err)
        }
        if !exists {
            return fmt.Errorf("%w: user %s not found", ErrUserNotFound, cmd.AssigneeID)
        }
    }

    return nil
}
```

### 2. Дополнительные ошибки

```go
// errors.go (добавить к существующим)
var (
    // ... существующие ошибки ...

    ErrUserNotFound = errors.New("user not found")
)
```

## Unit тесты

```go
// assign_task_test.go
package task_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "flowra/internal/domain/task"
    "flowra/internal/infrastructure/eventstore"
    taskusecase "flowra/internal/usecase/task"
    "flowra/tests/mocks"
)

func TestAssignTaskUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    userRepo := mocks.NewMockUserRepository()

    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    assignUseCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

    // Создаем задачу
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:    uuid.New(),
        Title:     "Test Task",
        CreatedBy: uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Создаем пользователя в моке
    assigneeID := uuid.New()
    userRepo.AddUser(assigneeID, "alice", "Alice Smith")

    // Назначаем исполнителя
    assignerID := uuid.New()
    assignCmd := taskusecase.AssignTaskCommand{
        TaskID:     createResult.TaskID,
        AssigneeID: &assigneeID,
        AssignedBy: assignerID,
    }

    // Act
    result, err := assignUseCase.Execute(context.Background(), assignCmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, createResult.TaskID, result.TaskID)
    assert.Equal(t, 2, result.Version)
    require.Len(t, result.Events, 1)

    // Проверяем событие
    event, ok := result.Events[0].(task.TaskAssigneeChangedEvent)
    require.True(t, ok, "Expected TaskAssigneeChangedEvent")
    assert.Equal(t, createResult.TaskID, event.TaskID)
    assert.Nil(t, event.OldAssigneeID)
    assert.Equal(t, &assigneeID, event.NewAssigneeID)
    assert.Equal(t, assignerID, event.ChangedBy)
}

func TestAssignTaskUseCase_Unassign(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    userRepo := mocks.NewMockUserRepository()

    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    assignUseCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

    // Создаем задачу с assignee
    assigneeID := uuid.New()
    userRepo.AddUser(assigneeID, "bob", "Bob Johnson")

    createCmd := taskusecase.CreateTaskCommand{
        ChatID:     uuid.New(),
        Title:      "Test Task",
        AssigneeID: &assigneeID,
        CreatedBy:  uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Снимаем assignee (nil)
    unassignCmd := taskusecase.AssignTaskCommand{
        TaskID:     createResult.TaskID,
        AssigneeID: nil, // снятие
        AssignedBy: uuid.New(),
    }
    result, err := assignUseCase.Execute(context.Background(), unassignCmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, 2, result.Version)
    require.Len(t, result.Events, 1)

    event := result.Events[0].(task.TaskAssigneeChangedEvent)
    assert.Equal(t, &assigneeID, event.OldAssigneeID)
    assert.Nil(t, event.NewAssigneeID)
}

func TestAssignTaskUseCase_Reassign(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    userRepo := mocks.NewMockUserRepository()

    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    assignUseCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

    // Создаем двух пользователей
    alice := uuid.New()
    bob := uuid.New()
    userRepo.AddUser(alice, "alice", "Alice")
    userRepo.AddUser(bob, "bob", "Bob")

    // Создаем задачу, назначенную на Alice
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:     uuid.New(),
        Title:      "Test Task",
        AssigneeID: &alice,
        CreatedBy:  uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Переназначаем на Bob
    reassignCmd := taskusecase.AssignTaskCommand{
        TaskID:     createResult.TaskID,
        AssigneeID: &bob,
        AssignedBy: uuid.New(),
    }
    result, err := assignUseCase.Execute(context.Background(), reassignCmd)

    // Assert
    require.NoError(t, err)
    require.Len(t, result.Events, 1)

    event := result.Events[0].(task.TaskAssigneeChangedEvent)
    assert.Equal(t, &alice, event.OldAssigneeID)
    assert.Equal(t, &bob, event.NewAssigneeID)
}

func TestAssignTaskUseCase_Idempotent(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    userRepo := mocks.NewMockUserRepository()

    createUseCase := taskusecase.NewCreateTaskUseCase(eventStore)
    assignUseCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

    assigneeID := uuid.New()
    userRepo.AddUser(assigneeID, "alice", "Alice")

    // Создаем задачу, уже назначенную на Alice
    createCmd := taskusecase.CreateTaskCommand{
        ChatID:     uuid.New(),
        Title:      "Test Task",
        AssigneeID: &assigneeID,
        CreatedBy:  uuid.New(),
    }
    createResult, err := createUseCase.Execute(context.Background(), createCmd)
    require.NoError(t, err)

    // Act: Повторно назначаем на Alice
    assignCmd := taskusecase.AssignTaskCommand{
        TaskID:     createResult.TaskID,
        AssigneeID: &assigneeID,
        AssignedBy: uuid.New(),
    }
    result, err := assignUseCase.Execute(context.Background(), assignCmd)

    // Assert: Не должно быть новых событий
    require.NoError(t, err)
    assert.Empty(t, result.Events, "Should not generate event for idempotent operation")
    assert.Equal(t, 1, result.Version, "Version should not change")
}

func TestAssignTaskUseCase_ValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        setupMock   func(*mocks.MockUserRepository)
        cmd         taskusecase.AssignTaskCommand
        expectedErr error
    }{
        {
            name: "Empty TaskID",
            setupMock: func(m *mocks.MockUserRepository) {},
            cmd: taskusecase.AssignTaskCommand{
                TaskID:     uuid.Nil,
                AssigneeID: ptr(uuid.New()),
                AssignedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrInvalidTaskID,
        },
        {
            name: "Empty AssignedBy",
            setupMock: func(m *mocks.MockUserRepository) {},
            cmd: taskusecase.AssignTaskCommand{
                TaskID:     uuid.New(),
                AssigneeID: ptr(uuid.New()),
                AssignedBy: uuid.Nil,
            },
            expectedErr: taskusecase.ErrInvalidUserID,
        },
        {
            name: "User Not Found",
            setupMock: func(m *mocks.MockUserRepository) {
                // не добавляем пользователя
            },
            cmd: taskusecase.AssignTaskCommand{
                TaskID:     uuid.New(),
                AssigneeID: ptr(uuid.New()),
                AssignedBy: uuid.New(),
            },
            expectedErr: taskusecase.ErrUserNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            eventStore := eventstore.NewInMemoryEventStore()
            userRepo := mocks.NewMockUserRepository()
            tt.setupMock(userRepo)

            useCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

            // Act
            result, err := useCase.Execute(context.Background(), tt.cmd)

            // Assert
            require.Error(t, err)
            assert.ErrorIs(t, err, tt.expectedErr)
            assert.Empty(t, result.Events)
        })
    }
}

func TestAssignTaskUseCase_TaskNotFound(t *testing.T) {
    // Arrange
    eventStore := eventstore.NewInMemoryEventStore()
    userRepo := mocks.NewMockUserRepository()

    assigneeID := uuid.New()
    userRepo.AddUser(assigneeID, "alice", "Alice")

    useCase := taskusecase.NewAssignTaskUseCase(eventStore, userRepo)

    cmd := taskusecase.AssignTaskCommand{
        TaskID:     uuid.New(), // не существует
        AssigneeID: &assigneeID,
        AssignedBy: uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.ErrorIs(t, err, taskusecase.ErrTaskNotFound)
    assert.Empty(t, result.Events)
}

// Helper
func ptr[T any](v T) *T {
    return &v
}
```

## Mock User Repository

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
```

## Интеграция с Tag Parser (будущее)

```go
// В tag processor
func (p *TagProcessor) ProcessAssigneeTag(msg Message, tag ParsedTag) error {
    // Парсим @username
    username := strings.TrimPrefix(tag.Value, "@")

    // Резолвим username в UserID
    user, err := p.userRepository.GetByUsername(context.Background(), username)
    if err != nil {
        return err
    }
    if user == nil {
        p.sendErrorMessage(msg.ChatID, fmt.Sprintf("❌ User @%s not found", username))
        return nil
    }

    // Выполняем команду
    cmd := taskusecase.AssignTaskCommand{
        TaskID:     msg.RelatedTaskID,
        AssigneeID: &user.ID,
        AssignedBy: msg.AuthorID,
    }

    result, err := p.assignTaskUseCase.Execute(context.Background(), cmd)
    if err != nil {
        p.sendErrorMessage(msg.ChatID, fmt.Sprintf("❌ %s", err.Error()))
        return err
    }

    p.sendConfirmation(msg.ChatID, fmt.Sprintf("✅ Task assigned to @%s", username))
    return nil
}
```

## Checklist

- [ ] Создать интерфейс `UserRepository` в `internal/usecase/shared/interfaces.go`
- [ ] Реализовать `AssignTaskUseCase` в `assign_task.go`
- [ ] Добавить валидацию существования пользователя
- [ ] Реализовать Mock User Repository для тестов
- [ ] Написать unit тест для успешного назначения
- [ ] Написать тест для снятия assignee
- [ ] Написать тест для переназначения
- [ ] Написать тест для идемпотентности
- [ ] Написать тесты для validation errors
- [ ] Написать тест для TaskNotFound и UserNotFound
- [ ] Проверить покрытие тестами (>80%)

## Критерии приемки

- ✅ Use case назначает исполнителя задаче
- ✅ Use case снимает назначение (nil assignee)
- ✅ Валидируется существование пользователя
- ✅ Идемпотентность работает
- ✅ Обрабатывается UserNotFound
- ✅ Mock repository используется в тестах
- ✅ Покрытие тестами >80%

## Следующие шаги

После завершения переходим к:
- **Task 05**: ChangePriorityUseCase и SetDueDateUseCase
- **Task 06**: Use Case Testing Strategy

## Референсы

- [Task 03: ChangeStatus UseCase](03-change-status-usecase.md)
- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
- [Dependency Injection in Go](https://blog.drewolson.org/dependency-injection-in-go)
