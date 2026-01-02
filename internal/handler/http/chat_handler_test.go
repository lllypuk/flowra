package httphandler_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up chat auth context.
func setupChatAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

// Helper function to create a test chat.
func createTestChat(t *testing.T, workspaceID, creatorID uuid.UUID) *chat.Chat {
	t.Helper()
	ch, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)
	return ch
}

// Helper function to build workspace chats URL.
func workspaceChatsURL(workspaceID uuid.UUID) string {
	return "/api/v1/workspaces/" + workspaceID.String() + "/chats"
}

// Helper function to build chat URL.
func chatURL(chatID uuid.UUID) string {
	return "/api/v1/chats/" + chatID.String()
}

// Helper function to build chat participants URL.
func chatParticipantsURL(chatID uuid.UUID) string {
	return "/api/v1/chats/" + chatID.String() + "/participants"
}

// Helper function to build chat participant URL.
func chatParticipantURL(chatID, userID uuid.UUID) string {
	return "/api/v1/chats/" + chatID.String() + "/participants/" + userID.String()
}

func TestChatHandler_Create(t *testing.T) {
	t.Run("successful create discussion chat", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "discussion", "is_public": true}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("successful create task chat", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "New Task", "type": "task", "is_public": false}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("create with participants", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		participant1 := uuid.NewUUID()
		participant2 := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "discussion", "is_public": false, "participant_ids": ["` + participant1.String() + `", "` + participant2.String() + `"]}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "discussion"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		// No auth context set

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid workspace ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "discussion"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/workspaces/invalid/chats", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid chat type", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "invalid_type"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("task without name", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "", "type": "task"}`
		req := httptest.NewRequest(stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("name too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		longName := strings.Repeat("a", 101)
		reqBody := `{"name": "` + longName + `", "type": "discussion"}`
		req := httptest.NewRequest(
			stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(
			stdhttp.MethodPost, workspaceChatsURL(workspaceID), strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.Create(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestChatHandler_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		// Create a test chat
		testChat := createTestChat(t, workspaceID, userID)
		mockService.AddChat(testChat)

		req := httptest.NewRequest(stdhttp.MethodGet, chatURL(testChat.ID()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("chat not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentChatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatURL(nonExistentChatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentChatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("access denied for non-participant", func(t *testing.T) {
		e := echo.New()
		creatorID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		// Create a private chat
		testChat := createTestChat(t, workspaceID, creatorID)
		mockService.AddChat(testChat)

		req := httptest.NewRequest(stdhttp.MethodGet, chatURL(testChat.ID()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, otherUserID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/chats/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatURL(chatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		// No auth context

		err := handler.Get(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestChatHandler_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		// Create test chats
		chat1 := createTestChat(t, workspaceID, userID)
		chat2 := createTestChat(t, workspaceID, userID)
		mockService.AddChat(chat1)
		mockService.AddChat(chat2)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceChatsURL(workspaceID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list with type filter", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		// Create a chat
		testChat := createTestChat(t, workspaceID, userID)
		mockService.AddChat(testChat)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceChatsURL(workspaceID)+"?type=discussion", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceChatsURL(workspaceID)+"?limit=10&offset=5", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		setupChatAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid workspace ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/workspaces/invalid/chats", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, workspaceChatsURL(workspaceID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("workspace_id")
		c.SetParamValues(workspaceID.String())

		// No auth context

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestChatHandler_Update(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		// Create a task chat that can be renamed
		testChat, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
		require.NoError(t, err)
		err = testChat.ConvertToTask("Original Title", userID)
		require.NoError(t, err)
		mockService.AddChat(testChat)

		reqBody := `{"name": "Updated Chat Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(testChat.ID()), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, userID)

		err = handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("chat not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentChatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "Updated Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(nonExistentChatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentChatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("empty name", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": ""}`
		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("name too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		longName := strings.Repeat("a", 101)
		reqBody := `{"name": "` + longName + `"}`
		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "New Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/chats/invalid", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(chatID), strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"name": "New Name"}`
		req := httptest.NewRequest(stdhttp.MethodPut, chatURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		// No auth context

		err := handler.Update(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestChatHandler_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		testChat := createTestChat(t, workspaceID, userID)
		mockService.AddChat(testChat)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatURL(testChat.ID()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("chat not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentChatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatURL(nonExistentChatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentChatID.String())

		setupChatAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/chats/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatURL(chatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		// No auth context

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestChatHandler_AddParticipant(t *testing.T) {
	t.Run("successful add participant", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		newParticipantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		testChat := createTestChat(t, workspaceID, userID)
		mockService.AddChat(testChat)

		reqBody := `{"user_id": "` + newParticipantID.String() + `", "role": "member"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatParticipantsURL(testChat.ID()), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("add admin participant", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		newParticipantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		testChat := createTestChat(t, workspaceID, userID)
		mockService.AddChat(testChat)

		reqBody := `{"user_id": "` + newParticipantID.String() + `", "role": "admin"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatParticipantsURL(testChat.ID()), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testChat.ID().String())

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("missing user_id", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"role": "member"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatParticipantsURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("chat not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentChatID := uuid.NewUUID()
		newParticipantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"user_id": "` + newParticipantID.String() + `", "role": "member"}`
		req := httptest.NewRequest(
			stdhttp.MethodPost, chatParticipantsURL(nonExistentChatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentChatID.String())

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		newParticipantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"user_id": "` + newParticipantID.String() + `", "role": "member"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/chats/invalid/participants", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPost, chatParticipantsURL(chatID), strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		setupChatAuthContext(c, userID)

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()
		newParticipantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		reqBody := `{"user_id": "` + newParticipantID.String() + `", "role": "member"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatParticipantsURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(chatID.String())

		// No auth context

		err := handler.AddParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestChatHandler_RemoveParticipant(t *testing.T) {
	t.Run("successful remove participant", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		participantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		testChat := createTestChat(t, workspaceID, userID)
		err := testChat.AddParticipant(participantID, chat.RoleMember)
		require.NoError(t, err)
		mockService.AddChat(testChat)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatParticipantURL(testChat.ID(), participantID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(testChat.ID().String(), participantID.String())

		setupChatAuthContext(c, userID)

		err = handler.RemoveParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("chat not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentChatID := uuid.NewUUID()
		participantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatParticipantURL(nonExistentChatID, participantID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(nonExistentChatID.String(), participantID.String())

		setupChatAuthContext(c, userID)

		err := handler.RemoveParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		participantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		invalidURL := "/api/v1/chats/invalid/participants/" + participantID.String()
		req := httptest.NewRequest(stdhttp.MethodDelete, invalidURL, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("invalid", participantID.String())

		setupChatAuthContext(c, userID)

		err := handler.RemoveParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		invalidURL := "/api/v1/chats/" + chatID.String() + "/participants/invalid"
		req := httptest.NewRequest(stdhttp.MethodDelete, invalidURL, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(chatID.String(), "invalid")

		setupChatAuthContext(c, userID)

		err := handler.RemoveParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()
		participantID := uuid.NewUUID()

		mockService := httphandler.NewMockChatService()
		handler := httphandler.NewChatHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, chatParticipantURL(chatID, participantID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(chatID.String(), participantID.String())

		// No auth context

		err := handler.RemoveParticipant(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNewChatHandler(t *testing.T) {
	mockService := httphandler.NewMockChatService()
	handler := httphandler.NewChatHandler(mockService)
	assert.NotNil(t, handler)
}

func TestToChatResponse(t *testing.T) {
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()

	ch, err := chat.NewChat(workspaceID, chat.TypeDiscussion, true, userID)
	require.NoError(t, err)

	resp := httphandler.ToChatResponse(ch)

	assert.Equal(t, ch.ID(), resp.ID)
	assert.Equal(t, ch.WorkspaceID(), resp.WorkspaceID)
	assert.Equal(t, string(ch.Type()), resp.Type)
	assert.Equal(t, ch.IsPublic(), resp.IsPublic)
	assert.Equal(t, ch.CreatedBy(), resp.CreatedBy)
	assert.NotEmpty(t, resp.CreatedAt)
}

func TestToChatResponseFromDTO(t *testing.T) {
	userID := uuid.NewUUID()
	workspaceID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	dto := &chatapp.Chat{
		ID:          chatID,
		WorkspaceID: workspaceID,
		Type:        chat.TypeTask,
		Title:       "Test Task",
		IsPublic:    false,
		CreatedBy:   userID,
		Participants: []chatapp.Participant{
			{
				UserID: userID,
				Role:   chat.RoleAdmin,
			},
		},
	}

	status := "open"
	dto.Status = &status

	resp := httphandler.ToChatResponseFromDTO(dto)

	assert.Equal(t, chatID, resp.ID)
	assert.Equal(t, workspaceID, resp.WorkspaceID)
	assert.Equal(t, "task", resp.Type)
	assert.Equal(t, "Test Task", resp.Name)
	assert.False(t, resp.IsPublic)
	assert.Equal(t, userID, resp.CreatedBy)
	assert.NotNil(t, resp.Status)
	assert.Equal(t, "open", *resp.Status)
	assert.Len(t, resp.Participants, 1)
}

func TestChatErrors(t *testing.T) {
	// Verify error variables are defined and have expected messages
	assert.Contains(t, httphandler.ErrChatNotFound.Error(), "chat not found")
	assert.Contains(t, httphandler.ErrNotChatMember.Error(), "not a member")
	assert.Contains(t, httphandler.ErrNotChatAdmin.Error(), "admin")
	assert.Contains(t, httphandler.ErrCannotRemoveCreator.Error(), "creator")
	assert.Contains(t, httphandler.ErrInvalidChatType.Error(), "chat type")
	assert.Contains(t, httphandler.ErrParticipantNotFound.Error(), "participant")
	assert.Contains(t, httphandler.ErrParticipantExists.Error(), "already exists")
	assert.Contains(t, httphandler.ErrTooManyParticipants.Error(), "too many")
	assert.Contains(t, httphandler.ErrDirectChatMaxMembers.Error(), "direct")
	assert.Contains(t, httphandler.ErrChatNameRequired.Error(), "required")
	assert.Contains(t, httphandler.ErrChatNameTooLong.Error(), "too long")
	assert.Contains(t, httphandler.ErrInvalidParticipantIDs.Error(), "participant")
}

func TestMockChatService(t *testing.T) {
	t.Run("create and get chat", func(t *testing.T) {
		mockService := httphandler.NewMockChatService()
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		cmd := chatapp.CreateChatCommand{
			WorkspaceID: workspaceID,
			Title:       "Test Chat",
			Type:        chat.TypeDiscussion,
			IsPublic:    true,
			CreatedBy:   userID,
		}

		result, err := mockService.CreateChat(context.Background(), cmd)
		require.NoError(t, err)
		assert.NotNil(t, result.Value)

		query := chatapp.GetChatQuery{
			ChatID:      result.Value.ID(),
			RequestedBy: userID,
		}

		getResult, err := mockService.GetChat(context.Background(), query)
		require.NoError(t, err)
		assert.NotNil(t, getResult.Chat)
	})

	t.Run("list chats", func(t *testing.T) {
		mockService := httphandler.NewMockChatService()
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		// Create chats
		for range 3 {
			cmd := chatapp.CreateChatCommand{
				WorkspaceID: workspaceID,
				Type:        chat.TypeDiscussion,
				IsPublic:    true,
				CreatedBy:   userID,
			}
			_, _ = mockService.CreateChat(context.Background(), cmd)
		}

		query := chatapp.ListChatsQuery{
			WorkspaceID: workspaceID,
			Limit:       10,
			Offset:      0,
			RequestedBy: userID,
		}

		result, err := mockService.ListChats(context.Background(), query)
		require.NoError(t, err)
		assert.Len(t, result.Chats, 3)
	})

	t.Run("delete chat", func(t *testing.T) {
		mockService := httphandler.NewMockChatService()
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		cmd := chatapp.CreateChatCommand{
			WorkspaceID: workspaceID,
			Type:        chat.TypeDiscussion,
			IsPublic:    true,
			CreatedBy:   userID,
		}

		result, err := mockService.CreateChat(context.Background(), cmd)
		require.NoError(t, err)

		err = mockService.DeleteChat(context.Background(), result.Value.ID(), userID)
		require.NoError(t, err)

		// Verify chat is deleted
		query := chatapp.GetChatQuery{
			ChatID:      result.Value.ID(),
			RequestedBy: userID,
		}
		_, err = mockService.GetChat(context.Background(), query)
		assert.Error(t, err)
	})
}
