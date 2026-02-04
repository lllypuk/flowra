# Task 007: Tag System Frontend Integration

**Status**: Pending
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

### Phase 1: System Message Identification

- [ ] Add `is_system` or `sender_type` field to message model
- [ ] Bot user ID is `00000000-0000-0000-0000-000000000000`
- [ ] Update message query to include system flag
- [ ] Update API response to include system flag

**Files to modify:**

| File | Change |
|------|--------|
| `internal/application/message/repository.go` | Add IsSystem to ReadModel |
| `internal/infrastructure/repository/mongodb/message_repository.go` | Query bot user messages |
| `internal/handler/http/message_handler.go` | Include is_system in response |

### Phase 2: System Message Template

- [ ] Create `web/templates/components/system-message.html`
- [ ] Add conditional rendering in message list
- [ ] Style system messages distinctly

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

### Phase 3: Message Grouping Logic

- [ ] Add grouping logic in template or handler
- [ ] Group consecutive system messages by timestamp proximity
- [ ] Render grouped messages in single container

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

### Phase 4: Human-Readable Formatting

- [ ] Update `formatter.go` to include actor name
- [ ] Format dates in human-readable style
- [ ] Use proper username display (not @handle)

**Formatter changes:**

```go
func (f *Formatter) formatSuccess(app TagApplication, actorName string) string {
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

### Phase 5: Connect Action Buttons

- [ ] Add HTMX attributes to sidebar form controls
- [ ] Handle loading states during requests
- [ ] Show success/error feedback
- [ ] Refresh affected UI components

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

## Open Design Questions

These questions were deferred from original implementation and should be addressed:

### 1. Collapsible System Messages

**Question**: Should system messages be collapsible/expandable in UI?

**Options**:
- A) Always visible (current)
- B) Collapsible per message
- C) Collapsible as a group ("Show N system messages")
- D) Auto-collapse after N seconds

**Recommendation**: Option C - group collapse for cleaner history

### 2. Batch Changes Handling

**Question**: How to handle rapid consecutive changes?

**Example**: User quickly changes status, then priority, then assignee within seconds.

**Options**:
- A) Show each change individually
- B) Batch into single message ("John changed status, priority, and assignee")
- C) Debounce changes (wait 2s, then show combined)

**Recommendation**: Option B for user-initiated changes, Option A for tag-based changes

### 3. Suppress System Messages

**Question**: Should we allow suppressing system messages for automated/bulk operations?

**Use case**: Bot updating 50 tasks at once shouldn't flood history.

**Options**:
- A) Never suppress
- B) Add `silent=true` parameter to action endpoints
- C) Auto-suppress if same actor makes >5 changes in <10s
- D) Per-workspace setting

**Recommendation**: Option B with audit log retention

### 4. External Integration Messages

**Question**: How to handle changes made via API by external integrations?

**Options**:
- A) Show as regular system messages
- B) Show with integration name ("Jira sync changed status...")
- C) Don't show, only log
- D) Configurable per integration

**Recommendation**: Option B for transparency

---

## Affected Files

| File | Change |
|------|--------|
| `internal/application/message/repository.go` | Add IsSystem field |
| `internal/domain/tag/formatter.go` | Add actor name parameter |
| `internal/domain/tag/handler.go` | Pass actor name to formatter |
| `internal/handler/http/message_handler.go` | Include system flag in response |
| `web/templates/chat/message-list.html` | Add conditional system rendering |
| `web/templates/components/system-message.html` | New - system message template |
| `web/templates/chat/task-sidebar.html` | Add HTMX action attributes |
| `web/static/css/main.css` | Add system message styles |

---

## Testing Plan

### Unit Tests

- [ ] Test message grouping algorithm
- [ ] Test human-readable formatter output
- [ ] Test system message identification

### Integration Tests

- [ ] Test action endpoint triggers system message
- [ ] Test HTMX response headers
- [ ] Test WebSocket broadcast after action

### Manual Testing

1. Send message with multiple tags
2. Verify system messages render compactly
3. Verify consecutive messages are grouped
4. Change status via sidebar dropdown
5. Verify system message appears in chat
6. Verify chat updates via WebSocket

---

## Success Criteria

1. [ ] System messages render with distinct compact style
2. [ ] Consecutive system messages are visually grouped
3. [ ] Messages show actor names (e.g., "John changed status...")
4. [ ] Dates formatted in human-readable style
5. [ ] All sidebar controls trigger corresponding action endpoints
6. [ ] Changes via sidebar appear as system messages in chat
7. [ ] Design questions resolved and documented
