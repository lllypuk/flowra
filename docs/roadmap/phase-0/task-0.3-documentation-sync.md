# Task 0.3: Documentation Sync

**Приоритет:** 🟡 MEDIUM
**Статус:** ✅ COMPLETED (2025-11-11)
**Время:** 1 час (completed in ~1.5 hours)
**Зависимости:** Task 0.1, Task 0.2 (для актуальных метрик)

---

## Проблема

README и архитектурная документация устарели и не отражают текущее состояние проекта.

**Текущее состояние:**
- ❌ README.md содержит устаревшие метрики
- ❌ Отсутствует секция "Current Status"
- ❌ docs/01-architecture.md не документирует Tag Processing
- ❌ Нет примеров использования API
- ❌ Quick Start не включает тестирование

---

## Цель

Синхронизировать документацию с текущим состоянием кода:
1. Обновить README.md
2. Обновить docs/01-architecture.md
3. Создать docs/API_USAGE.md

---

## Файлы для обновления/создания

```
├── README.md                    (UPDATE - основной README)
├── docs/
│   ├── 01-architecture.md       (UPDATE - архитектура)
│   └── API_USAGE.md             (NEW - примеры использования)
```

---

## Детальный план реализации

### 1. README.md Updates

#### Секция: Project Metrics (обновить)

**Текущее (устаревшее):**
```markdown
## Project Metrics
- Lines of Code: ~18,000
- Use Cases: 30+
- Test Coverage: Domain 85%+, Application 60%
```

**Новое:**
```markdown
## Project Metrics

**Version:** 0.4.0-alpha
**Status:** Active Development (Phase 2-3, 82% Complete)

- **Lines of Code:** ~23,000 LOC
- **Use Cases:** 40+ implemented
- **Test Coverage:**
  - Domain Layer: 90%+ ✅
  - Application Layer: 75%+ ✅ (was 64.7%, fixed in Phase 0)
  - Infrastructure: Not yet implemented
- **Domains:** 6 Event-Sourced Aggregates (Chat, Message, Task, Notification, User, Workspace)
- **Events:** 30+ domain events
```

---

#### Секция: Current Status (добавить новую)

**Добавить после Project Metrics:**

```markdown
## Current Status

### ✅ Completed
- **Domain Layer (90%+)** - Fully functional
  - 6 Event-Sourced aggregates with comprehensive business logic
  - Tag Processing System for chat commands
  - 30+ domain events

- **Application Layer (75%+)** - Complete
  - 40+ use cases (commands + queries)
  - All domains have >75% test coverage
  - Full CQRS implementation

- **Testing Infrastructure (85%)** - Solid foundation
  - Mocks, Fixtures, Test Utilities
  - Integration test helpers
  - MongoDB v2 test setup

### 🚧 In Progress
- **Infrastructure Layer (30%)**
  - ✅ In-memory Event Store (functional for testing)
  - ✅ MongoDB v2 connection setup
  - ✅ Redis client setup
  - ⏳ MongoDB/Redis repositories (not implemented)
  - ⏳ Event Bus (not implemented)

- **Interface Layer (0%)** - Not started
  - HTTP handlers, middleware, WebSocket

- **Entry Points (0%)** - Not started
  - API server, Worker service

### 📋 Next Steps
See [Development Roadmap](docs/DEVELOPMENT_ROADMAP_2025.md) for detailed plan.

**Immediate focus:**
1. Infrastructure Layer (repositories, event bus)
2. HTTP/WebSocket interface layer
3. Entry points (cmd/api, cmd/worker)
4. Minimal HTMX frontend
```

---

#### Секция: Quick Start (обновить)

**Добавить секцию "Running Tests":**

```markdown
## Quick Start

### Prerequisites
- Go 1.25+
- Docker & Docker Compose
- MongoDB 6+ (via Docker)
- Redis (via Docker)

### Setup

1. **Clone repository:**
   ```bash
   git clone https://github.com/lllypuk/flowra.git
   cd flowra
   ```

2. **Start infrastructure:**
   ```bash
   docker-compose up -d mongodb redis keycloak
   ```

3. **Run tests:** (NEW)
   ```bash
   # Run all tests
   go test ./...

   # Run specific domain tests
   go test ./internal/domain/chat/...
   go test ./internal/application/chat/...

   # Run with coverage
   go test -cover ./internal/application/...

   # Integration tests (requires MongoDB)
   go test -tags=integration ./tests/integration/...
   ```

4. **Start application:** (when implemented)
   ```bash
   go run cmd/api/main.go
   ```

### Example: Using Chat Domain

**Create a chat and send a message with task command:**

```go
package main

import (
    "context"
    "github.com/google/uuid"

    "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/application/message"
)

func main() {
    // Setup (repositories, event store, etc.)
    // ...

    // Create a chat
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    chatResult, _ := createChatUC.Execute(ctx, chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "Project Planning",
        IsPublic:    true,
        CreatedBy:   userID,
    })

    // Send message with task command
    sendMsgUC := message.NewSendMessageUseCase(msgRepo, chatRepo, eventStore, tagProcessor)
    msgResult, _ := sendMsgUC.Execute(ctx, message.SendMessageCommand{
        ChatID:    chatResult.ChatID,
        Content:   "We need to implement authentication #createTask",
        SentBy:    userID,
    })

    // Task automatically created via Tag Processing!
}
```
```

---

### 2. docs/01-architecture.md Updates

#### Добавить диаграмму актуальных слоев

```markdown
## Architecture Layers (Updated 2025-11-11)

```
┌─────────────────────────────────────────────────────────────┐
│                    Interface Layer                          │
│  (Echo HTTP handlers, WebSocket, Middleware)                │
│                  [NOT IMPLEMENTED]                          │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────────────┐
│                 Application Layer ✅ 75%                    │
│  ┌──────────────┬──────────────┬───────────────────────┐   │
│  │ Chat         │ Message      │ Task/User/Workspace   │   │
│  │ UseCases     │ UseCases     │ Notification UseCases │   │
│  │ (12 cmd + 3q)│ (8 total)    │ (20 total)            │   │
│  └──────────────┴──────────────┴───────────────────────┘   │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Tag Processing System ✅                           │    │
│  │ (Integrated with Message UseCases)                 │    │
│  └────────────────────────────────────────────────────┘    │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────────────┐
│                   Domain Layer ✅ 90%                       │
│  ┌──────────┬─────────┬──────┬──────────────┬──────────┐   │
│  │ Chat     │ Message │ Task │ Notification │ User     │   │
│  │ Aggregate│ Entity  │ Agg  │ Aggregate    │ Workspace│   │
│  └──────────┴─────────┴──────┴──────────────┴──────────┘   │
│                                                              │
│  30+ Domain Events (ChatCreated, StatusChanged, etc.)       │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────────────┐
│              Infrastructure Layer ⏳ 30%                    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Event Store:  ✅ In-Memory (testing)               │    │
│  │               ⏳ MongoDB (not impl)                │    │
│  └────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Repositories: ⏳ MongoDB (not impl)                │    │
│  │               ⏳ Redis (not impl)                  │    │
│  └────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Event Bus:    ⏳ Redis Pub/Sub (not impl)         │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```
```

---

#### Добавить секцию Tag Processing Integration

```markdown
## Tag Processing System

**Status:** ✅ Implemented and integrated

The Tag Processing System enables chat commands via hashtags in messages.

### Flow

1. **User sends message** with tag: `"Fix the bug #createTask"`
2. **SendMessageUseCase** saves message and publishes `MessagePosted` event
3. **TagProcessorHandler** listens to event, parses tags
4. **CommandExecutor** executes tag command (e.g., CreateTaskCommand)
5. **Task created** automatically from chat message

### Supported Tags

| Tag | Command | Description |
|-----|---------|-------------|
| `#createTask` | ConvertToTask | Convert chat to Task |
| `#createBug` | ConvertToBug | Convert chat to Bug with severity |
| `#createEpic` | ConvertToEpic | Convert chat to Epic |
| `#setStatus <status>` | ChangeStatus | Update task status |
| `#assign @user` | AssignUser | Assign task to user |
| `#setPriority <p>` | SetPriority | Set task priority |
| `#setDueDate <date>` | SetDueDate | Set task due date |
| `#setSeverity <s>` | SetSeverity | Set bug severity |

### Integration Points

- **Domain:** `internal/domain/tag/` (parser, executor)
- **Application:** Integrated into `SendMessageUseCase`
- **Event Flow:** `MessagePosted` → `TagProcessorHandler` → Command execution

### Example

```go
// User sends message
cmd := message.SendMessageCommand{
    ChatID:  chatID,
    Content: "Critical login issue #createBug severity:high",
    SentBy:  userID,
}

sendMsgUC.Execute(ctx, cmd)

// Result:
// 1. Message saved
// 2. MessagePosted event published
// 3. Tag processor detects #createBug
// 4. Chat converted to Bug with High severity
// 5. BugCreated event published
```
```

---

### 3. docs/API_USAGE.md (NEW)

Создать новый файл с примерами использования.

```markdown
# API Usage Examples

This document provides code examples for using the application layer use cases.

**Last Updated:** 2025-11-11
**Version:** 0.4.0-alpha

---

## Table of Contents

1. [Setup](#setup)
2. [Chat Domain](#chat-domain)
3. [Message Domain](#message-domain)
4. [Task Management](#task-management)
5. [Tag Processing](#tag-processing)
6. [Notifications](#notifications)

---

## Setup

### Dependency Injection

```go
package main

import (
    "context"
    "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/application/message"
    "github.com/lllypuk/flowra/internal/infrastructure/eventstore"
)

func setupUseCases() {
    // Infrastructure
    eventStore := eventstore.NewInMemoryEventStore()

    // Repositories (mocks for now)
    chatRepo := &MockChatRepository{}
    messageRepo := &MockMessageRepository{}
    userRepo := &MockUserRepository{}
    workspaceRepo := &MockWorkspaceRepository{}

    // Use Cases
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventStore, tagProcessor)

    // ... more use cases
}
```

---

## Chat Domain

### Create a Chat

```go
// Create Discussion chat
cmd := chat.CreateChatCommand{
    WorkspaceID: workspaceID,
    Type:        chatdomain.ChatTypeDiscussion,
    Title:       "Project Planning Meeting",
    IsPublic:    true,
    CreatedBy:   userID,
}

result, err := createChatUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Chat created: %s\n", result.ChatID)
```

### Get Chat Details

```go
query := chat.GetChatQuery{
    ChatID:      chatID,
    RequestedBy: userID,
}

result, err := getChatUC.Execute(ctx, query)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Chat: %s\n", result.Chat.Title)
fmt.Printf("Can Manage: %v\n", result.Permissions.CanManage)
```

### List Chats

```go
query := chat.ListChatsQuery{
    WorkspaceID: workspaceID,
    Type:        &chatdomain.ChatTypeTask,  // filter by type (optional)
    Limit:       20,
    Offset:      0,
    RequestedBy: userID,
}

result, err := listChatsUC.Execute(ctx, query)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d chats (Total: %d)\n", len(result.Chats), result.Total)
for _, chat := range result.Chats {
    fmt.Printf("- %s (%s)\n", chat.Title, chat.Type)
}
```

### Add Participant

```go
cmd := chat.AddParticipantCommand{
    ChatID:      chatID,
    UserID:      newUserID,
    Role:        chatdomain.ParticipantRoleMember,
    RequestedBy: adminUserID,
}

err := addParticipantUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}
```

### Convert Chat to Task

```go
cmd := chat.ConvertToTaskCommand{
    ChatID:      chatID,
    Title:       "Implement authentication",
    RequestedBy: userID,
}

result, err := convertToTaskUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Chat converted to Task: %s\n", result.TaskID)
```

---

## Message Domain

### Send Message

```go
cmd := message.SendMessageCommand{
    ChatID:   chatID,
    Content:  "Hello team! Let's discuss the architecture.",
    SentBy:   userID,
}

result, err := sendMessageUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Message sent: %s\n", result.MessageID)
```

### Send Message with Tag Command

```go
cmd := message.SendMessageCommand{
    ChatID:   chatID,
    Content:  "We need to fix the login bug #createBug severity:critical",
    SentBy:   userID,
}

result, err := sendMessageUC.Execute(ctx, cmd)
// Result:
// 1. Message created
// 2. Chat converted to Bug
// 3. Severity set to Critical
```

### Edit Message

```go
cmd := message.EditMessageCommand{
    MessageID:   messageID,
    NewContent:  "Updated content",
    EditedBy:    userID,
}

err := editMessageUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}
```

### Reply to Message (Thread)

```go
cmd := message.SendMessageCommand{
    ChatID:   chatID,
    ParentID: &parentMessageID,  // creates thread
    Content:  "I agree with your point!",
    SentBy:   userID,
}

result, err := sendMessageUC.Execute(ctx, cmd)
```

---

## Task Management

### Change Task Status

```go
cmd := chat.ChangeStatusCommand{
    ChatID:      taskChatID,
    Status:      chatdomain.TaskStatusInProgress,
    RequestedBy: userID,
}

err := changeStatusUC.Execute(ctx, cmd)
if err != nil {
    log.Fatal(err)
}
```

### Assign Task

```go
cmd := chat.AssignUserCommand{
    ChatID:      taskChatID,
    AssignedTo:  assigneeID,
    RequestedBy: managerID,
}

err := assignUserUC.Execute(ctx, cmd)
```

### Set Priority

```go
cmd := chat.SetPriorityCommand{
    ChatID:      taskChatID,
    Priority:    chatdomain.PriorityHigh,
    RequestedBy: userID,
}

err := setPriorityUC.Execute(ctx, cmd)
```

### Set Due Date

```go
dueDate := time.Now().Add(7 * 24 * time.Hour)  // 1 week from now

cmd := chat.SetDueDateCommand{
    ChatID:      taskChatID,
    DueDate:     &dueDate,
    RequestedBy: userID,
}

err := setDueDateUC.Execute(ctx, cmd)
```

---

## Tag Processing

### Supported Tag Commands

**Create Commands:**
```go
// Convert to Task
"Let's implement this feature #createTask"

// Convert to Bug with severity
"Login fails on mobile #createBug severity:high"

// Convert to Epic
"Q2 Roadmap planning #createEpic"
```

**Status Commands:**
```go
// Change status
"Starting work on this #setStatus inprogress"
"This is done #setStatus done"
```

**Assignment Commands:**
```go
// Assign to user
"Bob will handle this #assign @bob"
```

**Priority Commands:**
```go
// Set priority
"This is urgent #setPriority high"
```

**Due Date Commands:**
```go
// Set due date
"Deadline next week #setDueDate 2025-11-18"
```

**Bug Severity Commands:**
```go
// Set bug severity
"Critical issue #setSeverity critical"
```

### Integration Example

```go
// Setup tag processor
tagParser := tag.NewParser()
commandExecutor := tag.NewCommandExecutor(chatRepo, userRepo, messageRepo, eventStore)
tagProcessor := tag.NewTagProcessor(tagParser, commandExecutor)

// Integrate with SendMessageUseCase
sendMessageUC := message.NewSendMessageUseCase(
    messageRepo,
    chatRepo,
    eventStore,
    tagProcessor,  // injected
)

// Usage
cmd := message.SendMessageCommand{
    ChatID:   chatID,
    Content:  "Fix authentication bug #createBug severity:critical #setPriority high",
    SentBy:   userID,
}

result, _ := sendMessageUC.Execute(ctx, cmd)
// Chat is now a Bug with Critical severity and High priority
```

---

## Notifications

### Create Notification

```go
cmd := notification.CreateNotificationCommand{
    UserID:  recipientID,
    Type:    notificationdomain.NotificationTypeTaskAssigned,
    Title:   "New task assigned",
    Content: "You have been assigned to 'Implement Authentication'",
    Link:    fmt.Sprintf("/chats/%s", taskChatID),
}

result, err := createNotificationUC.Execute(ctx, cmd)
```

### List Unread Notifications

```go
query := notification.ListNotificationsQuery{
    UserID:     userID,
    UnreadOnly: true,
    Limit:      20,
    Offset:     0,
}

result, err := listNotificationsUC.Execute(ctx, query)

fmt.Printf("Unread: %d\n", len(result.Notifications))
for _, notif := range result.Notifications {
    fmt.Printf("- %s: %s\n", notif.Type, notif.Title)
}
```

### Mark as Read

```go
cmd := notification.MarkAsReadCommand{
    NotificationID: notificationID,
    UserID:         userID,
}

err := markAsReadUC.Execute(ctx, cmd)
```

---

## Complete Workflow Example

```go
func completeWorkflow() {
    ctx := context.Background()

    // 1. Create workspace
    workspaceCmd := workspace.CreateWorkspaceCommand{
        Name:      "Acme Corp",
        CreatedBy: adminID,
    }
    wsResult, _ := createWorkspaceUC.Execute(ctx, workspaceCmd)

    // 2. Create discussion chat
    chatCmd := chat.CreateChatCommand{
        WorkspaceID: wsResult.WorkspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "Sprint Planning",
        IsPublic:    true,
        CreatedBy:   adminID,
    }
    chatResult, _ := createChatUC.Execute(ctx, chatCmd)

    // 3. Send message with task command
    msgCmd := message.SendMessageCommand{
        ChatID:  chatResult.ChatID,
        Content: "We need authentication by next week #createTask #setPriority high #setDueDate 2025-11-18",
        SentBy:  adminID,
    }
    sendMessageUC.Execute(ctx, msgCmd)

    // 4. Assign task
    assignCmd := chat.AssignUserCommand{
        ChatID:      chatResult.ChatID,
        AssignedTo:  developerID,
        RequestedBy: adminID,
    }
    assignUserUC.Execute(ctx, assignCmd)

    // 5. Developer updates status
    statusCmd := chat.ChangeStatusCommand{
        ChatID:      chatResult.ChatID,
        Status:      chatdomain.TaskStatusInProgress,
        RequestedBy: developerID,
    }
    changeStatusUC.Execute(ctx, statusCmd)

    // 6. List tasks
    listQuery := chat.ListChatsQuery{
        WorkspaceID: wsResult.WorkspaceID,
        Type:        &chatdomain.ChatTypeTask,
        RequestedBy: adminID,
    }
    tasks, _ := listChatsUC.Execute(ctx, listQuery)

    fmt.Printf("Active tasks: %d\n", len(tasks.Chats))
}
```

---

## Testing Examples

See test files for comprehensive examples:
- `internal/application/chat/*_test.go`
- `internal/application/message/*_test.go`
- `internal/domain/tag/*_test.go`

---

## Next Steps

Once Infrastructure Layer is implemented:
- MongoDB persistence examples
- Event Bus integration examples
- WebSocket real-time updates
- HTTP API endpoint examples
```

---

## Критерии успеха

- ✅ **README.md обновлен** с актуальными метриками
- ✅ **Current Status секция** добавлена
- ✅ **Quick Start** включает примеры тестирования
- ✅ **docs/01-architecture.md** документирует Tag Processing
- ✅ **docs/API_USAGE.md** создан с полными примерами
- ✅ **Новый разработчик** может разобраться за 30 минут
- ✅ **Примеры кода работают** (проверены на актуальности)

---

## Checklist

### README.md
- [x] Update Project Metrics (LOC, use cases, coverage)
- [x] Add Current Status section (completed, in progress, next)
- [x] Update Quick Start with test commands
- [x] Add example code snippet for Chat + Message + Tag integration

### docs/01-architecture.md
- [x] Add updated architecture layers diagram
- [x] Document Tag Processing System integration
- [x] Update Event Flow with Tag Processing example
- [x] Add status indicators (✅ implemented, ⏳ in progress)

### docs/API_USAGE.md
- [x] Create new file
- [x] Add setup and dependency injection examples
- [x] Document all Chat domain use cases with examples
- [x] Document all Message domain use cases with examples
- [x] Document Task management use cases
- [x] Document Tag Processing with all supported tags
- [x] Add complete workflow example
- [x] Link to test files for more examples

---

## Следующий шаг

После завершения Phase 0 → **Phase 1: Infrastructure Layer**
