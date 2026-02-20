# Task: Route Task Field Changes Through Chat

## Status: Complete

## Goal

All changes to task/bug/epic fields (status, priority, assignee, due date) made from the right
panel sidebar must be routed through the chat message system. This creates a system message in the
associated chat, which is then processed by the tag system to apply the change. Activity is tracked
in the chat, not in a separate sidebar timeline.

The activity section in the task sidebar must be removed entirely.

## Background: Current Architecture

### Problem

The task sidebar currently sends HTMX `PUT` requests directly to the `TaskHandler`:

- `PUT /api/v1/workspaces/:wid/tasks/:task_id/status` → `TaskHandler.ChangeStatus`
- `PUT /api/v1/workspaces/:wid/tasks/:task_id/priority` → `TaskHandler.ChangePriority`
- `PUT /api/v1/workspaces/:wid/tasks/:task_id/assignee` → `TaskHandler.Assign`
- `PUT /api/v1/workspaces/:wid/tasks/:task_id/due-date` → `TaskHandler.SetDueDate`

These handlers update the task aggregate directly, bypassing the chat system entirely.
No system messages are created, no activity is tracked in the chat.

### Existing Infrastructure

The chat-based action system already exists and is fully functional:

- `ChatActionHandler` (`internal/handler/http/chat_action_handler.go`)
  - `POST /api/v1/workspaces/:wid/chats/:id/actions/status`
  - `POST /api/v1/workspaces/:wid/chats/:id/actions/priority`
  - `POST /api/v1/workspaces/:wid/chats/:id/actions/assignee`
  - `POST /api/v1/workspaces/:wid/chats/:id/actions/due-date`

- `ActionService` (`internal/service/action_service.go`)
  - Creates system messages with tag commands (e.g., `#status In Progress`)
  - Batches human-readable messages (e.g., "Alice changed status to In Progress")
  - Routes through `SendMessageUseCase` → tag processing → chat aggregate update

- Tag system processes `#status`, `#priority`, `#assignee`, `#due` commands and updates
  the chat aggregate (which is the source of truth for task metadata).

### Target Flow

```
User changes status in sidebar
    ↓
HTMX POST /api/v1/workspaces/:wid/tasks/:task_id/actions/status
    ↓
TaskActionHandler.ChangeStatus
    ├─ Load task read model → get ChatID
    └─ Delegate to ActionService.ChangeStatus(chatID, ...)
        ├─ Creates system message: "#status In Progress"
        ├─ SendMessageUseCase saves message (TypeSystem)
        ├─ Tag processor runs async → updates chat aggregate
        ├─ Task read model synced from chat aggregate
        └─ Bot response in chat: "✅ Alice changed status to In Progress"
```

## Detailed Step Files

Each step has a dedicated file with full implementation:

| Step | File | Description |
|------|------|-------------|
| 1 | [step-1-task-action-handler.md](step-1-task-action-handler.md) | New handler: full Go source code |
| 2 | [step-2-routes.md](step-2-routes.md) | Route registration and Container field |
| 3 | [step-3-container-wiring.md](step-3-container-wiring.md) | Wiring in `setupHTTPHandlers` |
| 4 | [step-4-sidebar-template.md](step-4-sidebar-template.md) | Sidebar changes, remove Activity section |
| 5 | [step-5-components.md](step-5-components.md) | `HxPost` support in `user_select` and `date_picker` |
| 6 | [step-6-tests.md](step-6-tests.md) | Full test file with mocks and cases |

---

## Implementation Plan (Summary)

### Step 1: New `TaskActionHandler`

**File**: `internal/handler/http/task_action_handler.go`

Create a new handler that:
1. Accepts task-scoped action requests
2. Resolves `taskID → chatID` using `TaskDetailService.GetTask`
3. Delegates to `ActionService` (the same service used by `ChatActionHandler`)

Interface to declare on the consumer side:

```go
type TaskActionTaskService interface {
    GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)
}
```

Endpoints:

| Method | Path | Handler |
|--------|------|---------|
| `POST` | `/api/v1/workspaces/:wid/tasks/:task_id/actions/status` | `ChangeStatus` |
| `POST` | `/api/v1/workspaces/:wid/tasks/:task_id/actions/priority` | `ChangePriority` |
| `POST` | `/api/v1/workspaces/:wid/tasks/:task_id/actions/assignee` | `ChangeAssignee` |
| `POST` | `/api/v1/workspaces/:wid/tasks/:task_id/actions/due-date` | `SetDueDate` |

Each handler:
1. Authenticate user
2. Parse `task_id` param
3. Call `GetTask` to resolve `chatID`
4. Parse action value from request body (form or JSON)
5. Call corresponding `ActionService` method with `chatID`
6. Return `204 No Content` with `HX-Trigger: taskUpdated` header

Form fields must match existing sidebar select `name` attributes:
- `status` for status changes
- `priority` for priority changes
- `assignee_id` for assignee changes
- `due_date` for due date changes

### Step 2: Register Routes in `cmd/api/routes.go`

Add to `registerTaskRoutes`:

```go
// Task field changes via chat message system
if c.TaskActionHandler != nil {
    tasks.POST("/:task_id/actions/status", c.TaskActionHandler.ChangeStatus)
    tasks.POST("/:task_id/actions/priority", c.TaskActionHandler.ChangePriority)
    tasks.POST("/:task_id/actions/assignee", c.TaskActionHandler.ChangeAssignee)
    tasks.POST("/:task_id/actions/due-date", c.TaskActionHandler.SetDueDate)
}
```

Keep the existing direct PUT endpoints (they are valid programmatic API endpoints).
Only the UI sidebar is being redirected.

### Step 3: Wire `TaskActionHandler` in `cmd/api/container.go`

Add `TaskActionHandler *httphandler.TaskActionHandler` field to `Container`.

Initialize in container setup, passing:
- `taskService` (for `GetTask`)
- `actionService` (already wired for `ChatActionHandler`)

### Step 4: Update `web/templates/task/sidebar.html`

**4a. Change HTMX calls from PUT to POST, and update URLs:**

Status field (line 36-38):
```html
<!-- Before -->
<select hx-put="/api/v1/tasks/{{.Task.ID}}/status"
        hx-trigger="change"
        hx-swap="none"

<!-- After -->
<select hx-post="/api/v1/workspaces/{{.Task.WorkspaceID}}/tasks/{{.Task.ID}}/actions/status"
        hx-trigger="change"
        hx-swap="none"
```

Priority field (line 54-56):
```html
<!-- Before -->
<select hx-put="/api/v1/tasks/{{.Task.ID}}/priority"

<!-- After -->
<select hx-post="/api/v1/workspaces/{{.Task.WorkspaceID}}/tasks/{{.Task.ID}}/actions/priority"
```

Assignee (user_select component, line 74-81):
```html
<!-- Before -->
"HxPut" (printf "/api/v1/tasks/%s/assignee" .Task.ID)

<!-- After -->
"HxPost" (printf "/api/v1/workspaces/%s/tasks/%s/actions/assignee" .Task.WorkspaceID .Task.ID)
```

Due Date (date_picker component, line 87-92):
```html
<!-- Before -->
"HxPut" (printf "/api/v1/tasks/%s/due-date" .Task.ID)

<!-- After -->
"HxPost" (printf "/api/v1/workspaces/%s/tasks/%s/actions/due-date" .Task.WorkspaceID .Task.ID)
```

**Note**: Verify that the `components/user_select` and `components/date_picker` templates support
`HxPost` parameter in addition to `HxPut`. If not, add `HxPost` support to these components.

**4b. Quick date buttons in sidebar** (lines 111-117):

The `setQuickDate` JS function currently calls `PUT /api/v1/tasks/:id/due-date` directly.
Update it to use the new action endpoint:

```javascript
function setQuickDate(taskId, workspaceId, daysFromNow) {
    var date = new Date();
    date.setDate(date.getDate() + daysFromNow);
    var dateStr = date.toISOString().split('T')[0];

    htmx.ajax('POST',
        '/api/v1/workspaces/' + workspaceId + '/tasks/' + taskId + '/actions/due-date',
        {
            values: { due_date: dateStr },
            swap: 'none'
        }
    );
}
```

Pass `WorkspaceID` to the JS call:
```html
onclick="setQuickDate('{{.Task.ID}}', '{{.Task.WorkspaceID}}', 0)"
```

**4c. Remove the Activity Timeline section** (lines 182-192):

Remove the entire block:
```html
<!-- Activity Timeline -->
<div class="field">
    <label>Activity</label>
    <div id="task-activity-{{.Task.ID}}"
         class="activity-timeline"
         hx-get="/partials/tasks/{{.Task.ID}}/activity"
         hx-trigger="load"
         hx-swap="innerHTML">
        {{template "components/loading" (dict "ID" "activity-loading")}}
    </div>
</div>
```

Also remove the `.activity-timeline` CSS class from the `<style>` block.

**4d. Update sidebar refresh logic**:

The existing `task.updated` WebSocket event listener (lines 212-218) already handles sidebar
refresh. Since tag processing is asynchronous (creates message → processes tags → updates task
model), there may be a brief delay before the task model reflects the new value.

The sidebar will refresh automatically when the `task.updated` WebSocket event arrives.
No change needed here — the existing pattern handles it correctly.

However, add optimistic UI: after the POST request completes (204 response), update the
select element's visual state immediately without waiting for the WebSocket refresh. HTMX handles
this with `hx-swap="none"` — the select already reflects the user's choice since they selected it.

### Step 5: Check Component Templates

**File**: `web/templates/components/user_select.html`
**File**: `web/templates/components/date_picker.html`

Verify these components support a `HxPost` parameter. If they only have `HxPut`:

- Add conditional logic: if `HxPost` is set, use `hx-post`; otherwise use `hx-put`
- Or rename the parameter to `HxUrl` + `HxMethod` for flexibility

Example for `user_select.html`:
```html
{{if .HxPost}}
<select hx-post="{{.HxPost}}" hx-trigger="change" hx-swap="none" name="{{.Name}}">
{{else}}
<select hx-put="{{.HxPut}}" hx-trigger="change" hx-swap="none" name="{{.Name}}">
{{end}}
```

### Step 6: Update Tests

**File**: `internal/handler/http/task_handler_test.go`

- The existing PUT endpoint tests remain valid (those endpoints still exist for API use)
- Add new test file `task_action_handler_test.go` covering:
  - `POST .../actions/status` → calls ActionService with correct chatID
  - `POST .../actions/priority` → calls ActionService
  - `POST .../actions/assignee` → calls ActionService
  - `POST .../actions/due-date` → calls ActionService
  - Error cases: task not found, invalid values, unauthenticated

**File**: `internal/handler/http/task_detail_template_handler_test.go`

- Verify sidebar renders without activity section

## Files to Change

| File | Change |
|------|--------|
| `internal/handler/http/task_action_handler.go` | **New file** — task action handler |
| `cmd/api/routes.go` | Register new task action routes |
| `cmd/api/container.go` | Wire `TaskActionHandler` |
| `web/templates/task/sidebar.html` | Change HTMX calls, remove activity section |
| `web/templates/components/user_select.html` | Add `HxPost` support |
| `web/templates/components/date_picker.html` | Add `HxPost` support |
| `internal/handler/http/task_action_handler_test.go` | **New file** — tests |

## Files NOT Changed

- `internal/handler/http/task_handler.go` — keep existing PUT endpoints (programmatic API)
- `internal/service/action_service.go` — no changes needed, already correct
- `internal/handler/http/chat_action_handler.go` — no changes needed
- `internal/domain/tag/` — no changes needed, tag system already handles these commands
- `web/templates/task/activity.html` — can be left as-is (no longer referenced from sidebar)

## Dependencies

- `ActionService` must be wired in container before `TaskActionHandler` can be created
  (already the case since `ChatActionHandler` uses it)
- `TaskDetailService.GetTask` used to resolve `taskID → chatID`

## Key Constraints

- Tag processing is **asynchronous** — changes appear in chat before the task read model
  is updated. The sidebar refreshes via WebSocket `task.updated` event, which fires
  after the tag processor updates the task model. This delay is typically < 1 second.
- The `ActionService` batcher groups rapid changes within a 500ms window. If a user
  quickly changes status and priority, they appear as one combined chat message.
- Direct PUT task endpoints remain available for programmatic API consumers.

## Acceptance Criteria

- [x] Changing status in sidebar creates a system message in the associated chat
- [x] Bot response appears in chat: "Alice changed status to In Progress"
- [x] Task read model updates after tag processing (sidebar refreshes)
- [x] Same flow works for priority, assignee, and due date changes
- [x] Activity section is absent from the task sidebar
- [x] Quick date buttons route through action endpoints
- [x] No activity is tracked in the sidebar (only in chat)
- [x] Existing direct PUT endpoints still work for API consumers
- [x] Tests pass for new task action handler
