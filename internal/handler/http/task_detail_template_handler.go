package httphandler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Task detail template handler constants.
const (
	defaultActivityLimit    = 50
	dueSoonDays             = 3
	maxMembersListLimitTask = 100
)

// TaskDetailService defines the interface for task operations needed by the detail view.
// Declared on the consumer side per project guidelines.
type TaskDetailService interface {
	// GetTask gets a task by ID.
	GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)

	// GetTaskByChatID gets a task by its associated chat ID.
	GetTaskByChatID(ctx context.Context, chatID uuid.UUID) (*taskapp.ReadModel, error)
}

// TaskEventService defines the interface for loading task events for activity timeline.
type TaskEventService interface {
	// GetEvents returns all events for a task (for activity timeline).
	GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error)
}

// ChatBasicInfoService defines the interface for loading basic chat information.
// Declared on the consumer side per project guidelines.
type ChatBasicInfoService interface {
	// GetChatBasicInfo returns minimal chat information needed for task details.
	GetChatBasicInfo(ctx context.Context, chatID uuid.UUID) (*ChatBasicInfo, error)
}

// ChatBasicInfo contains minimal chat information needed for task sidebar.
type ChatBasicInfo struct {
	ID          string
	WorkspaceID string
	Type        string
}

// TaskDetailMemberService is an alias for BoardMemberService to avoid interface duplication.
// We use the same interface since task details need the same member operations as the board.
type TaskDetailMemberService = BoardMemberService

// TaskSidebarViewData represents the data needed to render the task sidebar.
type TaskSidebarViewData struct {
	Task         TaskDetailViewData
	Statuses     []SelectOption
	Priorities   []SelectOption
	Participants []MemberViewData
	Token        string
}

// TaskDetailViewData represents task data for the detail view.
type TaskDetailViewData struct {
	ID           string
	WorkspaceID  string
	ChatID       string
	Title        string
	Description  string
	Type         string
	Status       string
	Priority     string
	AssigneeID   string
	DueDate      *time.Time
	IsOverdue    bool
	IsDueSoon    bool
	OverdueDays  int
	DaysUntilDue int
	CreatedAt    time.Time
}

// ActivityViewData represents a single activity item for the timeline.
type ActivityViewData struct {
	Actor      ActivityActorData
	ActionText string
	Details    bool
	OldValue   string
	NewValue   string
	CreatedAt  time.Time
}

// ActivityActorData represents the actor who performed an activity.
type ActivityActorData struct {
	ID       string
	Username string
}

// SelectOption represents an option for select dropdowns.
type SelectOption struct {
	Value string
	Label string
}

// TaskDetailTemplateHandler provides handlers for rendering task detail views.
type TaskDetailTemplateHandler struct {
	renderer        *TemplateRenderer
	logger          *slog.Logger
	taskService     TaskDetailService
	eventService    TaskEventService
	memberService   TaskDetailMemberService
	chatInfoService ChatBasicInfoService
}

// NewTaskDetailTemplateHandler creates a new task detail template handler.
func NewTaskDetailTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	taskService TaskDetailService,
	eventService TaskEventService,
	memberService TaskDetailMemberService,
	chatInfoService ChatBasicInfoService,
) *TaskDetailTemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskDetailTemplateHandler{
		renderer:        renderer,
		logger:          logger,
		taskService:     taskService,
		eventService:    eventService,
		memberService:   memberService,
		chatInfoService: chatInfoService,
	}
}

// SetupTaskDetailRoutes registers task detail-related partial routes.
func (h *TaskDetailTemplateHandler) SetupTaskDetailRoutes(e *echo.Echo) {
	// Task detail partials (protected)
	partials := e.Group("/partials", RequireAuth)
	partials.GET("/tasks/:task_id/sidebar", h.TaskSidebarPartial)
	partials.GET("/tasks/:task_id/activity", h.TaskActivityPartial)
	partials.GET("/tasks/:task_id/edit-title", h.TaskEditTitleForm)
	partials.GET("/tasks/:task_id/title-display", h.TaskTitleDisplay)
	partials.GET("/tasks/:task_id/edit-description", h.TaskEditDescriptionForm)
	partials.GET("/tasks/:task_id/description-display", h.TaskDescriptionDisplay)
	partials.GET("/tasks/:task_id/quick-edit", h.TaskQuickEditPopover)

	// Chat-based task details (for sidebar in chat view)
	partials.GET("/chats/:chat_id/task-details", h.TaskDetailsByChatID)
}

// TaskDetailsByChatID returns task details for a chat (resolves task by chat_id).
func (h *TaskDetailTemplateHandler) TaskDetailsByChatID(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	chatID, err := uuid.ParseUUID(c.Param("chat_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid chat ID")
	}

	if h.taskService == nil {
		return c.String(http.StatusInternalServerError, "Task service unavailable")
	}

	// Get basic chat info first to check type
	var chatInfo *ChatBasicInfo
	if h.chatInfoService != nil {
		chatInfo, _ = h.chatInfoService.GetChatBasicInfo(c.Request().Context(), chatID)
	}

	// Check if this chat type supports task details
	if chatInfo != nil && !isTaskChatType(chatInfo.Type) {
		return h.renderPartial(c, "chat/task-sidebar-error", map[string]any{
			"Message": "This chat does not have task details",
		})
	}

	// Get task by chat ID
	taskModel, err := h.taskService.GetTaskByChatID(c.Request().Context(), chatID)
	if err != nil {
		return h.renderPartial(c, "chat/task-sidebar-error", map[string]any{
			"Message": "Task details not available",
		})
	}

	// Fallback if chat info not available
	if chatInfo == nil {
		chatInfo = &ChatBasicInfo{
			ID:          chatID.String(),
			WorkspaceID: "", // Will need to be handled in template
			Type:        string(taskModel.EntityType),
		}
	}

	// Get workspace members for assignee dropdown
	var participants []MemberViewData
	if h.memberService != nil && chatInfo.WorkspaceID != "" {
		workspaceID, parseErr := uuid.ParseUUID(chatInfo.WorkspaceID)
		if parseErr == nil {
			participants, _ = h.memberService.ListWorkspaceMembers(
				c.Request().Context(), workspaceID, 0, maxMembersListLimitTask)
		}
	}

	// Build data structure matching template expectations
	innerData := map[string]any{
		"Task":         h.convertToDetailView(taskModel),
		"Chat":         chatInfo,
		"Statuses":     getStatusOptions(),
		"Priorities":   getPriorityOptions(),
		"Participants": participants,
	}

	data := map[string]any{
		"Data": innerData,
	}

	return h.renderPartial(c, "chat/task-sidebar", data)
}

// isTaskChatType checks if a chat type supports task details.
func isTaskChatType(chatType string) bool {
	return chatType == chatTypeTask || chatType == chatTypeBug || chatType == chatTypeEpic
}

// TaskSidebarPartial returns the full task sidebar as HTML partial.
func (h *TaskDetailTemplateHandler) TaskSidebarPartial(c echo.Context) error {
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

	// Get workspace members for assignee dropdown via chat info
	var participants []MemberViewData
	if h.memberService != nil && h.chatInfoService != nil {
		chatInfo, chatErr := h.chatInfoService.GetChatBasicInfo(c.Request().Context(), taskModel.ChatID)
		if chatErr == nil && chatInfo != nil && chatInfo.WorkspaceID != "" {
			workspaceID, parseErr := uuid.ParseUUID(chatInfo.WorkspaceID)
			if parseErr == nil {
				participants, _ = h.memberService.ListWorkspaceMembers(
					c.Request().Context(), workspaceID, 0, maxMembersListLimitTask)
			}
		}
	}

	data := TaskSidebarViewData{
		Task:         h.convertToDetailView(taskModel),
		Statuses:     getStatusOptions(),
		Priorities:   getPriorityOptions(),
		Participants: participants,
		Token:        "", // TODO: Get JWT token for WebSocket auth
	}

	return h.renderPartial(c, "task/sidebar", data)
}

// TaskActivityPartial returns the activity timeline for a task.
func (h *TaskDetailTemplateHandler) TaskActivityPartial(c echo.Context) error {
	user := h.getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	taskID, err := uuid.ParseUUID(c.Param("task_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid task ID")
	}

	var activities []ActivityViewData

	if h.eventService != nil {
		events, eventsErr := h.eventService.GetEvents(c.Request().Context(), taskID)
		if eventsErr == nil {
			activities = h.convertEventsToActivities(events)
		}
	}

	data := map[string]any{
		"Activities": activities,
	}

	return h.renderPartial(c, "task/activity", data)
}

// TaskEditTitleForm returns the inline title edit form.
func (h *TaskDetailTemplateHandler) TaskEditTitleForm(c echo.Context) error {
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

	data := map[string]any{
		"Task": h.convertToDetailView(taskModel),
	}

	return h.renderPartial(c, "task/edit-title", data)
}

// TaskTitleDisplay returns the title display view (after canceling edit).
func (h *TaskDetailTemplateHandler) TaskTitleDisplay(c echo.Context) error {
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

	data := map[string]any{
		"Task": h.convertToDetailView(taskModel),
	}

	return h.renderPartial(c, "task/title-display", data)
}

// TaskEditDescriptionForm returns the inline description edit form.
func (h *TaskDetailTemplateHandler) TaskEditDescriptionForm(c echo.Context) error {
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

	data := map[string]any{
		"Task": h.convertToDetailView(taskModel),
	}

	return h.renderPartial(c, "task/edit-description", data)
}

// TaskDescriptionDisplay returns the description display view (after canceling edit).
func (h *TaskDetailTemplateHandler) TaskDescriptionDisplay(c echo.Context) error {
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

	data := map[string]any{
		"Task": h.convertToDetailView(taskModel),
	}

	return h.renderPartial(c, "task/description-display", data)
}

// TaskQuickEditPopover returns the quick edit popover for a task.
func (h *TaskDetailTemplateHandler) TaskQuickEditPopover(c echo.Context) error {
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

	// Get workspace members for assignee dropdown
	var participants []MemberViewData

	data := map[string]any{
		"Task":         h.convertToDetailView(taskModel),
		"Statuses":     getStatusOptions(),
		"Priorities":   getPriorityOptions(),
		"Participants": participants,
	}

	return h.renderPartial(c, "task/quick-edit", data)
}

// convertToDetailView converts a task read model to detail view data.
func (h *TaskDetailTemplateHandler) convertToDetailView(t *taskapp.ReadModel) TaskDetailViewData {
	view := TaskDetailViewData{
		ID:        t.ID.String(),
		ChatID:    t.ChatID.String(),
		Title:     t.Title,
		Type:      string(t.EntityType),
		Status:    string(t.Status),
		Priority:  string(t.Priority),
		DueDate:   t.DueDate,
		CreatedAt: t.CreatedAt,
	}

	if t.AssignedTo != nil {
		view.AssigneeID = t.AssignedTo.String()
	}

	// Calculate overdue and due soon status
	if t.DueDate != nil && t.Status != task.StatusDone {
		now := time.Now()
		dueDate := *t.DueDate

		if dueDate.Before(now) {
			view.IsOverdue = true
			view.OverdueDays = int(now.Sub(dueDate).Hours() / hoursPerDay)
		} else {
			daysUntil := int(dueDate.Sub(now).Hours() / hoursPerDay)
			view.DaysUntilDue = daysUntil
			if daysUntil <= dueSoonDays {
				view.IsDueSoon = true
			}
		}
	}

	return view
}

// convertEventsToActivities converts domain events to activity view data.
func (h *TaskDetailTemplateHandler) convertEventsToActivities(events []event.DomainEvent) []ActivityViewData {
	activities := make([]ActivityViewData, 0, len(events))

	for _, e := range events {
		activity := h.convertEventToActivity(e)
		if activity != nil {
			activities = append(activities, *activity)
		}
	}

	// Return in reverse chronological order (newest first)
	for i, j := 0, len(activities)-1; i < j; i, j = i+1, j-1 {
		activities[i], activities[j] = activities[j], activities[i]
	}

	// Limit to default activity limit
	if len(activities) > defaultActivityLimit {
		activities = activities[:defaultActivityLimit]
	}

	return activities
}

// convertEventToActivity converts a single domain event to activity view data.
func (h *TaskDetailTemplateHandler) convertEventToActivity(e event.DomainEvent) *ActivityViewData {
	metadata := e.Metadata()

	activity := &ActivityViewData{
		Actor: ActivityActorData{
			ID:       metadata.UserID,
			Username: "user", // TODO: Load from user service
		},
		CreatedAt: e.OccurredAt(),
	}

	switch e.EventType() {
	case "task.created":
		activity.ActionText = "created this task"
	case "task.status_changed":
		activity.ActionText = "changed status"
		activity.Details = true
		// TODO: Extract old/new values from event data
	case "task.priority_changed":
		activity.ActionText = "changed priority"
		activity.Details = true
	case "task.assigned":
		activity.ActionText = "assigned this task"
		activity.Details = true
	case "task.unassigned":
		activity.ActionText = "removed assignee"
	case "task.due_date_set":
		activity.ActionText = "set due date"
		activity.Details = true
	case "task.due_date_cleared":
		activity.ActionText = "cleared due date"
	case "task.title_updated":
		activity.ActionText = "updated title"
	case "task.description_updated":
		activity.ActionText = "updated description"
	default:
		// Skip unknown event types
		return nil
	}

	return activity
}

// getStatusOptions returns the available status options for dropdowns.
func getStatusOptions() []SelectOption {
	return []SelectOption{
		{Value: string(task.StatusToDo), Label: "To Do"},
		{Value: string(task.StatusInProgress), Label: "In Progress"},
		{Value: string(task.StatusInReview), Label: "In Review"},
		{Value: string(task.StatusDone), Label: "Done"},
	}
}

// getPriorityOptions returns the available priority options for dropdowns.
func getPriorityOptions() []SelectOption {
	return []SelectOption{
		{Value: string(task.PriorityLow), Label: "Low"},
		{Value: string(task.PriorityMedium), Label: "Medium"},
		{Value: string(task.PriorityHigh), Label: "High"},
		{Value: string(task.PriorityCritical), Label: "Critical"},
	}
}

// getUserView extracts user information from the context for templates.
func (h *TaskDetailTemplateHandler) getUserView(c echo.Context) *UserView {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return nil
	}

	return &UserView{
		ID: userID.String(),
	}
}

// renderPartial renders a template without the base layout.
func (h *TaskDetailTemplateHandler) renderPartial(c echo.Context, templateName string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, data, c)
}
