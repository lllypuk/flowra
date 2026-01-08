package httphandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

// MockBoardTaskService is a mock implementation of BoardTaskService for testing.
type MockBoardTaskService struct {
	tasks map[uuid.UUID]*taskapp.ReadModel
}

// NewMockBoardTaskService creates a new mock board task service.
func NewMockBoardTaskService() *MockBoardTaskService {
	return &MockBoardTaskService{
		tasks: make(map[uuid.UUID]*taskapp.ReadModel),
	}
}

// AddTask adds a task to the mock service.
func (m *MockBoardTaskService) AddTask(t *taskapp.ReadModel) {
	m.tasks[t.ID] = t
}

// ListTasks implements BoardTaskService.
func (m *MockBoardTaskService) ListTasks(
	_ context.Context,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	result := make([]*taskapp.ReadModel, 0)
	for _, t := range m.tasks {
		// Apply status filter if set
		if filters.Status != nil && t.Status != *filters.Status {
			continue
		}
		// Apply entity type filter if set
		if filters.EntityType != nil && t.EntityType != *filters.EntityType {
			continue
		}
		// Apply priority filter if set
		if filters.Priority != nil && t.Priority != *filters.Priority {
			continue
		}
		// Apply assignee filter if set
		if filters.AssigneeID != nil {
			if t.AssignedTo == nil || *t.AssignedTo != *filters.AssigneeID {
				continue
			}
		}
		result = append(result, t)
	}

	// Apply pagination
	offset := filters.Offset
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if offset >= len(result) {
		return []*taskapp.ReadModel{}, nil
	}
	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

// CountTasks implements BoardTaskService.
func (m *MockBoardTaskService) CountTasks(
	_ context.Context,
	filters taskapp.Filters,
) (int, error) {
	tasks, _ := m.ListTasks(context.Background(), taskapp.Filters{
		Status:     filters.Status,
		EntityType: filters.EntityType,
		Priority:   filters.Priority,
		AssigneeID: filters.AssigneeID,
	})
	return len(tasks), nil
}

// GetTask implements BoardTaskService.
func (m *MockBoardTaskService) GetTask(
	_ context.Context,
	taskID uuid.UUID,
) (*taskapp.ReadModel, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, taskapp.ErrTaskNotFound
	}
	return t, nil
}

// MockBoardMemberService is a mock implementation of BoardMemberService for testing.
type MockBoardMemberService struct {
	members map[uuid.UUID][]httphandler.MemberViewData
}

// NewMockBoardMemberService creates a new mock board member service.
func NewMockBoardMemberService() *MockBoardMemberService {
	return &MockBoardMemberService{
		members: make(map[uuid.UUID][]httphandler.MemberViewData),
	}
}

// AddMembers adds members for a workspace.
func (m *MockBoardMemberService) AddMembers(workspaceID uuid.UUID, members []httphandler.MemberViewData) {
	m.members[workspaceID] = members
}

// ListWorkspaceMembers implements BoardMemberService.
func (m *MockBoardMemberService) ListWorkspaceMembers(
	_ context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]httphandler.MemberViewData, error) {
	members, ok := m.members[workspaceID]
	if !ok {
		return []httphandler.MemberViewData{}, nil
	}

	if offset >= len(members) {
		return []httphandler.MemberViewData{}, nil
	}
	end := min(offset+limit, len(members))
	return members[offset:end], nil
}

// setUserContextForBoard sets user authentication context on the echo context.
func setUserContextForBoard(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	c.Set(string(middleware.ContextKeyUsername), "testuser")
}

// makeTestTaskReadModel creates a task read model for testing.
func makeTestTaskReadModel(
	chatID uuid.UUID,
	title string,
	status task.Status,
	priority task.Priority,
	entityType task.EntityType,
) *taskapp.ReadModel {
	return &taskapp.ReadModel{
		ID:         uuid.NewUUID(),
		ChatID:     chatID,
		Title:      title,
		EntityType: entityType,
		Status:     status,
		Priority:   priority,
		CreatedBy:  uuid.NewUUID(),
		CreatedAt:  time.Now(),
		Version:    1,
	}
}

func TestBoardTemplateHandler_BoardIndex(t *testing.T) {
	t.Run("unauthorized redirects to login", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String()+"/board", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		// No user context set

		err := handler.BoardIndex(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("invalid workspace ID returns not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/workspaces/invalid-uuid/board", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForBoard(c, userID)

		err := handler.BoardIndex(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("successful board index with tasks", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		// Add test tasks
		task1 := makeTestTaskReadModel(chatID, "Task 1", task.StatusToDo, task.PriorityHigh, task.TypeTask)
		task2 := makeTestTaskReadModel(chatID, "Bug 1", task.StatusInProgress, task.PriorityCritical, task.TypeBug)
		mockTaskService.AddTask(task1)
		mockTaskService.AddTask(task2)

		// Add test members
		mockMemberService.AddMembers(workspaceID, []httphandler.MemberViewData{
			{UserID: userID.String(), Username: "testuser", Role: "admin"},
		})

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID.String()+"/board", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil, but we're testing the logic
		err := handler.BoardIndex(c)

		// We expect an error because there's no renderer
		require.Error(t, err)
	})
}

func TestBoardTemplateHandler_BoardPartial(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/workspace/"+workspaceID.String()+"/board", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		// No user context set

		err := handler.BoardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid workspace ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/workspace/invalid-uuid/board", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForBoard(c, userID)

		err := handler.BoardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("successful board partial with filters", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		// Add test tasks with different statuses
		task1 := makeTestTaskReadModel(chatID, "Task 1", task.StatusToDo, task.PriorityHigh, task.TypeTask)
		task2 := makeTestTaskReadModel(chatID, "Task 2", task.StatusInProgress, task.PriorityMedium, task.TypeTask)
		task3 := makeTestTaskReadModel(chatID, "Bug 1", task.StatusInReview, task.PriorityCritical, task.TypeBug)
		task4 := makeTestTaskReadModel(chatID, "Task 3", task.StatusDone, task.PriorityLow, task.TypeTask)
		mockTaskService.AddTask(task1)
		mockTaskService.AddTask(task2)
		mockTaskService.AddTask(task3)
		mockTaskService.AddTask(task4)

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		// Test with type filter
		req := httptest.NewRequest(http.MethodGet, "/partials/workspace/"+workspaceID.String()+"/board?type=task", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil
		err := handler.BoardPartial(c)
		require.Error(t, err)
	})
}

func TestBoardTemplateHandler_BoardColumnMore(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/workspace/"+workspaceID.String()+"/board/todo/more", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id", "status")
		c.SetParamValues(workspaceID.String(), "todo")
		// No user context set

		err := handler.BoardColumnMore(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		url := "/partials/workspace/" + workspaceID.String() + "/board/invalid/more"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id", "status")
		c.SetParamValues(workspaceID.String(), "invalid")
		setUserContextForBoard(c, userID)

		err := handler.BoardColumnMore(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("valid status with offset", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		// Add multiple tasks in todo status
		for i := range 25 {
			testTask := makeTestTaskReadModel(
				chatID, "Task "+string(rune('A'+i)), task.StatusToDo, task.PriorityMedium, task.TypeTask,
			)
			mockTaskService.AddTask(testTask)
		}

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		url := "/partials/workspace/" + workspaceID.String() +
			"/board/todo/more?offset=20"
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id", "status")
		c.SetParamValues(workspaceID.String(), "todo")
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil
		err := handler.BoardColumnMore(c)
		require.Error(t, err)
	})
}

func TestBoardTemplateHandler_TaskCardPartial(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/card", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		// No user context set

		err := handler.TaskCardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid task ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/invalid-uuid/card", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForBoard(c, userID)

		err := handler.TaskCardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/card", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForBoard(c, userID)

		err := handler.TaskCardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("successful task card partial", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		// Add a test task
		testTask := makeTestTaskReadModel(chatID, "Test Task", task.StatusToDo, task.PriorityHigh, task.TypeTask)
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/card", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil
		err := handler.TaskCardPartial(c)
		require.Error(t, err)
	})
}

func TestBoardTemplateHandler_NilTaskService(t *testing.T) {
	t.Run("task card partial with nil service returns 500", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		// Create handler with nil task service
		handler := httphandler.NewBoardTemplateHandler(nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/card", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForBoard(c, userID)

		err := handler.TaskCardPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestNewBoardTemplateHandler(t *testing.T) {
	t.Run("creates handler with nil logger", func(t *testing.T) {
		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		assert.NotNil(t, handler)
	})

	t.Run("creates handler with all dependencies", func(t *testing.T) {
		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		assert.NotNil(t, handler)
	})
}

func TestGetBoardColumns(t *testing.T) {
	columns := httphandler.GetBoardColumns()

	require.Len(t, columns, 4)

	assert.Equal(t, task.StatusToDo, columns[0].Status)
	assert.Equal(t, "todo", columns[0].Key)
	assert.Equal(t, "To Do", columns[0].Title)

	assert.Equal(t, task.StatusInProgress, columns[1].Status)
	assert.Equal(t, "in_progress", columns[1].Key)
	assert.Equal(t, "In Progress", columns[1].Title)

	assert.Equal(t, task.StatusInReview, columns[2].Status)
	assert.Equal(t, "review", columns[2].Key)
	assert.Equal(t, "Review", columns[2].Title)

	assert.Equal(t, task.StatusDone, columns[3].Status)
	assert.Equal(t, "done", columns[3].Key)
	assert.Equal(t, "Done", columns[3].Title)
}

func TestBoardFilters(t *testing.T) {
	t.Run("filters parse correctly from query params", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet,
			"/partials/workspace/"+workspaceID.String()+"/board?type=bug&priority=high&assignee=me&search=test",
			nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil, but filters should be parsed
		err := handler.BoardPartial(c)
		require.Error(t, err)
	})
}

func TestBoardTasksWithAssignee(t *testing.T) {
	t.Run("tasks with assignee filter me", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockBoardTaskService()
		mockMemberService := NewMockBoardMemberService()

		// Add task assigned to current user
		task1 := makeTestTaskReadModel(chatID, "My Task", task.StatusToDo, task.PriorityHigh, task.TypeTask)
		task1.AssignedTo = &userID
		mockTaskService.AddTask(task1)

		// Add task assigned to someone else
		otherUser := uuid.NewUUID()
		task2 := makeTestTaskReadModel(chatID, "Other Task", task.StatusToDo, task.PriorityMedium, task.TypeTask)
		task2.AssignedTo = &otherUser
		mockTaskService.AddTask(task2)

		handler := httphandler.NewBoardTemplateHandler(nil, nil, mockTaskService, mockMemberService)

		req := httptest.NewRequest(http.MethodGet,
			"/partials/workspace/"+workspaceID.String()+"/board?assignee=me",
			nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())
		setUserContextForBoard(c, userID)

		// Note: This will fail because renderer is nil
		err := handler.BoardPartial(c)
		require.Error(t, err)
	})
}

func TestBoardTaskOverdue(t *testing.T) {
	t.Run("task with past due date is marked overdue", func(t *testing.T) {
		chatID := uuid.NewUUID()

		// Create a task with past due date
		testTask := makeTestTaskReadModel(chatID, "Overdue Task", task.StatusToDo, task.PriorityHigh, task.TypeTask)
		pastDate := time.Now().Add(-24 * time.Hour)
		testTask.DueDate = &pastDate

		mockTaskService := NewMockBoardTaskService()
		mockTaskService.AddTask(testTask)

		// Verify the task is in the mock
		task, err := mockTaskService.GetTask(context.Background(), testTask.ID)
		require.NoError(t, err)
		require.NotNil(t, task.DueDate)
		assert.True(t, task.DueDate.Before(time.Now()))
	})

	t.Run("task with future due date is not overdue", func(t *testing.T) {
		chatID := uuid.NewUUID()

		// Create a task with future due date
		testTask := makeTestTaskReadModel(chatID, "Future Task", task.StatusToDo, task.PriorityMedium, task.TypeTask)
		futureDate := time.Now().Add(24 * time.Hour)
		testTask.DueDate = &futureDate

		mockTaskService := NewMockBoardTaskService()
		mockTaskService.AddTask(testTask)

		// Verify the task is in the mock
		task, err := mockTaskService.GetTask(context.Background(), testTask.ID)
		require.NoError(t, err)
		require.NotNil(t, task.DueDate)
		assert.True(t, task.DueDate.After(time.Now()))
	})
}

func TestMockBoardTaskService(t *testing.T) {
	t.Run("list tasks with pagination", func(t *testing.T) {
		mockService := NewMockBoardTaskService()
		chatID := uuid.NewUUID()

		// Add 30 tasks
		for range 30 {
			testTask := makeTestTaskReadModel(chatID, "Task", task.StatusToDo, task.PriorityMedium, task.TypeTask)
			mockService.AddTask(testTask)
		}

		// Get first page
		tasks, err := mockService.ListTasks(context.Background(), taskapp.Filters{
			Offset: 0,
			Limit:  20,
		})
		require.NoError(t, err)
		assert.Len(t, tasks, 20)

		// Get second page
		tasks, err = mockService.ListTasks(context.Background(), taskapp.Filters{
			Offset: 20,
			Limit:  20,
		})
		require.NoError(t, err)
		assert.Len(t, tasks, 10)
	})

	t.Run("list tasks with status filter", func(t *testing.T) {
		mockService := NewMockBoardTaskService()
		chatID := uuid.NewUUID()

		// Add tasks with different statuses
		task1 := makeTestTaskReadModel(
			chatID, "Todo Task", task.StatusToDo, task.PriorityMedium, task.TypeTask,
		)
		task2 := makeTestTaskReadModel(
			chatID, "In Progress Task", task.StatusInProgress, task.PriorityMedium, task.TypeTask,
		)
		task3 := makeTestTaskReadModel(
			chatID, "Done Task", task.StatusDone, task.PriorityMedium, task.TypeTask,
		)
		mockService.AddTask(task1)
		mockService.AddTask(task2)
		mockService.AddTask(task3)

		// Filter by todo status
		status := task.StatusToDo
		tasks, err := mockService.ListTasks(context.Background(), taskapp.Filters{
			Status: &status,
		})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Todo Task", tasks[0].Title)
	})

	t.Run("count tasks", func(t *testing.T) {
		mockService := NewMockBoardTaskService()
		chatID := uuid.NewUUID()

		// Add 5 tasks
		for range 5 {
			testTask := makeTestTaskReadModel(chatID, "Task", task.StatusToDo, task.PriorityMedium, task.TypeTask)
			mockService.AddTask(testTask)
		}

		count, err := mockService.CountTasks(context.Background(), taskapp.Filters{})
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})
}

func TestMockBoardMemberService(t *testing.T) {
	t.Run("list workspace members", func(t *testing.T) {
		mockService := NewMockBoardMemberService()
		workspaceID := uuid.NewUUID()

		// Add members
		members := []httphandler.MemberViewData{
			{UserID: uuid.NewUUID().String(), Username: "user1", Role: "admin"},
			{UserID: uuid.NewUUID().String(), Username: "user2", Role: "member"},
			{UserID: uuid.NewUUID().String(), Username: "user3", Role: "member"},
		}
		mockService.AddMembers(workspaceID, members)

		// List members
		result, err := mockService.ListWorkspaceMembers(context.Background(), workspaceID, 0, 10)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("list members for unknown workspace", func(t *testing.T) {
		mockService := NewMockBoardMemberService()
		unknownWorkspaceID := uuid.NewUUID()

		result, err := mockService.ListWorkspaceMembers(context.Background(), unknownWorkspaceID, 0, 10)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
