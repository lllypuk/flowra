# Use Cases Implementation Plan (All Domains)

Этот каталог содержит полный план реализации use cases для всех доменных моделей проекта.

## Обзор

**Цель**: Создать полнофункциональный слой application logic (use cases) для всех агрегатов с использованием Event Sourcing, CQRS и паттернов DDD.

**Текущее состояние**:
- ✅ Domain models полностью реализованы (Chat, Message, Task, User, Workspace, Notification, Tag)
- ✅ Tag.CommandExecutor частично реализует UseCase паттерн для Chat операций
- ❌ Полноценный UseCase слой отсутствует
- ❌ Application layer пуст

## Структура задач

### Phase 1: Архитектура и базовые компоненты

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 01** | [01-architecture.md](01-architecture.md) | 📝 Pending | 3-4 ч | Архитектура UseCase слоя, паттерны, shared компоненты |

### Phase 2: Chat Domain Use Cases

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 02** | [02-chat-usecases.md](02-chat-usecases.md) | 📝 Pending | 6-8 ч | Create, AddParticipant, RemoveParticipant, ConvertType, ChangeStatus, AssignUser, SetProperties |

### Phase 3: Message Domain Use Cases

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 03** | [03-message-usecases.md](03-message-usecases.md) | 📝 Pending | 5-7 ч | SendMessage, EditMessage, DeleteMessage, AddReaction, RemoveReaction, AddAttachment |

### Phase 4: User Domain Use Cases

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 04** | [04-user-usecases.md](04-user-usecases.md) | 📝 Pending | 3-4 ч | RegisterUser, UpdateProfile, GetUser, ListUsers |

### Phase 5: Workspace Domain Use Cases

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 05** | [05-workspace-usecases.md](05-workspace-usecases.md) | 📝 Pending | 4-5 ч | CreateWorkspace, UpdateWorkspace, CreateInvite, AcceptInvite, RevokeInvite |

### Phase 6: Notification Domain Use Cases

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 06** | [06-notification-usecases.md](06-notification-usecases.md) | 📝 Pending | 3-4 ч | CreateNotification, MarkAsRead, GetNotifications, DeleteNotification |

### Phase 7: Integration & Testing

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 07** | [07-integration-testing.md](07-integration-testing.md) | 📝 Pending | 4-5 ч | Cross-domain integration, E2E tests, test infrastructure |

### Phase 8: Tag Integration Refactoring

| Задача | Файл | Статус | Оценка | Описание |
|--------|------|--------|--------|----------|
| **Task 08** | [08-tag-integration.md](08-tag-integration.md) | 📝 Pending | 2-3 ч | Рефакторинг Tag.CommandExecutor для использования UseCase |

## Порядок выполнения

Задачи выполняются **последовательно**:

```
Task 01 (Architecture)
   ↓
Task 02 (Chat) ←─── Приоритет 1 (основной агрегат)
   ↓
Task 03 (Message) ← Приоритет 2 (core messaging)
   ↓
Task 04 (User) ←──── Приоритет 3 (базовая функциональность)
   ↓
Task 05 (Workspace) ─ Приоритет 4
   ↓
Task 06 (Notification) ─ Приоритет 5
   ↓
Task 07 (Integration Testing)
   ↓
Task 08 (Tag Integration) ← Финальный рефакторинг
```

## Общая оценка времени

| Phase | Задачи | Оценка |
|-------|--------|--------|
| Phase 1 | Architecture | 3-4 ч |
| Phase 2 | Chat UseCases | 6-8 ч |
| Phase 3 | Message UseCases | 5-7 ч |
| Phase 4 | User UseCases | 3-4 ч |
| Phase 5 | Workspace UseCases | 4-5 ч |
| Phase 6 | Notification UseCases | 3-4 ч |
| Phase 7 | Integration Testing | 4-5 ч |
| Phase 8 | Tag Refactoring | 2-3 ч |

**Итого**: ~30-40 часов работы

## Результаты после завершения

### 1. Полный UseCase слой

```
internal/application/
├── shared/
│   ├── interfaces.go          # Общие интерфейсы (UseCase, Command, Result)
│   ├── base.go                # Базовая функциональность
│   ├── errors.go              # Общие ошибки
│   └── validation.go          # Общие валидаторы
├── chat/
│   ├── commands.go            # Все команды для Chat
│   ├── results.go             # Результаты
│   ├── errors.go              # Специфичные ошибки
│   ├── create_chat.go         # ✅
│   ├── add_participant.go     # ✅
│   ├── remove_participant.go  # ✅
│   ├── convert_to_task.go     # ✅
│   ├── convert_to_bug.go      # ✅
│   ├── convert_to_epic.go     # ✅
│   ├── change_status.go       # ✅
│   ├── assign_user.go         # ✅
│   ├── set_priority.go        # ✅
│   ├── set_due_date.go        # ✅
│   ├── rename.go              # ✅
│   ├── set_severity.go        # ✅
│   └── *_test.go              # Полное покрытие
├── message/
│   ├── commands.go
│   ├── send_message.go        # ✅
│   ├── edit_message.go        # ✅
│   ├── delete_message.go      # ✅
│   ├── add_reaction.go        # ✅
│   ├── remove_reaction.go     # ✅
│   ├── add_attachment.go      # ✅
│   └── *_test.go
├── user/
│   ├── commands.go
│   ├── register_user.go       # ✅
│   ├── update_profile.go      # ✅
│   ├── get_user.go            # ✅ (query)
│   ├── list_users.go          # ✅ (query)
│   └── *_test.go
├── workspace/
│   ├── commands.go
│   ├── create_workspace.go    # ✅
│   ├── update_workspace.go    # ✅
│   ├── create_invite.go       # ✅
│   ├── accept_invite.go       # ✅
│   ├── revoke_invite.go       # ✅
│   └── *_test.go
└── notification/
    ├── commands.go
    ├── create_notification.go # ✅
    ├── mark_as_read.go        # ✅
    ├── get_notifications.go   # ✅ (query)
    └── *_test.go
```

### 2. Infrastructure Components

```
internal/infrastructure/
├── eventstore/
│   ├── eventstore.go          # Интерфейс Event Store
│   ├── inmemory.go            # In-memory для тестов
│   └── mongodb.go             # MongoDB реализация (будущее)
├── repository/
│   ├── chat/
│   │   ├── repository.go      # Интерфейс
│   │   ├── eventstore.go      # Event-sourced реализация
│   │   └── readmodel.go       # Projection для queries
│   ├── message/
│   │   └── mongodb.go         # CRUD репозиторий
│   ├── user/
│   │   └── mongodb.go
│   ├── workspace/
│   │   └── mongodb.go
│   └── notification/
│       └── mongodb.go
└── eventbus/
    ├── eventbus.go            # Интерфейс
    ├── inmemory.go            # In-memory для тестов
    └── redis.go               # Redis pub/sub (будущее)
```

### 3. Тестовая инфраструктура

```
tests/
├── mocks/
│   ├── chat_repository.go
│   ├── message_repository.go
│   ├── user_repository.go
│   ├── workspace_repository.go
│   ├── eventstore.go
│   └── eventbus.go
├── fixtures/
│   ├── chat.go                # Test data builders
│   ├── message.go
│   ├── user.go
│   └── workspace.go
├── testutil/
│   ├── db.go                  # Database helpers
│   ├── context.go             # Context helpers
│   └── assert.go              # Custom assertions
└── integration/
    ├── chat_test.go
    ├── message_test.go
    ├── user_test.go
    ├── workspace_test.go
    └── e2e_test.go            # End-to-end workflows
```

## Принципы разработки

### 1. Clean Architecture

```
Handler (HTTP/WebSocket)
    ↓
UseCase (Application Logic)
    ↓
Domain (Business Logic)
    ↓
Repository (Data Access)
```

### 2. CQRS Separation

- **Commands**: Изменяют состояние (через Event Store)
- **Queries**: Читают состояние (через Read Models)

### 3. Event Sourcing

- Все изменения агрегатов через события
- Event Store как source of truth
- Projections для query optimization

### 4. Dependency Injection

- UseCase зависит от интерфейсов
- Конкретные реализации инжектируются
- Легко тестируется с моками

### 5. Test-Driven Development

1. Пишем тест
2. Реализуем минимальный код
3. Рефакторим
4. Повторяем

## Ключевые паттерны

### Command Pattern

```go
type Command interface {
    CommandName() string
}

type CreateChatCommand struct {
    WorkspaceID uuid.UUID
    Title       string
    Type        string
    CreatedBy   uuid.UUID
}
```

### Result Pattern

```go
type Result[T any] struct {
    Value   T
    Events  []event.Event
    Version int
    Error   error
}

func (r Result[T]) IsSuccess() bool { return r.Error == nil }
func (r Result[T]) IsFailure() bool { return r.Error != nil }
```

### UseCase Interface

```go
type UseCase[TCommand any, TResult any] interface {
    Execute(ctx context.Context, cmd TCommand) (TResult, error)
}
```

### Repository Pattern

```go
type Repository interface {
    Load(ctx context.Context, id uuid.UUID) (*Chat, error)
    Save(ctx context.Context, chat *Chat) error
}

type ReadModelRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*ChatReadModel, error)
    FindByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*ChatReadModel, error)
}
```

## Интеграция с существующим кодом

### Tag.CommandExecutor → UseCase

Текущий код в `internal/domain/tag/executor.go` напрямую работает с Chat aggregate. После реализации UseCase:

```go
// До
func (e *CommandExecutor) executeCreateTask(ctx context.Context, cmd CreateTaskCommand, actorID uuid.UUID) error {
    c, err := e.chatRepo.Load(ctx, chatID)
    // ...
    c.ConvertToTask(cmd.Title, userID)
    // ...
}

// После
func (e *CommandExecutor) executeCreateTask(ctx context.Context, cmd CreateTaskCommand, actorID uuid.UUID) error {
    usecaseCmd := chat.ConvertToTaskCommand{
        ChatID:    cmd.ChatID,
        Title:     cmd.Title,
        ActorID:   actorID,
    }
    _, err := e.convertToTaskUseCase.Execute(ctx, usecaseCmd)
    return err
}
```

## Готовность к интеграции

После завершения всех задач:

✅ **HTTP Handlers готовы к подключению**
```go
func (h *ChatHandler) CreateChat(c echo.Context) error {
    cmd := chat.CreateChatCommand{ /* ... */ }
    result, err := h.createChatUseCase.Execute(c.Request().Context(), cmd)
    // ...
}
```

✅ **WebSocket handlers готовы**
```go
func (ws *WebSocketHandler) HandleSendMessage(conn *websocket.Conn, msg IncomingMessage) {
    cmd := message.SendMessageCommand{ /* ... */ }
    result, err := ws.sendMessageUseCase.Execute(context.Background(), cmd)
    // ...
}
```

✅ **Tag integration готова**
```go
executor := tag.NewCommandExecutor(
    chatUseCases,      // вместо прямого chatRepo
    messageUseCases,
    userRepo,
)
```

✅ **Event handlers готовы**
```go
eventBus.Subscribe(chat.ChatCreatedEvent, func(evt event.Event) {
    // Create notification
    notificationUseCase.Execute(ctx, CreateNotificationCommand{ /* ... */ })
})
```

## Следующие шаги после завершения

1. **Repository implementations** (MongoDB)
   - Event Store persistence
   - Read Model projections
   - CRUD repositories

2. **HTTP Handlers**
   - REST API endpoints
   - HTMX integration
   - Request/Response DTOs

3. **WebSocket Handlers**
   - Real-time message delivery
   - Presence tracking
   - Event broadcasting

4. **Event Bus** (Redis)
   - Pub/Sub для событий
   - Event handlers для notifications
   - Cross-domain integration

5. **Authentication & Authorization**
   - Keycloak integration
   - JWT validation
   - Permission checks в UseCases

## Ресурсы

- [CLAUDE.md](../../../CLAUDE.md) - Общая документация проекта
- [Domain Models](../../internal/domain/) - Доменные модели
- [Task UseCases Plan](../01-impl-task/) - Пример детального плана (для Task)
- [Tag Grammar](../02-impl-tag-grammar/) - Tag parsing система
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)

## Статус обновления

**Дата создания**: 2025-10-19
**Версия**: 1.0
**Статус**: Все задачи задокументированы, готовы к реализации
**Автор**: Claude Code

---

## Quick Start

Для начала работы:

1. Прочитайте [Task 01: Architecture](01-architecture.md)
2. Реализуйте базовую структуру UseCase слоя
3. Начните с [Task 02: Chat UseCases](02-chat-usecases.md)
4. Следуйте последовательности задач
5. Запускайте тесты после каждой задачи

**Важно**: Не пропускайте Task 01! Архитектура определяет структуру для всех остальных задач.
