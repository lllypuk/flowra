# Task 007a: Implement New Tag Commands

## Status: ✅ Complete
## Priority: High
## Parent: Task 007
## Depends on: Task 007 (design review)
## Completed: 2026-01-16

---

## Objective

Add new tags for participant management and chat lifecycle:
- `#invite @username` - Add participant to chat
- `#remove @username` - Remove participant from chat
- `#close` - Close/archive chat or task
- `#reopen` - Reopen closed chat or task
- `#delete` - Delete chat or task (soft delete)

---

## Implementation Checklist

### 1. Add Command Structs

**File:** `internal/domain/tag/commands.go`

```go
// InviteUserCommand - command to add a participant to the chat
type InviteUserCommand struct {
    ChatID   uuid.UUID
    Username string     // @alex format
    UserID   *uuid.UUID // resolved ID (set by executor)
}

func (c InviteUserCommand) CommandType() string {
    return "InviteUser"
}

// RemoveUserCommand - command to remove a participant from the chat
type RemoveUserCommand struct {
    ChatID   uuid.UUID
    Username string
    UserID   *uuid.UUID
}

func (c RemoveUserCommand) CommandType() string {
    return "RemoveUser"
}

// CloseChatCommand - command to close/archive a chat
type CloseChatCommand struct {
    ChatID uuid.UUID
}

func (c CloseChatCommand) CommandType() string {
    return "CloseChat"
}

// ReopenChatCommand - command to reopen a closed chat
type ReopenChatCommand struct {
    ChatID uuid.UUID
}

func (c ReopenChatCommand) CommandType() string {
    return "ReopenChat"
}

// DeleteChatCommand - command to delete a chat (soft delete)
type DeleteChatCommand struct {
    ChatID uuid.UUID
}

func (c DeleteChatCommand) CommandType() string {
    return "DeleteChat"
}
```

### 2. Register Tags in Parser

**File:** `internal/domain/tag/parser.go`

Add to `NewParser()` function:

```go
knownTags: map[string]tagDef{
    // ... existing tags ...

    // Participant management
    "invite": {valueType: ValueTypeUsername},
    "remove": {valueType: ValueTypeUsername},

    // Chat lifecycle
    "close":  {valueType: ValueTypeNone},  // no value needed
    "reopen": {valueType: ValueTypeNone},
    "delete": {valueType: ValueTypeNone},
},
```

Add `ValueTypeNone` constant if not exists:
```go
const (
    ValueTypeString   ValueType = "string"
    ValueTypeUsername ValueType = "username"
    ValueTypeDate     ValueType = "date"
    ValueTypeEnum     ValueType = "enum"
    ValueTypeNone     ValueType = "none"  // New: tag without value
)
```

### 3. Add Processors

**File:** `internal/domain/tag/processor.go`

Add processing logic for new tags in `ProcessTags()`:

```go
case "invite":
    if err := validateUsername(tag.Value); err != nil {
        result.Errors = append(result.Errors, TagError{
            TagKey:   tag.Key,
            TagValue: tag.Value,
            Error:    err,
            Severity: ErrorSeverityError,
        })
        continue
    }
    cmd := InviteUserCommand{
        ChatID:   chatID,
        Username: tag.Value,
    }
    result.AppliedTags = append(result.AppliedTags, TagApplication{
        TagKey:   tag.Key,
        TagValue: tag.Value,
        Command:  cmd,
        Success:  true,
    })

case "remove":
    if err := validateUsername(tag.Value); err != nil {
        // ... error handling
    }
    cmd := RemoveUserCommand{
        ChatID:   chatID,
        Username: tag.Value,
    }
    // ... add to AppliedTags

case "close":
    if currentEntityType == "" {
        result.Errors = append(result.Errors, TagError{
            TagKey:   tag.Key,
            Error:    errors.New("cannot close a discussion, convert to task/bug/epic first"),
            Severity: ErrorSeverityError,
        })
        continue
    }
    cmd := CloseChatCommand{ChatID: chatID}
    // ... add to AppliedTags

case "reopen":
    cmd := ReopenChatCommand{ChatID: chatID}
    // ... add to AppliedTags

case "delete":
    cmd := DeleteChatCommand{ChatID: chatID}
    // ... add to AppliedTags
```

### 4. Add Validators

**File:** `internal/domain/tag/validators.go`

Reuse existing `validateUsername()` for invite/remove tags.

Add validation for close/delete:
```go
// ValidateCloseOperation validates if chat can be closed
func ValidateCloseOperation(chatType string) error {
    if chatType == "" || chatType == "Discussion" {
        return errors.New("cannot close a discussion")
    }
    return nil
}
```

### 5. Add Executors

**File:** `internal/domain/tag/executor.go`

```go
func (e *CommandExecutor) Execute(ctx context.Context, cmd Command, actorID uuid.UUID) error {
    switch c := cmd.(type) {
    // ... existing cases ...

    case InviteUserCommand:
        return e.executeInviteUser(ctx, c, actorID)
    case RemoveUserCommand:
        return e.executeRemoveUser(ctx, c, actorID)
    case CloseChatCommand:
        return e.executeCloseChat(ctx, c, actorID)
    case ReopenChatCommand:
        return e.executeReopenChat(ctx, c, actorID)
    case DeleteChatCommand:
        return e.executeDeleteChat(ctx, c, actorID)
    }
    return nil
}

func (e *CommandExecutor) executeInviteUser(ctx context.Context, cmd InviteUserCommand, actorID uuid.UUID) error {
    // Resolve username to userID
    username := strings.TrimPrefix(cmd.Username, "@")
    user, err := e.userRepo.FindByUsername(ctx, username)
    if err != nil {
        return fmt.Errorf("user @%s not found", username)
    }

    // Call AddParticipant use case
    addCmd := chatapp.AddParticipantCommand{
        ChatID:  cmd.ChatID,
        UserID:  user.ID(),
        Role:    chat.RoleMember,
        AddedBy: actorID,
    }
    _, err = e.chatUseCases.AddParticipant.Execute(ctx, addCmd)
    return err
}

func (e *CommandExecutor) executeRemoveUser(ctx context.Context, cmd RemoveUserCommand, actorID uuid.UUID) error {
    // Resolve username to userID
    username := strings.TrimPrefix(cmd.Username, "@")
    user, err := e.userRepo.FindByUsername(ctx, username)
    if err != nil {
        return fmt.Errorf("user @%s not found", username)
    }

    // Call RemoveParticipant use case
    removeCmd := chatapp.RemoveParticipantCommand{
        ChatID:    cmd.ChatID,
        UserID:    user.ID(),
        RemovedBy: actorID,
    }
    _, err = e.chatUseCases.RemoveParticipant.Execute(ctx, removeCmd)
    return err
}

func (e *CommandExecutor) executeCloseChat(ctx context.Context, cmd CloseChatCommand, actorID uuid.UUID) error {
    closeCmd := chatapp.CloseChatCommand{
        ChatID:   cmd.ChatID,
        ClosedBy: actorID,
    }
    _, err := e.chatUseCases.CloseChat.Execute(ctx, closeCmd)
    return err
}

func (e *CommandExecutor) executeReopenChat(ctx context.Context, cmd ReopenChatCommand, actorID uuid.UUID) error {
    reopenCmd := chatapp.ReopenChatCommand{
        ChatID:     cmd.ChatID,
        ReopenedBy: actorID,
    }
    _, err := e.chatUseCases.ReopenChat.Execute(ctx, reopenCmd)
    return err
}

func (e *CommandExecutor) executeDeleteChat(ctx context.Context, cmd DeleteChatCommand, actorID uuid.UUID) error {
    return e.chatUseCases.DeleteChat.Execute(ctx, cmd.ChatID, actorID)
}
```

### 6. Update ChatUseCases Struct

**File:** `internal/domain/tag/chat_usecases.go`

```go
type ChatUseCases struct {
    // Creation
    ConvertToTask *chatApp.ConvertToTaskUseCase
    ConvertToBug  *chatApp.ConvertToBugUseCase
    ConvertToEpic *chatApp.ConvertToEpicUseCase

    // Entity management
    ChangeStatus  *chatApp.ChangeStatusUseCase
    AssignUser    *chatApp.AssignUserUseCase
    SetPriority   *chatApp.SetPriorityUseCase
    SetDueDate    *chatApp.SetDueDateUseCase
    Rename        *chatApp.RenameChatUseCase
    SetSeverity   *chatApp.SetSeverityUseCase

    // Participant management (NEW)
    AddParticipant    *chatApp.AddParticipantUseCase
    RemoveParticipant *chatApp.RemoveParticipantUseCase

    // Lifecycle (NEW)
    CloseChat  *chatApp.CloseChatUseCase
    ReopenChat *chatApp.ReopenChatUseCase
    DeleteChat *chatApp.DeleteChatUseCase
}
```

### 7. Create New Use Cases

**File:** `internal/application/chat/close_chat.go`

```go
package chat

type CloseChatCommand struct {
    ChatID   uuid.UUID
    ClosedBy uuid.UUID
}

type CloseChatUseCase struct {
    repo ChatCommandRepository
}

func NewCloseChatUseCase(repo ChatCommandRepository) *CloseChatUseCase {
    return &CloseChatUseCase{repo: repo}
}

func (uc *CloseChatUseCase) Execute(ctx context.Context, cmd CloseChatCommand) (Result, error) {
    chat, err := uc.repo.Load(ctx, cmd.ChatID)
    if err != nil {
        return Result{}, err
    }

    if err := chat.Close(cmd.ClosedBy); err != nil {
        return Result{}, err
    }

    if err := uc.repo.Save(ctx, chat); err != nil {
        return Result{}, err
    }

    return Result{Value: chat}, nil
}
```

**File:** `internal/application/chat/reopen_chat.go`

```go
package chat

type ReopenChatCommand struct {
    ChatID     uuid.UUID
    ReopenedBy uuid.UUID
}

type ReopenChatUseCase struct {
    repo ChatCommandRepository
}

func NewReopenChatUseCase(repo ChatCommandRepository) *ReopenChatUseCase {
    return &ReopenChatUseCase{repo: repo}
}

func (uc *ReopenChatUseCase) Execute(ctx context.Context, cmd ReopenChatCommand) (Result, error) {
    chat, err := uc.repo.Load(ctx, cmd.ChatID)
    if err != nil {
        return Result{}, err
    }

    if err := chat.Reopen(cmd.ReopenedBy); err != nil {
        return Result{}, err
    }

    if err := uc.repo.Save(ctx, chat); err != nil {
        return Result{}, err
    }

    return Result{Value: chat}, nil
}
```

### 8. Add Domain Methods

**File:** `internal/domain/chat/chat.go`

```go
// Close marks the chat as closed
func (c *Chat) Close(closedBy uuid.UUID) error {
    if c.chatType == TypeDiscussion {
        return errors.New("cannot close a discussion")
    }
    if c.status == StatusClosed {
        return errors.New("chat is already closed")
    }

    previousStatus := c.status
    c.status = StatusClosed
    c.recordEvent(ChatClosed{
        ChatID:         c.id,
        ClosedBy:       closedBy,
        PreviousStatus: previousStatus,
        ClosedAt:       time.Now(),
    })
    return nil
}

// Reopen reopens a closed chat
func (c *Chat) Reopen(reopenedBy uuid.UUID) error {
    if c.status != StatusClosed {
        return errors.New("chat is not closed")
    }

    // Return to default status for entity type
    c.status = c.defaultStatusForType()
    c.recordEvent(ChatReopened{
        ChatID:     c.id,
        ReopenedBy: reopenedBy,
        NewStatus:  c.status,
        ReopenedAt: time.Now(),
    })
    return nil
}

func (c *Chat) defaultStatusForType() string {
    switch c.chatType {
    case TypeTask:
        return "To Do"
    case TypeBug:
        return "New"
    case TypeEpic:
        return "Planned"
    default:
        return ""
    }
}
```

### 9. Add Events

**File:** `internal/domain/chat/events.go`

```go
type ChatClosed struct {
    ChatID         uuid.UUID `json:"chat_id"`
    ClosedBy       uuid.UUID `json:"closed_by"`
    PreviousStatus string    `json:"previous_status"`
    ClosedAt       time.Time `json:"closed_at"`
}

func (e ChatClosed) EventType() string { return "ChatClosed" }
func (e ChatClosed) AggregateID() uuid.UUID { return e.ChatID }

type ChatReopened struct {
    ChatID     uuid.UUID `json:"chat_id"`
    ReopenedBy uuid.UUID `json:"reopened_by"`
    NewStatus  string    `json:"new_status"`
    ReopenedAt time.Time `json:"reopened_at"`
}

func (e ChatReopened) EventType() string { return "ChatReopened" }
func (e ChatReopened) AggregateID() uuid.UUID { return e.ChatID }
```

### 10. Update Container

**File:** `cmd/api/container.go`

Add initialization of new use cases in `createChatUseCasesForTags()`:

```go
func (c *Container) createChatUseCasesForTags() *tag.ChatUseCases {
    return &tag.ChatUseCases{
        // ... existing ...

        AddParticipant:    c.AddParticipantUC,
        RemoveParticipant: c.RemoveParticipantUC,
        CloseChat:         chatapp.NewCloseChatUseCase(c.ChatRepo),
        ReopenChat:        chatapp.NewReopenChatUseCase(c.ChatRepo),
        DeleteChat:        c.DeleteChatUC,
    }
}
```

---

## Testing

### Unit Tests

**File:** `internal/domain/tag/parser_test.go`
- Test parsing `#invite @username`
- Test parsing `#remove @username`
- Test parsing `#close` (no value)
- Test parsing `#reopen`
- Test parsing `#delete`

**File:** `internal/domain/tag/processor_test.go`
- Test processing invite with valid username
- Test processing invite with invalid username
- Test processing close on discussion (should fail)
- Test processing close on task (should succeed)

**File:** `internal/domain/tag/executor_test.go`
- Test execution of InviteUserCommand
- Test execution of RemoveUserCommand
- Test execution of CloseChatCommand
- Test execution of ReopenChatCommand

**File:** `internal/domain/chat/chat_test.go`
- Test Chat.Close() on task
- Test Chat.Close() on discussion (should fail)
- Test Chat.Reopen() on closed chat
- Test Chat.Reopen() on open chat (should fail)

### Integration Tests

**File:** `tests/integration/tag_commands_test.go`
- Full flow: send message with `#invite @user` → participant added
- Full flow: send message with `#close` → chat closed
- Full flow: send message with `#reopen` → chat reopened

---

## Acceptance Criteria

- [x] All 5 new tags parse correctly
- [x] Username validation works for invite/remove
- [x] Close fails on Discussion type
- [x] Close/Reopen produce correct events
- [x] Events saved to EventStore
- [x] ReadModel updated correctly
- [ ] Bot response generated for each operation (deferred to 007c)
- [x] All tests pass
