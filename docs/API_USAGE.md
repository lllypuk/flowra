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
7. [Complete Workflow Example](#complete-workflow-example)
8. [Testing Examples](#testing-examples)

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

    // Repositories (mocks for now, MongoDB later)
    chatRepo := &MockChatRepository{}
    messageRepo := &MockMessageRepository{}
    userRepo := &MockUserRepository{}
    workspaceRepo := &MockWorkspaceRepository{}
    taskRepo := &MockTaskRepository{}
    notificationRepo := &MockNotificationRepository{}

    // Tag Processing
    tagParser := tag.NewParser()
    tagExecutor := tag.NewCommandExecutor(chatRepo, userRepo, messageRepo, eventStore)
    tagProcessor := tag.NewTagProcessor(tagParser, tagExecutor)

    // Use Cases
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventStore, tagProcessor)
    createNotificationUC := notification.NewCreateNotificationUseCase(eventStore, notificationRepo)

    // ... more use cases
}
```

---

## Chat Domain

### Create a Chat

```go
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
// 4. BugCreated and SeveritySet events published
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

### Delete Message

```go
cmd := message.DeleteMessageCommand{
    MessageID:   messageID,
    DeletedBy:   userID,
}

err := deleteMessageUC.Execute(ctx, cmd)
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

### Set Severity (Bug Only)

```go
cmd := chat.SetSeverityCommand{
    ChatID:      bugChatID,
    Severity:    chatdomain.SeverityCritical,
    RequestedBy: userID,
}

err := setSeverityUC.Execute(ctx, cmd)
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

### Mark All as Read

```go
cmd := notification.MarkAllAsReadCommand{
    UserID: userID,
}

err := markAllAsReadUC.Execute(ctx, cmd)
```

---

## Complete Workflow Example

```go
func completeWorkflow(ctx context.Context) {
    // 1. Create workspace
    workspaceCmd := workspace.CreateWorkspaceCommand{
        Name:      "Acme Corp",
        CreatedBy: adminID,
    }
    wsResult, _ := createWorkspaceUC.Execute(ctx, workspaceCmd)
    workspaceID := wsResult.WorkspaceID

    // 2. Create discussion chat
    chatCmd := chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "Sprint Planning",
        IsPublic:    true,
        CreatedBy:   adminID,
    }
    chatResult, _ := createChatUC.Execute(ctx, chatCmd)
    chatID := chatResult.ChatID

    // 3. Send message with task command
    msgCmd := message.SendMessageCommand{
        ChatID:  chatID,
        Content: "We need authentication by next week #createTask #setPriority high #setDueDate 2025-11-18",
        SentBy:  adminID,
    }
    sendMessageUC.Execute(ctx, msgCmd)
    // Now chatID is a Task with High priority and Due Date set

    // 4. Assign task
    assignCmd := chat.AssignUserCommand{
        ChatID:      chatID,
        AssignedTo:  developerID,
        RequestedBy: adminID,
    }
    assignUserUC.Execute(ctx, assignCmd)

    // 5. Developer updates status
    statusCmd := chat.ChangeStatusCommand{
        ChatID:      chatID,
        Status:      chatdomain.TaskStatusInProgress,
        RequestedBy: developerID,
    }
    changeStatusUC.Execute(ctx, statusCmd)

    // 6. Developer adds comment (new message in thread)
    commentCmd := message.SendMessageCommand{
        ChatID:  chatID,
        Content: "Working on OAuth 2.0 integration",
        SentBy:  developerID,
    }
    sendMessageUC.Execute(ctx, commentCmd)

    // 7. List tasks
    listQuery := chat.ListChatsQuery{
        WorkspaceID: workspaceID,
        Type:        &chatdomain.ChatTypeTask,
        RequestedBy: adminID,
    }
    tasks, _ := listChatsUC.Execute(ctx, listQuery)

    fmt.Printf("Active tasks in workspace: %d\n", len(tasks.Chats))
    for _, task := range tasks.Chats {
        fmt.Printf("- %s (Priority: %s, Status: %s)\n",
            task.Title, task.Priority, task.Status)
    }
}
```

---

## Testing Examples

### Unit Testing with Fixtures

```go
func TestCreateChat(t *testing.T) {
    // Create test data using fluent API
    cmd := fixtures.NewCreateChatCommand().
        WithTitle("Test Chat").
        WithType(chatdomain.ChatTypeDiscussion).
        WithCreator(fixtures.UserID1).
        Build()

    // Execute use case
    result, err := uc.Execute(context.Background(), cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, result.ChatID)
}
```

### Integration Testing with MongoDB

```go
func TestSendMessageIntegration(t *testing.T) {
    // Setup in-memory MongoDB (testcontainers)
    container, db := testutil.SetupMongoDB(t)
    defer testutil.TeardownMongoDB(t, container)

    // Create repositories
    msgRepo := persistence.NewMongoMessageRepository(db)
    chatRepo := persistence.NewMongoChatRepository(db)

    // Create and execute use case
    cmd := message.SendMessageCommand{
        ChatID:  chatID,
        Content: "Test message",
        SentBy:  userID,
    }
    result, err := sendMessageUC.Execute(context.Background(), cmd)

    // Verify in database
    retrievedMsg, err := msgRepo.GetByID(context.Background(), result.MessageID)
    assert.NoError(t, err)
    assert.Equal(t, "Test message", retrievedMsg.Content)
}
```

### Mock Testing

```go
func TestTagProcessing(t *testing.T) {
    // Setup mocks
    mockChatRepo := &MockChatRepository{}
    mockUserRepo := &MockUserRepository{}

    // Create use case with mocks
    executor := tag.NewCommandExecutor(
        mockChatRepo,
        mockUserRepo,
        mockMessageRepo,
        eventStore,
    )

    // Execute tag processing
    tags := []tag.Tag{
        {Name: "createBug", Params: map[string]string{"severity": "critical"}},
    }
    err := executor.Execute(ctx, chatID, tags)

    // Assert expectations
    assert.NoError(t, err)
    assert.True(t, mockChatRepo.ConvertToBugCalled)
}
```

---

## See Also

For more detailed examples and test cases:
- `internal/application/chat/*_test.go` - Chat use case tests
- `internal/application/message/*_test.go` - Message use case tests
- `internal/domain/tag/*_test.go` - Tag processing tests
- `tests/testutil/fixtures.go` - Test data builders
- `tests/integration/*_test.go` - Integration tests

---

## Next Steps

Once Infrastructure Layer is implemented:
- MongoDB persistence examples
- Event Bus integration examples
- WebSocket real-time updates
- HTTP API endpoint examples

See [Development Roadmap](./DEVELOPMENT_ROADMAP_2025.md) for implementation timeline.
