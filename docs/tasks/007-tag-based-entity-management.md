# Task 007: Tag-Based Entity Management (All Changes Through Chat)

**Status**: ✅ Complete (007a ✅, 007b ✅, 007c ✅)  
**Priority**: High  
**Depends on**: Tasks 001-006 (Event Architecture)  
**Completed**: 2026-01-16

---

## Overview

All entity changes (tasks, chats, participants) must go through the tag system via chat messages. This ensures:
1. Complete audit trail in chat history
2. Full visibility of who changed what and when
3. Consistent event flow through the established event architecture
4. Natural collaboration - changes are discussed in context

---

## Current State Analysis

### Operations That GO THROUGH Tags (recorded in chat history):

| Tag | Operation | Status |
|-----|-----------|--------|
| `#task <title>` | Convert Discussion → Task | Implemented |
| `#bug <title>` | Convert Discussion → Bug | Implemented |
| `#epic <title>` | Convert Discussion → Epic | Implemented |
| `#status <status>` | Change status | Implemented |
| `#assignee @user` | Assign user | Implemented |
| `#priority <Low\|Medium\|High>` | Set priority | Implemented |
| `#due <date>` | Set due date | Implemented |
| `#title <new-title>` | Rename | Implemented |
| `#severity <level>` | Set bug severity | Implemented |

### Operations That BYPASS Tags (NOT recorded in chat history):

**TaskHandler (`/api/v1/tasks`):**
- `POST /tasks` - Create task directly
- `PUT /tasks/:id/status` - Change status
- `PUT /tasks/:id/assign` - Assign user
- `PUT /tasks/:id/priority` - Change priority
- `PUT /tasks/:id/due-date` - Set due date
- `DELETE /tasks/:id` - Delete task

**ChatHandler (`/api/v1/chats`):**
- `POST /chats` - Create chat
- `PUT /chats/:id` - Rename chat
- `DELETE /chats/:id` - Delete chat
- `POST /chats/:id/participants` - Add participant
- `DELETE /chats/:id/participants/:user_id` - Remove participant

---

## Target Architecture

### Principle: "All Changes Through Messages"

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER ACTIONS                                 │
├─────────────────────────────────────────────────────────────────────┤
│  UI Button Click    │   API Call    │   Direct Message with Tag     │
│  (Status dropdown)  │   (Mobile)    │   "#status Done"              │
└──────────┬──────────┴───────┬───────┴────────────┬──────────────────┘
           │                  │                    │
           ▼                  ▼                    │
┌─────────────────────────────────────────────────┐│
│           HTTP Handler (Modified)                ││
│  Creates system message with appropriate tag     ││
│  POST /chats/:id/messages                        ││
│  { content: "#status Done", type: "system" }     │◄──────────────────┘
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│           SendMessageUseCase                     │
│  1. Validate & save message                      │
│  2. Process tags (async)                         │
│  3. Execute commands via CommandExecutor         │
│  4. Generate bot response (success/error)        │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│           Chat Domain (existing)                 │
│  Events: StatusChanged, UserAssigned, etc.       │
│  Saved to: EventStore → Outbox → EventBus       │
└─────────────────────────────────────────────────┘
```

### Message Types

Add new message type for system-generated messages:

```go
const (
    MessageTypeUser    MessageType = "user"     // User typed message
    MessageTypeSystem  MessageType = "system"   // System-generated (from UI actions)
    MessageTypeBot     MessageType = "bot"      // Bot responses
)
```

System messages:
- Have special visual styling (smaller, gray, inline)
- Show who performed the action
- Contain the tag that was executed
- Are grouped with consecutive changes

---

## Implementation Phases

### Phase 1: Add Missing Tags

New tags to implement:

| Tag | Operation | Notes |
|-----|-----------|-------|
| `#invite @user` | Add participant | Resolves username to userID |
| `#remove @user` | Remove participant | Cannot remove creator |
| `#close` | Close/archive chat | Sets status to "Closed" |
| `#reopen` | Reopen closed chat | Returns to previous status |
| `#delete` | Delete chat/task | Soft delete, requires confirmation |

### Phase 2: Modify HTTP Handlers

Convert direct-action handlers to message-based:

1. **Status Change**: `PUT /tasks/:id/status`
   - Instead of calling `ChangeStatusUseCase` directly
   - Create message: `#status {new_status}`
   - Return message ID + pending status

2. **Assignment**: `PUT /tasks/:id/assign`
   - Create message: `#assignee @{username}` or `#assignee @none`

3. **Priority**: `PUT /tasks/:id/priority`
   - Create message: `#priority {priority}`

4. **Due Date**: `PUT /tasks/:id/due-date`
   - Create message: `#due {date}` or `#due` (to clear)

5. **Participants**: `POST/DELETE /chats/:id/participants`
   - Create message: `#invite @{username}` or `#remove @{username}`

6. **Rename**: `PUT /chats/:id`
   - Create message: `#title {new_name}`

### Phase 3: System Message Handling

1. Add `MessageType` field to Message domain
2. Update SendMessageUseCase to handle system messages
3. System messages skip certain validations (e.g., empty text after tag)
4. Add `ActorID` to track who initiated the action (different from message author for system messages)

### Phase 4: UI Integration

1. System messages rendered differently (compact, inline)
2. Group consecutive system messages from same actor
3. Show "X changed status to Done" instead of raw tag
4. Action buttons in UI POST to new endpoints that create messages

### Phase 5: Remove/Deprecate Direct Handlers

1. Mark direct action endpoints as deprecated
2. Add deprecation warnings in API responses
3. Eventually remove (after migration period)

---

## Detailed Task Breakdown

### Task 7.1: Add New Tag Commands

**Files to modify:**
- `internal/domain/tag/commands.go` - Add new command structs
- `internal/domain/tag/parser.go` - Register new tags
- `internal/domain/tag/processor.go` - Handle new tag processing
- `internal/domain/tag/executor.go` - Execute new commands
- `internal/domain/tag/validators.go` - Add validation rules

**New commands:**
```go
type InviteUserCommand struct {
    ChatID   uuid.UUID
    Username string
    UserID   *uuid.UUID  // resolved
}

type RemoveUserCommand struct {
    ChatID   uuid.UUID
    Username string
    UserID   *uuid.UUID  // resolved
}

type CloseChatCommand struct {
    ChatID uuid.UUID
}

type ReopenChatCommand struct {
    ChatID uuid.UUID
}

type DeleteChatCommand struct {
    ChatID uuid.UUID
}
```

### Task 7.2: Add Message Type Support

**Files to modify:**
- `internal/domain/message/message.go` - Add Type field
- `internal/domain/message/events.go` - Include type in events
- `internal/infrastructure/repository/mongodb/message_repository.go` - Handle type
- `internal/application/message/send_message.go` - Accept type parameter

**New types:**
```go
type Type string

const (
    TypeUser   Type = "user"
    TypeSystem Type = "system"
    TypeBot    Type = "bot"
)
```

### Task 7.3: Create Message-Based Action Service

**New file:** `internal/service/action_service.go`

Service that converts UI actions to messages:

```go
type ActionService struct {
    sendMessageUC *message.SendMessageUseCase
    userRepo      UserRepository
}

func (s *ActionService) ChangeStatus(ctx context.Context, chatID uuid.UUID, status string, actorID uuid.UUID) error {
    content := fmt.Sprintf("#status %s", status)
    cmd := message.SendMessageCommand{
        ChatID:   chatID,
        AuthorID: actorID,
        Content:  content,
        Type:     message.TypeSystem,
    }
    _, err := s.sendMessageUC.Execute(ctx, cmd)
    return err
}

func (s *ActionService) InviteUser(ctx context.Context, chatID uuid.UUID, username string, actorID uuid.UUID) error {
    content := fmt.Sprintf("#invite @%s", username)
    // ...
}
```

### Task 7.4: Modify Existing HTTP Handlers

**Files to modify:**
- `internal/handler/http/task_handler.go`
- `internal/handler/http/chat_handler.go`

Example modification:

```go
// Before:
func (h *TaskHandler) ChangeStatus(c echo.Context) error {
    cmd := taskapp.ChangeStatusCommand{...}
    result, err := h.taskService.ChangeStatus(ctx, cmd)
    // ...
}

// After:
func (h *TaskHandler) ChangeStatus(c echo.Context) error {
    err := h.actionService.ChangeStatus(ctx, chatID, status, userID)
    if err != nil {
        return handleError(c, err)
    }
    return httpserver.RespondOK(c, map[string]any{
        "message": "Status change initiated",
        "status": "pending",
    })
}
```

### Task 7.5: Add Chat Use Cases for New Operations

**Files to create/modify:**
- `internal/application/chat/close_chat.go` - Close chat use case
- `internal/application/chat/reopen_chat.go` - Reopen chat use case
- `internal/domain/chat/chat.go` - Add Close/Reopen methods

### Task 7.6: Update Tag Executor Integration

**Files to modify:**
- `internal/domain/tag/executor.go` - Add execution for new commands
- `internal/domain/tag/chat_usecases.go` - Add new use case references

```go
type ChatUseCases struct {
    // Existing
    ConvertToTask *chatApp.ConvertToTaskUseCase
    ChangeStatus  *chatApp.ChangeStatusUseCase
    // ...

    // New
    AddParticipant    *chatApp.AddParticipantUseCase
    RemoveParticipant *chatApp.RemoveParticipantUseCase
    CloseChat         *chatApp.CloseChatUseCase
    ReopenChat        *chatApp.ReopenChatUseCase
    DeleteChat        *chatApp.DeleteChatUseCase
}
```

### Task 7.7: Update Frontend Templates

**Files to modify:**
- `web/templates/components/message.html` - Render system messages differently
- `web/templates/chat/chat_view.html` - Update action buttons
- `web/static/js/chat.js` - POST to message endpoint instead of direct action

### Task 7.8: Add Bot Response Completion

**Files to modify:**
- `internal/application/message/send_message.go` - Enable bot response sending
- `internal/domain/tag/formatter.go` - Improve response formatting

Currently bot response is TODO in `processTagsAsync()`. Need to:
1. Implement bot message creation
2. Send via WebSocket to chat participants
3. Save bot message to database

---

## Migration Strategy

1. **Phase A**: Add new functionality (7.1-7.6) alongside existing
2. **Phase B**: Update frontend to use message-based actions (7.7)
3. **Phase C**: Add deprecation warnings to direct endpoints
4. **Phase D**: Remove deprecated endpoints after 2 releases

---

## API Changes

### New Endpoints

None required - all actions go through existing message endpoint:
```
POST /api/v1/chats/:chat_id/messages
```

### Modified Response Format

For system messages triggered by UI actions:

```json
{
    "id": "message-uuid",
    "chat_id": "chat-uuid",
    "type": "system",
    "content": "#status Done",
    "author_id": "user-uuid",
    "created_at": "2025-01-15T10:30:00Z",
    "tag_results": [
        {
            "tag": "status",
            "value": "Done",
            "success": true
        }
    ]
}
```

### Deprecated Endpoints

Mark these as deprecated (still functional during migration):
- `PUT /api/v1/tasks/:id/status`
- `PUT /api/v1/tasks/:id/assign`
- `PUT /api/v1/tasks/:id/priority`
- `PUT /api/v1/tasks/:id/due-date`
- `PUT /api/v1/chats/:id` (for rename)
- `POST /api/v1/chats/:id/participants`
- `DELETE /api/v1/chats/:id/participants/:user_id`

---

## Testing Requirements

### Unit Tests
- New tag parsing tests
- New command execution tests
- ActionService tests
- System message handling tests

### Integration Tests
- Full flow: UI action → Message → Tag processing → Domain event
- Verify events are published correctly
- Verify chat history contains all changes

### E2E Tests
- UI workflow: click button → see message in chat → see entity updated
- Concurrent changes from multiple users
- Error handling for invalid tag values

---

## Rollback Plan

If issues arise:
1. Re-enable direct action handlers (remove deprecation)
2. ActionService can be bypassed
3. Frontend can switch back to direct API calls
4. No data migration needed - both paths produce same events

---

## Success Criteria

1. All entity changes appear in chat history
2. No direct modification without chat message
3. System messages are visually distinct but clear
4. Existing frontend functionality preserved
5. API backward compatibility during migration
6. All tests passing
7. No performance degradation

---

## Open Questions

1. Should system messages be collapsible/expandable in UI?
2. How to handle rapid consecutive changes (batch into single message)?
3. Should we allow suppressing system messages for automated/bulk operations?
4. How to handle changes made via API by external integrations?

---

## Files Summary

### New Files
- `internal/service/action_service.go`
- `internal/application/chat/close_chat.go`
- `internal/application/chat/reopen_chat.go`
- Tests for all new functionality

### Modified Files
- `internal/domain/tag/commands.go`
- `internal/domain/tag/parser.go`
- `internal/domain/tag/processor.go`
- `internal/domain/tag/executor.go`
- `internal/domain/tag/validators.go`
- `internal/domain/tag/chat_usecases.go`
- `internal/domain/message/message.go`
- `internal/domain/message/events.go`
- `internal/application/message/send_message.go`
- `internal/handler/http/task_handler.go`
- `internal/handler/http/chat_handler.go`
- `internal/infrastructure/repository/mongodb/message_repository.go`
- `cmd/api/container.go`
- Frontend templates and JS
