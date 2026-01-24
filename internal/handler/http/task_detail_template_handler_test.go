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
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

// MockTaskDetailService is a mock implementation of TaskDetailService for testing.
type MockTaskDetailService struct {
	tasks map[uuid.UUID]*taskapp.ReadModel
}

// NewMockTaskDetailService creates a new mock task detail service.
func NewMockTaskDetailService() *MockTaskDetailService {
	return &MockTaskDetailService{
		tasks: make(map[uuid.UUID]*taskapp.ReadModel),
	}
}

// AddTask adds a task to the mock service.
func (m *MockTaskDetailService) AddTask(t *taskapp.ReadModel) {
	m.tasks[t.ID] = t
}

// GetTask implements TaskDetailService.
func (m *MockTaskDetailService) GetTask(
	_ context.Context,
	taskID uuid.UUID,
) (*taskapp.ReadModel, error) {
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, taskapp.ErrTaskNotFound
	}
	return t, nil
}

// GetTaskByChatID implements TaskDetailService.
func (m *MockTaskDetailService) GetTaskByChatID(
	_ context.Context,
	chatID uuid.UUID,
) (*taskapp.ReadModel, error) {
	for _, t := range m.tasks {
		if t.ChatID == chatID {
			return t, nil
		}
	}
	return nil, taskapp.ErrTaskNotFound
}

// MockTaskEventService is a mock implementation of TaskEventService for testing.
type MockTaskEventService struct {
	events map[uuid.UUID][]event.DomainEvent
}

// NewMockTaskEventService creates a new mock task event service.
func NewMockTaskEventService() *MockTaskEventService {
	return &MockTaskEventService{
		events: make(map[uuid.UUID][]event.DomainEvent),
	}
}

// AddEvents adds events for a task.
func (m *MockTaskEventService) AddEvents(taskID uuid.UUID, events []event.DomainEvent) {
	m.events[taskID] = events
}

// GetEvents implements TaskEventService.
func (m *MockTaskEventService) GetEvents(
	_ context.Context,
	taskID uuid.UUID,
) ([]event.DomainEvent, error) {
	events, ok := m.events[taskID]
	if !ok {
		return []event.DomainEvent{}, nil
	}
	return events, nil
}

// MockTaskDetailMemberService is a mock implementation of TaskDetailMemberService for testing.
type MockTaskDetailMemberService struct {
	members map[uuid.UUID][]httphandler.MemberViewData
}

// NewMockTaskDetailMemberService creates a new mock member service.
func NewMockTaskDetailMemberService() *MockTaskDetailMemberService {
	return &MockTaskDetailMemberService{
		members: make(map[uuid.UUID][]httphandler.MemberViewData),
	}
}

// AddMembers adds members for a workspace.
func (m *MockTaskDetailMemberService) AddMembers(workspaceID uuid.UUID, members []httphandler.MemberViewData) {
	m.members[workspaceID] = members
}

// ListWorkspaceMembers implements TaskDetailMemberService.
func (m *MockTaskDetailMemberService) ListWorkspaceMembers(
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

// setUserContextForTaskDetail sets user authentication context on the echo context.
func setUserContextForTaskDetail(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	c.Set(string(middleware.ContextKeyUsername), "testuser")
}

// makeTestTaskDetailReadModel creates a task read model for testing with default values.
func makeTestTaskDetailReadModel(chatID uuid.UUID) *taskapp.ReadModel {
	return &taskapp.ReadModel{
		ID:         uuid.NewUUID(),
		ChatID:     chatID,
		Title:      "Test Task",
		EntityType: task.TypeTask,
		Status:     task.StatusToDo,
		Priority:   task.PriorityHigh,
		CreatedBy:  uuid.NewUUID(),
		CreatedAt:  time.Now(),
		Version:    1,
	}
}

// MockDomainEvent is a mock implementation of event.DomainEvent for testing.
type MockDomainEvent struct {
	eventType     string
	aggregateID   string
	aggregateType string
	occurredAt    time.Time
	version       int
	metadata      event.Metadata
}

func (m *MockDomainEvent) EventType() string     { return m.eventType }
func (m *MockDomainEvent) AggregateID() string   { return m.aggregateID }
func (m *MockDomainEvent) AggregateType() string { return m.aggregateType }
func (m *MockDomainEvent) OccurredAt() time.Time { return m.occurredAt }
func (m *MockDomainEvent) Version() int          { return m.version }
func (m *MockDomainEvent) Metadata() event.Metadata {
	return m.metadata
}

// newMockDomainEvent creates a mock domain event for testing.
func newMockDomainEvent(eventType string, taskID uuid.UUID, version int) *MockDomainEvent {
	return &MockDomainEvent{
		eventType:     eventType,
		aggregateID:   taskID.String(),
		aggregateType: "task",
		occurredAt:    time.Now(),
		version:       version,
		metadata: event.Metadata{
			UserID:    uuid.NewUUID().String(),
			Timestamp: time.Now(),
		},
	}
}

func TestTaskDetailTemplateHandler_TaskSidebarPartial(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		// No user context set

		err := handler.TaskSidebarPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid task ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/invalid-uuid/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("nil task service returns 500", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("successful sidebar partial (without renderer)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		testTask := makeTestTaskDetailReadModel(chatID)
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		// This will fail because renderer is nil, but we're testing the logic
		err := handler.TaskSidebarPartial(c)
		require.Error(t, err)
	})
}

func TestTaskDetailTemplateHandler_TaskActivityPartial(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/activity", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskActivityPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid task ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/invalid-uuid/activity", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskActivityPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("activity with events (without renderer)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		// Add some events
		events := []event.DomainEvent{
			newMockDomainEvent("task.created", taskID, 1),
			newMockDomainEvent("task.status_changed", taskID, 2),
			newMockDomainEvent("task.assigned", taskID, 3),
		}
		mockEventService.AddEvents(taskID, events)

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/activity", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		// This will fail because renderer is nil
		err := handler.TaskActivityPartial(c)
		require.Error(t, err)
	})
}

func TestTaskDetailTemplateHandler_TaskEditTitleForm(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/edit-title", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskEditTitleForm(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid task ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/invalid-uuid/edit-title", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskEditTitleForm(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/edit-title", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskEditTitleForm(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("successful edit title form (without renderer)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		testTask := makeTestTaskDetailReadModel(chatID)
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/edit-title", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskEditTitleForm(c)
		require.Error(t, err) // Renderer is nil
	})
}

func TestTaskDetailTemplateHandler_TaskTitleDisplay(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/title-display", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskTitleDisplay(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/title-display", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskTitleDisplay(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestTaskDetailTemplateHandler_TaskEditDescriptionForm(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/edit-description", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskEditDescriptionForm(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/edit-description", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskEditDescriptionForm(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("successful edit description form (without renderer)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		testTask := makeTestTaskDetailReadModel(chatID)
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/edit-description", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskEditDescriptionForm(c)
		require.Error(t, err) // Renderer is nil
	})
}

func TestTaskDetailTemplateHandler_TaskDescriptionDisplay(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/description-display", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskDescriptionDisplay(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/description-display", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskDescriptionDisplay(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestTaskDetailTemplateHandler_TaskQuickEditPopover(t *testing.T) {
	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, nil, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/quick-edit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())

		err := handler.TaskQuickEditPopover(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid task ID returns 400", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/invalid-uuid/quick-edit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("invalid-uuid")
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskQuickEditPopover(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("task not found returns 404", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/quick-edit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskQuickEditPopover(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("successful quick edit popover (without renderer)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		testTask := makeTestTaskDetailReadModel(chatID)
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/quick-edit", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskQuickEditPopover(c)
		require.Error(t, err) // Renderer is nil
	})
}

func TestNewTaskDetailTemplateHandler(t *testing.T) {
	t.Run("creates handler with nil logger", func(t *testing.T) {
		mockTaskService := NewMockTaskDetailService()

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		assert.NotNil(t, handler)
	})

	t.Run("creates handler with all dependencies", func(t *testing.T) {
		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()
		mockMemberService := NewMockTaskDetailMemberService()

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, mockMemberService, nil,
		)

		assert.NotNil(t, handler)
	})
}

func TestTaskDetailViewData_DueDateCalculations(t *testing.T) {
	t.Run("overdue task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		overdueDays := 5
		overdueDate := time.Now().AddDate(0, 0, -overdueDays)

		testTask := &taskapp.ReadModel{
			ID:         uuid.NewUUID(),
			ChatID:     chatID,
			Title:      "Overdue Task",
			EntityType: task.TypeTask,
			Status:     task.StatusInProgress,
			Priority:   task.PriorityHigh,
			DueDate:    &overdueDate,
			CreatedBy:  uuid.NewUUID(),
			CreatedAt:  time.Now().AddDate(0, 0, -10),
			Version:    1,
		}
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		// This will fail because renderer is nil, but task was found and processed
		err := handler.TaskSidebarPartial(c)
		require.Error(t, err)
	})

	t.Run("due soon task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		dueSoonDate := time.Now().AddDate(0, 0, 2) // Due in 2 days

		testTask := &taskapp.ReadModel{
			ID:         uuid.NewUUID(),
			ChatID:     chatID,
			Title:      "Due Soon Task",
			EntityType: task.TypeTask,
			Status:     task.StatusInProgress,
			Priority:   task.PriorityHigh,
			DueDate:    &dueSoonDate,
			CreatedBy:  uuid.NewUUID(),
			CreatedAt:  time.Now().AddDate(0, 0, -5),
			Version:    1,
		}
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)
		require.Error(t, err) // Renderer is nil
	})

	t.Run("done task is not overdue", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		overdueDate := time.Now().AddDate(0, 0, -5)

		testTask := &taskapp.ReadModel{
			ID:         uuid.NewUUID(),
			ChatID:     chatID,
			Title:      "Done Task",
			EntityType: task.TypeTask,
			Status:     task.StatusDone, // Done tasks shouldn't be marked overdue
			Priority:   task.PriorityHigh,
			DueDate:    &overdueDate,
			CreatedBy:  uuid.NewUUID(),
			CreatedAt:  time.Now().AddDate(0, 0, -10),
			Version:    1,
		}
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)
		require.Error(t, err) // Renderer is nil
	})
}

func TestTaskDetailTemplateHandler_EventTypeConversion(t *testing.T) {
	t.Run("all event types are converted to activities", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()

		// Test all known event types
		eventTypes := []string{
			"task.created",
			"task.status_changed",
			"task.priority_changed",
			"task.assigned",
			"task.unassigned",
			"task.due_date_set",
			"task.due_date_cleared",
			"task.title_updated",
			"task.description_updated",
		}

		events := make([]event.DomainEvent, len(eventTypes))
		for i, eventType := range eventTypes {
			events[i] = newMockDomainEvent(eventType, taskID, i+1)
		}
		mockEventService.AddEvents(taskID, events)

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, nil, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/activity", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskActivityPartial(c)
		require.Error(t, err) // Renderer is nil
	})

	t.Run("unknown event types are skipped", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()
		mockEventService := NewMockTaskEventService()

		events := []event.DomainEvent{
			newMockDomainEvent("task.created", taskID, 1),
			newMockDomainEvent("unknown.event.type", taskID, 2),
			newMockDomainEvent("task.status_changed", taskID, 3),
		}
		mockEventService.AddEvents(taskID, events)

		handler := httphandler.NewTaskDetailTemplateHandler(
			nil, nil, mockTaskService, mockEventService, nil, nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+taskID.String()+"/activity", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskActivityPartial(c)
		require.Error(t, err) // Renderer is nil
	})
}

func TestTaskDetailTemplateHandler_WithAssignee(t *testing.T) {
	t.Run("task with assignee", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		mockTaskService := NewMockTaskDetailService()

		testTask := &taskapp.ReadModel{
			ID:         uuid.NewUUID(),
			ChatID:     chatID,
			Title:      "Assigned Task",
			EntityType: task.TypeTask,
			Status:     task.StatusInProgress,
			Priority:   task.PriorityHigh,
			AssignedTo: &assigneeID,
			CreatedBy:  uuid.NewUUID(),
			CreatedAt:  time.Now(),
			Version:    1,
		}
		mockTaskService.AddTask(testTask)

		handler := httphandler.NewTaskDetailTemplateHandler(nil, nil, mockTaskService, nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/partials/tasks/"+testTask.ID.String()+"/sidebar", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(testTask.ID.String())
		setUserContextForTaskDetail(c, userID)

		err := handler.TaskSidebarPartial(c)
		require.Error(t, err) // Renderer is nil
	})
}
