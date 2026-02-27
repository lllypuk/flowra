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

// resolveActorAndChat extracts the authenticated user ID and resolves the task's chat ID.
func (h *TaskActionHandler) resolveActorAndTask(c echo.Context) (uuid.UUID, *taskapp.ReadModel, error) {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return uuid.UUID(""), nil, httpserver.RespondErrorWithCode(
			c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("task_id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return uuid.UUID(""), nil, httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	taskModel, getErr := h.taskService.GetTask(c.Request().Context(), taskID)
	if getErr != nil {
		return uuid.UUID(""), nil, httpserver.RespondError(c, getErr)
	}

	return userID, taskModel, nil
}

// ChangeStatus handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/status.
// Sends a #status tag message to the task's associated chat.
func (h *TaskActionHandler) ChangeStatus(c echo.Context) error {
	userID, taskModel, err := h.resolveActorAndTask(c)
	if err != nil {
		return err
	}
	if taskModel == nil {
		return nil
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

	status, statusErr := parseStatus(req.Status)
	if statusErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, "INVALID_STATUS", statusErr.Error())
	}

	// Idempotent no-op: same status requested.
	if taskModel.Status == status {
		c.Response().Header().Set("Hx-Trigger", "taskUpdated")
		return c.NoContent(http.StatusNoContent)
	}

	if _, actionErr := h.actionService.ChangeStatus(
		c.Request().Context(),
		taskModel.ChatID,
		string(status),
		userID,
	); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("Hx-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// ChangePriority handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/priority.
// Sends a #priority tag message to the task's associated chat.
func (h *TaskActionHandler) ChangePriority(c echo.Context) error {
	userID, taskModel, err := h.resolveActorAndTask(c)
	if err != nil {
		return err
	}
	if taskModel == nil {
		return nil
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

	priority := parsePriorityStrict(req.Priority)
	if priority == "" {
		return httpserver.RespondErrorWithCode(
			c,
			http.StatusBadRequest,
			"INVALID_PRIORITY",
			"priority must be Low, Medium, High, or Critical",
		)
	}

	// Idempotent no-op: same priority requested.
	if taskModel.Priority == priority {
		c.Response().Header().Set("Hx-Trigger", "taskUpdated")
		return c.NoContent(http.StatusNoContent)
	}

	if _, actionErr := h.actionService.SetPriority(
		c.Request().Context(),
		taskModel.ChatID,
		string(priority),
		userID,
	); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("Hx-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// ChangeAssignee handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/assignee.
// Sends an #assignee tag message to the task's associated chat.
// An empty assignee_id clears the current assignee.
func (h *TaskActionHandler) ChangeAssignee(c echo.Context) error {
	ctx := c.Request().Context()
	userID, taskModel, err := h.resolveActorAndTask(c)
	if err != nil {
		return err
	}
	if taskModel == nil {
		return nil
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

	// Idempotent no-op: same assignee requested.
	if sameUUIDPtr(taskModel.AssignedTo, assigneeID) {
		c.Response().Header().Set("Hx-Trigger", "taskUpdated")
		return c.NoContent(http.StatusNoContent)
	}

	if _, actionErr := h.actionService.AssignUser(ctx, taskModel.ChatID, assigneeID, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("Hx-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}

// SetDueDate handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/actions/due-date.
// Sends a #due tag message to the task's associated chat.
// An empty due_date clears the current due date.
func (h *TaskActionHandler) SetDueDate(c echo.Context) error {
	ctx := c.Request().Context()
	userID, taskModel, err := h.resolveActorAndTask(c)
	if err != nil {
		return err
	}
	if taskModel == nil {
		return nil
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

	// Idempotent no-op: same due date requested.
	if sameDate(taskModel.DueDate, dueDate) {
		c.Response().Header().Set("Hx-Trigger", "taskUpdated")
		return c.NoContent(http.StatusNoContent)
	}

	if _, actionErr := h.actionService.SetDueDate(ctx, taskModel.ChatID, dueDate, userID); actionErr != nil {
		return httpserver.RespondError(c, actionErr)
	}

	c.Response().Header().Set("Hx-Trigger", "taskUpdated")
	return c.NoContent(http.StatusNoContent)
}
