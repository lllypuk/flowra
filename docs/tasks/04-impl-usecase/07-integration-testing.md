# Task 07: Integration & End-to-End Testing

**Дата:** 2025-10-19
**Статус:** 📝 Pending
**Зависимости:** Tasks 01-06 (все UseCases)
**Оценка:** 4-5 часов

## Цель

Создать полную тестовую инфраструктуру для интеграционного и end-to-end тестирования UseCases с фокусом на cross-domain взаимодействия.

## Контекст

После реализации всех UseCases нужно:
- Протестировать взаимодействие между доменами
- Проверить end-to-end workflows (полные сценарии пользователя)
- Настроить тестовую инфраструктуру (mocks, fixtures, helpers)
- Обеспечить покрытие >80%

## Типы тестов

### 1. Unit Tests (уже реализованы)
- Каждый UseCase тестируется отдельно
- Используются моки для зависимостей
- Покрытие: валидация, бизнес-логика, обработка ошибок

### 2. Integration Tests (эта задача)
- Тестирование взаимодействия между UseCases
- Реальные репозитории (in-memory)
- Проверка Event Bus integration
- Проверка транзакционности

### 3. End-to-End Tests (эта задача)
- Полные пользовательские сценарии
- Несколько доменов взаимодействуют
- Проверка eventual consistency
- Проверка event handlers

## Структура тестов

```
tests/
├── mocks/                          # Mock implementations
│   ├── chat_repository.go
│   ├── message_repository.go
│   ├── user_repository.go
│   ├── workspace_repository.go
│   ├── notification_repository.go
│   ├── eventstore.go
│   ├── eventbus.go
│   └── keycloak_client.go
│
├── fixtures/                       # Test data builders
│   ├── chat_fixtures.go
│   ├── message_fixtures.go
│   ├── user_fixtures.go
│   ├── workspace_fixtures.go
│   └── notification_fixtures.go
│
├── testutil/                       # Test utilities
│   ├── context.go                  # Context helpers
│   ├── assert.go                   # Custom assertions
│   └── suite.go                    # Test suite base
│
├── integration/                    # Integration tests
│   ├── chat_integration_test.go
│   ├── message_integration_test.go
│   ├── workspace_integration_test.go
│   ├── notification_integration_test.go
│   └── eventbus_integration_test.go
│
└── e2e/                            # End-to-end tests
    ├── chat_workflow_test.go
    ├── task_workflow_test.go
    ├── messaging_workflow_test.go
    └── workspace_workflow_test.go
```

## 1. Mock Implementations

### Chat Repository Mock

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

func (r *ChatRepository) SaveCallCount() int {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.calls["Save"]
}

func (r *ChatRepository) LoadCallCount() int {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.calls["Load"]
}
```

### Event Bus Mock

```go
// tests/mocks/eventbus.go
package mocks

import (
    "context"
    "sync"

    "github.com/flowra/flowra/internal/domain/event"
)

type EventBus struct {
    mu          sync.RWMutex
    published   []event.Event
    subscribers map[string][]event.Handler
}

func NewEventBus() *EventBus {
    return &EventBus{
        published:   []event.Event{},
        subscribers: make(map[string][]event.Handler),
    }
}

func (b *EventBus) Publish(ctx context.Context, evt event.Event) error {
    b.mu.Lock()
    b.published = append(b.published, evt)
    handlers := b.subscribers[evt.EventType()]
    b.mu.Unlock()

    // Синхронный вызов handlers (для тестов)
    for _, handler := range handlers {
        if err := handler(ctx, evt); err != nil {
            return err
        }
    }

    return nil
}

func (b *EventBus) Subscribe(eventType string, handler event.Handler) {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *EventBus) PublishCallCount() int {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return len(b.published)
}

func (b *EventBus) PublishedEvents() []event.Event {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return append([]event.Event{}, b.published...)
}
```

## 2. Test Fixtures

```go
// tests/fixtures/chat_fixtures.go
package fixtures

import (
    "github.com/google/uuid"
    "github.com/flowra/flowra/internal/domain/chat"
    domainUUID "github.com/flowra/flowra/internal/domain/uuid"
)

type ChatBuilder struct {
    workspaceID uuid.UUID
    title       string
    chatType    chat.Type
    createdBy   uuid.UUID
}

func NewChatBuilder() *ChatBuilder {
    return &ChatBuilder{
        workspaceID: uuid.New(),
        title:       "Test Chat",
        chatType:    chat.TypeDiscussion,
        createdBy:   uuid.New(),
    }
}

func (b *ChatBuilder) WithWorkspace(id uuid.UUID) *ChatBuilder {
    b.workspaceID = id
    return b
}

func (b *ChatBuilder) WithTitle(title string) *ChatBuilder {
    b.title = title
    return b
}

func (b *ChatBuilder) AsTask() *ChatBuilder {
    b.chatType = chat.TypeTask
    return b
}

func (b *ChatBuilder) CreatedBy(userID uuid.UUID) *ChatBuilder {
    b.createdBy = userID
    return b
}

func (b *ChatBuilder) Build() *chat.Chat {
    wsID := domainUUID.FromGoogleUUID(b.workspaceID)
    creatorID := domainUUID.FromGoogleUUID(b.createdBy)

    c := chat.NewChat(wsID, b.title, b.chatType, creatorID)
    c.AddParticipant(creatorID, chat.RoleAdmin)

    return c
}
```

## 3. Integration Tests

### Event Bus Integration Test

```go
// tests/integration/eventbus_integration_test.go
package integration

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/flowra/flowra/internal/application/chat"
    "github.com/flowra/flowra/internal/application/notification"
    "github.com/flowra/flowra/internal/application/eventhandlers"
    domainChat "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/tests/mocks"
)

func TestEventBusIntegration_ChatCreated_CreatesNotification(t *testing.T) {
    // Arrange
    chatRepo := mocks.NewChatRepository()
    notificationRepo := mocks.NewNotificationRepository()
    eventBus := mocks.NewEventBus()

    // Setup UseCases
    createChatUseCase := chat.NewCreateChatUseCase(chatRepo, eventBus)
    createNotificationUseCase := notification.NewCreateNotificationUseCase(notificationRepo)

    // Setup Event Handler
    notificationHandler := eventhandlers.NewNotificationEventHandler(createNotificationUseCase)
    eventBus.Subscribe(domainChat.EventTypeChatCreated, notificationHandler.HandleChatCreated)

    // Act: создание чата
    cmd := chat.CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "Test Chat",
        Type:        domainChat.TypeDiscussion,
        CreatedBy:   uuid.New(),
    }

    result, err := createChatUseCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result.Value)

    // Проверка, что notification создан
    notifications := notificationRepo.FindAll()
    assert.Len(t, notifications, 1, "notification should be created via event handler")
    assert.Equal(t, cmd.CreatedBy, notifications[0].UserID())
}
```

### Cross-Domain Integration Test

```go
// tests/integration/chat_message_integration_test.go
package integration

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    chatApp "github.com/flowra/flowra/internal/application/chat"
    messageApp "github.com/flowra/flowra/internal/application/message"
    "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/tests/mocks"
)

func TestChatMessageIntegration_SendMessage_RequiresParticipation(t *testing.T) {
    // Arrange
    chatRepo := mocks.NewChatRepository()
    chatReadModelRepo := mocks.NewChatReadModelRepository()
    messageRepo := mocks.NewMessageRepository()
    eventBus := mocks.NewEventBus()

    createChatUseCase := chatApp.NewCreateChatUseCase(chatRepo, eventBus)
    sendMessageUseCase := messageApp.NewSendMessageUseCase(
        messageRepo,
        chatReadModelRepo,
        eventBus,
    )

    // Act: создание чата
    userID := uuid.New()
    createChatCmd := chatApp.CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "Test Chat",
        Type:        chat.TypeDiscussion,
        CreatedBy:   userID,
    }

    chatResult, err := createChatUseCase.Execute(context.Background(), createChatCmd)
    require.NoError(t, err)

    chatID := chatResult.Value.ID().ToGoogleUUID()

    // Sync chat to read model
    chatReadModelRepo.AddChat(chatResult.Value)

    // Act: отправка сообщения участником (должно пройти)
    sendMsgCmd := messageApp.SendMessageCommand{
        ChatID:   chatID,
        Content:  "Hello!",
        AuthorID: userID,
    }

    msgResult, err := sendMessageUseCase.Execute(context.Background(), sendMsgCmd)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, msgResult.Value)
    assert.Equal(t, "Hello!", msgResult.Value.Content())

    // Act: отправка сообщения НЕ участником (должно упасть)
    nonParticipantID := uuid.New()
    sendMsgCmd2 := messageApp.SendMessageCommand{
        ChatID:   chatID,
        Content:  "I'm not a participant",
        AuthorID: nonParticipantID,
    }

    msgResult2, err2 := sendMessageUseCase.Execute(context.Background(), sendMsgCmd2)

    // Assert
    assert.Error(t, err2)
    assert.ErrorIs(t, err2, messageApp.ErrNotChatParticipant)
}
```

## 4. End-to-End Tests

### Complete Task Workflow

```go
// tests/e2e/task_workflow_test.go
package e2e

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    chatApp "github.com/flowra/flowra/internal/application/chat"
    messageApp "github.com/flowra/flowra/internal/application/message"
    notificationApp "github.com/flowra/flowra/internal/application/notification"
    "github.com/flowra/flowra/internal/domain/chat"
    "github.com/flowra/flowra/tests/fixtures"
    "github.com/flowra/flowra/tests/mocks"
    "github.com/flowra/flowra/tests/testutil"
)

func TestE2E_TaskWorkflow(t *testing.T) {
    // Setup
    suite := testutil.NewTestSuite(t)

    user1 := uuid.New()
    user2 := uuid.New()
    workspaceID := uuid.New()

    // Сценарий:
    // 1. User1 создает Discussion чат
    // 2. User1 добавляет User2 как участника
    // 3. User1 отправляет сообщение с тегом !task
    // 4. Чат конвертируется в Task
    // 5. User1 назначает задачу на User2
    // 6. User2 получает notification
    // 7. User2 меняет статус на "In Progress"
    // 8. User1 получает notification

    ctx := context.Background()

    // Step 1: Create chat
    createChatCmd := chatApp.CreateChatCommand{
        WorkspaceID: workspaceID,
        Title:       "Discuss new feature",
        Type:        chat.TypeDiscussion,
        CreatedBy:   user1,
    }

    chatResult, err := suite.ChatUseCases.CreateChat.Execute(ctx, createChatCmd)
    require.NoError(t, err)
    chatID := chatResult.Value.ID().ToGoogleUUID()

    // Step 2: Add participant
    addParticipantCmd := chatApp.AddParticipantCommand{
        ChatID:  chatID,
        UserID:  user2,
        Role:    chat.RoleMember,
        AddedBy: user1,
    }

    _, err = suite.ChatUseCases.AddParticipant.Execute(ctx, addParticipantCmd)
    require.NoError(t, err)

    // Step 3: Send message with !task tag
    sendMsgCmd := messageApp.SendMessageCommand{
        ChatID:   chatID,
        Content:  "Let's create a task for this !task",
        AuthorID: user1,
    }

    msgResult, err := suite.MessageUseCases.SendMessage.Execute(ctx, sendMsgCmd)
    require.NoError(t, err)

    // TODO: Tag processing integration
    // For now, manually convert to task

    // Step 4: Convert to Task
    convertCmd := chatApp.ConvertToTaskCommand{
        ChatID:      chatID,
        Title:       "New feature task",
        ConvertedBy: user1,
    }

    _, err = suite.ChatUseCases.ConvertToTask.Execute(ctx, convertCmd)
    require.NoError(t, err)

    // Step 5: Assign to User2
    assignCmd := chatApp.AssignUserCommand{
        ChatID:     chatID,
        AssigneeID: &user2,
        AssignedBy: user1,
    }

    _, err = suite.ChatUseCases.AssignUser.Execute(ctx, assignCmd)
    require.NoError(t, err)

    // Step 6: Check User2 notifications
    listNotificationsQuery := notificationApp.ListNotificationsQuery{
        UserID:     user2,
        UnreadOnly: true,
    }

    notificationsResult, err := suite.NotificationUseCases.ListNotifications.Execute(ctx, listNotificationsQuery)
    require.NoError(t, err)
    assert.GreaterOrEqual(t, len(notificationsResult.Value), 1, "User2 should have notification about assignment")

    // Step 7: User2 changes status
    changeStatusCmd := chatApp.ChangeStatusCommand{
        ChatID:    chatID,
        Status:    chat.StatusInProgress,
        ChangedBy: user2,
    }

    _, err = suite.ChatUseCases.ChangeStatus.Execute(ctx, changeStatusCmd)
    require.NoError(t, err)

    // Step 8: Check User1 notifications
    listNotificationsQuery2 := notificationApp.ListNotificationsQuery{
        UserID:     user1,
        UnreadOnly: true,
    }

    notificationsResult2, err := suite.NotificationUseCases.ListNotifications.Execute(ctx, listNotificationsQuery2)
    require.NoError(t, err)
    assert.GreaterOrEqual(t, len(notificationsResult2.Value), 1, "User1 should have notification about status change")

    // Verify final state
    getChat Query := chatApp.GetChatQuery{
        ChatID: chatID,
        UserID: user1,
    }

    finalChatResult, err := suite.ChatUseCases.GetChat.Execute(ctx, getChatQuery)
    require.NoError(t, err)
    assert.Equal(t, chat.TypeTask, finalChatResult.Value.Type())
    assert.Equal(t, chat.StatusInProgress, finalChatResult.Value.Status())
    assert.Equal(t, user2, finalChatResult.Value.AssigneeID().ToGoogleUUID())
}
```

## 5. Test Suite Helper

```go
// tests/testutil/suite.go
package testutil

import (
    "testing"

    chatApp "github.com/flowra/flowra/internal/application/chat"
    messageApp "github.com/flowra/flowra/internal/application/message"
    notificationApp "github.com/flowra/flowra/internal/application/notification"
    userApp "github.com/flowra/flowra/internal/application/user"
    workspaceApp "github.com/flowra/flowra/internal/application/workspace"
    "github.com/flowra/flowra/tests/mocks"
)

type TestSuite struct {
    t *testing.T

    // Repositories
    ChatRepo         *mocks.ChatRepository
    MessageRepo      *mocks.MessageRepository
    UserRepo         *mocks.UserRepository
    WorkspaceRepo    *mocks.WorkspaceRepository
    NotificationRepo *mocks.NotificationRepository
    EventBus         *mocks.EventBus

    // Use Cases
    ChatUseCases         *ChatUseCases
    MessageUseCases      *MessageUseCases
    UserUseCases         *UserUseCases
    WorkspaceUseCases    *WorkspaceUseCases
    NotificationUseCases *NotificationUseCases
}

type ChatUseCases struct {
    CreateChat       *chatApp.CreateChatUseCase
    AddParticipant   *chatApp.AddParticipantUseCase
    ConvertToTask    *chatApp.ConvertToTaskUseCase
    ChangeStatus     *chatApp.ChangeStatusUseCase
    AssignUser       *chatApp.AssignUserUseCase
    GetChat          *chatApp.GetChatUseCase
}

// ... аналогично для других доменов ...

func NewTestSuite(t *testing.T) *TestSuite {
    suite := &TestSuite{
        t:                t,
        ChatRepo:         mocks.NewChatRepository(),
        MessageRepo:      mocks.NewMessageRepository(),
        UserRepo:         mocks.NewUserRepository(),
        WorkspaceRepo:    mocks.NewWorkspaceRepository(),
        NotificationRepo: mocks.NewNotificationRepository(),
        EventBus:         mocks.NewEventBus(),
    }

    // Initialize use cases
    suite.ChatUseCases = &ChatUseCases{
        CreateChat:     chatApp.NewCreateChatUseCase(suite.ChatRepo, suite.EventBus),
        AddParticipant: chatApp.NewAddParticipantUseCase(suite.ChatRepo, suite.EventBus),
        // ... и т.д.
    }

    // ... аналогично для других доменов ...

    return suite
}
```

## Checklist

- [ ] Создать все mock implementations
- [ ] Создать fixture builders для всех доменов
- [ ] Создать test utilities (context, assertions, suite)
- [ ] Написать integration tests для Event Bus
- [ ] Написать integration tests для cross-domain interactions
- [ ] Написать E2E test: Task workflow
- [ ] Написать E2E test: Messaging workflow
- [ ] Написать E2E test: Workspace workflow
- [ ] Настроить CI/CD для запуска тестов
- [ ] Проверить coverage (цель: >80%)

## Coverage Goals

| Layer | Target |
|-------|--------|
| Domain | >90% |
| Application (UseCases) | >85% |
| Integration | >70% |
| E2E | Key workflows covered |

## Следующие шаги

- **Task 08**: Tag Integration Refactoring
- Repository implementations (MongoDB)
- HTTP handlers with integration tests
