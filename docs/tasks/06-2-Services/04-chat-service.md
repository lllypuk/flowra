# Task 04: ChatService

**Приоритет:** 🟡 High
**Статус:** Complete
**Зависит от:** MongoDB репозитории (готовы)

---

## Описание

Реализовать `ChatService` — фасад над существующими chat юзкейсами. Сервис должен реализовать интерфейс `httphandler.ChatService` и заменить `MockChatService`.

---

## Текущее состояние

### Mock реализация (internal/handler/http/chat_handler.go)

```go
type MockChatService struct {
    chats   map[string]*chatapp.GetChatResult
    counter int
}

func NewMockChatService() *MockChatService
func (m *MockChatService) CreateChat(...) (chatapp.Result, error)
func (m *MockChatService) GetChat(...) (*chatapp.GetChatResult, error)
func (m *MockChatService) ListChats(...) (*chatapp.ListChatsResult, error)
func (m *MockChatService) RenameChat(...) (chatapp.Result, error)
func (m *MockChatService) AddParticipant(...) (chatapp.Result, error)
func (m *MockChatService) RemoveParticipant(...) (chatapp.Result, error)
func (m *MockChatService) DeleteChat(...) error
```

### Использование в container.go

```go
// container.go:438
mockChatService := httphandler.NewMockChatService()
c.ChatHandler = httphandler.NewChatHandler(mockChatService)
```

---

## Интерфейс (internal/handler/http/chat_handler.go)

```go
type ChatService interface {
    CreateChat(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)
    GetChat(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)
    ListChats(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)
    RenameChat(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error)
    AddParticipant(ctx context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error)
    RemoveParticipant(ctx context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error)
    DeleteChat(ctx context.Context, chatID, deletedBy uuid.UUID) error
}
```

---

## Существующие юзкейсы (internal/application/chat/)

| Юзкейс | Файл | Используется в ChatService |
|--------|------|---------------------------|
| `CreateChatUseCase` | `create_chat.go` | ✅ Да |
| `GetChatUseCase` | `get_chat.go` | ✅ Да |
| `ListChatsUseCase` | `list_chats.go` | ✅ Да |
| `RenameChatUseCase` | `rename_chat.go` | ✅ Да |
| `AddParticipantUseCase` | `add_participant.go` | ✅ Да |
| `RemoveParticipantUseCase` | `remove_participant.go` | ✅ Да |
| `ListParticipantsUseCase` | `list_participants.go` | Опционально |
| `ConvertToTaskUseCase` | `convert_to_task.go` | Отдельный endpoint |
| `ConvertToBugUseCase` | `convert_to_bug.go` | Отдельный endpoint |
| `ChangeStatusUseCase` | `change_status.go` | Отдельный endpoint |

---

## Реализация

### Файл: internal/service/chat_service.go

```go
package service

import (
    "context"

    "github.com/google/uuid"
    chatapp "github.com/lllypuk/flowra/internal/application/chat"
)

// ChatService реализует httphandler.ChatService
type ChatService struct {
    createUC    *chatapp.CreateChatUseCase
    getUC       *chatapp.GetChatUseCase
    listUC      *chatapp.ListChatsUseCase
    renameUC    *chatapp.RenameChatUseCase
    addPartUC   *chatapp.AddParticipantUseCase
    removePartUC *chatapp.RemoveParticipantUseCase

    // Repository для delete (use case может отсутствовать)
    commandRepo chatapp.CommandRepository
}

// ChatServiceConfig содержит зависимости для ChatService.
type ChatServiceConfig struct {
    CreateUC        *chatapp.CreateChatUseCase
    GetUC           *chatapp.GetChatUseCase
    ListUC          *chatapp.ListChatsUseCase
    RenameUC        *chatapp.RenameChatUseCase
    AddPartUC       *chatapp.AddParticipantUseCase
    RemovePartUC    *chatapp.RemoveParticipantUseCase
    CommandRepo     chatapp.CommandRepository
}

// NewChatService создаёт новый ChatService.
func NewChatService(cfg ChatServiceConfig) *ChatService {
    return &ChatService{
        createUC:     cfg.CreateUC,
        getUC:        cfg.GetUC,
        listUC:       cfg.ListUC,
        renameUC:     cfg.RenameUC,
        addPartUC:    cfg.AddPartUC,
        removePartUC: cfg.RemovePartUC,
        commandRepo:  cfg.CommandRepo,
    }
}

// CreateChat создаёт новый чат.
func (s *ChatService) CreateChat(
    ctx context.Context,
    cmd chatapp.CreateChatCommand,
) (chatapp.Result, error) {
    return s.createUC.Execute(ctx, cmd)
}

// GetChat возвращает чат по ID.
func (s *ChatService) GetChat(
    ctx context.Context,
    query chatapp.GetChatQuery,
) (*chatapp.GetChatResult, error) {
    return s.getUC.Execute(ctx, query)
}

// ListChats возвращает список чатов workspace.
func (s *ChatService) ListChats(
    ctx context.Context,
    query chatapp.ListChatsQuery,
) (*chatapp.ListChatsResult, error) {
    return s.listUC.Execute(ctx, query)
}

// RenameChat переименовывает чат.
func (s *ChatService) RenameChat(
    ctx context.Context,
    cmd chatapp.RenameChatCommand,
) (chatapp.Result, error) {
    return s.renameUC.Execute(ctx, cmd)
}

// AddParticipant добавляет участника в чат.
func (s *ChatService) AddParticipant(
    ctx context.Context,
    cmd chatapp.AddParticipantCommand,
) (chatapp.Result, error) {
    return s.addPartUC.Execute(ctx, cmd)
}

// RemoveParticipant удаляет участника из чата.
func (s *ChatService) RemoveParticipant(
    ctx context.Context,
    cmd chatapp.RemoveParticipantCommand,
) (chatapp.Result, error) {
    return s.removePartUC.Execute(ctx, cmd)
}

// DeleteChat удаляет чат.
func (s *ChatService) DeleteChat(
    ctx context.Context,
    chatID, deletedBy uuid.UUID,
) error {
    // Загрузить чат
    chat, err := s.commandRepo.Load(ctx, chatID)
    if err != nil {
        return err
    }

    // Применить команду удаления (если есть в domain)
    // или soft delete через флаг
    chat.Delete(deletedBy)

    // Сохранить
    return s.commandRepo.Save(ctx, chat)
}
```

---

## Event Sourcing

Chat использует Event Sourcing. Важные моменты:

1. **Юзкейсы работают с агрегатом:**
   - Load aggregate from EventStore
   - Apply command → generate events
   - Save events to EventStore

2. **Read Model обновляется автоматически** в `MongoChatRepository.Save()`

3. **Queries используют Read Model** — быстрые запросы без replay событий

---

## Инициализация юзкейсов

```go
// В container.go

// Event store уже есть: c.EventStore

// Chat use cases
createChatUC := chatapp.NewCreateChatUseCase(c.EventStore)
getChatUC := chatapp.NewGetChatUseCase(c.ChatRepo)
listChatsUC := chatapp.NewListChatsUseCase(c.ChatRepo)
renameChatUC := chatapp.NewRenameChatUseCase(c.ChatRepo)
addParticipantUC := chatapp.NewAddParticipantUseCase(c.ChatRepo)
removeParticipantUC := chatapp.NewRemoveParticipantUseCase(c.ChatRepo)

// Создание сервиса
chatService := service.NewChatService(service.ChatServiceConfig{
    CreateUC:     createChatUC,
    GetUC:        getChatUC,
    ListUC:       listChatsUC,
    RenameUC:     renameChatUC,
    AddPartUC:    addParticipantUC,
    RemovePartUC: removeParticipantUC,
    CommandRepo:  c.ChatRepo,
})
```

---

## Зависимости

### Входящие
- Chat use cases из `internal/application/chat/`
- `chatapp.CommandRepository` и `chatapp.QueryRepository`
- `EventStore` для event-sourced операций

### Репозитории

```go
type CommandRepository interface {
    Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)
    Save(ctx context.Context, c *chat.Chat) error
    GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

type QueryRepository interface {
    FindByID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)
    FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters Filters) ([]*ReadModel, error)
    Count(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

---

## Дополнительные методы (для будущих endpoints)

```go
// ConvertToTask конвертирует чат в task.
func (s *ChatService) ConvertToTask(ctx context.Context, cmd chatapp.ConvertToTaskCommand) (chatapp.Result, error)

// ConvertToBug конвертирует чат в bug.
func (s *ChatService) ConvertToBug(ctx context.Context, cmd chatapp.ConvertToBugCommand) (chatapp.Result, error)

// ChangeStatus изменяет статус чата.
func (s *ChatService) ChangeStatus(ctx context.Context, cmd chatapp.ChangeStatusCommand) (chatapp.Result, error)

// AssignUser назначает пользователя.
func (s *ChatService) AssignUser(ctx context.Context, cmd chatapp.AssignUserCommand) (chatapp.Result, error)
```

---

## Тестирование

### Unit tests

```go
// internal/service/chat_service_test.go

func TestChatService_CreateChat(t *testing.T) {
    // Test cases:
    // 1. Successfully create chat
    // 2. Validation error → error from use case
    // 3. Workspace not found → error
}

func TestChatService_GetChat(t *testing.T) {
    // 1. Chat exists → returns chat
    // 2. Chat not found → ErrNotFound
}

func TestChatService_ListChats(t *testing.T) {
    // 1. Workspace has chats → returns list
    // 2. Empty workspace → empty list
    // 3. Filters work correctly
}

func TestChatService_RenameChat(t *testing.T) {
    // 1. Successfully rename
    // 2. Chat not found → error
    // 3. Permission denied → error
}

func TestChatService_AddParticipant(t *testing.T) {
    // 1. Successfully add participant
    // 2. Already participant → error
    // 3. Chat not found → error
}

func TestChatService_DeleteChat(t *testing.T) {
    // 1. Successfully delete (soft delete)
    // 2. Chat not found → error
}
```

---

## Чеклист

- [x] Создать файл `internal/service/chat_service.go`
- [x] Определить `ChatServiceConfig` struct
- [x] Реализовать `NewChatService()`
- [x] Реализовать `CreateChat()` через use case
- [x] Реализовать `GetChat()` через use case
- [x] Реализовать `ListChats()` через use case
- [x] Реализовать `RenameChat()` через use case
- [x] Реализовать `AddParticipant()` через use case
- [x] Реализовать `RemoveParticipant()` через use case
- [x] Реализовать `DeleteChat()` через event sourcing
- [x] Написать unit tests
- [ ] Обновить `container.go` (Task 06)

---

## Критерии приёмки

- [x] `ChatService` реализует `httphandler.ChatService`
- [x] Event Sourcing работает корректно
- [ ] Read Model обновляется при изменениях
- [x] Unit test coverage > 80%
- [ ] Handler тесты проходят с real сервисом

---

## Заметки

- Chat использует Event Sourcing — важно понимать flow: Load → Apply → Save
- ✅ DeleteChat генерирует событие ChatDeleted (soft delete) - добавлен Deleted event в domain/chat/events.go
- ✅ Добавлен метод Delete() в domain/chat/chat.go с applyDeleted()
- Рассмотреть публикацию событий в EventBus для real-time обновлений
- Фильтры в ListChats: по типу (task, bug, epic), по статусу, по assignee

---

*Создано: 2026-01-06*
