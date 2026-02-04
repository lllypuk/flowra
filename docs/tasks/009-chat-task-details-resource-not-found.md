# Task 009: Chat Task Details "Resource Not Found" Errors

**Status**: Pending
**Priority**: Medium
**Depends on**: None
**Created**: 2026-02-04
**Discovered**: Frontend testing with agent-browser

---

## Overview

When opening some chats in the frontend, multiple "Resource not found" error notifications appear. The task details sidebar shows "Loading task details..." and then errors are displayed via toast notifications.

---

## Symptoms

1. Opening a chat shows multiple "Resource not found" toast notifications
2. Task Details sidebar shows "Loading task details..." indefinitely
3. Errors repeat (multiple identical notifications)
4. Occurs on chats that should have task data but don't load correctly

**Screenshot evidence**: `/tmp/07-chat-view.png` from frontend testing session

---

## Technical Analysis

### HTMX Request Flow

```
Chat Page Load
  ↓
task-sidebar.html (lines 97-102)
  ↓
HTMX GET: /partials/chats/{chat_id}/task-details
  ↓
TaskDetailsByChatID() handler
  ├─ GetTaskByChatID() → Returns 404 if task not found
  └─ 404 Response
      ↓
app.js (lines 146-147) catches error
  ↓
Shows toast: "Resource not found."
```

### Template Location

**File**: `web/templates/chat/task-sidebar.html` (lines 97-102)

```html
{{else}}
<div class="task-details"
     hx-get="/partials/chats/{{.Data.Chat.ID}}/task-details"
     hx-trigger="load"
     hx-swap="outerHTML">
    <p>Loading task details...</p>
</div>
{{end}}
```

### Handler Location

**File**: `internal/handler/http/task_detail_template_handler.go` (lines 155-209)

```go
func (h *TaskDetailTemplateHandler) TaskDetailsByChatID(c echo.Context) error {
    // ...
    taskModel, err := h.taskService.GetTaskByChatID(c.Request().Context(), chatID)
    if err != nil {
        return c.String(http.StatusNotFound, "Task not found for this chat")
    }
    // ...
}
```

---

## Root Causes

### 1. Missing WorkspaceID in Task ReadModel (HIGH)

**File**: `internal/application/task/repository.go` (lines 66-78)

The `ReadModel` struct does NOT include `WorkspaceID`:

```go
type ReadModel struct {
    ID         uuid.UUID
    ChatID     uuid.UUID
    Title      string
    EntityType taskdomain.EntityType
    Status     taskdomain.Status
    Priority   taskdomain.Priority
    AssignedTo *uuid.UUID
    DueDate    *time.Time
    CreatedBy  uuid.UUID
    CreatedAt  time.Time
    Version    int
    // WorkspaceID is MISSING!
}
```

**Impact**: Cannot fetch workspace members for assignee dropdown.

### 2. Empty Participants List (MEDIUM)

**File**: `internal/handler/http/task_detail_template_handler.go` (lines 192-201)

```go
var participants []MemberViewData  // Always empty!

innerData := map[string]any{
    "Task":         h.convertToDetailView(taskModel),
    "Chat":         chatInfo,
    "Statuses":     getStatusOptions(),
    "Priorities":   getPriorityOptions(),
    "Participants": participants,  // EMPTY!
}
```

The TODO comment at lines 235-239 shows this is not implemented:
```go
// TODO: Get workspace ID from task's chat
var participants []MemberViewData
// if h.memberService != nil {
//     participants, _ = h.memberService.ListWorkspaceMembers(...)
// }
```

### 3. No Graceful Error Handling (MEDIUM)

When task is not found, handler returns 404 with plain text:
```go
return c.String(http.StatusNotFound, "Task not found for this chat")
```

Instead, it should return an empty/error state HTML that HTMX can display gracefully.

### 4. Event Sourcing Race Condition (LOW)

Task might exist as events but read model projection hasn't run yet, causing temporary 404s.

---

## Affected Files

| File | Line | Issue |
|------|------|-------|
| `internal/application/task/repository.go` | 66-78 | Missing WorkspaceID field |
| `internal/handler/http/task_detail_template_handler.go` | 155-209 | Handler returns 404 |
| `internal/handler/http/task_detail_template_handler.go` | 235-239 | TODO: participants not loaded |
| `web/templates/chat/task-sidebar.html` | 97-102 | HTMX trigger |
| `web/static/js/app.js` | 146-147 | Error shown as toast |

---

## Implementation Plan

### Phase 1: Add WorkspaceID to Task ReadModel

- [ ] Add `WorkspaceID uuid.UUID` to `taskapp.ReadModel` struct
- [ ] Update task read model repository to store/retrieve WorkspaceID
- [ ] Update event handler that builds read model to include WorkspaceID
- [ ] Run migration to update existing task_read_model documents

### Phase 2: Implement Participants Loading

- [ ] Inject `MemberService` into `TaskDetailTemplateHandler`
- [ ] Fetch workspace members using WorkspaceID from task
- [ ] Pass participants to template for assignee dropdown

### Phase 3: Improve Error Handling

- [ ] Return HTML partial for error state instead of plain text 404
- [ ] Create `notification/task-sidebar-error.html` template
- [ ] Show "No task data available" instead of error toast
- [ ] Handle Discussion chats that don't have task data (hide sidebar)

### Phase 4: Handle Non-Task Chats

- [ ] Check chat type before loading task sidebar
- [ ] If chat is Discussion type, don't request task details
- [ ] Only show task sidebar for Task/Bug/Epic chats

---

## Testing Plan

### Manual Testing

1. Create a new Task chat
2. Open the chat immediately after creation
3. Verify task details load without errors
4. Change task status/priority/assignee
5. Verify changes reflect in sidebar

### Edge Cases

1. Open Discussion chat - should NOT show task sidebar
2. Open chat during task creation - handle race condition
3. Rapid navigation between chats - no duplicate requests

---

## Success Criteria

1. [ ] No "Resource not found" errors when opening task chats
2. [ ] Task Details sidebar loads correctly with all fields
3. [ ] Assignee dropdown shows workspace members
4. [ ] Discussion chats don't trigger task details loading
5. [ ] Graceful error state for edge cases
