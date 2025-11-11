# Task 0.1: Chat UseCases Testing (БЛОКЕР)

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** ✅ Completed
**Время:** 3-4 часа (Completed)
**Зависимости:** Нет

---

## Проблема

Chat domain имеет **0% test coverage** при 12 реализованных use cases. Это наибольший риск проекта - ключевая функциональность может содержать критические баги.

**Текущее состояние:**
- ✅ 12 command use cases реализованы
- ❌ 0 unit tests
- ❌ Coverage: 0%
- ⚠️ Risk: HIGH - критическая функциональность не протестирована

---

## Цель

Создать comprehensive test suite для всех Chat use cases с покрытием >85%.

---

## Файлы для создания

```
internal/application/chat/
├── create_chat_test.go          (8 тестов)
├── participants_test.go         (12 тестов: Add/Remove)
├── convert_test.go              (12 тестов: Task/Bug/Epic)
├── management_test.go           (15 тестов: Status/Assign/Priority/DueDate)
├── rename_severity_test.go      (10 тестов)
└── test_setup.go                (mocks setup)

Итого: ~60 unit tests
```

---

## Детальный план реализации

### 1. Test Setup (test_setup.go)

Создать вспомогательные функции и моки для всех тестов.

```go
package chat_test

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/mock"

    "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
)

// Mock EventStore
type MockEventStore struct {
    mock.Mock
}

func (m *MockEventStore) SaveEvents(ctx context.Context, aggregateID uuid.UUID, events []shared.DomainEvent, expectedVersion int) error {
    args := m.Called(ctx, aggregateID, events, expectedVersion)
    return args.Error(0)
}

func (m *MockEventStore) LoadEvents(ctx context.Context, aggregateID uuid.UUID) ([]shared.DomainEvent, error) {
    args := m.Called(ctx, aggregateID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]shared.DomainEvent), args.Error(1)
}

// Mock UserRepository
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
    args := m.Called(ctx, userID)
    return args.Bool(0), args.Error(1)
}

// Mock WorkspaceRepository
type MockWorkspaceRepository struct {
    mock.Mock
}

func (m *MockWorkspaceRepository) Exists(ctx context.Context, workspaceID uuid.UUID) (bool, error) {
    args := m.Called(ctx, workspaceID)
    return args.Bool(0), args.Error(1)
}

func (m *MockWorkspaceRepository) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
    args := m.Called(ctx, workspaceID, userID)
    return args.Bool(0), args.Error(1)
}

// Test Fixtures
func createTestChat(t *testing.T, chatType chatdomain.ChatType) *chatdomain.Chat {
    t.Helper()

    workspaceID := uuid.New()
    createdBy := uuid.New()

    chat, err := chatdomain.NewChat(workspaceID, chatType, "Test Chat", true, createdBy)
    if err != nil {
        t.Fatalf("Failed to create test chat: %v", err)
    }

    return chat
}

func setupChatUseCaseTest(t *testing.T) (*MockEventStore, *MockUserRepository, *MockWorkspaceRepository) {
    t.Helper()

    return &MockEventStore{}, &MockUserRepository{}, &MockWorkspaceRepository{}
}
```

---

### 2. CreateChatUseCase Tests (create_chat_test.go)

**Тесты (8 total):**

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

func TestCreateChatUseCase_Success_Discussion(t *testing.T) {
    // Arrange
    eventStore, userRepo, workspaceRepo := setupChatUseCaseTest(t)

    workspaceID := uuid.New()
    createdBy := uuid.New()

    workspaceRepo.On("Exists", mock.Anything, workspaceID).Return(true, nil)
    workspaceRepo.On("IsMember", mock.Anything, workspaceID, createdBy).Return(true, nil)
    userRepo.On("Exists", mock.Anything, createdBy).Return(true, nil)
    eventStore.On("SaveEvents", mock.Anything, mock.Anything, mock.Anything, 0).Return(nil)

    useCase := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)

    cmd := chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "New Discussion",
        IsPublic:    true,
        CreatedBy:   createdBy,
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, result.ChatID)
    assert.Equal(t, chatdomain.ChatTypeDiscussion, result.Type)
    assert.Equal(t, "New Discussion", result.Title)

    eventStore.AssertExpectations(t)
    workspaceRepo.AssertExpectations(t)
    userRepo.AssertExpectations(t)
}

func TestCreateChatUseCase_Success_Task(t *testing.T) {
    // Test creating Task type chat
    // Similar structure to above
}

func TestCreateChatUseCase_Success_Bug(t *testing.T) {
    // Test creating Bug type chat
}

func TestCreateChatUseCase_Success_Epic(t *testing.T) {
    // Test creating Epic type chat
}

func TestCreateChatUseCase_Error_WorkspaceNotFound(t *testing.T) {
    // Arrange
    eventStore, userRepo, workspaceRepo := setupChatUseCaseTest(t)

    workspaceID := uuid.New()
    createdBy := uuid.New()

    workspaceRepo.On("Exists", mock.Anything, workspaceID).Return(false, nil)

    useCase := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)

    cmd := chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "New Discussion",
        IsPublic:    true,
        CreatedBy:   createdBy,
    }

    // Act
    _, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "workspace not found")
}

func TestCreateChatUseCase_Error_UserNotMember(t *testing.T) {
    // Test when user is not a workspace member
}

func TestCreateChatUseCase_Error_InvalidTitle(t *testing.T) {
    // Test with empty or invalid title
}

func TestCreateChatUseCase_Error_InvalidType(t *testing.T) {
    // Test with invalid chat type
}
```

---

### 3. Participants Tests (participants_test.go)

**Тесты (12 total):**

```go
func TestAddParticipantUseCase_Success(t *testing.T) {
    // Happy path: add participant to chat
}

func TestAddParticipantUseCase_Success_MultipleParticipants(t *testing.T) {
    // Add multiple participants sequentially
}

func TestAddParticipantUseCase_Error_ChatNotFound(t *testing.T) {
    // Chat doesn't exist
}

func TestAddParticipantUseCase_Error_UserNotFound(t *testing.T) {
    // User doesn't exist
}

func TestAddParticipantUseCase_Error_AlreadyParticipant(t *testing.T) {
    // User already in chat
}

func TestAddParticipantUseCase_Error_NotAuthorized(t *testing.T) {
    // Requester is not chat admin
}

func TestRemoveParticipantUseCase_Success(t *testing.T) {
    // Happy path: remove participant
}

func TestRemoveParticipantUseCase_Success_SelfRemove(t *testing.T) {
    // User removes themselves
}

func TestRemoveParticipantUseCase_Error_ChatNotFound(t *testing.T) {
    // Chat doesn't exist
}

func TestRemoveParticipantUseCase_Error_UserNotParticipant(t *testing.T) {
    // User not in chat
}

func TestRemoveParticipantUseCase_Error_CannotRemoveCreator(t *testing.T) {
    // Cannot remove chat creator
}

func TestRemoveParticipantUseCase_Error_NotAuthorized(t *testing.T) {
    // Requester is not admin
}
```

---

### 4. Convert Tests (convert_test.go)

**Тесты (12 total):**

```go
func TestConvertToTaskUseCase_Success_FromDiscussion(t *testing.T) {
    // Convert Discussion → Task
}

func TestConvertToTaskUseCase_Success_WithTitle(t *testing.T) {
    // Convert with custom task title
}

func TestConvertToBugUseCase_Success_FromDiscussion(t *testing.T) {
    // Convert Discussion → Bug
}

func TestConvertToBugUseCase_Success_WithSeverity(t *testing.T) {
    // Convert with severity set
}

func TestConvertToEpicUseCase_Success_FromDiscussion(t *testing.T) {
    // Convert Discussion → Epic
}

func TestConvertToEpicUseCase_Success_WithDescription(t *testing.T) {
    // Convert with epic description
}

func TestConvertToTaskUseCase_Error_ChatNotFound(t *testing.T) {
    // Chat doesn't exist
}

func TestConvertToTaskUseCase_Error_AlreadyTask(t *testing.T) {
    // Chat is already a Task
}

func TestConvertToTaskUseCase_Error_NotAuthorized(t *testing.T) {
    // User not authorized to convert
}

func TestConvertToBugUseCase_Error_InvalidSeverity(t *testing.T) {
    // Invalid severity value
}

func TestConvertToEpicUseCase_Error_PrivateChat(t *testing.T) {
    // Cannot convert private chat to Epic
}

func TestConvertToTaskUseCase_EventPublished(t *testing.T) {
    // Verify ChatConvertedToTask event is published
}
```

---

### 5. Management Tests (management_test.go)

**Тесты (15 total):**

```go
func TestChangeStatusUseCase_Success_ToInProgress(t *testing.T) {
    // Change status: New → In Progress
}

func TestChangeStatusUseCase_Success_ToDone(t *testing.T) {
    // Change status: In Progress → Done
}

func TestChangeStatusUseCase_Error_InvalidTransition(t *testing.T) {
    // Invalid status transition (e.g., New → Done)
}

func TestChangeStatusUseCase_Error_NotTask(t *testing.T) {
    // Cannot change status on Discussion
}

func TestAssignUserUseCase_Success(t *testing.T) {
    // Assign user to task
}

func TestAssignUserUseCase_Success_Reassign(t *testing.T) {
    // Reassign task to different user
}

func TestAssignUserUseCase_Error_UserNotFound(t *testing.T) {
    // Assignee doesn't exist
}

func TestAssignUserUseCase_Error_UserNotParticipant(t *testing.T) {
    // Assignee not in chat
}

func TestSetPriorityUseCase_Success_High(t *testing.T) {
    // Set priority to High
}

func TestSetPriorityUseCase_Success_Low(t *testing.T) {
    // Set priority to Low
}

func TestSetPriorityUseCase_Error_InvalidPriority(t *testing.T) {
    // Invalid priority value
}

func TestSetDueDateUseCase_Success_FutureDate(t *testing.T) {
    // Set due date in future
}

func TestSetDueDateUseCase_Success_ClearDueDate(t *testing.T) {
    // Clear existing due date
}

func TestSetDueDateUseCase_Error_PastDate(t *testing.T) {
    // Cannot set due date in past
}

func TestSetDueDateUseCase_Error_NotTask(t *testing.T) {
    // Cannot set due date on Discussion
}
```

---

### 6. Rename/Severity Tests (rename_severity_test.go)

**Тесты (10 total):**

```go
func TestRenameChatUseCase_Success(t *testing.T) {
    // Rename chat successfully
}

func TestRenameChatUseCase_Success_SameTitle(t *testing.T) {
    // Rename to same title (no-op)
}

func TestRenameChatUseCase_Error_EmptyTitle(t *testing.T) {
    // Cannot rename to empty string
}

func TestRenameChatUseCase_Error_TooLongTitle(t *testing.T) {
    // Title exceeds max length
}

func TestRenameChatUseCase_Error_NotAuthorized(t *testing.T) {
    // User not authorized to rename
}

func TestSetSeverityUseCase_Success_Critical(t *testing.T) {
    // Set severity to Critical
}

func TestSetSeverityUseCase_Success_Low(t *testing.T) {
    // Set severity to Low
}

func TestSetSeverityUseCase_Error_NotBug(t *testing.T) {
    // Cannot set severity on non-Bug
}

func TestSetSeverityUseCase_Error_InvalidSeverity(t *testing.T) {
    // Invalid severity value
}

func TestSetSeverityUseCase_EventPublished(t *testing.T) {
    // Verify SeverityChanged event published
}
```

---

## Тестовое покрытие

Каждый use case должен тестировать:

### Happy Path
- ✅ Успешное выполнение команды
- ✅ Корректные возвращаемые значения
- ✅ Events опубликованы

### Error Cases
- ❌ Validation errors (empty fields, invalid types)
- ❌ Authorization errors (not member, not admin)
- ❌ Not found errors (chat, user, workspace)
- ❌ Business rule violations (duplicate participant, invalid transition)

### Edge Cases
- ⚠️ Boundary values (max length titles, dates)
- ⚠️ Concurrent modifications (optimistic locking)
- ⚠️ State transitions (valid/invalid status changes)

---

## Критерии успеха

- ✅ **60 unit tests** созданы
- ✅ **Coverage Chat domain: 0% → 85%+**
- ✅ **Application Layer overall: 64.7% → 75%+**
- ✅ **Все тесты проходят** (green)
- ✅ **No regressions** в других доменах
- ✅ **Mock expectations** проверены
- ✅ **Event publishing** верифицирован

---

## Референсы

**Примеры для копирования структуры:**
- `internal/application/message/send_message_test.go`
- `internal/application/message/edit_message_test.go`
- `internal/application/task/change_status_test.go`

**Mock utilities:**
- `tests/testutil/mocks.go` (если есть shared mocks)

---

## Зависимости

**Packages to import:**
```go
import (
    "testing"
    "context"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"

    "github.com/lllypuk/flowra/internal/application/chat"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/application/shared"
)
```

---

## Следующий шаг

После завершения этой задачи → **Task 0.2: Chat Query UseCases Implementation**
