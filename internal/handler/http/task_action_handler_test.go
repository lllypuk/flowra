package httphandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

// mockTaskActionTaskService implements TaskActionTaskService for testing.
type mockTaskActionTaskService struct {
	task    *taskapp.ReadModel
	taskErr error
}

func (m *mockTaskActionTaskService) GetTask(
	_ context.Context,
	_ uuid.UUID,
) (*taskapp.ReadModel, error) {
	return m.task, m.taskErr
}

// mockTaskActionService implements TaskActionService for testing.
type mockTaskActionService struct {
	lastChatID   uuid.UUID
	lastStatus   string
	lastPriority string
	lastAssignee *uuid.UUID
	lastDueDate  *time.Time
	actionErr    error
}

func (m *mockTaskActionService) ChangeStatus(
	_ context.Context,
	chatID uuid.UUID,
	newStatus string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	m.lastChatID = chatID
	m.lastStatus = newStatus
	return &appcore.ActionResult{Success: true}, m.actionErr
}

func (m *mockTaskActionService) SetPriority(
	_ context.Context,
	chatID uuid.UUID,
	priority string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	m.lastChatID = chatID
	m.lastPriority = priority
	return &appcore.ActionResult{Success: true}, m.actionErr
}

func (m *mockTaskActionService) AssignUser(
	_ context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	m.lastChatID = chatID
	m.lastAssignee = assigneeID
	return &appcore.ActionResult{Success: true}, m.actionErr
}

func (m *mockTaskActionService) SetDueDate(
	_ context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	m.lastChatID = chatID
	m.lastDueDate = dueDate
	return &appcore.ActionResult{Success: true}, m.actionErr
}

// newTestTaskActionContext creates an Echo context with a user ID and task_id param set.
func newTestTaskActionContext(
	t *testing.T,
	body string,
	taskID uuid.UUID,
) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("task_id")
	c.SetParamValues(taskID.String())

	testUserID := uuid.NewUUID()
	c.Set(string(middleware.ContextKeyUserID), testUserID)

	return c, rec
}

func TestTaskActionHandler_ChangeStatus(t *testing.T) {
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	task := &taskapp.ReadModel{
		ID:     taskID,
		ChatID: chatID,
	}

	t.Run("success", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "status=In+Progress", taskID)

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, chatID, actionSvc.lastChatID)
		assert.Equal(t, "In Progress", actionSvc.lastStatus)
		assert.Equal(t, "taskUpdated", rec.Header().Get("Hx-Trigger"))
	})

	t.Run("missing status", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "", taskID)

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{taskErr: taskapp.ErrTaskNotFound}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "status=Done", taskID)

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("action service error", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{actionErr: errors.New("send message failed")}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "status=Done", taskID)

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("status=Done"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues(taskID.String())
		// No user ID injected — middleware.GetUserID returns zero value

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestTaskActionHandler_ChangePriority(t *testing.T) {
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	task := &taskapp.ReadModel{ID: taskID, ChatID: chatID}

	t.Run("success", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "priority=High", taskID)

		err := h.ChangePriority(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, chatID, actionSvc.lastChatID)
		assert.Equal(t, "High", actionSvc.lastPriority)
	})

	t.Run("missing priority", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "", taskID)

		err := h.ChangePriority(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTaskActionHandler_ChangeAssignee(t *testing.T) {
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()

	task := &taskapp.ReadModel{ID: taskID, ChatID: chatID}

	t.Run("assign user", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		body := "assignee_id=" + assigneeID.String()
		c, rec := newTestTaskActionContext(t, body, taskID)

		err := h.ChangeAssignee(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, chatID, actionSvc.lastChatID)
		require.NotNil(t, actionSvc.lastAssignee)
		assert.Equal(t, assigneeID, *actionSvc.lastAssignee)
	})

	t.Run("clear assignee (empty assignee_id)", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "assignee_id=", taskID)

		err := h.ChangeAssignee(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Nil(t, actionSvc.lastAssignee)
	})

	t.Run("invalid assignee_id format", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "assignee_id=not-a-uuid", taskID)

		err := h.ChangeAssignee(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTaskActionHandler_SetDueDate(t *testing.T) {
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	task := &taskapp.ReadModel{ID: taskID, ChatID: chatID}

	t.Run("set due date", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "due_date=2026-03-15", taskID)

		err := h.SetDueDate(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, chatID, actionSvc.lastChatID)
		require.NotNil(t, actionSvc.lastDueDate)

		expected := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected, *actionSvc.lastDueDate)
	})

	t.Run("clear due date (empty value)", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "due_date=", taskID)

		err := h.SetDueDate(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Nil(t, actionSvc.lastDueDate)
	})

	t.Run("invalid date format", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{task: task}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		c, rec := newTestTaskActionContext(t, "due_date=15-03-2026", taskID)

		err := h.SetDueDate(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTaskActionHandler_ResolveChatID(t *testing.T) {
	t.Run("invalid task_id param", func(t *testing.T) {
		taskSvc := &mockTaskActionTaskService{}
		actionSvc := &mockTaskActionService{}
		h := httphandler.NewTaskActionHandler(taskSvc, actionSvc)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("status=Done"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("task_id")
		c.SetParamValues("not-a-uuid")
		testUserID := uuid.NewUUID()
		c.Set(string(middleware.ContextKeyUserID), testUserID)

		err := h.ChangeStatus(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
