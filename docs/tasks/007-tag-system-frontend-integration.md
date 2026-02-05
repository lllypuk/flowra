# Task 007: Tag System Frontend Integration

**Status**: Complete
**Priority**: Medium
**Depends on**: None
**Created**: 2026-02-04
**Source**: Deferred from Tag-Based Entity Management implementation

---

## Overview

The tag system backend is fully implemented with parsing, validation, command execution, and bot response generation. This task focuses on frontend integration to properly render system messages, group consecutive changes, and connect action buttons to the backend endpoints.

---

## Current Implementation

### Tag System Architecture

```
User Message Flow:
  User types: "#task 'New feature' #priority High"
        ↓
  Parser.Parse() → extracts tags
        ↓
  Processor.ProcessTags() → validates, creates commands
        ↓
  Executor.Execute() → applies changes to domain
        ↓
  Formatter.GenerateBotResponse() → creates feedback
        ↓
  Handler.sendBotResponse() → saves system message
        ↓
  WebSocket broadcast → UI update
```

### Key Backend Files

| File | Lines | Description |
|------|-------|-------------|
| `internal/domain/tag/parser.go` | 1-291 | Tag parsing with 13 registered tags |
| `internal/domain/tag/processor.go` | 1-348 | Validation and command generation |
| `internal/domain/tag/executor.go` | 1-343 | Command execution with retry logic |
| `internal/domain/tag/formatter.go` | 1-96 | Bot response generation |
| `internal/domain/tag/handler.go` | 1-160 | Orchestration of tag processing |
| `internal/handler/http/chat_action_handler.go` | 1-302 | REST API action endpoints |

### Supported Tags

| Tag | Value Type | Example |
|-----|------------|---------|
| `#task` | String (title) | `#task "New feature"` |
| `#bug` | String (title) | `#bug "Login broken"` |
| `#epic` | String (title) | `#epic "Q1 Release"` |
| `#status` | Enum | `#status "In Progress"` |
| `#priority` | Enum | `#priority High` |
| `#assignee` | Username | `#assignee @john` |
| `#due` | Date | `#due 2026-03-15` |
| `#title` | String | `#title "Updated title"` |
| `#severity` | Enum | `#severity Critical` |
| `#invite` | Username | `#invite @jane` |
| `#remove` | Username | `#remove @bob` |
| `#close` | None | `#close` |
| `#reopen` | None | `#reopen` |

### Bot Response Format

**File**: `internal/domain/tag/formatter.go:33-86`

```
Success messages:
  "✅ Task created: {title}"
  "✅ Status changed to {status}"
  "✅ Assigned to: @{username}"
  "✅ Priority changed to {priority}"
  "✅ Due date set to {date}"
  "✅ Chat closed"
  etc.

Error messages:
  "❌ {error message}"
  "⚠️ {warning message}"
```

### Action Endpoints

**File**: `internal/handler/http/chat_action_handler.go`

| Endpoint | Method | Input | Response |
|----------|--------|-------|----------|
| `/api/v1/chats/:id/actions/status` | POST | `{"status": "..."}` | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/priority` | POST | `{"priority": "..."}` | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/assignee` | POST | `{"assignee_id": "..."}` | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/due-date` | POST | `{"due_date": "..."}` | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/close` | POST | - | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/reopen` | POST | - | HX-Trigger: chatUpdated |
| `/api/v1/chats/:id/actions/rename` | POST | `{"title": "..."}` | HX-Trigger: chatUpdated |

---

## Requirements

### 1. System Message Rendering

System messages (bot responses) should be visually distinct from user messages:

**Current state**: All messages render identically with user avatar and name.

**Desired state**:
- System messages have no avatar or generic system icon
- Compact inline display (no bubble/card wrapper)
- Muted text color
- Smaller font size
- Centered or left-aligned without indentation

**Visual example:**

```
┌─────────────────────────────────────────────┐
│ [Avatar] John                               │
│ ┌─────────────────────────────────┐         │
│ │ Let's track this as a task      │         │
│ │ #task "Update documentation"    │         │
│ └─────────────────────────────────┘         │
│                                             │
│   ✅ Task created: Update documentation     │  ← system message (compact)
│                                             │
│ [Avatar] Jane                               │
│ ┌─────────────────────────────────┐         │
│ │ I'll handle this #assignee @jane │        │
│ └─────────────────────────────────┘         │
│                                             │
│   ✅ Assigned to: @jane                     │  ← system message (compact)
│                                             │
└─────────────────────────────────────────────┘
```

### 2. Group Consecutive System Messages

When multiple tags are processed in one message, group the resulting system messages:

**Current state**: Each success/error is a separate message bubble.

**Desired state**: Consecutive system messages from same action grouped into single block.

**Visual example:**

```
Bad (current):
│   ✅ Task created: Update docs             │
│   ✅ Priority changed to High              │
│   ✅ Assigned to: @jane                    │

Good (desired):
│   ✅ Task created: Update docs             │
│   ✅ Priority changed to High              │
│   ✅ Assigned to: @jane                    │
│   ─────────────────────────────            │  ← single grouped block
```

### 3. Human-Readable Status Messages

Transform raw tag responses to user-friendly language:

| Raw Message | Human-Readable |
|-------------|----------------|
| `✅ Status changed to "In Progress"` | `✅ John changed status to In Progress` |
| `✅ Assigned to: @jane` | `✅ John assigned this to Jane` |
| `✅ Priority changed to High` | `✅ John set priority to High` |
| `✅ Due date set to 2026-03-15` | `✅ John set due date to March 15, 2026` |

### 4. Action Buttons in UI

Connect sidebar form controls to action endpoints:

**Status dropdown** → POST `/api/v1/chats/:id/actions/status`
**Priority dropdown** → POST `/api/v1/chats/:id/actions/priority`
**Assignee dropdown** → POST `/api/v1/chats/:id/actions/assignee`
**Due date picker** → POST `/api/v1/chats/:id/actions/due-date`
**Close button** → POST `/api/v1/chats/:id/actions/close`

---

## Implementation Plan

### Phase 1: System Message Identification ✅

- [x] Add `is_system` or `sender_type` field to message model (already exists: `Type` with user/system/bot)
- [x] Bot user ID is `00000000-0000-0000-0000-000000000000`
- [x] Update message query to include system flag (already in repository)
- [x] Update API response to include system flag

**Completed changes:**

| File | Change |
|------|--------|
| `internal/domain/message/message.go` | Already has `Type`, `IsSystemMessage()`, `IsBotMessage()` |
| `internal/infrastructure/repository/mongodb/message_repository.go` | Already persists `type` and `actor_id` |
| `internal/handler/http/message_handler.go` | Added `Type`, `IsSystem`, `ActorID` to `MessageResponse` |
| `internal/domain/tag/handler.go` | Updated to use `NewMessageWithType` with `TypeBot` |

### Phase 2: System Message Template ✅

- [x] Create `web/templates/components/system-message.html` (not needed - updated existing `message.html`)
- [x] Add conditional rendering in message list (already exists with `IsSystemMessage`/`IsBotMessage`)
- [x] Style system messages distinctly

**Completed changes:**

| File | Change |
|------|--------|
| `web/templates/components/message.html` | Updated to hide avatar and header for bot messages, compact styling |

**Template structure:**

```html
{{define "system-message"}}
<div class="message-system">
    <span class="system-icon">ℹ️</span>
    <span class="system-text">{{.Content}}</span>
    <time class="system-time">{{.CreatedAt | formatTime}}</time>
</div>
{{end}}
```

**CSS additions:**

```css
.message-system {
    padding: 0.25rem 1rem;
    color: var(--muted-color);
    font-size: 0.875rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.message-system .system-icon {
    font-size: 0.75rem;
}

.message-system .system-text {
    flex: 1;
}

.message-system .system-time {
    font-size: 0.75rem;
    opacity: 0.7;
}
```

### Phase 3: Message Grouping Logic ✅

- [x] Add grouping logic in template or handler
- [x] Group consecutive system messages by timestamp proximity (5 seconds)
- [x] Render grouped messages in single container

**Completed changes:**

| File | Change |
|------|--------|
| `internal/handler/http/chat_template_handler.go` | Added `IsGroupStart`/`IsGroupEnd` to `MessageViewData`, `applyMessageGrouping()` function |
| `web/templates/components/message.html` | Added `group-start`/`group-end` CSS classes and styling |

**Grouping algorithm:**

```go
func groupMessages(messages []Message) []MessageGroup {
    var groups []MessageGroup
    var currentGroup *MessageGroup

    for _, msg := range messages {
        if msg.IsSystem {
            if currentGroup != nil && currentGroup.IsSystem {
                // Add to existing system group if within 5 seconds
                if msg.CreatedAt.Sub(currentGroup.LastTime) < 5*time.Second {
                    currentGroup.Messages = append(currentGroup.Messages, msg)
                    currentGroup.LastTime = msg.CreatedAt
                    continue
                }
            }
            // Start new system group
            currentGroup = &MessageGroup{IsSystem: true, Messages: []Message{msg}}
            groups = append(groups, *currentGroup)
        } else {
            // User message - no grouping
            groups = append(groups, MessageGroup{Messages: []Message{msg}})
            currentGroup = nil
        }
    }
    return groups
}
```

### Phase 3.5: Batch UI Changes (Deferred)

**Status**: Deferred to future enhancement. Phase 3 grouping already provides visual grouping of consecutive messages.

- [ ] Implement debounce/batching for rapid UI changes
- [ ] Collect changes within 2-second window
- [ ] Generate combined message: "John changed status to X, priority to Y, and assigned to Z"

**Batching approach:**

```go
type PendingChanges struct {
    ActorID   uuid.UUID
    ChatID    uuid.UUID
    Changes   []Change
    FirstTime time.Time
}

type Change struct {
    Type  string // "status", "priority", "assignee", "due_date"
    Value string
}

// In action handler: collect changes, flush after 2s idle or on different actor/chat
func (b *ChangeBatcher) AddChange(actorID, chatID uuid.UUID, change Change) {
    // If pending changes exist for same actor+chat and within window, append
    // Otherwise flush existing and start new batch
}

func (b *ChangeBatcher) formatBatchMessage(changes []Change, actorName string) string {
    // "John changed status to In Progress, priority to High, and assigned to Jane"
}
```
```

### Phase 4: Human-Readable Formatting ✅

- [x] Update `formatter.go` to include actor name
- [x] Format dates in human-readable style
- [x] Use proper username display (not @handle)
- [x] Support integration names for external API changes

**Completed changes:**

| File | Change |
|------|--------|
| `internal/domain/tag/formatter.go` | Added `ActorInfo`, `GenerateBotResponseWithActor()`, `formatHumanReadableDate()` |
| `internal/domain/tag/repositories.go` | Added `FindByID` to `UserRepository` |
| `internal/domain/tag/handler.go` | Added `userRepo`, `getActorInfo()`, uses `GenerateBotResponseWithActor()` |
| `internal/service/action_service.go` | All actions now generate human-readable messages with actor names |

**Formatter changes:**

```go
type ActorInfo struct {
    Name          string
    IsIntegration bool
    Integration   string // "Jira sync", "GitHub webhook", etc.
}

func (f *Formatter) formatSuccess(app TagApplication, actor ActorInfo) string {
    actorName := actor.Name
    if actor.IsIntegration {
        actorName = actor.Integration // "Jira sync", "GitHub webhook"
    }

    switch cmd := app.Command.(type) {
    case ChangeStatusCommand:
        return fmt.Sprintf("✅ %s changed status to %s", actorName, cmd.Status)
    case AssignUserCommand:
        if cmd.Username == "" {
            return fmt.Sprintf("✅ %s removed the assignee", actorName)
        }
        return fmt.Sprintf("✅ %s assigned this to %s", actorName, cmd.Username)
    // ... etc
    }
}
```

**Integration identification:**

```go
// In action handler or middleware
func getActorInfo(ctx context.Context) ActorInfo {
    // Check for X-Integration-Name header or API key metadata
    if integrationName := ctx.Value("integration_name"); integrationName != nil {
        return ActorInfo{IsIntegration: true, Integration: integrationName.(string)}
    }
    // Regular user
    user := auth.UserFromContext(ctx)
    return ActorInfo{Name: user.DisplayName}
}
```

### Phase 5: Connect Action Buttons ✅

- [x] Add HTMX attributes to sidebar form controls
- [x] Handle loading states during requests
- [x] Show success/error feedback (via chatUpdated HX-Trigger)
- [x] Refresh affected UI components (via WebSocket)

**Completed changes:**

| File | Change |
|------|--------|
| `web/templates/chat/task-sidebar.html` | Updated all form controls to use `/api/v1/chats/:id/actions/*` endpoints with HTMX |

**Example for status dropdown:**

```html
<select name="status"
        hx-post="/api/v1/chats/{{.ChatID}}/actions/status"
        hx-trigger="change"
        hx-swap="none"
        hx-vals='js:{"status": this.value}'
        hx-indicator="#status-loading">
    {{range .Statuses}}
    <option value="{{.Value}}" {{if eq .Value $.CurrentStatus}}selected{{end}}>
        {{.Label}}
    </option>
    {{end}}
</select>
<span id="status-loading" class="htmx-indicator">Saving...</span>
```

---

## Design Decisions

Resolved on 2026-02-05:

### 1. Collapsible System Messages

**Decision**: Always visible (no collapsing)

System messages remain visible at all times. No collapse/expand functionality needed.

### 2. Batch Changes Handling

**Decision**: Batch into single message

When user makes rapid consecutive changes via UI (status, priority, assignee within seconds), combine them into a single system message:
- Example: "John changed status to In Progress, priority to High, and assigned to Jane"

Note: Tag-based changes from a single message are already grouped by the formatter.

### 3. Suppress System Messages

**Decision**: Never suppress

All changes always create visible system messages in chat. No silent mode for bulk operations - full transparency is preferred.

### 4. External Integration Messages

**Decision**: Show with integration name

Changes from external integrations display the integration source:
- Example: "Jira sync changed status to Done"
- Example: "GitHub webhook closed this task"

This provides transparency about where changes originate.

---

## Affected Files (Actual)

| File | Change |
|------|--------|
| `internal/handler/http/message_handler.go` | Added `Type`, `IsSystem`, `ActorID` to `MessageResponse` |
| `internal/handler/http/chat_template_handler.go` | Added `IsGroupStart`/`IsGroupEnd`, `applyMessageGrouping()` |
| `internal/domain/tag/formatter.go` | Added `ActorInfo`, `GenerateBotResponseWithActor()`, `formatHumanReadableDate()` |
| `internal/domain/tag/handler.go` | Added `userRepo`, `getActorInfo()`, use `NewMessageWithType` with `TypeBot` |
| `internal/domain/tag/repositories.go` | Added `FindByID` to `UserRepository` |
| `internal/service/action_service.go` | All actions generate human-readable messages with actor names |
| `web/templates/components/message.html` | Updated compact bot message styles, grouping CSS |
| `web/templates/chat/task-sidebar.html` | Updated to use `/api/v1/chats/:id/actions/*` endpoints |

---

## Testing Plan

### Unit Tests

- [ ] Test message grouping algorithm
- [ ] Test human-readable formatter output
- [ ] Test system message identification
- [ ] Test change batcher collects and flushes correctly
- [ ] Test batch message formatting with multiple changes
- [ ] Test ActorInfo with regular user vs integration

### Integration Tests

- [ ] Test action endpoint triggers system message
- [ ] Test HTMX response headers
- [ ] Test WebSocket broadcast after action
- [ ] Test rapid changes batched into single message
- [ ] Test integration header results in correct actor name

### Manual Testing

1. Send message with multiple tags
2. Verify system messages render compactly
3. Verify consecutive messages are grouped
4. Change status via sidebar dropdown
5. Verify system message appears in chat
6. Verify chat updates via WebSocket
7. Quickly change status, priority, assignee - verify single batched message
8. Test API call with `X-Integration-Name: Jira` header - verify "Jira changed..." message

---

## Success Criteria

1. [x] System messages render with distinct compact style
2. [x] Consecutive system messages are visually grouped
3. [x] Messages show actor names (e.g., "John changed status...")
4. [x] Dates formatted in human-readable style
5. [x] All sidebar controls trigger corresponding action endpoints
6. [x] Changes via sidebar appear as system messages in chat
7. [x] Design questions resolved and documented
8. [ ] Rapid UI changes batched into single message (deferred to future enhancement)
9. [x] External integration changes show integration name (ActorInfo.Integration support added)
