# Task 02: Chat Domain Use Cases

**Дата:** 2025-10-19
**Статус:** 📝 Pending
**Зависимости:** Task 01 (Architecture)
**Оценка:** 6-8 часов

## Цель

Реализовать все Use Cases для Chat агрегата с полным тестовым покрытием. Chat является центральным агрегатом системы, поэтому его реализация критична для всего проекта.

## Контекст

**Chat aggregate поддерживает:**
- 4 типа чатов: Discussion, Task, Bug, Epic
- Event Sourcing с версионированием
- Участники с ролями (Admin, Member)
- Конвертация типов (Discussion → Task/Bug/Epic)
- Типизированные свойства (status, priority, assignee, dueDate, severity)
- 12 событий (ChatCreated, TypeChanged, StatusChanged, UserAssigned, и т.д.)

## Use Cases для реализации

### Command Use Cases (изменяют состояние)

| UseCase | Операция | Приоритет |
|---------|----------|-----------|
| CreateChatUseCase | Создание нового чата | Критичный |
| AddParticipantUseCase | Добавление участника | Критичный |
| RemoveParticipantUseCase | Удаление участника | Критичный |
| ConvertToTaskUseCase | Конвертация в Task | Высокий |
| ConvertToBugUseCase | Конвертация в Bug | Высокий |
| ConvertToEpicUseCase | Конвертация в Epic | Высокий |
| ChangeStatusUseCase | Изменение статуса | Высокий |
| AssignUserUseCase | Назначение пользователя | Высокий |
| SetPriorityUseCase | Установка приоритета | Средний |
| SetDueDateUseCase | Установка дедлайна | Средний |
| RenameChatUseCase | Переименование | Средний |
| SetSeverityUseCase | Установка severity (Bug) | Средний |

### Query Use Cases (только чтение)

| UseCase | Операция | Приоритет |
|---------|----------|-----------|
| GetChatUseCase | Получение чата по ID | Критичный |
| ListChatsUseCase | Список чатов workspace | Критичный |
| ListParticipantsUseCase | Список участников | Высокий |

## Структура файлов

```
internal/application/chat/
├── commands.go            # Все команды (создание, модификация)
├── queries.go             # Запросы для CQRS
├── results.go             # Результаты выполнения
├── errors.go              # Специфичные ошибки Chat domain
│
├── create_chat.go         # CreateChatUseCase
├── add_participant.go     # AddParticipantUseCase
├── remove_participant.go  # RemoveParticipantUseCase
├── convert_to_task.go     # ConvertToTaskUseCase
├── convert_to_bug.go      # ConvertToBugUseCase
├── convert_to_epic.go     # ConvertToEpicUseCase
├── change_status.go       # ChangeStatusUseCase
├── assign_user.go         # AssignUserUseCase
├── set_priority.go        # SetPriorityUseCase
├── set_due_date.go        # SetDueDateUseCase
├── rename_chat.go         # RenameChatUseCase
├── set_severity.go        # SetSeverityUseCase
│
├── get_chat.go            # GetChatUseCase (query)
├── list_chats.go          # ListChatsUseCase (query)
├── list_participants.go   # ListParticipantsUseCase (query)
│
└── *_test.go              # Тесты для каждого UseCase
```

## Детальное описание

### 1. Commands (commands.go)

```go
package chat

import (
    "time"

    "github.com/google/uuid"
    "github.com/flowra/flowra/internal/domain/chat"
)

// CreateChatCommand - создание нового чата
type CreateChatCommand struct {
    WorkspaceID uuid.UUID
    Title       string
    Type        chat.Type       // Discussion, Task, Bug, Epic
    CreatedBy   uuid.UUID
}

func (c CreateChatCommand) CommandName() string { return "CreateChat" }

// AddParticipantCommand - добавление участника
type AddParticipantCommand struct {
    ChatID      uuid.UUID
    UserID      uuid.UUID
    Role        chat.Role       // Admin, Member
    AddedBy     uuid.UUID
}

func (c AddParticipantCommand) CommandName() string { return "AddParticipant" }

// RemoveParticipantCommand - удаление участника
type RemoveParticipantCommand struct {
    ChatID      uuid.UUID
    UserID      uuid.UUID
    RemovedBy   uuid.UUID
}

func (c RemoveParticipantCommand) CommandName() string { return "RemoveParticipant" }

// ConvertToTaskCommand - конвертация в Task
type ConvertToTaskCommand struct {
    ChatID      uuid.UUID
    Title       string          // Новый заголовок (опционально)
    ConvertedBy uuid.UUID
}

func (c ConvertToTaskCommand) CommandName() string { return "ConvertToTask" }

// ConvertToBugCommand - конвертация в Bug
type ConvertToBugCommand struct {
    ChatID      uuid.UUID
    Title       string
    ConvertedBy uuid.UUID
}

func (c ConvertToBugCommand) CommandName() string { return "ConvertToBug" }

// ConvertToEpicCommand - конвертация в Epic
type ConvertToEpicCommand struct {
    ChatID      uuid.UUID
    Title       string
    ConvertedBy uuid.UUID
}

func (c ConvertToEpicCommand) CommandName() string { return "ConvertToEpic" }

// ChangeStatusCommand - изменение статуса
type ChangeStatusCommand struct {
    ChatID      uuid.UUID
    Status      chat.Status     // зависит от типа чата
    ChangedBy   uuid.UUID
}

func (c ChangeStatusCommand) CommandName() string { return "ChangeStatus" }

// AssignUserCommand - назначение пользователя
type AssignUserCommand struct {
    ChatID      uuid.UUID
    AssigneeID  *uuid.UUID      // nil = снять assignee
    AssignedBy  uuid.UUID
}

func (c AssignUserCommand) CommandName() string { return "AssignUser" }

// SetPriorityCommand - установка приоритета
type SetPriorityCommand struct {
    ChatID      uuid.UUID
    Priority    chat.Priority   // Low, Medium, High, Critical
    SetBy       uuid.UUID
}

func (c SetPriorityCommand) CommandName() string { return "SetPriority" }

// SetDueDateCommand - установка дедлайна
type SetDueDateCommand struct {
    ChatID      uuid.UUID
    DueDate     *time.Time      // nil = снять дедлайн
    SetBy       uuid.UUID
}

func (c SetDueDateCommand) CommandName() string { return "SetDueDate" }

// RenameChatCommand - переименование чата
type RenameChatCommand struct {
    ChatID      uuid.UUID
    NewTitle    string
    RenamedBy   uuid.UUID
}

func (c RenameChatCommand) CommandName() string { return "RenameChat" }

// SetSeverityCommand - установка severity (только для Bug)
type SetSeverityCommand struct {
    ChatID      uuid.UUID
    Severity    chat.Severity   // Minor, Major, Critical, Blocker
    SetBy       uuid.UUID
}

func (c SetSeverityCommand) CommandName() string { return "SetSeverity" }
```

### 2. Queries (queries.go)

```go
package chat

import "github.com/google/uuid"

// GetChatQuery - получение чата по ID
type GetChatQuery struct {
    ChatID uuid.UUID
    UserID uuid.UUID           // для проверки доступа
}

func (q GetChatQuery) QueryName() string { return "GetChat" }

// ListChatsQuery - список чатов workspace
type ListChatsQuery struct {
    WorkspaceID uuid.UUID
    UserID      uuid.UUID       // для фильтрации доступных
    Type        *chat.Type      // фильтр по типу (опционально)
    Limit       int
    Offset      int
}

func (q ListChatsQuery) QueryName() string { return "ListChats" }

// ListParticipantsQuery - список участников чата
type ListParticipantsQuery struct {
    ChatID uuid.UUID
    UserID uuid.UUID           // для проверки доступа
}

func (q ListParticipantsQuery) QueryName() string { return "ListParticipants" }
```

### 3. Results (results.go)

```go
package chat

import (
    "github.com/flowra/flowra/internal/application/shared"
    "github.com/flowra/flowra/internal/domain/chat"
)

// ChatResult - результат command UseCase
type ChatResult = shared.EventSourcedResult[*chat.Chat]

// ChatQueryResult - результат query UseCase
type ChatQueryResult = shared.Result[*chat.Chat]

// ChatsQueryResult - результат для списка чатов
type ChatsQueryResult = shared.Result[[]*chat.Chat]

// ParticipantsQueryResult - результат для списка участников
type ParticipantsQueryResult = shared.Result[[]chat.Participant]
```

### 4. Errors (errors.go)

```go
package chat

import (
    "errors"

    "github.com/flowra/flowra/internal/application/shared"
)

var (
    // Validation errors
    ErrInvalidChatType      = errors.New("invalid chat type")
    ErrInvalidStatus        = errors.New("invalid status for chat type")
    ErrInvalidPriority      = errors.New("invalid priority")
    ErrInvalidSeverity      = errors.New("invalid severity")
    ErrInvalidRole          = errors.New("invalid participant role")

    // Business logic errors
    ErrChatNotFound         = errors.New("chat not found")
    ErrUserNotParticipant   = errors.New("user is not a participant")
    ErrUserAlreadyParticipant = errors.New("user is already a participant")
    ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
    ErrNotAdmin             = errors.New("user is not an admin")
    ErrCannotConvertType    = errors.New("cannot convert chat type")
    ErrSeverityOnlyForBugs  = errors.New("severity can only be set on bugs")

    // Authorization errors
    ErrNotAuthorized        = shared.ErrUnauthorized
)
```

### 5. Пример реализации: CreateChatUseCase

```go
// File: create_chat.go
package chat

import (
    "context"
    "fmt"

    "github.com/flowra/flowra/internal/application/shared"
    "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/internal/domain/event"
    domainUUID "github.com/flowra/flowra/internal/domain/uuid"
)

type CreateChatUseCase struct {
    chatRepo chat.Repository
    eventBus event.Bus
}

func NewCreateChatUseCase(
    chatRepo chat.Repository,
    eventBus event.Bus,
) *CreateChatUseCase {
    return &CreateChatUseCase{
        chatRepo: chatRepo,
        eventBus: eventBus,
    }
}

func (uc *CreateChatUseCase) Execute(
    ctx context.Context,
    cmd CreateChatCommand,
) (ChatResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return ChatResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Создание агрегата
    workspaceID := domainUUID.FromGoogleUUID(cmd.WorkspaceID)
    creatorID := domainUUID.FromGoogleUUID(cmd.CreatedBy)

    chatAggregate := chat.NewChat(workspaceID, cmd.Title, cmd.Type, creatorID)

    // Добавление создателя как Admin
    if err := chatAggregate.AddParticipant(creatorID, chat.RoleAdmin); err != nil {
        return ChatResult{}, fmt.Errorf("failed to add creator: %w", err)
    }

    // Сохранение
    if err := uc.chatRepo.Save(ctx, chatAggregate); err != nil {
        return ChatResult{}, fmt.Errorf("failed to save chat: %w", err)
    }

    // Публикация событий
    events := chatAggregate.GetUncommittedEvents()
    for _, evt := range events {
        if err := uc.eventBus.Publish(ctx, evt); err != nil {
            // Rollback сохранения (если нужно)
            return ChatResult{}, fmt.Errorf("failed to publish event: %w", err)
        }
    }

    chatAggregate.MarkEventsAsCommitted()

    return ChatResult{
        Result: shared.Result[*chat.Chat]{
            Value:   chatAggregate,
            Version: chatAggregate.Version(),
        },
        Events: events,
    }, nil
}

func (uc *CreateChatUseCase) validate(cmd CreateChatCommand) error {
    if err := shared.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
        return err
    }
    if err := shared.ValidateRequired("title", cmd.Title); err != nil {
        return err
    }
    if err := shared.ValidateMaxLength("title", cmd.Title, 200); err != nil {
        return err
    }
    if err := shared.ValidateEnum("type", string(cmd.Type), []string{
        string(chat.TypeDiscussion),
        string(chat.TypeTask),
        string(chat.TypeBug),
        string(chat.TypeEpic),
    }); err != nil {
        return err
    }
    if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
        return err
    }
    return nil
}
```

### 6. Пример теста

```go
// File: create_chat_test.go
package chat_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/flowra/flowra/internal/application/chat"
    domainChat "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/tests/mocks"
)

func TestCreateChatUseCase_Success(t *testing.T) {
    // Arrange
    chatRepo := mocks.NewChatRepository()
    eventBus := mocks.NewEventBus()
    useCase := chat.NewCreateChatUseCase(chatRepo, eventBus)

    cmd := chat.CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "Test Chat",
        Type:        domainChat.TypeDiscussion,
        CreatedBy:   uuid.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result.Value)
    assert.Equal(t, cmd.Title, result.Value.Title())
    assert.Equal(t, cmd.Type, result.Value.Type())
    assert.Len(t, result.Events, 2) // ChatCreated + ParticipantAdded

    // Verify Save was called
    assert.Equal(t, 1, chatRepo.SaveCallCount())

    // Verify events published
    assert.Equal(t, 2, eventBus.PublishCallCount())
}

func TestCreateChatUseCase_ValidationError_EmptyTitle(t *testing.T) {
    chatRepo := mocks.NewChatRepository()
    eventBus := mocks.NewEventBus()
    useCase := chat.NewCreateChatUseCase(chatRepo, eventBus)

    cmd := chat.CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "", // пустой
        Type:        domainChat.TypeDiscussion,
        CreatedBy:   uuid.New(),
    }

    result, err := useCase.Execute(context.Background(), cmd)

    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
    assert.Nil(t, result.Value)
}

func TestCreateChatUseCase_ValidationError_InvalidType(t *testing.T) {
    chatRepo := mocks.NewChatRepository()
    eventBus := mocks.NewEventBus()
    useCase := chat.NewCreateChatUseCase(chatRepo, eventBus)

    cmd := chat.CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "Test",
        Type:        "InvalidType", // невалидный
        CreatedBy:   uuid.New(),
    }

    result, err := useCase.Execute(context.Background(), cmd)

    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}
```

## Специальные требования

### 1. Авторизация

Все UseCases должны проверять права доступа:

```go
func (uc *AddParticipantUseCase) authorize(ctx context.Context, chatAggregate *chat.Chat, cmd AddParticipantCommand) error {
    userID, err := shared.GetUserID(ctx)
    if err != nil {
        return shared.ErrUnauthorized
    }

    // Проверка, что пользователь - admin чата
    if !chatAggregate.IsParticipantAdmin(userID) {
        return ErrNotAdmin
    }

    return nil
}
```

### 2. Optimistic Locking

Для UseCases, изменяющих существующие агрегаты:

```go
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (ChatResult, error) {
    // Загрузка агрегата
    chatAggregate, err := uc.chatRepo.Load(ctx, domainUUID.FromGoogleUUID(cmd.ChatID))
    if err != nil {
        return ChatResult{}, ErrChatNotFound
    }

    // Сохранение с проверкой версии
    expectedVersion := chatAggregate.Version()

    // ... выполнение бизнес-логики ...

    if err := uc.chatRepo.Save(ctx, chatAggregate); err != nil {
        if errors.Is(err, chat.ErrVersionMismatch) {
            return ChatResult{}, shared.ErrConcurrentUpdate
        }
        return ChatResult{}, err
    }

    // ...
}
```

### 3. Event Publishing Order

События публикуются в том же порядке, в котором они были созданы:

```go
events := chatAggregate.GetUncommittedEvents()
for _, evt := range events {
    if err := uc.eventBus.Publish(ctx, evt); err != nil {
        return ChatResult{}, fmt.Errorf("failed to publish event: %w", err)
    }
}
```

## Checklist

### Phase 1: Commands Structure
- [ ] Создать `commands.go` со всеми командами
- [ ] Создать `queries.go` со всеми запросами
- [ ] Создать `results.go`
- [ ] Создать `errors.go`

### Phase 2: Command UseCases (приоритет по важности)
- [ ] CreateChatUseCase + tests
- [ ] AddParticipantUseCase + tests
- [ ] RemoveParticipantUseCase + tests
- [ ] ConvertToTaskUseCase + tests
- [ ] ConvertToBugUseCase + tests
- [ ] ConvertToEpicUseCase + tests
- [ ] ChangeStatusUseCase + tests
- [ ] AssignUserUseCase + tests
- [ ] SetPriorityUseCase + tests
- [ ] SetDueDateUseCase + tests
- [ ] RenameChatUseCase + tests
- [ ] SetSeverityUseCase + tests

### Phase 3: Query UseCases
- [ ] GetChatUseCase + tests
- [ ] ListChatsUseCase + tests
- [ ] ListParticipantsUseCase + tests

### Phase 4: Integration Testing
- [ ] End-to-end workflow tests
- [ ] Cross-domain integration tests

## Оценка времени

| Группа | Оценка |
|--------|--------|
| Структура (commands, queries, results, errors) | 1 час |
| CreateChat, AddParticipant, RemoveParticipant | 2 часа |
| Convert* UseCases (Task, Bug, Epic) | 1.5 часа |
| ChangeStatus, AssignUser, SetPriority, SetDueDate | 1.5 часа |
| Rename, SetSeverity | 0.5 часа |
| Query UseCases | 1 час |
| Integration tests | 1 час |

**Итого**: ~8 часов

## Следующие шаги

После завершения Chat UseCases:
- **Task 03**: Message UseCases
- Интеграция Tag.CommandExecutor с Chat UseCases (Task 08)

## Референсы

- [Chat Domain Model](../../internal/domain/chat/)
- [Task 01: Architecture](01-architecture.md)
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)
