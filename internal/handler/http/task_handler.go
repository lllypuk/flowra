package httphandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Validation constants for task handler.
const (
	maxTaskTitleLength       = 200
	maxTaskDescriptionLength = 5000
	defaultTaskListLimit     = 20
	maxTaskListLimit         = 100
)

// Task handler errors.
var (
	ErrTaskNotFound            = errors.New("task not found")
	ErrTaskTitleRequired       = errors.New("task title is required")
	ErrTaskTitleTooLong        = errors.New("task title is too long")
	ErrTaskDescriptionTooLong  = errors.New("task description is too long")
	ErrInvalidTaskPriority     = errors.New("invalid task priority")
	ErrInvalidTaskStatus       = errors.New("invalid task status")
	ErrInvalidTaskDueDate      = errors.New("invalid task due date")
	ErrNotAuthorizedForTask    = errors.New("not authorized for this task operation")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

// CreateTaskRequest represents the request to create a task.
type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
	ChatID      *string `json:"chat_id"`
	EntityType  string  `json:"entity_type"`
}

// ChangeStatusRequest represents the request to change task status.
type ChangeStatusRequest struct {
	Status string `json:"status"`
}

// AssignTaskRequest represents the request to assign a task.
type AssignTaskRequest struct {
	AssigneeID *string `json:"assignee_id"`
}

// ChangePriorityRequest represents the request to change task priority.
type ChangePriorityRequest struct {
	Priority string `json:"priority"`
}

// SetDueDateRequest represents the request to set task due date.
type SetDueDateRequest struct {
	DueDate *string `json:"due_date"`
}

// TaskResponse represents a task in API responses.
type TaskResponse struct {
	ID          string  `json:"id"`
	ChatID      string  `json:"chat_id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	EntityType  string  `json:"entity_type"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	ReporterID  string  `json:"reporter_id"`
	DueDate     *string `json:"due_date,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
	Version     int     `json:"version"`
}

// TaskListResponse represents a list of tasks in API responses.
type TaskListResponse struct {
	Tasks   []TaskResponse `json:"tasks"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

// TaskService defines the interface for task operations.
// Declared on the consumer side per project guidelines.
type TaskService interface {
	// CreateTask creates a new task.
	CreateTask(ctx context.Context, cmd taskapp.CreateTaskCommand) (taskapp.TaskResult, error)

	// ChangeStatus changes the task status.
	ChangeStatus(ctx context.Context, cmd taskapp.ChangeStatusCommand) (taskapp.TaskResult, error)

	// AssignTask assigns a task to a user.
	AssignTask(ctx context.Context, cmd taskapp.AssignTaskCommand) (taskapp.TaskResult, error)

	// ChangePriority changes the task priority.
	ChangePriority(ctx context.Context, cmd taskapp.ChangePriorityCommand) (taskapp.TaskResult, error)

	// SetDueDate sets or removes the task due date.
	SetDueDate(ctx context.Context, cmd taskapp.SetDueDateCommand) (taskapp.TaskResult, error)

	// GetTask gets a task by ID.
	GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error)

	// ListTasks lists tasks with filters.
	ListTasks(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error)

	// CountTasks counts tasks with filters.
	CountTasks(ctx context.Context, filters taskapp.Filters) (int, error)

	// DeleteTask deletes a task.
	DeleteTask(ctx context.Context, taskID uuid.UUID, deletedBy uuid.UUID) error
}

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	taskService TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

// RegisterRoutes registers task routes with the router.
func (h *TaskHandler) RegisterRoutes(r *httpserver.Router) {
	// Task creation (workspace-scoped)
	r.Workspace().POST("/tasks", h.Create)
	r.Workspace().GET("/tasks", h.List)

	// Task operations (authenticated routes with task ID)
	r.Auth().GET("/tasks/:id", h.Get)
	r.Auth().PUT("/tasks/:id/status", h.ChangeStatus)
	r.Auth().PUT("/tasks/:id/assign", h.Assign)
	r.Auth().PUT("/tasks/:id/priority", h.ChangePriority)
	r.Auth().PUT("/tasks/:id/due-date", h.SetDueDate)
	r.Auth().DELETE("/tasks/:id", h.Delete)
}

// Create handles POST /api/v1/workspaces/:workspace_id/tasks.
// Creates a new task.
func (h *TaskHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	var req CreateTaskRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate request
	if valErr := validateCreateTaskRequest(&req); valErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Error())
	}

	// Parse ChatID - can come from request body or be derived from workspace
	var chatID uuid.UUID
	if req.ChatID != nil && *req.ChatID != "" {
		var parseErr error
		chatID, parseErr = uuid.ParseUUID(*req.ChatID)
		if parseErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
		}
	} else {
		// If no chat ID provided, we need to handle this based on business logic
		// For now, return an error
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "CHAT_ID_REQUIRED", "chat_id is required")
	}

	// Parse optional fields
	var assigneeID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		parsed, parseErr := uuid.ParseUUID(*req.AssigneeID)
		if parseErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_ASSIGNEE_ID", "invalid assignee ID format")
		}
		assigneeID = &parsed
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, parseErr := time.Parse("2006-01-02", *req.DueDate)
		if parseErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_DUE_DATE", "invalid due date format, expected YYYY-MM-DD")
		}
		dueDate = &parsed
	}

	// Parse entity type with default
	entityType := parseEntityType(req.EntityType)

	// Parse priority with default
	priority := parsePriority(req.Priority)

	cmd := taskapp.CreateTaskCommand{
		ChatID:     chatID,
		Title:      req.Title,
		EntityType: entityType,
		Priority:   priority,
		AssigneeID: assigneeID,
		DueDate:    dueDate,
		CreatedBy:  userID,
	}

	result, err := h.taskService.CreateTask(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	resp := ToTaskResponseFromResult(result, req.Title, chatID, entityType, priority, assigneeID, dueDate, userID)
	return httpserver.RespondCreated(c, resp)
}

// Get handles GET /api/v1/tasks/:id.
// Gets a task by ID.
func (h *TaskHandler) Get(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	taskModel, err := h.taskService.GetTask(c.Request().Context(), taskID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	resp := ToTaskResponseFromReadModel(taskModel)
	return httpserver.RespondOK(c, resp)
}

// List handles GET /api/v1/workspaces/:workspace_id/tasks.
// Lists tasks with filtering and pagination.
func (h *TaskHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	// Parse filters
	filters := parseTaskFilters(c)

	// Get tasks
	tasks, err := h.taskService.ListTasks(c.Request().Context(), filters)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Count total for pagination
	total, countErr := h.taskService.CountTasks(c.Request().Context(), filters)
	if countErr != nil {
		return httpserver.RespondError(c, countErr)
	}

	// Build response
	taskResponses := make([]TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		taskResponses = append(taskResponses, ToTaskResponseFromReadModel(t))
	}

	hasMore := filters.Offset+len(tasks) < total

	resp := TaskListResponse{
		Tasks:   taskResponses,
		Total:   total,
		HasMore: hasMore,
	}

	return httpserver.RespondOK(c, resp)
}

// ChangeStatus handles PUT /api/v1/tasks/:id/status.
// Changes the task status.
func (h *TaskHandler) ChangeStatus(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	var req ChangeStatusRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	status, statusErr := parseStatus(req.Status)
	if statusErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_STATUS", statusErr.Error())
	}

	cmd := taskapp.ChangeStatusCommand{
		TaskID:    taskID,
		NewStatus: status,
		ChangedBy: userID,
	}

	result, err := h.taskService.ChangeStatus(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Get updated task
	taskModel, getErr := h.taskService.GetTask(c.Request().Context(), result.TaskID)
	if getErr != nil {
		// Return minimal response if get fails
		return httpserver.RespondOK(c, map[string]any{
			"id":      result.TaskID.String(),
			"version": result.Version,
			"message": "status updated successfully",
		})
	}

	resp := ToTaskResponseFromReadModel(taskModel)
	return httpserver.RespondOK(c, resp)
}

// Assign handles PUT /api/v1/tasks/:id/assign.
// Assigns a task to a user.
func (h *TaskHandler) Assign(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	var req AssignTaskRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var assigneeID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		parsed, assigneeErr := uuid.ParseUUID(*req.AssigneeID)
		if assigneeErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_ASSIGNEE_ID", "invalid assignee ID format")
		}
		assigneeID = &parsed
	}

	cmd := taskapp.AssignTaskCommand{
		TaskID:     taskID,
		AssigneeID: assigneeID,
		AssignedBy: userID,
	}

	result, err := h.taskService.AssignTask(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Get updated task
	taskModel, getErr := h.taskService.GetTask(c.Request().Context(), result.TaskID)
	if getErr != nil {
		return httpserver.RespondOK(c, map[string]any{
			"id":      result.TaskID.String(),
			"version": result.Version,
			"message": "assignee updated successfully",
		})
	}

	resp := ToTaskResponseFromReadModel(taskModel)
	return httpserver.RespondOK(c, resp)
}

// ChangePriority handles PUT /api/v1/tasks/:id/priority.
// Changes the task priority.
func (h *TaskHandler) ChangePriority(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	var req ChangePriorityRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	priority := parsePriorityStrict(req.Priority)
	if priority == "" {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_PRIORITY", "priority must be Low, Medium, High, or Critical")
	}

	cmd := taskapp.ChangePriorityCommand{
		TaskID:    taskID,
		Priority:  priority,
		ChangedBy: userID,
	}

	result, err := h.taskService.ChangePriority(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Get updated task
	taskModel, getErr := h.taskService.GetTask(c.Request().Context(), result.TaskID)
	if getErr != nil {
		return httpserver.RespondOK(c, map[string]any{
			"id":      result.TaskID.String(),
			"version": result.Version,
			"message": "priority updated successfully",
		})
	}

	resp := ToTaskResponseFromReadModel(taskModel)
	return httpserver.RespondOK(c, resp)
}

// SetDueDate handles PUT /api/v1/tasks/:id/due-date.
// Sets or removes the task due date.
func (h *TaskHandler) SetDueDate(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	var req SetDueDateRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, dueDateErr := time.Parse("2006-01-02", *req.DueDate)
		if dueDateErr != nil {
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "INVALID_DUE_DATE", "invalid due date format, expected YYYY-MM-DD")
		}
		dueDate = &parsed
	}

	cmd := taskapp.SetDueDateCommand{
		TaskID:    taskID,
		DueDate:   dueDate,
		ChangedBy: userID,
	}

	result, err := h.taskService.SetDueDate(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	// Get updated task
	taskModel, getErr := h.taskService.GetTask(c.Request().Context(), result.TaskID)
	if getErr != nil {
		return httpserver.RespondOK(c, map[string]any{
			"id":      result.TaskID.String(),
			"version": result.Version,
			"message": "due date updated successfully",
		})
	}

	resp := ToTaskResponseFromReadModel(taskModel)
	return httpserver.RespondOK(c, resp)
}

// Delete handles DELETE /api/v1/tasks/:id.
// Deletes a task.
func (h *TaskHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskIDStr := c.Param("id")
	taskID, parseErr := uuid.ParseUUID(taskIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	err := h.taskService.DeleteTask(c.Request().Context(), taskID, userID)
	if err != nil {
		return httpserver.RespondError(c, err)
	}

	return httpserver.RespondNoContent(c)
}

// Helper functions

func validateCreateTaskRequest(req *CreateTaskRequest) error {
	if req.Title == "" {
		return ErrTaskTitleRequired
	}
	if len(req.Title) > maxTaskTitleLength {
		return ErrTaskTitleTooLong
	}
	if len(req.Description) > maxTaskDescriptionLength {
		return ErrTaskDescriptionTooLong
	}
	return nil
}

func parseEntityType(s string) task.EntityType {
	switch s {
	case "task", "Task":
		return task.TypeTask
	case "bug", "Bug":
		return task.TypeBug
	case "epic", "Epic":
		return task.TypeEpic
	default:
		return task.TypeTask
	}
}

func parsePriority(s string) task.Priority {
	switch s {
	case "low", "Low":
		return task.PriorityLow
	case "medium", "Medium":
		return task.PriorityMedium
	case "high", "High":
		return task.PriorityHigh
	case "critical", "Critical", "urgent", "Urgent":
		return task.PriorityCritical
	default:
		return task.PriorityMedium
	}
}

// parsePriorityStrict parses priority strictly - returns empty string for invalid values.
// Used in ChangePriority where we need to validate the input.
func parsePriorityStrict(s string) task.Priority {
	switch s {
	case "low", "Low":
		return task.PriorityLow
	case "medium", "Medium":
		return task.PriorityMedium
	case "high", "High":
		return task.PriorityHigh
	case "critical", "Critical", "urgent", "Urgent":
		return task.PriorityCritical
	default:
		return ""
	}
}

func parseStatus(s string) (task.Status, error) {
	switch s {
	case "backlog", "Backlog":
		return task.StatusBacklog, nil
	case "open", "todo", "to_do", "To Do", "ToDo":
		return task.StatusToDo, nil
	case "in_progress", "In Progress", "InProgress":
		return task.StatusInProgress, nil
	case "review", "in_review", "In Review", "InReview":
		return task.StatusInReview, nil
	case "done", "Done":
		return task.StatusDone, nil
	case "cancelled", "Cancelled", "canceled", "Canceled":
		return task.StatusCancelled, nil
	default:
		return "", ErrInvalidTaskStatus
	}
}

func parseTaskFilters(c echo.Context) taskapp.Filters {
	filters := taskapp.Filters{
		Limit:  defaultTaskListLimit,
		Offset: 0,
	}

	filters.Status = parseStatusFilter(c.QueryParam("status"))
	filters.AssigneeID = parseUUIDFilter(c.QueryParam("assignee_id"))
	filters.Priority = parsePriorityFilter(c.QueryParam("priority"))
	filters.ChatID = parseUUIDFilter(c.QueryParam("chat_id"))

	limit, offset := parseTaskPagination(c, filters.Limit)
	filters.Limit = limit
	filters.Offset = offset

	return filters
}

func parseStatusFilter(s string) *task.Status {
	if s == "" {
		return nil
	}
	status, err := parseStatus(s)
	if err != nil {
		return nil
	}
	return &status
}

func parseUUIDFilter(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.ParseUUID(s)
	if err != nil {
		return nil
	}
	return &id
}

func parsePriorityFilter(s string) *task.Priority {
	if s == "" {
		return nil
	}
	priority := parsePriority(s)
	return &priority
}

func parseTaskPagination(c echo.Context, defaultLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if perPageStr := c.QueryParam("per_page"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil && perPage > 0 {
			limit = min(perPage, maxTaskListLimit)
		}
	}

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			offset = (page - 1) * limit
		}
	}

	return limit, offset
}

// ToTaskResponseFromReadModel converts a ReadModel to TaskResponse.
func ToTaskResponseFromReadModel(rm *taskapp.ReadModel) TaskResponse {
	resp := TaskResponse{
		ID:         rm.ID.String(),
		ChatID:     rm.ChatID.String(),
		Title:      rm.Title,
		Status:     string(rm.Status),
		Priority:   string(rm.Priority),
		EntityType: string(rm.EntityType),
		ReporterID: rm.CreatedBy.String(),
		CreatedAt:  rm.CreatedAt.Format(time.RFC3339),
		Version:    rm.Version,
	}

	if rm.AssignedTo != nil {
		assigneeStr := rm.AssignedTo.String()
		resp.AssigneeID = &assigneeStr
	}

	if rm.DueDate != nil {
		dueDateStr := rm.DueDate.Format("2006-01-02")
		resp.DueDate = &dueDateStr
	}

	return resp
}

// ToTaskResponseFromResult creates a TaskResponse from a TaskResult after creation.
func ToTaskResponseFromResult(
	result taskapp.TaskResult,
	title string,
	chatID uuid.UUID,
	entityType task.EntityType,
	priority task.Priority,
	assigneeID *uuid.UUID,
	dueDate *time.Time,
	createdBy uuid.UUID,
) TaskResponse {
	resp := TaskResponse{
		ID:         result.TaskID.String(),
		ChatID:     chatID.String(),
		Title:      title,
		Status:     string(task.StatusToDo),
		Priority:   string(priority),
		EntityType: string(entityType),
		ReporterID: createdBy.String(),
		CreatedAt:  time.Now().Format(time.RFC3339),
		Version:    result.Version,
	}

	if assigneeID != nil {
		assigneeStr := assigneeID.String()
		resp.AssigneeID = &assigneeStr
	}

	if dueDate != nil {
		dueDateStr := dueDate.Format("2006-01-02")
		resp.DueDate = &dueDateStr
	}

	return resp
}

// MockTaskService is a mock implementation of TaskService for testing.
type MockTaskService struct {
	tasks map[uuid.UUID]*taskapp.ReadModel
}

// NewMockTaskService creates a new mock task service.
func NewMockTaskService() *MockTaskService {
	return &MockTaskService{
		tasks: make(map[uuid.UUID]*taskapp.ReadModel),
	}
}

// AddTask adds a task to the mock service.
func (m *MockTaskService) AddTask(t *taskapp.ReadModel) {
	m.tasks[t.ID] = t
}

// CreateTask creates a task in the mock service.
func (m *MockTaskService) CreateTask(
	_ context.Context,
	cmd taskapp.CreateTaskCommand,
) (taskapp.TaskResult, error) {
	taskID := uuid.NewUUID()

	rm := &taskapp.ReadModel{
		ID:         taskID,
		ChatID:     cmd.ChatID,
		Title:      cmd.Title,
		EntityType: cmd.EntityType,
		Status:     task.StatusToDo,
		Priority:   cmd.Priority,
		AssignedTo: cmd.AssigneeID,
		DueDate:    cmd.DueDate,
		CreatedBy:  cmd.CreatedBy,
		CreatedAt:  time.Now(),
		Version:    1,
	}

	m.tasks[taskID] = rm

	return taskapp.NewSuccessResult(taskID, 1, nil), nil
}

// GetTask gets a task from the mock service.
func (m *MockTaskService) GetTask(_ context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, taskapp.ErrTaskNotFound
	}
	return t, nil
}

// ListTasks lists tasks from the mock service.
func (m *MockTaskService) ListTasks(_ context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error) {
	result := make([]*taskapp.ReadModel, 0)

	for _, t := range m.tasks {
		// Apply filters
		if filters.Status != nil && t.Status != *filters.Status {
			continue
		}
		if filters.AssigneeID != nil && (t.AssignedTo == nil || *t.AssignedTo != *filters.AssigneeID) {
			continue
		}
		if filters.Priority != nil && t.Priority != *filters.Priority {
			continue
		}
		if filters.ChatID != nil && t.ChatID != *filters.ChatID {
			continue
		}

		result = append(result, t)
	}

	// Apply pagination
	start := filters.Offset
	if start > len(result) {
		start = len(result)
	}
	end := start + filters.Limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// CountTasks counts tasks in the mock service.
func (m *MockTaskService) CountTasks(_ context.Context, filters taskapp.Filters) (int, error) {
	count := 0

	for _, t := range m.tasks {
		// Apply filters
		if filters.Status != nil && t.Status != *filters.Status {
			continue
		}
		if filters.AssigneeID != nil && (t.AssignedTo == nil || *t.AssignedTo != *filters.AssigneeID) {
			continue
		}
		if filters.Priority != nil && t.Priority != *filters.Priority {
			continue
		}
		if filters.ChatID != nil && t.ChatID != *filters.ChatID {
			continue
		}

		count++
	}

	return count, nil
}

// ChangeStatus changes task status in the mock service.
func (m *MockTaskService) ChangeStatus(
	_ context.Context,
	cmd taskapp.ChangeStatusCommand,
) (taskapp.TaskResult, error) {
	t, ok := m.tasks[cmd.TaskID]
	if !ok {
		return taskapp.TaskResult{}, taskapp.ErrTaskNotFound
	}

	t.Status = cmd.NewStatus
	t.Version++

	return taskapp.NewSuccessResult(cmd.TaskID, t.Version, nil), nil
}

// AssignTask assigns a task in the mock service.
func (m *MockTaskService) AssignTask(
	_ context.Context,
	cmd taskapp.AssignTaskCommand,
) (taskapp.TaskResult, error) {
	t, ok := m.tasks[cmd.TaskID]
	if !ok {
		return taskapp.TaskResult{}, taskapp.ErrTaskNotFound
	}

	t.AssignedTo = cmd.AssigneeID
	t.Version++

	return taskapp.NewSuccessResult(cmd.TaskID, t.Version, nil), nil
}

// ChangePriority changes task priority in the mock service.
func (m *MockTaskService) ChangePriority(
	_ context.Context,
	cmd taskapp.ChangePriorityCommand,
) (taskapp.TaskResult, error) {
	t, ok := m.tasks[cmd.TaskID]
	if !ok {
		return taskapp.TaskResult{}, taskapp.ErrTaskNotFound
	}

	t.Priority = cmd.Priority
	t.Version++

	return taskapp.NewSuccessResult(cmd.TaskID, t.Version, nil), nil
}

// SetDueDate sets task due date in the mock service.
func (m *MockTaskService) SetDueDate(
	_ context.Context,
	cmd taskapp.SetDueDateCommand,
) (taskapp.TaskResult, error) {
	t, ok := m.tasks[cmd.TaskID]
	if !ok {
		return taskapp.TaskResult{}, taskapp.ErrTaskNotFound
	}

	t.DueDate = cmd.DueDate
	t.Version++

	return taskapp.NewSuccessResult(cmd.TaskID, t.Version, nil), nil
}

// DeleteTask deletes a task from the mock service.
func (m *MockTaskService) DeleteTask(_ context.Context, taskID uuid.UUID, _ uuid.UUID) error {
	if _, ok := m.tasks[taskID]; !ok {
		return taskapp.ErrTaskNotFound
	}

	delete(m.tasks, taskID)
	return nil
}
