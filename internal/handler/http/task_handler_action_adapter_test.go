package httphandler_test

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyTaskWriteActionService struct {
	statusCalls   int
	priorityCalls int
	assigneeCalls int
	dueDateCalls  int

	lastChatID   uuid.UUID
	lastStatus   string
	lastPriority string
	lastAssignee *uuid.UUID
	lastDueDate  *time.Time
}

func (s *spyTaskWriteActionService) ChangeStatus(
	_ context.Context,
	chatID uuid.UUID,
	newStatus string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	s.statusCalls++
	s.lastChatID = chatID
	s.lastStatus = newStatus
	return &appcore.ActionResult{Success: true}, nil
}

func (s *spyTaskWriteActionService) SetPriority(
	_ context.Context,
	chatID uuid.UUID,
	priority string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	s.priorityCalls++
	s.lastChatID = chatID
	s.lastPriority = priority
	return &appcore.ActionResult{Success: true}, nil
}

func (s *spyTaskWriteActionService) AssignUser(
	_ context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	s.assigneeCalls++
	s.lastChatID = chatID
	s.lastAssignee = assigneeID
	return &appcore.ActionResult{Success: true}, nil
}

func (s *spyTaskWriteActionService) SetDueDate(
	_ context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	s.dueDateCalls++
	s.lastChatID = chatID
	s.lastDueDate = dueDate
	return &appcore.ActionResult{Success: true}, nil
}

func TestTaskHandler_ChangeStatus_UsesActionServiceWhenConfigured(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	mockService := httphandler.NewMockTaskService()
	testTask := &taskapp.ReadModel{
		ID:         uuid.NewUUID(),
		ChatID:     chatID,
		Title:      "Task",
		EntityType: task.TypeTask,
		Status:     task.StatusToDo,
		Priority:   task.PriorityMedium,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		Version:    1,
	}
	mockService.AddTask(testTask)
	actionSpy := &spyTaskWriteActionService{}

	handler := httphandler.NewTaskHandler(mockService, actionSpy)

	reqBody := `{"status": "in_progress"}`
	req := httptest.NewRequest(
		stdhttp.MethodPut,
		taskURL(workspaceID, testTask.ID)+"/status",
		strings.NewReader(reqBody),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("workspace_id", "task_id")
	c.SetParamValues(workspaceID.String(), testTask.ID.String())
	setupTaskAuthContext(c, userID)

	err := handler.ChangeStatus(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, 1, actionSpy.statusCalls)
	assert.Equal(t, chatID, actionSpy.lastChatID)
	assert.Equal(t, string(task.StatusInProgress), actionSpy.lastStatus)
}

func TestTaskHandler_SetDueDate_IdempotentNoopWhenDateUnchanged(t *testing.T) {
	e := echo.New()
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	mockService := httphandler.NewMockTaskService()
	testTask := &taskapp.ReadModel{
		ID:         uuid.NewUUID(),
		ChatID:     chatID,
		Title:      "Task",
		EntityType: task.TypeTask,
		Status:     task.StatusToDo,
		Priority:   task.PriorityMedium,
		DueDate:    &dueDate,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		Version:    1,
	}
	mockService.AddTask(testTask)
	actionSpy := &spyTaskWriteActionService{}

	handler := httphandler.NewTaskHandler(mockService, actionSpy)

	reqBody := `{"due_date": "2026-03-15"}`
	req := httptest.NewRequest(
		stdhttp.MethodPut,
		taskURL(workspaceID, testTask.ID)+"/due-date",
		strings.NewReader(reqBody),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("workspace_id", "task_id")
	c.SetParamValues(workspaceID.String(), testTask.ID.String())
	setupTaskAuthContext(c, userID)

	err := handler.SetDueDate(c)
	require.NoError(t, err)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, 0, actionSpy.dueDateCalls)
}
