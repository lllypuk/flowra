package httphandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBoardChatCreator struct {
	returnID  uuid.UUID
	createErr error

	calls      int
	workspace  uuid.UUID
	userID     uuid.UUID
	chatType   string
	title      string
	priority   taskdomain.Priority
	assigneeID *uuid.UUID
	dueDate    *time.Time
}

func (m *mockBoardChatCreator) CreateChat(
	_ context.Context,
	workspaceID, userID uuid.UUID,
	chatType, title string,
	priority taskdomain.Priority,
	assigneeID *uuid.UUID,
	dueDate *time.Time,
) (uuid.UUID, error) {
	m.calls++
	m.workspace = workspaceID
	m.userID = userID
	m.chatType = chatType
	m.title = title
	m.priority = priority
	m.assigneeID = assigneeID
	m.dueDate = dueDate

	if m.createErr != nil {
		return "", m.createErr
	}
	return m.returnID, nil
}

func setUserContextForTaskCreate(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	c.Set(string(middleware.ContextKeyUsername), "testuser")
}

func TestBoardTemplateHandler_TaskCreate_UsesSingleChatPath(t *testing.T) {
	e := echo.New()
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	handler := httphandler.NewBoardTemplateHandler(nil, nil, nil, nil)
	chatCreator := &mockBoardChatCreator{returnID: uuid.NewUUID()}
	handler.SetChatCreator(chatCreator)

	form := url.Values{}
	form.Set("workspace_id", workspaceID.String())
	form.Set("title", "Task from board")
	form.Set("type", "task")
	req := httptest.NewRequest(http.MethodPost, "/partials/task/create", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContextForTaskCreate(c, userID)

	err := handler.TaskCreate(c)
	require.Error(t, err) // renderer is nil; verify command path before rendering

	require.Equal(t, 1, chatCreator.calls)
	assert.Equal(t, workspaceID, chatCreator.workspace)
	assert.Equal(t, userID, chatCreator.userID)
	assert.Equal(t, "task", chatCreator.chatType)
	assert.Equal(t, "Task from board", chatCreator.title)
	assert.Equal(t, taskdomain.PriorityMedium, chatCreator.priority)
	assert.Nil(t, chatCreator.assigneeID)
	assert.Nil(t, chatCreator.dueDate)
}

func TestBoardTemplateHandler_TaskCreate_PropagatesOptionalFields(t *testing.T) {
	e := echo.New()
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()

	handler := httphandler.NewBoardTemplateHandler(nil, nil, nil, nil)
	chatCreator := &mockBoardChatCreator{
		returnID:  uuid.NewUUID(),
		createErr: errors.New("stop after capture"),
	}
	handler.SetChatCreator(chatCreator)

	form := url.Values{}
	form.Set("workspace_id", workspaceID.String())
	form.Set("title", "Bug from board")
	form.Set("type", "bug")
	form.Set("priority", "high")
	form.Set("assignee_id", assigneeID.String())
	form.Set("due_date", "2026-03-10")

	req := httptest.NewRequest(http.MethodPost, "/partials/task/create", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContextForTaskCreate(c, userID)

	err := handler.TaskCreate(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	require.Equal(t, 1, chatCreator.calls)
	assert.Equal(t, taskdomain.PriorityHigh, chatCreator.priority)
	require.NotNil(t, chatCreator.assigneeID)
	assert.Equal(t, assigneeID, *chatCreator.assigneeID)
	require.NotNil(t, chatCreator.dueDate)
	assert.Equal(t, "2026-03-10", chatCreator.dueDate.Format("2006-01-02"))
}
