package httphandler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
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

// UserLookupService resolves user IDs to display names for the activity timeline.
type UserLookupService interface {
	// GetDisplayName returns the display name for a user ID. Returns empty string if not found.
	GetDisplayName(ctx context.Context, userID string) string
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
	IsDueToday   bool
	OverdueDays  int
	DaysUntilDue int
	CreatedAt    time.Time
	Attachments  []TaskAttachmentViewData
}

// TaskAttachmentViewData represents an attachment in the task detail view.
type TaskAttachmentViewData struct {
	FileID   string
	FileName string
	FileSize int64
	MimeType string
	URL      string
	IsImage  bool
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
	userLookup      UserLookupService
}

// NewTaskDetailTemplateHandler creates a new task detail template handler.
func NewTaskDetailTemplateHandler(
	renderer *TemplateRenderer,
	logger *slog.Logger,
	taskService TaskDetailService,
	eventService TaskEventService,
	memberService TaskDetailMemberService,
	chatInfoService ChatBasicInfoService,
	userLookup UserLookupService,
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
		userLookup:      userLookup,
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
	user := getUserView(c)
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
		"Statuses":     getChatStatusOptions(chatInfo.Type),
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
	user := getUserView(c)
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
	user := getUserView(c)
	if user == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	taskID, err := uuid.ParseUUID(c.Param("task_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid task ID")
	}

	// Parse pagination: page parameter (1-based)
	page := 1
	if p := c.QueryParam("page"); p != "" {
		if parsed, parseErr := strconv.Atoi(p); parseErr == nil && parsed > 0 {
			page = parsed
		}
	}

	var activities []ActivityViewData

	activities, hasMore := h.loadPaginatedActivities(c.Request().Context(), taskID, page)

	data := map[string]any{
		"Activities": activities,
		"TaskID":     taskID.String(),
		"HasMore":    hasMore,
		"NextPage":   page + 1,
	}

	// Use pagination-only template for subsequent pages to avoid nested wrappers
	templateName := "task/activity"
	if page > 1 {
		templateName = "task/activity-page"
	}

	return h.renderPartial(c, templateName, data)
}

// TaskEditTitleForm returns the inline title edit form.
func (h *TaskDetailTemplateHandler) TaskEditTitleForm(c echo.Context) error {
	user := getUserView(c)
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
	user := getUserView(c)
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
	user := getUserView(c)
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
	user := getUserView(c)
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
	user := getUserView(c)
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

	for _, a := range t.Attachments {
		view.Attachments = append(view.Attachments, TaskAttachmentViewData{
			FileID:   a.FileID.String(),
			FileName: a.FileName,
			FileSize: a.FileSize,
			MimeType: a.MimeType,
			URL:      fmt.Sprintf("/api/v1/files/%s/%s", a.FileID.String(), url.PathEscape(a.FileName)),
			IsImage:  strings.HasPrefix(a.MimeType, "image/"),
		})
	}

	h.calculateDueStatus(&view, t)

	return view
}

// calculateDueStatus sets overdue/due-soon/due-today flags on the view data.
func (h *TaskDetailTemplateHandler) calculateDueStatus(view *TaskDetailViewData, t *taskapp.ReadModel) {
	if t.DueDate == nil || t.Status == task.StatusDone {
		return
	}

	now := time.Now()
	dueDate := *t.DueDate

	if dueDate.Before(now) {
		view.IsOverdue = true
		view.OverdueDays = int(now.Sub(dueDate).Hours() / hoursPerDay)
		return
	}

	daysUntil := int(dueDate.Sub(now).Hours() / hoursPerDay)
	view.DaysUntilDue = daysUntil

	switch {
	case daysUntil == 0:
		view.IsDueToday = true
	case daysUntil <= dueSoonDays:
		view.IsDueSoon = true
	}
}

// convertEventsToActivities converts domain events to activity view data.
func (h *TaskDetailTemplateHandler) convertEventsToActivities(
	ctx context.Context,
	events []event.DomainEvent,
) []ActivityViewData {
	activities := make([]ActivityViewData, 0, len(events))

	for _, e := range events {
		activity := h.convertEventToActivity(ctx, e)
		if activity != nil {
			activities = append(activities, *activity)
		}
	}

	// Return in reverse chronological order (newest first)
	for i, j := 0, len(activities)-1; i < j; i, j = i+1, j-1 {
		activities[i], activities[j] = activities[j], activities[i]
	}

	return activities
}

// loadPaginatedActivities loads and paginates activity items for a task.
func (h *TaskDetailTemplateHandler) loadPaginatedActivities(
	ctx context.Context,
	taskID uuid.UUID,
	page int,
) ([]ActivityViewData, bool) {
	if h.eventService == nil {
		return nil, false
	}

	events, err := h.eventService.GetEvents(ctx, taskID)
	if err != nil {
		return nil, false
	}

	allActivities := h.convertEventsToActivities(ctx, events)

	start := (page - 1) * defaultActivityLimit
	if start >= len(allActivities) {
		return nil, false
	}

	end := min(start+defaultActivityLimit, len(allActivities))

	return allActivities[start:end], end < len(allActivities)
}

// extractActorID returns the actor user ID from an event, preferring event-specific
// ChangedBy/CreatedBy fields over metadata (which may not have UserID set).
func extractActorID(e event.DomainEvent) string {
	switch te := e.(type) {
	case *task.Created:
		return te.CreatedBy.String()
	case *task.StatusChanged:
		return te.ChangedBy.String()
	case *task.PriorityChanged:
		return te.ChangedBy.String()
	case *task.AssigneeChanged:
		return te.ChangedBy.String()
	case *task.DueDateChanged:
		return te.ChangedBy.String()
	case *task.AttachmentAdded:
		return te.AddedBy.String()
	case *task.AttachmentRemoved:
		return te.RemovedBy.String()
	}
	// Fall back to metadata
	return e.Metadata().UserID
}

// convertEventToActivity converts a single domain event to activity view data.
func (h *TaskDetailTemplateHandler) convertEventToActivity(ctx context.Context, e event.DomainEvent) *ActivityViewData {
	actorID := extractActorID(e)
	username := h.resolveUsername(ctx, actorID)

	activity := &ActivityViewData{
		Actor: ActivityActorData{
			ID:       actorID,
			Username: username,
		},
		CreatedAt: e.OccurredAt(),
	}

	switch te := e.(type) {
	case *task.Created:
		activity.ActionText = "created this task"
	case *task.StatusChanged:
		activity.ActionText = "changed status"
		activity.Details = true
		activity.OldValue = string(te.OldStatus)
		activity.NewValue = string(te.NewStatus)
	case *task.PriorityChanged:
		activity.ActionText = "changed priority"
		activity.Details = true
		activity.OldValue = string(te.OldPriority)
		activity.NewValue = string(te.NewPriority)
	case *task.AssigneeChanged:
		if te.NewAssignee != nil {
			activity.ActionText = "assigned this task"
			activity.Details = true
			activity.NewValue = h.resolveUsername(ctx, te.NewAssignee.String())
			if te.OldAssignee != nil {
				activity.OldValue = h.resolveUsername(ctx, te.OldAssignee.String())
			}
		} else {
			activity.ActionText = "removed assignee"
		}
	case *task.DueDateChanged:
		if te.NewDueDate != nil {
			activity.ActionText = "set due date"
			activity.Details = true
			activity.NewValue = te.NewDueDate.Format("Jan 2, 2006")
			if te.OldDueDate != nil {
				activity.OldValue = te.OldDueDate.Format("Jan 2, 2006")
			}
		} else {
			activity.ActionText = "cleared due date"
		}
	case *task.Updated:
		activity.ActionText = "updated title"
	case *task.Deleted:
		activity.ActionText = "deleted this task"
	case *task.AttachmentAdded:
		activity.ActionText = "added attachment"
		activity.Details = true
		activity.NewValue = te.FileName
	case *task.AttachmentRemoved:
		activity.ActionText = "removed attachment"
	default:
		// Check by event type string for any events not matched by type assertion
		switch e.EventType() {
		case "task.title_updated":
			activity.ActionText = "updated title"
		case "task.description_updated":
			activity.ActionText = "updated description"
		default:
			return nil
		}
	}

	return activity
}

// resolveUsername resolves a user ID to a display name.
func (h *TaskDetailTemplateHandler) resolveUsername(ctx context.Context, userID string) string {
	if h.userLookup != nil && userID != "" {
		if name := h.userLookup.GetDisplayName(ctx, userID); name != "" {
			return name
		}
	}
	if userID != "" && len(userID) > 8 {
		return fmt.Sprintf("user-%s", userID[:8])
	}
	return "Unknown"
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

// getChatStatusOptions returns status options for chat sidebar based on chat type.
func getChatStatusOptions(chatType string) []SelectOption {
	switch strings.ToLower(chatType) {
	case chatTypeBug:
		return []SelectOption{
			{Value: "New", Label: "New"},
			{Value: "Investigating", Label: "Investigating"},
			{Value: "Fixed", Label: "Fixed"},
			{Value: "Verified", Label: "Verified"},
		}
	case chatTypeEpic:
		return []SelectOption{
			{Value: "Planned", Label: "Planned"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Completed", Label: "Completed"},
		}
	case chatTypeTask:
		return []SelectOption{
			{Value: "To Do", Label: "To Do"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Done", Label: "Done"},
		}
	default:
		return []SelectOption{
			{Value: "To Do", Label: "To Do"},
			{Value: "In Progress", Label: "In Progress"},
			{Value: "Done", Label: "Done"},
		}
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

// renderPartial renders a template without the base layout.
func (h *TaskDetailTemplateHandler) renderPartial(c echo.Context, templateName string, data any) error {
	if h.renderer == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
	}

	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderer.Render(c.Response().Writer, templateName, data, c)
}
