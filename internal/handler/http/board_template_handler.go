package httphandler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Board template handler constants.
const (
	defaultBoardColumnLimit = 20
	maxBoardColumnLimit     = 100
	maxMembersListLimit     = 100
	boardColumnsCount       = 4

	// Priority string constants.
	priorityStringLow      = "low"
	priorityStringMedium   = "medium"
	priorityStringHigh     = "high"
	priorityStringCritical = "critical"
)

// BoardTaskService defines the interface for task operations needed by the board.
// Declared on the consumer side per project guidelines.
type BoardTaskService interface {
	// ListTasks lists tasks with filters.
	ListTasks(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error)

	// CountTasks counts tasks with filters.
	CountTasks(ctx context.Context, filters taskapp.Filters) (int, error)

	// GetTask gets a task by ID.
	GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)
}

// BoardMemberService defines the interface for member operations needed by the board.
// Declared on the consumer side per project guidelines.
type BoardMemberService interface {
	// ListWorkspaceMembers lists members of a workspace.
	ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]MemberViewData, error)
}

// BoardViewData represents the data needed to render the board page.
type BoardViewData struct {
	Workspace  WorkspaceViewData
	TotalTasks int
	Filters    BoardFilters
	Members    []MemberViewData
	Token      string
	Columns    []ColumnViewData
}

// BoardFilters represents the current filter state.
type BoardFilters struct {
	Type     string
	Assignee string
	Priority string
	Search   string
}

// ColumnViewData represents a single column in the board.
type ColumnViewData struct {
	Status      string
	Title       string
	Tasks       []TaskCardViewData
	Count       int
	TotalCount  int
	HasMore     bool
	WorkspaceID string
}

// TaskCardViewData represents a task card for display.
type TaskCardViewData struct {
	ID          string
	WorkspaceID string
	ChatID      string
	Title       string
	Type        string
	Priority    string
	Status      string
	Assignee    *TaskAssigneeData
	DueDate     *time.Time
	IsOverdue   bool
}

// TaskAssigneeData represents assignee information for a task card.
type TaskAssigneeData struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
}

// BoardColumnStatus represents the status columns shown on the board.
type BoardColumnStatus struct {
	Status task.Status
	Key    string
	Title  string
}

// GetBoardColumns returns the columns to display on the Kanban board.
func GetBoardColumns() []BoardColumnStatus {
	return []BoardColumnStatus{
		{Status: task.StatusToDo, Key: "todo", Title: "To Do"},
		{Status: task.StatusInProgress, Key: "in_progress", Title: "In Progress"},
		{Status: task.StatusInReview, Key: "review", Title: "Review"},
		{Status: task.StatusDone, Key: "done", Title: "Done"},
	}
}

// BoardTemplateHandler provides handlers for rendering the Kanban board.
type BoardTemplateHandler struct {
	renderer      *TemplateRenderer
	logger        *slog.Logger
	taskService   BoardTaskService
	memberService BoardMemberService
}

// NewBoardTemplateHandler creates a new board template handler.
func NewBoardTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	taskService BoardTaskService,
	memberService BoardMemberService,
) *BoardTemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BoardTemplateHandler{
		renderer:      renderer,
		logger:        logger,
		taskService:   taskService,
		memberService: memberService,
	}
}

// SetupBoardRoutes registers board-related page and partial routes.
func (h *BoardTemplateHandler) SetupBoardRoutes(e *echo.Echo) {
	// Board pages (protected)
	workspaces := e.Group("/workspaces", RequireAuth)
	workspaces.GET("/:workspace_id/board", h.BoardIndex)

	// Board partials (protected)
	partials := e.Group("/partials", RequireAuth)
	partials.GET("/workspace/:workspace_id/board", h.BoardPartial)
	partials.GET("/workspace/:workspace_id/board/:status/more", h.BoardColumnMore)
	partials.GET("/tasks/:task_id/card", h.TaskCardPartial)
}

// BoardIndex renders the main board page.
func (h *BoardTemplateHandler) BoardIndex(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		return h.renderNotFound(c)
	}

	// Parse filters from query params
	filters := h.parseFilters(c)

	// Get workspace members for filter dropdown
	var members []MemberViewData
	if h.memberService != nil {
		members, _ = h.memberService.ListWorkspaceMembers(c.Request().Context(), workspaceID, 0, maxMembersListLimit)
	}

	// Count total tasks
	var totalTasks int
	if h.taskService != nil {
		taskFilters := h.buildTaskFilters(workspaceID, filters, user.ID)
		totalTasks, _ = h.taskService.CountTasks(c.Request().Context(), taskFilters)
	}

	data := BoardViewData{
		Workspace: WorkspaceViewData{
			ID: workspaceID.String(),
		},
		TotalTasks: totalTasks,
		Filters:    filters,
		Members:    members,
		Token:      "", // TODO: Get JWT token for WebSocket auth
	}

	return h.render(c, "board/index", "Board", data)
}

// BoardPartial returns all columns with tasks as HTML partial for HTMX.
func (h *BoardTemplateHandler) BoardPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid workspace ID")
	}

	// Parse filters
	filters := h.parseFilters(c)

	// Build columns
	columns := h.buildColumns(c.Request().Context(), workspaceID, filters, user.ID)

	data := map[string]any{
		"Columns": columns,
	}

	return h.renderPartial(c, "board/columns", data)
}

// BoardColumnMore returns additional tasks for a column (pagination).
func (h *BoardTemplateHandler) BoardColumnMore(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	workspaceID, err := uuid.ParseUUID(c.Param("workspace_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid workspace ID")
	}

	statusKey := c.Param("status")
	status := h.parseStatusKey(statusKey)
	if status == nil {
		return c.String(http.StatusBadRequest, "Invalid status")
	}

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	// Parse filters
	filters := h.parseFilters(c)

	// Build task filters for this column
	taskFilters := h.buildTaskFilters(workspaceID, filters, user.ID)
	taskFilters.Status = status
	taskFilters.Offset = offset
	taskFilters.Limit = defaultBoardColumnLimit

	// Get tasks
	var tasks []*taskapp.ReadModel
	var totalCount int
	if h.taskService != nil {
		tasks, _ = h.taskService.ListTasks(c.Request().Context(), taskFilters)
		totalCount, _ = h.taskService.CountTasks(c.Request().Context(), taskFilters)
	}

	// Convert to view data
	taskCards := h.convertTasksToCards(tasks, workspaceID.String())

	data := map[string]any{
		"Tasks":       taskCards,
		"Status":      statusKey,
		"WorkspaceID": workspaceID.String(),
		"Offset":      offset + len(taskCards),
		"TotalCount":  totalCount,
		"HasMore":     offset+len(taskCards) < totalCount,
	}

	return h.renderPartial(c, "board/column_more", data)
}

// TaskCardPartial returns a single task card as HTML partial.
func (h *BoardTemplateHandler) TaskCardPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	taskID, err := uuid.ParseUUID(c.Param("task_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid task ID")
	}

	if h.taskService == nil {
		return c.String(http.StatusInternalServerError, "Task service unavailable")
	}

	taskModel, err := h.taskService.GetTask(c.Request().Context(), taskID)
	if err != nil {
		return c.String(http.StatusNotFound, "Task not found")
	}

	// We need workspace ID - get it from the task's chat
	// For now, use an empty string as we'd need to look this up
	workspaceID := "" // TODO: Get workspace ID from task or chat

	card := h.convertTaskToCard(taskModel, workspaceID)

	return h.renderPartial(c, "components/task_card", card)
}

// buildColumns builds all column data for the board.
func (h *BoardTemplateHandler) buildColumns(
	ctx context.Context,
	workspaceID uuid.UUID,
	filters BoardFilters,
	userID string,
) []ColumnViewData {
	columns := make([]ColumnViewData, 0, boardColumnsCount)
	boardColumns := GetBoardColumns()

	for _, col := range boardColumns {
		// Build filters for this column
		taskFilters := h.buildTaskFilters(workspaceID, filters, userID)
		taskFilters.Status = &col.Status
		taskFilters.Offset = 0
		taskFilters.Limit = defaultBoardColumnLimit

		var tasks []*taskapp.ReadModel
		var totalCount int

		if h.taskService != nil {
			tasks, _ = h.taskService.ListTasks(ctx, taskFilters)
			totalCount, _ = h.taskService.CountTasks(ctx, taskFilters)
		}

		taskCards := h.convertTasksToCards(tasks, workspaceID.String())

		columns = append(columns, ColumnViewData{
			Status:      col.Key,
			Title:       col.Title,
			Tasks:       taskCards,
			Count:       len(taskCards),
			TotalCount:  totalCount,
			HasMore:     len(taskCards) < totalCount,
			WorkspaceID: workspaceID.String(),
		})
	}

	return columns
}

// buildTaskFilters builds task filters from board filters.
func (h *BoardTemplateHandler) buildTaskFilters(
	_ uuid.UUID, // workspaceID - reserved for future workspace-scoped filtering
	filters BoardFilters,
	userID string,
) taskapp.Filters {
	taskFilters := taskapp.Filters{}

	// TODO: Add workspace filter when supported by repository

	// Filter by entity type
	if filters.Type != "" {
		entityType := parseEntityTypeFromString(filters.Type)
		if entityType != nil {
			taskFilters.EntityType = entityType
		}
	}

	// Filter by priority
	if filters.Priority != "" {
		priority := parsePriorityFromString(filters.Priority)
		if priority != nil {
			taskFilters.Priority = priority
		}
	}

	// Filter by assignee
	if filters.Assignee != "" {
		switch filters.Assignee {
		case "unassigned":
			// TODO: Support unassigned filter in repository
		case "me":
			if uid, err := uuid.ParseUUID(userID); err == nil {
				taskFilters.AssigneeID = &uid
			}
		default:
			if uid, err := uuid.ParseUUID(filters.Assignee); err == nil {
				taskFilters.AssigneeID = &uid
			}
		}
	}

	// TODO: Add search filter when supported by repository

	return taskFilters
}

// convertTasksToCards converts task read models to view data.
func (h *BoardTemplateHandler) convertTasksToCards(
	tasks []*taskapp.ReadModel,
	workspaceID string,
) []TaskCardViewData {
	cards := make([]TaskCardViewData, 0, len(tasks))
	for _, t := range tasks {
		cards = append(cards, h.convertTaskToCard(t, workspaceID))
	}
	return cards
}

// convertTaskToCard converts a single task read model to view data.
func (h *BoardTemplateHandler) convertTaskToCard(
	t *taskapp.ReadModel,
	workspaceID string,
) TaskCardViewData {
	card := TaskCardViewData{
		ID:          t.ID.String(),
		WorkspaceID: workspaceID,
		ChatID:      t.ChatID.String(),
		Title:       t.Title,
		Type:        string(t.EntityType),
		Priority:    string(t.Priority),
		Status:      string(t.Status),
		DueDate:     t.DueDate,
	}

	// Check if overdue
	if t.DueDate != nil && t.Status != task.StatusDone {
		card.IsOverdue = t.DueDate.Before(time.Now())
	}

	// TODO: Load assignee details from user service
	if t.AssignedTo != nil {
		card.Assignee = &TaskAssigneeData{
			ID:       t.AssignedTo.String(),
			Username: "user", // TODO: Load from user service
		}
	}

	return card
}

// parseFilters extracts filter values from query parameters.
func (h *BoardTemplateHandler) parseFilters(c echo.Context) BoardFilters {
	return BoardFilters{
		Type:     strings.TrimSpace(c.QueryParam("type")),
		Assignee: strings.TrimSpace(c.QueryParam("assignee")),
		Priority: strings.TrimSpace(c.QueryParam("priority")),
		Search:   strings.TrimSpace(c.QueryParam("search")),
	}
}

// parseStatusKey converts a status key to a task.Status.
func (h *BoardTemplateHandler) parseStatusKey(key string) *task.Status {
	switch key {
	case "todo":
		s := task.StatusToDo
		return &s
	case "in_progress":
		s := task.StatusInProgress
		return &s
	case "review":
		s := task.StatusInReview
		return &s
	case "done":
		s := task.StatusDone
		return &s
	default:
		return nil
	}
}

// getUserView extracts user information from the context for templates.
func (h *BoardTemplateHandler) getUserView(c echo.Context) *UserView {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return nil
	}

	return &UserView{
		ID: userID.String(),
	}
}

// render renders a full page with the base layout.
func (h *BoardTemplateHandler) render(c echo.Context, templateName string, title string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	pageData := PageData{
		Title: title,
		User:  h.getUserView(c),
		Data:  data,
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, pageData, c)
}

// renderPartial renders a template without the base layout.
func (h *BoardTemplateHandler) renderPartial(c echo.Context, templateName string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, data, c)
}

// renderNotFound renders a 404 error page.
func (h *BoardTemplateHandler) renderNotFound(c echo.Context) error {
	return c.String(http.StatusNotFound, "Page not found")
}

// parseEntityTypeFromString converts a string to task.EntityType.
//

func parseEntityTypeFromString(s string) *task.EntityType {
	switch strings.ToLower(s) {
	case chatTypeTask:
		t := task.TypeTask
		return &t
	case chatTypeBug:
		t := task.TypeBug
		return &t
	case chatTypeEpic:
		t := task.TypeEpic
		return &t
	default:
		return nil
	}
}

// parsePriorityFromString converts a string to task.Priority.
//

func parsePriorityFromString(s string) *task.Priority {
	switch strings.ToLower(s) {
	case priorityStringLow:
		p := task.PriorityLow
		return &p
	case priorityStringMedium:
		p := task.PriorityMedium
		return &p
	case priorityStringHigh:
		p := task.PriorityHigh
		return &p
	case priorityStringCritical:
		p := task.PriorityCritical
		return &p
	default:
		return nil
	}
}
