# Step 2: Register Task Action Routes

## Status: Complete

## Goal

Register the four new task action endpoints in `cmd/api/routes.go` under the existing
workspace-scoped task route group, and add a `TaskActionHandler` field to the `Container`.

## Files to Change

- `cmd/api/routes.go` — add action routes to `registerTaskRoutes`
- `cmd/api/container.go` — add `TaskActionHandler` field to the `Container` struct

---

## Change 1: `cmd/api/routes.go`

### Location

Function `registerTaskRoutes` (currently at line ~181).

### Current Code

```go
func registerTaskRoutes(r *httpserver.Router, c *Container) {
	tasks := r.NewWorkspaceRouteGroup("/tasks")

	if c.TaskHandler != nil {
		tasks.POST("", c.TaskHandler.Create)
		tasks.GET("", c.TaskHandler.List)
		tasks.GET("/:task_id", c.TaskHandler.Get)
		tasks.PUT("/:task_id/status", c.TaskHandler.ChangeStatus)
		tasks.PUT("/:task_id/assignee", c.TaskHandler.Assign)
		tasks.PUT("/:task_id/priority", c.TaskHandler.ChangePriority)
		tasks.PUT("/:task_id/due-date", c.TaskHandler.SetDueDate)
		tasks.DELETE("/:task_id", c.TaskHandler.Delete)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := createPlaceholderHandler("Task")
		tasks.POST("", placeholder)
		tasks.GET("", placeholder)
		tasks.GET("/:task_id", placeholder)
		tasks.PUT("/:task_id/status", placeholder)
		tasks.PUT("/:task_id/assignee", placeholder)
		tasks.PUT("/:task_id/priority", placeholder)
		tasks.PUT("/:task_id/due-date", placeholder)
		tasks.DELETE("/:task_id", placeholder)
	}
}
```

### Updated Code

Add the task action block **after** the existing `TaskHandler` block:

```go
func registerTaskRoutes(r *httpserver.Router, c *Container) {
	tasks := r.NewWorkspaceRouteGroup("/tasks")

	if c.TaskHandler != nil {
		tasks.POST("", c.TaskHandler.Create)
		tasks.GET("", c.TaskHandler.List)
		tasks.GET("/:task_id", c.TaskHandler.Get)
		tasks.PUT("/:task_id/status", c.TaskHandler.ChangeStatus)
		tasks.PUT("/:task_id/assignee", c.TaskHandler.Assign)
		tasks.PUT("/:task_id/priority", c.TaskHandler.ChangePriority)
		tasks.PUT("/:task_id/due-date", c.TaskHandler.SetDueDate)
		tasks.DELETE("/:task_id", c.TaskHandler.Delete)
	} else {
		// Placeholder endpoints when handler is not initialized
		placeholder := createPlaceholderHandler("Task")
		tasks.POST("", placeholder)
		tasks.GET("", placeholder)
		tasks.GET("/:task_id", placeholder)
		tasks.PUT("/:task_id/status", placeholder)
		tasks.PUT("/:task_id/assignee", placeholder)
		tasks.PUT("/:task_id/priority", placeholder)
		tasks.PUT("/:task_id/due-date", placeholder)
		tasks.DELETE("/:task_id", placeholder)
	}

	// Task field changes routed through the chat message system.
	// These endpoints create system messages in the task's associated chat
	// instead of updating the task aggregate directly.
	if c.TaskActionHandler != nil {
		tasks.POST("/:task_id/actions/status", c.TaskActionHandler.ChangeStatus)
		tasks.POST("/:task_id/actions/priority", c.TaskActionHandler.ChangePriority)
		tasks.POST("/:task_id/actions/assignee", c.TaskActionHandler.ChangeAssignee)
		tasks.POST("/:task_id/actions/due-date", c.TaskActionHandler.SetDueDate)
	}
}
```

### Resulting Routes

The following routes will be registered under `/api/v1/workspaces/:workspace_id/tasks`:

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| `POST` | `/:task_id/actions/status` | `TaskActionHandler.ChangeStatus` | Status → chat |
| `POST` | `/:task_id/actions/priority` | `TaskActionHandler.ChangePriority` | Priority → chat |
| `POST` | `/:task_id/actions/assignee` | `TaskActionHandler.ChangeAssignee` | Assignee → chat |
| `POST` | `/:task_id/actions/due-date` | `TaskActionHandler.SetDueDate` | Due date → chat |

Full path examples:
- `POST /api/v1/workspaces/abc123/tasks/def456/actions/status`
- `POST /api/v1/workspaces/abc123/tasks/def456/actions/priority`
- `POST /api/v1/workspaces/abc123/tasks/def456/actions/assignee`
- `POST /api/v1/workspaces/abc123/tasks/def456/actions/due-date`

All four routes are workspace-scoped and require authentication (handled by the router group).

---

## Change 2: `cmd/api/container.go`

### Location

The `Container` struct definition (currently around line 82).

### Current Handler Fields

```go
// HTTP Handlers
AuthHandler         *httphandler.AuthHandler
WorkspaceHandler    *httphandler.WorkspaceHandler
ChatHandler         *httphandler.ChatHandler
ChatActionHandler   *httphandler.ChatActionHandler
MessageHandler      *httphandler.MessageHandler
FileHandler         *httphandler.FileHandler
TaskHandler         *httphandler.TaskHandler
NotificationHandler *httphandler.NotificationHandler
UserHandler         *httphandler.UserHandler
WSHandler           *wshandler.Handler
```

### Updated Handler Fields

Add `TaskActionHandler` after `TaskHandler`:

```go
// HTTP Handlers
AuthHandler         *httphandler.AuthHandler
WorkspaceHandler    *httphandler.WorkspaceHandler
ChatHandler         *httphandler.ChatHandler
ChatActionHandler   *httphandler.ChatActionHandler
MessageHandler      *httphandler.MessageHandler
FileHandler         *httphandler.FileHandler
TaskHandler         *httphandler.TaskHandler
TaskActionHandler   *httphandler.TaskActionHandler
NotificationHandler *httphandler.NotificationHandler
UserHandler         *httphandler.UserHandler
WSHandler           *wshandler.Handler
```

---

## Notes

- The nil guard `if c.TaskActionHandler != nil` matches the pattern already used for
  `ChatActionHandler` in `registerChatRoutes`. No placeholder needed here since the
  action routes are additive — the PUT endpoints remain for programmatic API use.
- The existing `PUT /:task_id/status` etc. endpoints are intentionally kept. They serve
  programmatic API consumers (e.g. integrations, future mobile clients) and are not
  touched by this change.
- Initialization of `TaskActionHandler` in the container is covered in **Step 3**.
