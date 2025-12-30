# Task 10: Chat Query UseCases Implementation

**Дата:** 2025-10-22
**Статус:** ✅ Complete
**Зависимости:** Task 09 (Chat Tests)
**Оценка:** 2 часа
**Приоритет:** 🟡 ВЫСОКИЙ

## Проблема

Chat Query UseCases не реализованы, что делает невозможным:
- Получение информации о чате по ID
- Вывод списка чатов workspace
- Просмотр участников чата

Без Query UseCases функциональность Chat агрегата неполная.

## Цель

Реализовать 3 Query UseCases для чтения данных Chat агрегата с полным тестовым покрытием.

## Query UseCases для реализации

### 1. GetChatUseCase - Получение чата по ID

**Приоритет:** Критический
**Оценка:** 30 минут

#### Описание

Получает агрегат Chat по его ID, восстанавливая состояние из EventStore.

#### Интерфейс

```go
// File: internal/application/chat/queries.go
package chat

import (
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// GetChatQuery - запрос на получение чата по ID
type GetChatQuery struct {
    ChatID domainUUID.UUID
}

func (q GetChatQuery) QueryName() string {
    return "GetChat"
}
```

#### Реализация

```go
// File: internal/application/chat/get_chat.go
package chat

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
)

// GetChatUseCase получает чат по ID
type GetChatUseCase struct {
    eventStore shared.EventStore
}

// NewGetChatUseCase создает новый GetChatUseCase
func NewGetChatUseCase(eventStore shared.EventStore) *GetChatUseCase {
    return &GetChatUseCase{
        eventStore: eventStore,
    }
}

// Execute выполняет получение чата
func (uc *GetChatUseCase) Execute(ctx context.Context, query GetChatQuery) (QueryResult, error) {
    // Валидация
    if err := uc.validate(query); err != nil {
        return QueryResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Загрузка событий из EventStore
    events, err := uc.eventStore.LoadEvents(ctx, query.ChatID.String())
    if err != nil {
        return QueryResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return QueryResult{}, ErrChatNotFound
    }

    // Восстановление агрегата из событий
    chatAggregate := &chat.Chat{}
    if err := chatAggregate.LoadFromHistory(events); err != nil {
        return QueryResult{}, fmt.Errorf("failed to load from history: %w", err)
    }

    return QueryResult{
        Aggregate: chatAggregate,
        Version:   chatAggregate.Version(),
    }, nil
}

func (uc *GetChatUseCase) validate(query GetChatQuery) error {
    if query.ChatID.IsNil() {
        return shared.NewValidationError("chatID", "must be a valid UUID")
    }
    return nil
}
```

#### Тесты

```go
// File: internal/application/chat/get_chat_test.go
package chat_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/chat"
    domainChat "github.com/lllypuk/flowra/internal/domain/chat"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/tests/mocks"
)

func TestGetChatUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewGetChatUseCase(eventStore)

    chatID := domainUUID.New()
    workspaceID := domainUUID.New()
    creatorID := domainUUID.New()

    // Создаем события для чата
    events := []interface{}{
        domainChat.ChatCreatedEvent{
            ChatID:      chatID,
            WorkspaceID: workspaceID,
            Type:        domainChat.TypeDiscussion,
            IsPublic:    true,
            CreatedBy:   creatorID,
        },
    }
    eventStore.SetLoadEventsResult(events, nil)

    query := chat.GetChatQuery{
        ChatID: chatID,
    }

    // Act
    result, err := useCase.Execute(context.Background(), query)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result.Aggregate)
    assert.Equal(t, chatID, result.Aggregate.ID())
    assert.Equal(t, domainChat.TypeDiscussion, result.Aggregate.Type())
}

func TestGetChatUseCase_Error_NotFound(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    eventStore.SetLoadEventsResult([]interface{}{}, nil) // Нет событий
    useCase := chat.NewGetChatUseCase(eventStore)

    query := chat.GetChatQuery{
        ChatID: domainUUID.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), query)

    // Assert
    require.Error(t, err)
    assert.ErrorIs(t, err, chat.ErrChatNotFound)
    assert.Nil(t, result.Aggregate)
}

func TestGetChatUseCase_ValidationError_InvalidChatID(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewGetChatUseCase(eventStore)

    query := chat.GetChatQuery{
        ChatID: domainUUID.Nil(), // Невалидный
    }

    // Act
    result, err := useCase.Execute(context.Background(), query)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}

func TestGetChatUseCase_EventStoreError(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    eventStore.SetLoadEventsError(errors.New("database error"))
    useCase := chat.NewGetChatUseCase(eventStore)

    query := chat.GetChatQuery{
        ChatID: domainUUID.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), query)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "database error")
}
```

**Тесты:** 4
**Время:** 15 минут

---

### 2. ListChatsUseCase - Список чатов workspace

**Приоритет:** Критический
**Оценка:** 40 минут

#### Описание

Возвращает список всех чатов в workspace с поддержкой фильтрации и пагинации.

#### Интерфейс

```go
// File: internal/application/chat/queries.go (добавить)

// ListChatsQuery - запрос на получение списка чатов
type ListChatsQuery struct {
    WorkspaceID domainUUID.UUID
    Type        *chat.Type // фильтр по типу (опционально)
    Limit       int        // default: 50, max: 100
    Offset      int        // для pagination
}

func (q ListChatsQuery) QueryName() string {
    return "ListChats"
}
```

#### Реализация

```go
// File: internal/application/chat/list_chats.go
package chat

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
)

const (
    DefaultLimit = 50
    MaxLimit     = 100
)

// ListChatsUseCase получает список чатов workspace
type ListChatsUseCase struct {
    eventStore shared.EventStore
}

// NewListChatsUseCase создает новый ListChatsUseCase
func NewListChatsUseCase(eventStore shared.EventStore) *ListChatsUseCase {
    return &ListChatsUseCase{
        eventStore: eventStore,
    }
}

// Execute выполняет получение списка чатов
func (uc *ListChatsUseCase) Execute(ctx context.Context, query ListChatsQuery) (ListQueryResult, error) {
    // Валидация
    if err := uc.validate(&query); err != nil {
        return ListQueryResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Установка дефолтных значений
    if query.Limit == 0 {
        query.Limit = DefaultLimit
    }

    // Загрузка всех чатов workspace из EventStore
    // ПРИМЕЧАНИЕ: В реальной реализации нужен специальный индекс по workspace_id
    // Для простоты используем LoadEventsByPrefix
    prefix := fmt.Sprintf("workspace:%s:chat:", query.WorkspaceID.String())
    allEvents, err := uc.eventStore.LoadEventsByPrefix(ctx, prefix)
    if err != nil {
        return ListQueryResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    // Группировка событий по chat_id и восстановление агрегатов
    chatsMap := make(map[string]*chat.Chat)
    for _, event := range allEvents {
        chatID := extractChatID(event)
        if chatID == "" {
            continue
        }

        if _, exists := chatsMap[chatID]; !exists {
            chatsMap[chatID] = &chat.Chat{}
        }
        chatsMap[chatID].ApplyEvent(event)
    }

    // Фильтрация по типу
    chats := make([]*chat.Chat, 0, len(chatsMap))
    for _, c := range chatsMap {
        if query.Type != nil && c.Type() != *query.Type {
            continue
        }
        chats = append(chats, c)
    }

    // Сортировка по дате создания (desc)
    sort.Slice(chats, func(i, j int) bool {
        return chats[i].CreatedAt().After(chats[j].CreatedAt())
    })

    // Pagination
    total := len(chats)
    start := query.Offset
    end := query.Offset + query.Limit

    if start >= total {
        return ListQueryResult{
            Chats:  []*chat.Chat{},
            Total:  total,
            Limit:  query.Limit,
            Offset: query.Offset,
        }, nil
    }

    if end > total {
        end = total
    }

    return ListQueryResult{
        Chats:  chats[start:end],
        Total:  total,
        Limit:  query.Limit,
        Offset: query.Offset,
    }, nil
}

func (uc *ListChatsUseCase) validate(query *ListChatsQuery) error {
    if query.WorkspaceID.IsNil() {
        return shared.NewValidationError("workspaceID", "must be a valid UUID")
    }
    if query.Limit > MaxLimit {
        query.Limit = MaxLimit
    }
    if query.Offset < 0 {
        query.Offset = 0
    }
    return nil
}

func extractChatID(event interface{}) string {
    // Извлечение chat_id из события
    // Зависит от структуры событий
    switch e := event.(type) {
    case chat.ChatCreatedEvent:
        return e.ChatID.String()
    case chat.TypeChangedEvent:
        return e.ChatID.String()
    // ... другие события
    default:
        return ""
    }
}
```

#### Результат

```go
// File: internal/application/chat/results.go (добавить)

// ListQueryResult - результат для списка чатов
type ListQueryResult struct {
    Chats  []*chat.Chat
    Total  int // общее количество
    Limit  int
    Offset int
}
```

#### Тесты

```go
// File: internal/application/chat/list_chats_test.go
package chat_test

func TestListChatsUseCase_Success(t *testing.T) {
    // Создать 5 чатов в workspace
    // Запросить список
    // Проверить, что получили все 5
}

func TestListChatsUseCase_Success_WithTypeFilter(t *testing.T) {
    // Создать чаты разных типов (Discussion, Task, Bug)
    // Запросить только Task
    // Проверить фильтрацию
}

func TestListChatsUseCase_Success_Pagination(t *testing.T) {
    // Создать 25 чатов
    // Запросить с Limit=10, Offset=0
    // Проверить, что получили первые 10
    // Запросить с Limit=10, Offset=10
    // Проверить, что получили следующие 10
}

func TestListChatsUseCase_Success_EmptyResult(t *testing.T) {
    // Workspace без чатов
    // Проверить пустой список
}

func TestListChatsUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {}

func TestListChatsUseCase_ValidationError_LimitTooLarge(t *testing.T) {
    // Limit > MaxLimit
    // Проверить, что Limit установлен в MaxLimit
}
```

**Тесты:** 6
**Время:** 25 минут

---

### 3. ListParticipantsUseCase - Список участников чата

**Приоритет:** Высокий
**Оценка:** 30 минут

#### Описание

Возвращает список всех участников чата с их ролями.

#### Интерфейс

```go
// File: internal/application/chat/queries.go (добавить)

// ListParticipantsQuery - запрос на получение участников чата
type ListParticipantsQuery struct {
    ChatID domainUUID.UUID
}

func (q ListParticipantsQuery) QueryName() string {
    return "ListParticipants"
}
```

#### Реализация

```go
// File: internal/application/chat/list_participants.go
package chat

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
)

// ListParticipantsUseCase получает список участников чата
type ListParticipantsUseCase struct {
    eventStore shared.EventStore
}

// NewListParticipantsUseCase создает новый ListParticipantsUseCase
func NewListParticipantsUseCase(eventStore shared.EventStore) *ListParticipantsUseCase {
    return &ListParticipantsUseCase{
        eventStore: eventStore,
    }
}

// Execute выполняет получение участников
func (uc *ListParticipantsUseCase) Execute(ctx context.Context, query ListParticipantsQuery) (ParticipantsQueryResult, error) {
    // Валидация
    if err := uc.validate(query); err != nil {
        return ParticipantsQueryResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Загрузка чата
    events, err := uc.eventStore.LoadEvents(ctx, query.ChatID.String())
    if err != nil {
        return ParticipantsQueryResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return ParticipantsQueryResult{}, ErrChatNotFound
    }

    // Восстановление агрегата
    chatAggregate := &chat.Chat{}
    if err := chatAggregate.LoadFromHistory(events); err != nil {
        return ParticipantsQueryResult{}, fmt.Errorf("failed to load from history: %w", err)
    }

    // Получение участников
    participants := chatAggregate.Participants()

    return ParticipantsQueryResult{
        Participants: participants,
        Total:        len(participants),
    }, nil
}

func (uc *ListParticipantsUseCase) validate(query ListParticipantsQuery) error {
    if query.ChatID.IsNil() {
        return shared.NewValidationError("chatID", "must be a valid UUID")
    }
    return nil
}
```

#### Результат

```go
// File: internal/application/chat/results.go (добавить)

// ParticipantsQueryResult - результат для списка участников
type ParticipantsQueryResult struct {
    Participants []chat.Participant
    Total        int
}
```

#### Тесты

```go
// File: internal/application/chat/list_participants_test.go
package chat_test

func TestListParticipantsUseCase_Success(t *testing.T) {
    // Создать чат с 3 участниками
    // Получить список участников
    // Проверить, что вернулись все 3
}

func TestListParticipantsUseCase_Success_WithRoles(t *testing.T) {
    // Создать чат с Admin и Member
    // Проверить роли участников
}

func TestListParticipantsUseCase_Error_ChatNotFound(t *testing.T) {
    // Несуществующий chatID
    // Ожидаем ErrChatNotFound
}

func TestListParticipantsUseCase_ValidationError_InvalidChatID(t *testing.T) {}

func TestListParticipantsUseCase_EventStoreError(t *testing.T) {}
```

**Тесты:** 5
**Время:** 20 минут

---

## Структура файлов

После реализации структура будет:

```
internal/application/chat/
├── commands.go           ✅ (существует)
├── queries.go            ⚠️ СОЗДАТЬ
├── results.go            ⚠️ ОБНОВИТЬ (добавить ListQueryResult, ParticipantsQueryResult)
├── errors.go             ✅ (существует)
│
├── create_chat.go        ✅
├── add_participant.go    ✅
├── ... (другие commands) ✅
│
├── get_chat.go           ❌ СОЗДАТЬ
├── list_chats.go         ❌ СОЗДАТЬ
├── list_participants.go  ❌ СОЗДАТЬ
│
├── get_chat_test.go           ❌ СОЗДАТЬ
├── list_chats_test.go         ❌ СОЗДАТЬ
├── list_participants_test.go  ❌ СОЗДАТЬ
```

## Checklist

### Подготовка (10 минут)
- [x] Создать `queries.go` со всеми Query структурами
- [x] Обновить `results.go` с новыми типами результатов
- [x] Проверить mock EventStore поддерживает LoadEventsByPrefix

### Реализация (1.5 часа)
- [x] GetChatUseCase (30 мин)
  - [x] Реализация
  - [x] 4 теста
- [x] ListChatsUseCase (40 мин)
  - [x] Реализация
  - [x] 6 тестов
  - [x] Поддержка фильтрации
  - [x] Поддержка pagination
- [x] ListParticipantsUseCase (30 мин)
  - [x] Реализация
  - [x] 5 тестов

### Проверка (10 минут)
- [x] Запустить тесты: `go test ./internal/application/chat/... -v -run Query`
- [x] Проверить coverage
- [x] Проверить линтер

## Метрики успеха

- ✅ **3 Query UseCases** реализованы
- ✅ **15 unit тестов** создано
- ✅ **Coverage >85%** для query файлов
- ✅ **Все тесты проходят**

## Оценка времени

| Этап | Время |
|------|-------|
| Подготовка | 10 минут |
| GetChatUseCase | 30 минут |
| ListChatsUseCase | 40 минут |
| ListParticipantsUseCase | 30 минут |
| Проверка | 10 минут |
| **ИТОГО** | **2 часа** |

## Примечание о производительности

⚠️ **ВАЖНО:** Текущая реализация ListChatsUseCase использует EventStore, что может быть неэффективно для больших workspace.

**Для production:**
- Рассмотреть использование Read Model (CQRS projection)
- Создать MongoDB view с индексом по workspace_id
- Использовать Redis cache для часто запрашиваемых списков

**Для текущей фазы (UseCase layer):**
- Достаточно EventStore реализации
- Read Model будет создан в фазе инфраструктуры

## Следующие шаги

После завершения:
- [x] Обновить PROGRESS_TRACKER.md (Phase 2 Query UseCases)
- [x] Объединить с Task 09 (Chat Tests)
- [x] Полностью завершить Phase 2
- [ ] Перейти к infrastructure implementation

## Референсы

- Пример Query UseCases: `internal/application/message/get_message.go`
- Пример pagination: `internal/application/message/list_messages.go`
- EventStore interface: `internal/application/shared/eventstore.go`
