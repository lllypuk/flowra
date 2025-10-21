# Quick Start Guide - UseCase Implementation

Это краткое руководство для быстрого старта разработки UseCases.

## Шаг 1: Прочитайте архитектуру (15 мин)

```bash
# Обязательно к прочтению
docs/tasks/04-impl-usecase/01-architecture.md
```

**Ключевые концепции:**
- Command Pattern
- Result Pattern
- UseCase Interface
- Shared компоненты (validation, errors, context)

## Шаг 2: Создайте структуру (30 мин)

### Создание shared компонентов

```bash
mkdir -p internal/application/shared
```

Создайте файлы:
- `interfaces.go` - UseCase, Command, Query, Result
- `errors.go` - Общие ошибки
- `context.go` - Context utilities
- `validation.go` - Validation helpers

### Создание структуры домена

```bash
mkdir -p internal/application/chat
mkdir -p internal/application/message
mkdir -p internal/application/user
mkdir -p internal/application/workspace
mkdir -p internal/application/notification
```

## Шаг 3: Начните с Chat Domain (6-8 ч)

### Создайте базовые файлы

```bash
cd internal/application/chat

touch commands.go queries.go results.go errors.go
touch create_chat.go add_participant.go
```

### Шаблон для команды

```go
// commands.go
package chat

import (
    "github.com/google/uuid"
    "github.com/flowra/flowra/internal/domain/chat"
)

type CreateChatCommand struct {
    WorkspaceID uuid.UUID
    Title       string
    Type        chat.Type
    CreatedBy   uuid.UUID
}

func (c CreateChatCommand) CommandName() string {
    return "CreateChat"
}
```

### Шаблон для UseCase

```go
// create_chat.go
package chat

import (
    "context"
    "fmt"

    "github.com/flowra/flowra/internal/application/shared"
    "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/internal/domain/event"
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
    // 1. Валидация
    if err := uc.validate(cmd); err != nil {
        return ChatResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Создание агрегата
    // ... domain logic ...

    // 3. Сохранение
    if err := uc.chatRepo.Save(ctx, chatAggregate); err != nil {
        return ChatResult{}, fmt.Errorf("failed to save: %w", err)
    }

    // 4. Публикация событий
    events := chatAggregate.GetUncommittedEvents()
    for _, evt := range events {
        _ = uc.eventBus.Publish(ctx, evt)
    }
    chatAggregate.MarkEventsAsCommitted()

    // 5. Возврат результата
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
    return nil
}
```

### Шаблон для теста

```go
// create_chat_test.go
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
}
```

## Шаг 4: Создайте Mocks (1 ч)

```bash
mkdir -p tests/mocks
```

### Шаблон Mock Repository

```go
// tests/mocks/chat_repository.go
package mocks

import (
    "context"
    "sync"

    "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/internal/domain/uuid"
)

type ChatRepository struct {
    mu    sync.RWMutex
    chats map[string]*chat.Chat
    calls map[string]int
}

func NewChatRepository() *ChatRepository {
    return &ChatRepository{
        chats: make(map[string]*chat.Chat),
        calls: make(map[string]int),
    }
}

func (r *ChatRepository) Load(ctx context.Context, id uuid.UUID) (*chat.Chat, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    r.calls["Load"]++

    c, ok := r.chats[id.String()]
    if !ok {
        return nil, chat.ErrChatNotFound
    }

    return c, nil
}

func (r *ChatRepository) Save(ctx context.Context, c *chat.Chat) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.calls["Save"]++
    r.chats[c.ID().String()] = c

    return nil
}
```

## Шаг 5: Запустите тесты

```bash
# Запуск тестов для конкретного UseCase
go test ./internal/application/chat -v -run TestCreateChatUseCase

# Запуск всех тестов в application layer
go test ./internal/application/... -v

# Проверка покрытия
go test ./internal/application/chat -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Шаг 6: Следующие домены

После завершения Chat UseCases, переходите к:

1. **Message UseCases** (5-7 ч) - проще, чем Chat
2. **User UseCases** (3-4 ч) - самый простой
3. **Workspace UseCases** (4-5 ч) - Keycloak integration
4. **Notification UseCases** (3-4 ч) - Event handlers

## Шаг 7: Integration Tests (4-5 ч)

После реализации всех доменов:

```bash
mkdir -p tests/integration
mkdir -p tests/e2e
```

Смотрите `07-integration-testing.md` для деталей.

## Шаг 8: Tag Integration (2-3 ч)

Рефакторинг `internal/domain/tag/executor.go`:

```bash
# См. детали в 08-tag-integration.md
```

## Частые вопросы

### Q: Нужно ли создавать UseCase для каждой операции?

**A:** Да. Каждая бизнес-операция = отдельный UseCase. Это обеспечивает:
- Single Responsibility
- Легкую тестируемость
- Простоту поддержки

### Q: Где размещать валидацию?

**A:** В методе `validate()` UseCase. Доменная валидация остается в aggregate.

### Q: Когда использовать Event Sourcing?

**A:** Только для Chat и Task агрегатов. Message, User, Workspace - простые CRUD.

### Q: Как обрабатывать ошибки?

**A:**
1. Валидация → ValidationError
2. Not found → специфичная ошибка (ErrChatNotFound)
3. Авторизация → ErrUnauthorized/ErrForbidden
4. Инфраструктура → wrap с context

### Q: Нужно ли создавать интерфейсы для UseCases?

**A:** Необязательно. Конкретные типы достаточно. Интерфейс `UseCase[TCommand, TResult]` используется как маркер.

## Полезные команды

```bash
# Создание новой структуры UseCase
make new-usecase DOMAIN=chat NAME=CreateChat

# Генерация моков (если используете mockery)
mockery --name=Repository --dir=internal/domain/chat --output=tests/mocks

# Запуск линтера
golangci-lint run ./internal/application/...

# Форматирование импортов
goimports -w internal/application/

# Проверка покрытия всего application layer
go test ./internal/application/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## Чеклист прогресса

### Phase 1: Architecture ✅
- [ ] `internal/application/shared/interfaces.go`
- [ ] `internal/application/shared/errors.go`
- [ ] `internal/application/shared/context.go`
- [ ] `internal/application/shared/validation.go`

### Phase 2: Chat UseCases
- [ ] commands.go, queries.go, results.go, errors.go
- [ ] CreateChatUseCase
- [ ] AddParticipantUseCase
- [ ] RemoveParticipantUseCase
- [ ] ConvertToTaskUseCase
- [ ] ConvertToBugUseCase
- [ ] ConvertToEpicUseCase
- [ ] ChangeStatusUseCase
- [ ] AssignUserUseCase
- [ ] SetPriorityUseCase
- [ ] SetDueDateUseCase
- [ ] RenameChatUseCase
- [ ] SetSeverityUseCase

### Phase 3: Message UseCases
- [ ] SendMessageUseCase
- [ ] EditMessageUseCase
- [ ] DeleteMessageUseCase
- [ ] AddReactionUseCase
- [ ] RemoveReactionUseCase
- [ ] AddAttachmentUseCase

### Phase 4: User UseCases
- [ ] RegisterUserUseCase
- [ ] UpdateProfileUseCase
- [ ] GetUserUseCase

### Phase 5: Workspace UseCases
- [ ] CreateWorkspaceUseCase
- [ ] CreateInviteUseCase
- [ ] AcceptInviteUseCase

### Phase 6: Notification UseCases
- [ ] CreateNotificationUseCase
- [ ] MarkAsReadUseCase
- [ ] ListNotificationsUseCase

### Phase 7: Testing
- [ ] Mocks для всех репозиториев
- [ ] Integration tests
- [ ] E2E workflow tests

### Phase 8: Tag Integration
- [ ] Рефакторинг CommandExecutor
- [ ] Integration с SendMessageUseCase

## Оценка времени по фазам

| Phase | Оценка |
|-------|--------|
| Phase 1: Architecture | 3-4 ч |
| Phase 2: Chat | 6-8 ч |
| Phase 3: Message | 5-7 ч |
| Phase 4: User | 3-4 ч |
| Phase 5: Workspace | 4-5 ч |
| Phase 6: Notification | 3-4 ч |
| Phase 7: Testing | 4-5 ч |
| Phase 8: Tag Integration | 2-3 ч |

**Итого: 30-40 часов**

## Рекомендации

1. **Не пропускайте тесты** - пишите их сразу после UseCase
2. **Коммитьте часто** - после каждого UseCase
3. **Используйте TDD** - сначала тест, потом реализация
4. **Ревью код** - сверяйтесь с примерами из задач
5. **Запускайте линтер** - после каждого коммита

## Следующие шаги после завершения

1. MongoDB repository implementations
2. HTTP handlers (Echo)
3. WebSocket handlers
4. Event Bus (Redis)
5. Keycloak integration
6. HTMX frontend

---

**Готовы начать?** Откройте `01-architecture.md` и приступайте! 🚀
