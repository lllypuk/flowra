# Task 0.2: Chat Query UseCases Implementation

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** ✅ Completed
**Время:** 1.5-2 часа
**Зависимости:** Task 0.1 (желательно, но не обязательно)

---

## Проблема

Query use cases для Chat не реализованы. Невозможно получить данные чата для UI.

**Текущее состояние:**
- ✅ Command use cases реализованы (12 штук)
- ✅ Query use cases реализованы (3 штуки)
- ✅ UI может получить данные
- ✅ Разблокировано: Frontend development, API handlers

---

## Цель

Реализовать 3 query use cases с полным тестированием:
1. GetChatUseCase
2. ListChatsUseCase
3. ListParticipantsUseCase

---

## Файлы для создания

```
internal/application/chat/
├── queries.go                  (query definitions - NEW)
├── get_chat.go                 (NEW)
├── list_chats.go               (NEW)
├── list_participants.go        (NEW)
├── get_chat_test.go            (NEW - 4 tests)
├── list_chats_test.go          (NEW - 6 tests)
└── list_participants_test.go   (NEW - 5 tests)

Итого: 6 новых файлов, 15 unit tests
```

---

## Детальный план реализации

### 1. Query Definitions (queries.go)

Определить все query структуры и результаты.

```go
package chat

import (
    "time"
    "github.com/google/uuid"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

// GetChatQuery - запрос на получение чата
type GetChatQuery struct {
    ChatID      uuid.UUID
    RequestedBy uuid.UUID  // для проверки доступа
}

// GetChatResult - результат получения чата
type GetChatResult struct {
    Chat        *ChatDTO
    Permissions ChatPermissions  // read/write/admin
}

// ChatDTO - Data Transfer Object для чата
type ChatDTO struct {
    ID           uuid.UUID           `json:"id"`
    WorkspaceID  uuid.UUID           `json:"workspace_id"`
    Type         chatdomain.ChatType `json:"type"`
    Title        string              `json:"title"`
    IsPublic     bool                `json:"is_public"`
    CreatedBy    uuid.UUID           `json:"created_by"`
    CreatedAt    time.Time           `json:"created_at"`

    // Task-specific fields (optional)
    Status       *chatdomain.TaskStatus   `json:"status,omitempty"`
    AssignedTo   *uuid.UUID               `json:"assigned_to,omitempty"`
    Priority     *chatdomain.Priority     `json:"priority,omitempty"`
    DueDate      *time.Time               `json:"due_date,omitempty"`

    // Bug-specific fields (optional)
    Severity     *chatdomain.BugSeverity  `json:"severity,omitempty"`

    // Participants
    Participants []ParticipantDTO `json:"participants"`
}

// ChatPermissions - права пользователя на чат
type ChatPermissions struct {
    CanRead   bool `json:"can_read"`
    CanWrite  bool `json:"can_write"`
    CanManage bool `json:"can_manage"`  // admin rights
}

// ParticipantDTO - участник чата
type ParticipantDTO struct {
    UserID     uuid.UUID             `json:"user_id"`
    Role       chatdomain.ParticipantRole `json:"role"`
    JoinedAt   time.Time             `json:"joined_at"`
}

// ListChatsQuery - запрос на список чатов
type ListChatsQuery struct {
    WorkspaceID uuid.UUID
    Type        *chatdomain.ChatType  // optional filter
    Limit       int
    Offset      int
    RequestedBy uuid.UUID
}

// ListChatsResult - результат списка чатов
type ListChatsResult struct {
    Chats   []ChatDTO `json:"chats"`
    Total   int       `json:"total"`
    HasMore bool      `json:"has_more"`
}

// ListParticipantsQuery - запрос на список участников
type ListParticipantsQuery struct {
    ChatID      uuid.UUID
    RequestedBy uuid.UUID
}

// ListParticipantsResult - результат списка участников
type ListParticipantsResult struct {
    Participants []ParticipantDTO `json:"participants"`
}
```

---

### 2. GetChatUseCase Implementation (get_chat.go)

```go
package chat

import (
    "context"
    "fmt"
    "github.com/google/uuid"

    "github.com/lllypuk/flowra/internal/application/shared"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

// ChatRepository - репозиторий для query операций
type ChatRepository interface {
    Load(ctx context.Context, chatID uuid.UUID) (*chatdomain.Chat, error)
}

// GetChatUseCase - use case для получения чата
type GetChatUseCase struct {
    chatRepo   ChatRepository
    eventStore shared.EventStore
}

// NewGetChatUseCase - конструктор
func NewGetChatUseCase(chatRepo ChatRepository, eventStore shared.EventStore) *GetChatUseCase {
    return &GetChatUseCase{
        chatRepo:   chatRepo,
        eventStore: eventStore,
    }
}

// Execute - выполнить запрос
func (uc *GetChatUseCase) Execute(ctx context.Context, query GetChatQuery) (*GetChatResult, error) {
    // 1. Load chat from repository
    chat, err := uc.chatRepo.Load(ctx, query.ChatID)
    if err != nil {
        return nil, fmt.Errorf("failed to load chat: %w", err)
    }

    // 2. Check access permissions
    if !chat.IsPublic() && !chat.IsParticipant(query.RequestedBy) {
        return nil, fmt.Errorf("access denied: user is not a participant")
    }

    // 3. Build ChatDTO
    chatDTO := mapChatToDTO(chat)

    // 4. Calculate permissions
    permissions := calculatePermissions(chat, query.RequestedBy)

    return &GetChatResult{
        Chat:        chatDTO,
        Permissions: permissions,
    }, nil
}

// Helper: map domain Chat to DTO
func mapChatToDTO(chat *chatdomain.Chat) *ChatDTO {
    dto := &ChatDTO{
        ID:          chat.ID(),
        WorkspaceID: chat.WorkspaceID(),
        Type:        chat.Type(),
        Title:       chat.Title(),
        IsPublic:    chat.IsPublic(),
        CreatedBy:   chat.CreatedBy(),
        CreatedAt:   chat.CreatedAt(),
    }

    // Add task-specific fields if applicable
    if chat.Type() == chatdomain.ChatTypeTask || chat.Type() == chatdomain.ChatTypeBug || chat.Type() == chatdomain.ChatTypeEpic {
        status := chat.Status()
        dto.Status = &status

        if assignedTo := chat.AssignedTo(); assignedTo != nil {
            dto.AssignedTo = assignedTo
        }

        if priority := chat.Priority(); priority != nil {
            dto.Priority = priority
        }

        if dueDate := chat.DueDate(); dueDate != nil {
            dto.DueDate = dueDate
        }
    }

    // Add bug-specific fields
    if chat.Type() == chatdomain.ChatTypeBug {
        if severity := chat.Severity(); severity != nil {
            dto.Severity = severity
        }
    }

    // Map participants
    dto.Participants = make([]ParticipantDTO, 0)
    for _, p := range chat.Participants() {
        dto.Participants = append(dto.Participants, ParticipantDTO{
            UserID:   p.UserID,
            Role:     p.Role,
            JoinedAt: p.JoinedAt,
        })
    }

    return dto
}

// Helper: calculate user permissions
func calculatePermissions(chat *chatdomain.Chat, userID uuid.UUID) ChatPermissions {
    permissions := ChatPermissions{}

    // Public chats: everyone can read
    if chat.IsPublic() {
        permissions.CanRead = true
    }

    // Participants can read and write
    if chat.IsParticipant(userID) {
        permissions.CanRead = true
        permissions.CanWrite = true
    }

    // Creator and admins can manage
    if chat.CreatedBy() == userID {
        permissions.CanManage = true
    }

    // Check if user is admin role
    for _, p := range chat.Participants() {
        if p.UserID == userID && p.Role == chatdomain.ParticipantRoleAdmin {
            permissions.CanManage = true
            break
        }
    }

    return permissions
}
```

---

### 3. ListChatsUseCase Implementation (list_chats.go)

```go
package chat

import (
    "context"
    "fmt"
    "github.com/google/uuid"

    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

// ChatQueryRepository - extended repository with query methods
type ChatQueryRepository interface {
    ChatRepository
    FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, chatType *chatdomain.ChatType, limit, offset int) ([]chatdomain.Chat, error)
    CountByWorkspace(ctx context.Context, workspaceID uuid.UUID, chatType *chatdomain.ChatType) (int, error)
}

// ListChatsUseCase - use case для получения списка чатов
type ListChatsUseCase struct {
    chatRepo ChatQueryRepository
}

// NewListChatsUseCase - конструктор
func NewListChatsUseCase(chatRepo ChatQueryRepository) *ListChatsUseCase {
    return &ListChatsUseCase{
        chatRepo: chatRepo,
    }
}

// Execute - выполнить запрос
func (uc *ListChatsUseCase) Execute(ctx context.Context, query ListChatsQuery) (*ListChatsResult, error) {
    // 1. Validate pagination
    if query.Limit <= 0 {
        query.Limit = 20  // default
    }
    if query.Limit > 100 {
        query.Limit = 100  // max
    }

    // 2. Find chats
    chats, err := uc.chatRepo.FindByWorkspace(ctx, query.WorkspaceID, query.Type, query.Limit+1, query.Offset)
    if err != nil {
        return nil, fmt.Errorf("failed to find chats: %w", err)
    }

    // 3. Filter by user access (only participant or public chats)
    accessibleChats := make([]chatdomain.Chat, 0)
    for _, chat := range chats {
        if chat.IsPublic() || chat.IsParticipant(query.RequestedBy) {
            accessibleChats = append(accessibleChats, chat)
        }
    }

    // 4. Check if has more
    hasMore := len(accessibleChats) > query.Limit
    if hasMore {
        accessibleChats = accessibleChats[:query.Limit]
    }

    // 5. Map to DTOs
    chatDTOs := make([]ChatDTO, len(accessibleChats))
    for i, chat := range accessibleChats {
        chatDTOs[i] = *mapChatToDTO(&chat)
    }

    // 6. Count total (optional, for pagination info)
    total, err := uc.chatRepo.CountByWorkspace(ctx, query.WorkspaceID, query.Type)
    if err != nil {
        total = len(chatDTOs)  // fallback
    }

    return &ListChatsResult{
        Chats:   chatDTOs,
        Total:   total,
        HasMore: hasMore,
    }, nil
}
```

---

### 4. ListParticipantsUseCase Implementation (list_participants.go)

```go
package chat

import (
    "context"
    "fmt"
    "sort"
    "github.com/google/uuid"
)

// ListParticipantsUseCase - use case для получения списка участников
type ListParticipantsUseCase struct {
    chatRepo ChatRepository
}

// NewListParticipantsUseCase - конструктор
func NewListParticipantsUseCase(chatRepo ChatRepository) *ListParticipantsUseCase {
    return &ListParticipantsUseCase{
        chatRepo: chatRepo,
    }
}

// Execute - выполнить запрос
func (uc *ListParticipantsUseCase) Execute(ctx context.Context, query ListParticipantsQuery) (*ListParticipantsResult, error) {
    // 1. Load chat
    chat, err := uc.chatRepo.Load(ctx, query.ChatID)
    if err != nil {
        return nil, fmt.Errorf("failed to load chat: %w", err)
    }

    // 2. Check access (must be participant or public chat)
    if !chat.IsPublic() && !chat.IsParticipant(query.RequestedBy) {
        return nil, fmt.Errorf("access denied: user is not a participant")
    }

    // 3. Get participants
    participants := chat.Participants()

    // 4. Sort by join date (ascending)
    sort.Slice(participants, func(i, j int) bool {
        return participants[i].JoinedAt.Before(participants[j].JoinedAt)
    })

    // 5. Map to DTOs
    participantDTOs := make([]ParticipantDTO, len(participants))
    for i, p := range participants {
        participantDTOs[i] = ParticipantDTO{
            UserID:   p.UserID,
            Role:     p.Role,
            JoinedAt: p.JoinedAt,
        }
    }

    return &ListParticipantsResult{
        Participants: participantDTOs,
    }, nil
}
```

---

### 5. GetChatUseCase Tests (get_chat_test.go)

**Тесты (4 total):**

```go
package chat_test

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/chat"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

func TestGetChatUseCase_Success(t *testing.T) {
    // Arrange
    chatRepo, eventStore := setupQueryTest(t)

    workspaceID := uuid.New()
    createdBy := uuid.New()
    requestedBy := uuid.New()

    testChat := createTestChat(t, chatdomain.ChatTypeDiscussion)
    testChat.AddParticipant(requestedBy, chatdomain.ParticipantRoleMember)

    chatRepo.On("Load", mock.Anything, testChat.ID()).Return(testChat, nil)

    useCase := chat.NewGetChatUseCase(chatRepo, eventStore)

    query := chat.GetChatQuery{
        ChatID:      testChat.ID(),
        RequestedBy: requestedBy,
    }

    // Act
    result, err := useCase.Execute(context.Background(), query)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotNil(t, result.Chat)
    assert.Equal(t, testChat.ID(), result.Chat.ID)
    assert.True(t, result.Permissions.CanRead)
    assert.True(t, result.Permissions.CanWrite)
}

func TestGetChatUseCase_Error_ChatNotFound(t *testing.T) {
    // Test when chat doesn't exist
}

func TestGetChatUseCase_Error_AccessDenied(t *testing.T) {
    // Test when user is not participant and chat is private
}

func TestGetChatUseCase_Success_PublicChat(t *testing.T) {
    // Test when non-participant accesses public chat
}
```

---

### 6. ListChatsUseCase Tests (list_chats_test.go)

**Тесты (6 total):**

```go
func TestListChatsUseCase_Success_AllChats(t *testing.T) {
    // List all chats in workspace
}

func TestListChatsUseCase_Success_FilterByType_Task(t *testing.T) {
    // Filter only Task type chats
}

func TestListChatsUseCase_Success_Pagination(t *testing.T) {
    // Test pagination works correctly
}

func TestListChatsUseCase_Success_OnlyUserChats(t *testing.T) {
    // Only returns chats where user is participant or public
}

func TestListChatsUseCase_Success_IncludesPublicChats(t *testing.T) {
    // Public chats are included even if not participant
}

func TestListChatsUseCase_Error_InvalidWorkspace(t *testing.T) {
    // Workspace doesn't exist
}
```

---

### 7. ListParticipantsUseCase Tests (list_participants_test.go)

**Тесты (5 total):**

```go
func TestListParticipantsUseCase_Success(t *testing.T) {
    // List all participants
}

func TestListParticipantsUseCase_Error_ChatNotFound(t *testing.T) {
    // Chat doesn't exist
}

func TestListParticipantsUseCase_Error_NotParticipant(t *testing.T) {
    // User is not participant and chat is private
}

func TestListParticipantsUseCase_Success_IncludesRoles(t *testing.T) {
    // Verify roles are included
}

func TestListParticipantsUseCase_Success_SortedByJoinDate(t *testing.T) {
    // Verify sorting by join date
}
```

---

## Интеграция с существующим кодом

### Repository Interface Updates

Нужно расширить существующий ChatRepository:

```go
// internal/application/chat/repository.go (update existing)

type ChatRepository interface {
    // Event sourcing (existing)
    Load(ctx context.Context, chatID uuid.UUID) (*chatdomain.Chat, error)
    Save(ctx context.Context, chat *chatdomain.Chat) error

    // Query methods (NEW)
    FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, chatType *chatdomain.ChatType, limit, offset int) ([]chatdomain.Chat, error)
    CountByWorkspace(ctx context.Context, workspaceID uuid.UUID, chatType *chatdomain.ChatType) (int, error)
}
```

---

## Критерии успеха

- ✅ **3 query use cases реализованы** (Get, List, ListParticipants)
- ✅ **15 unit tests покрывают все сценарии**
- ✅ **Coverage >85%**
- ✅ **Pagination протестирована**
- ✅ **Authorization checks на месте**
- ✅ **DTOs корректно маппятся**
- ✅ **Public/Private access работает**

---

## Референсы

**Примеры для копирования структуры:**
- `internal/application/message/query_messages.go`
- `internal/application/user/get_user.go`

---

## Следующий шаг

После завершения этой задачи → **Task 0.3: Documentation Sync**
