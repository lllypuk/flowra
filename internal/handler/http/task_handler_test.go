package httphandler_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up task auth context.
func setupTaskAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

// Helper function to create a test task read model.
func createTestTaskReadModel(chatID, createdBy uuid.UUID) *taskapp.ReadModel {
	taskID := uuid.NewUUID()
	return &taskapp.ReadModel{
		ID:         taskID,
		ChatID:     chatID,
		Title:      "Test Task",
		EntityType: task.TypeTask,
		Status:     task.StatusToDo,
		Priority:   task.PriorityMedium,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
		Version:    1,
	}
}

// Helper function to build workspace tasks URL.
func workspaceTasksURL(workspaceID uuid.UUID) string {
	return "/api/v1/workspaces/" + workspaceID.String() + "/tasks"
}

// Helper function to build task URL.
func taskURL(taskID uuid.UUID) string {
	return "/api/v1/tasks/" + taskID.String()
}

func TestTaskHandler_Create(t *testing.T) {
	t.Run("successful create task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"title": "New Task", "chat_id": "` + chatID.String() + `", "priority": "high"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("create task with assignee and due date", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{
			"title": "Task with details",
			"chat_id": "` + chatID.String() + `",
			"priority": "critical",
			"assignee_id": "` + assigneeID.String() + `",
			"due_date": "2026-02-15",
			"entity_type": "bug"
		}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"title": "New Task", "chat_id": "` + chatID.String() + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("missing title", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"chat_id": "` + chatID.String() + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing chat_id", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"title": "New Task"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid due date format", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"title": "New Task", "chat_id": "` + chatID.String() + `", "due_date": "invalid-date"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceTasksURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestTaskHandler_Get(t *testing.T) {
	t.Run("successful get task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, taskURL(testTask.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("task not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, taskURL(taskID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(taskID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("invalid task ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/tasks/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")

		setupTaskAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestTaskHandler_List(t *testing.T) {
	t.Run("successful list tasks", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		task1 := createTestTaskReadModel(chatID, userID)
		task2 := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(task1)
		mockService.AddTask(task2)

		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceTasksURL(workspaceID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list with status filter", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		task1 := createTestTaskReadModel(chatID, userID)
		task1.Status = task.StatusToDo
		task2 := createTestTaskReadModel(chatID, userID)
		task2.Status = task.StatusDone
		mockService.AddTask(task1)
		mockService.AddTask(task2)

		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceTasksURL(workspaceID)+"?status=todo", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		for range 25 {
			mockService.AddTask(createTestTaskReadModel(chatID, userID))
		}

		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceTasksURL(workspaceID)+"?page=2&per_page=10", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupTaskAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})
}

func TestTaskHandler_ChangeStatus(t *testing.T) {
	t.Run("successful change status", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"status": "in_progress"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/status", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.ChangeStatus(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid status value", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"status": "invalid_status"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/status", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.ChangeStatus(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"status": "done"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(taskID)+"/status", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(taskID.String())

		setupTaskAuthContext(c, userID)

		err := handler.ChangeStatus(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})
}

func TestTaskHandler_Assign(t *testing.T) {
	t.Run("successful assign task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"assignee_id": "` + assigneeID.String() + `"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/assign", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Assign(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("unassign task (null assignee)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"assignee_id": null}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/assign", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Assign(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid assignee ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"assignee_id": "invalid-uuid"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/assign", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Assign(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestTaskHandler_ChangePriority(t *testing.T) {
	t.Run("successful change priority", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"priority": "high"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/priority", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.ChangePriority(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid priority value", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"priority": ""}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/priority", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.ChangePriority(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestTaskHandler_SetDueDate(t *testing.T) {
	t.Run("successful set due date", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"due_date": "2026-03-15"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/due-date", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.SetDueDate(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("clear due date (null)", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		dueDate := time.Now().Add(24 * time.Hour)
		testTask.DueDate = &dueDate
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"due_date": null}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/due-date", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.SetDueDate(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid due date format", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		reqBody := `{"due_date": "not-a-date"}`
		req := httptest.NewRequest(stdhttp.MethodPut, taskURL(testTask.ID)+"/due-date", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.SetDueDate(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestTaskHandler_Delete(t *testing.T) {
	t.Run("successful delete task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, taskURL(testTask.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testTask.ID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("delete non-existent task", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		taskID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, taskURL(taskID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(taskID.String())

		setupTaskAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("missing auth for delete", func(t *testing.T) {
		e := echo.New()
		taskID := uuid.NewUUID()

		mockService := httphandler.NewMockTaskService()
		handler := httphandler.NewTaskHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, taskURL(taskID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(taskID.String())

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNewTaskHandler(t *testing.T) {
	mockService := httphandler.NewMockTaskService()
	handler := httphandler.NewTaskHandler(mockService)
	assert.NotNil(t, handler)
}

func TestToTaskResponseFromReadModel(t *testing.T) {
	chatID := uuid.NewUUID()
	userID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	rm := &taskapp.ReadModel{
		ID:         uuid.NewUUID(),
		ChatID:     chatID,
		Title:      "Test Task",
		EntityType: task.TypeBug,
		Status:     task.StatusInProgress,
		Priority:   task.PriorityHigh,
		AssignedTo: &assigneeID,
		DueDate:    &dueDate,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
		Version:    5,
	}

	resp := httphandler.ToTaskResponseFromReadModel(rm)

	assert.Equal(t, rm.ID.String(), resp.ID)
	assert.Equal(t, chatID.String(), resp.ChatID)
	assert.Equal(t, "Test Task", resp.Title)
	assert.Equal(t, string(task.TypeBug), resp.EntityType)
	assert.Equal(t, string(task.StatusInProgress), resp.Status)
	assert.Equal(t, string(task.PriorityHigh), resp.Priority)
	assert.NotNil(t, resp.AssigneeID)
	assert.Equal(t, assigneeID.String(), *resp.AssigneeID)
	assert.NotNil(t, resp.DueDate)
	assert.Equal(t, "2026-03-15", *resp.DueDate)
	assert.Equal(t, userID.String(), resp.ReporterID)
	assert.Equal(t, 5, resp.Version)
}

func TestMockTaskService(t *testing.T) {
	t.Run("create and get task", func(t *testing.T) {
		mockService := httphandler.NewMockTaskService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		cmd := taskapp.CreateTaskCommand{
			ChatID:     chatID,
			Title:      "Mock Task",
			EntityType: task.TypeTask,
			Priority:   task.PriorityMedium,
			CreatedBy:  userID,
		}

		result, err := mockService.CreateTask(context.Background(), cmd)
		require.NoError(t, err)
		assert.True(t, result.Success)

		retrieved, err := mockService.GetTask(context.Background(), result.TaskID)
		require.NoError(t, err)
		assert.Equal(t, "Mock Task", retrieved.Title)
	})

	t.Run("list and count tasks", func(t *testing.T) {
		mockService := httphandler.NewMockTaskService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		for range 5 {
			mockService.AddTask(createTestTaskReadModel(chatID, userID))
		}

		tasks, err := mockService.ListTasks(context.Background(), taskapp.Filters{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, tasks, 5)

		count, err := mockService.CountTasks(context.Background(), taskapp.Filters{})
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("change status", func(t *testing.T) {
		mockService := httphandler.NewMockTaskService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		cmd := taskapp.ChangeStatusCommand{
			TaskID:    testTask.ID,
			NewStatus: task.StatusDone,
			ChangedBy: userID,
		}

		_, err := mockService.ChangeStatus(context.Background(), cmd)
		require.NoError(t, err)

		updated, err := mockService.GetTask(context.Background(), testTask.ID)
		require.NoError(t, err)
		assert.Equal(t, task.StatusDone, updated.Status)
	})

	t.Run("delete task", func(t *testing.T) {
		mockService := httphandler.NewMockTaskService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		testTask := createTestTaskReadModel(chatID, userID)
		mockService.AddTask(testTask)

		err := mockService.DeleteTask(context.Background(), testTask.ID, userID)
		require.NoError(t, err)

		_, err = mockService.GetTask(context.Background(), testTask.ID)
		assert.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	})
}
