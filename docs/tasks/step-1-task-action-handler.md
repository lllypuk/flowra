# Step 1: New `TaskActionHandler`

## Status: Complete

## Goal

Create a new HTTP handler that accepts task field change requests from the UI sidebar and routes
them through the `ActionService` (chat message system) instead of directly updating the task
aggregate. The handler resolves `taskID → chatID` and delegates to the existing `ActionService`.

## Context

`ActionService` (`internal/service/action_service.go`) already provides all necessary methods:
- `ChangeStatus(ctx, chatID, newStatus, actorID)`
- `SetPriority(ctx, chatID, priority, actorID)`
- `AssignUser(ctx, chatID, assigneeID, actorID)`
- `SetDueDate(ctx, chatID, dueDate, actorID)`

These create system messages with tags, which trigger the tag processor to update the chat
aggregate asynchronously. `ChatActionHandler` uses this same service for chat-level actions.
The new `TaskActionHandler` follows the same pattern but takes a `taskID` as input and resolves
it to a `chatID` by loading the task read model.

## File to Create

**`internal/handler/http/task_action_handler.go`**

## Full Implementation

```go
package httphandler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// TaskActionTaskService resolves a task by ID to get its associated chat ID.
// Declared on the consumer side per project guidelines.
type TaskActionTaskService interface {
	GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)
}

// TaskActionService defines the actions that can be triggered on a task's chat.
// Declared on the consumer side per project guidelines.
// This is the same interface as ActionService (see chat_action_handler.go),
// repeated here to keep consumer-side interface ownership and avoid coupling.
type TaskActionService interface {
	ChangeStatus(
		ctx context.Context,
		chatID uuid.UUID,
		newStatus string,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)

	SetPriority(
		ctx context.Context,
		chatID uuid.UUID,
		priority string,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)

	AssignUser(
		ctx context.Context,
		chatID uuid.UUID,
		assigneeID *uuid.UUID,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)

	SetDueDate(
		ctx context.Context,
		chatID uuid.UUID,
		dueDate *time.Time,
		actorID uuid.UUID,
	) (*appcore.ActionResult, error)
}

// TaskActionHandler routes task field changes through the chat message system.
// It resolves taskID → chatID and delegates to ActionService, which creates
// system messages that drive task updates via tag processing.
type TaskActionHandler struct {
	taskService   TaskActionTaskService
	actionService TaskActionService
}

// NewTaskActionHandler creates a new TaskActionHandler.
func NewTaskActionHandler(
	taskService TaskActionTaskService,
	actionService TaskActionService,
) *TaskActionHandler {
	return &TaskActionHandler{
		taskService:   taskService,
		actionService: actionService,
	}
}

// ChangeStatus handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/status.
// Sends a #status tag message to the task's associated chat.
func (h *TaskActionHandler) ChangeStatus(c echo.Context) error {
	ctx := c.Request().Context()

	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatID, err := h.resolveChatID(ctx, c)
	if err != nil {
		return err
	}

	var req struct {
		Status string `json:"status" form:"status"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}
	if req.Status == "" {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_STATUS", "status is required")
	}

	if _, actionErr := h.actionService.ChangeStatus(ctx, chatID, req.Status, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("HX-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// ChangePriority handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/priority.
// Sends a #priority tag message to the task's associated chat.
func (h *TaskActionHandler) ChangePriority(c echo.Context) error {
	ctx := c.Request().Context()

	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatID, err := h.resolveChatID(ctx, c)
	if err != nil {
		return err
	}

	var req struct {
		Priority string `json:"priority" form:"priority"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}
	if req.Priority == "" {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_PRIORITY", "priority is required")
	}

	if _, actionErr := h.actionService.SetPriority(ctx, chatID, req.Priority, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("HX-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// ChangeAssignee handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/assignee.
// Sends an #assignee tag message to the task's associated chat.
// An empty assignee_id clears the current assignee.
func (h *TaskActionHandler) ChangeAssignee(c echo.Context) error {
	ctx := c.Request().Context()

	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatID, err := h.resolveChatID(ctx, c)
	if err != nil {
		return err
	}

	var req struct {
		AssigneeID string `json:"assignee_id" form:"assignee_id"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var assigneeID *uuid.UUID
	if req.AssigneeID != "" {
		parsed, parseErr := uuid.ParseUUID(req.AssigneeID)
		if parseErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_ASSIGNEE_ID", "invalid assignee ID format")
		}
		assigneeID = &parsed
	}

	if _, actionErr := h.actionService.AssignUser(ctx, chatID, assigneeID, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("HX-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// SetDueDate handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/due-date.
// Sends a #due tag message to the task's associated chat.
// An empty due_date clears the current due date.
func (h *TaskActionHandler) SetDueDate(c echo.Context) error {
	ctx := c.Request().Context()

	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	chatID, err := h.resolveChatID(ctx, c)
	if err != nil {
		return err
	}

	var req struct {
		DueDate string `json:"due_date" form:"due_date"`
	}
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		parsed, parseErr := time.Parse("2006-01-02", req.DueDate)
		if parseErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_DATE", "invalid date format, use YYYY-MM-DD")
		}
		dueDate = &parsed
	}

	if _, actionErr := h.actionService.SetDueDate(ctx, chatID, dueDate, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("HX-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// resolveChatID extracts the task_id path param, loads the task, and returns its ChatID.
// Returns an echo error response (already written) on failure.
func (h *TaskActionHandler) resolveChatID(ctx context.Context, c echo.Context) (uuid.UUID, error) {
	taskIDStr := c.Param("task_id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return uuid.UUID(""), httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	taskModel, getErr := h.taskService.GetTask(ctx, taskID)
	if getErr != nil {
		return uuid.UUID(""), httpserver.RespondError(c, getErr)
	}

	return taskModel.ChatID, nil
}
```

## Key Design Decisions

### Consumer-Side Interface

`TaskActionTaskService` and `TaskActionService` are declared in this file (consumer side),
following the project's interface ownership rule. They are *not* imported from infrastructure
or service packages.

`TaskActionService` mirrors the `ActionService` interface already declared in
`chat_action_handler.go`. Both use the same underlying `service.ActionService` implementation.
Duplication is intentional — each handler owns its interface.

### `resolveChatID` Helper

The private `resolveChatID` method handles the two-step resolution:
1. Parse `task_id` from the path parameter
2. Load task read model to get `ChatID`

It writes the error response directly and returns the echo error, so callers simply do:
```go
chatID, err := h.resolveChatID(ctx, c)
if err != nil {
    return err
}
```

### HTTP Response

All action handlers return `204 No Content` with an `HX-Trigger: taskUpdated` header.
HTMX on the client side uses this to refresh the sidebar after the action completes.

The task read model update is **asynchronous** (tag processing happens in background).
The `taskUpdated` WebSocket event (published after tag processing) is what triggers
the sidebar reload with the new values.

### No Body in Response

`hx-swap="none"` is used on all sidebar selects. HTMX discards the response body.
The `HX-Trigger` header signals that an update happened; the sidebar listens for the
`task.updated` WebSocket event to reload content with actual updated values.

## Testing

See `step-6-tests.md` for the corresponding test file implementation.
