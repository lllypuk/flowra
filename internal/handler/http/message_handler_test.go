package httphandler_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up message auth context.
func setupMessageAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

// Helper function to create a test message.
func createTestMessage(t *testing.T, chatID, authorID uuid.UUID, content string) *message.Message {
	t.Helper()
	msg, err := message.NewMessage(chatID, authorID, content, uuid.UUID(""))
	require.NoError(t, err)
	return msg
}

// Helper function to build chat messages URL.
func chatMessagesURL(chatID uuid.UUID) string {
	return "/api/v1/chats/" + chatID.String() + "/messages"
}

// Helper function to build message URL.
func messageURL(messageID uuid.UUID) string {
	return "/api/v1/messages/" + messageID.String()
}

func TestMessageHandler_Send(t *testing.T) {
	t.Run("successful send message", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Hello, world!"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("send reply message", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		parentID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "This is a reply", "reply_to_id": "` + parentID.String() + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusCreated, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Hello"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		// No auth context set

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Hello"}`
		req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/chats/invalid/messages", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues("invalid")

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("empty content", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": ""}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("content too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		longContent := strings.Repeat("a", 10001)
		reqBody := `{"content": "` + longContent + `"}`
		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPost, chatMessagesURL(chatID), strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Send(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestMessageHandler_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Add some test messages
		msg1 := createTestMessage(t, chatID, userID, "Message 1")
		msg2 := createTestMessage(t, chatID, userID, "Message 2")
		mockService.AddMessage(msg1)
		mockService.AddMessage(msg2)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID)+"?limit=10&offset=5", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("list with max limit", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Request more than max limit
		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID)+"?limit=200", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid chat ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/chats/invalid/messages", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues("invalid")

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		// No auth context

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestMessageHandler_Edit(t *testing.T) {
	t.Run("successful edit", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Create a test message
		testMessage := createTestMessage(t, chatID, userID, "Original content")
		mockService.AddMessage(testMessage)

		reqBody := `{"content": "Updated content"}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(testMessage.ID()), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testMessage.ID().String())

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("message not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentMessageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Updated content"}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(nonExistentMessageID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentMessageID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("not message author", func(t *testing.T) {
		e := echo.New()
		authorID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Create a message by authorID
		testMessage := createTestMessage(t, chatID, authorID, "Original content")
		mockService.AddMessage(testMessage)

		reqBody := `{"content": "Updated content"}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(testMessage.ID()), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testMessage.ID().String())

		// Try to edit as a different user
		setupMessageAuthContext(c, otherUserID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("empty content", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		messageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": ""}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(messageID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(messageID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("content too long", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		messageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		longContent := strings.Repeat("a", 10001)
		reqBody := `{"content": "` + longContent + `"}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(messageID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(messageID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid message ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Updated content"}`
		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/messages/invalid", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		messageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(messageID), strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(messageID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		messageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		reqBody := `{"content": "Updated content"}`
		req := httptest.NewRequest(stdhttp.MethodPut, messageURL(messageID), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(messageID.String())

		// No auth context

		err := handler.Edit(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestMessageHandler_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Create a test message
		testMessage := createTestMessage(t, chatID, userID, "Message to delete")
		mockService.AddMessage(testMessage)

		req := httptest.NewRequest(stdhttp.MethodDelete, messageURL(testMessage.ID()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testMessage.ID().String())

		setupMessageAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("message not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		nonExistentMessageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, messageURL(nonExistentMessageID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(nonExistentMessageID.String())

		setupMessageAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("not message author", func(t *testing.T) {
		e := echo.New()
		authorID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		// Create a message by authorID
		testMessage := createTestMessage(t, chatID, authorID, "Message content")
		mockService.AddMessage(testMessage)

		req := httptest.NewRequest(stdhttp.MethodDelete, messageURL(testMessage.ID()), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(testMessage.ID().String())

		// Try to delete as a different user
		setupMessageAuthContext(c, otherUserID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("invalid message ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/messages/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")

		setupMessageAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()
		messageID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, messageURL(messageID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(messageID.String())

		// No auth context

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNewMessageHandler(t *testing.T) {
	mockService := httphandler.NewMockMessageService()
	handler := httphandler.NewMessageHandler(mockService)
	assert.NotNil(t, handler)
}

func TestToMessageResponse(t *testing.T) {
	t.Run("basic message", func(t *testing.T) {
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		msg, err := message.NewMessage(chatID, userID, "Test content", uuid.UUID(""))
		require.NoError(t, err)

		resp := httphandler.ToMessageResponse(msg)

		assert.Equal(t, msg.ID(), resp.ID)
		assert.Equal(t, msg.ChatID(), resp.ChatID)
		assert.Equal(t, msg.AuthorID(), resp.SenderID)
		assert.Equal(t, msg.Content(), resp.Content)
		assert.NotEmpty(t, resp.CreatedAt)
		assert.False(t, resp.IsDeleted)
		assert.Nil(t, resp.ReplyToID)
		assert.Nil(t, resp.EditedAt)
	})

	t.Run("reply message", func(t *testing.T) {
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		parentID := uuid.NewUUID()

		msg, err := message.NewMessage(chatID, userID, "Reply content", parentID)
		require.NoError(t, err)

		resp := httphandler.ToMessageResponse(msg)

		assert.NotNil(t, resp.ReplyToID)
		assert.Equal(t, parentID, *resp.ReplyToID)
	})

	t.Run("edited message", func(t *testing.T) {
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		msg, err := message.NewMessage(chatID, userID, "Original content", uuid.UUID(""))
		require.NoError(t, err)

		// Edit the message
		err = msg.EditContent("Updated content", userID)
		require.NoError(t, err)

		resp := httphandler.ToMessageResponse(msg)

		assert.NotNil(t, resp.EditedAt)
		assert.Equal(t, "Updated content", resp.Content)
	})
}

func TestMessageErrors(t *testing.T) {
	// Verify error variables are defined and have expected messages
	assert.Contains(t, httphandler.ErrMessageNotFound.Error(), "message not found")
	assert.Contains(t, httphandler.ErrNotMessageAuthor.Error(), "author")
	assert.Contains(t, httphandler.ErrMessageEmpty.Error(), "empty")
	assert.Contains(t, httphandler.ErrMessageTooLong.Error(), "too long")
	assert.Contains(t, httphandler.ErrMessageDeleted.Error(), "deleted")
	assert.Contains(t, httphandler.ErrParentNotFound.Error(), "parent")
	assert.Contains(t, httphandler.ErrNotChatParticipant.Error(), "participant")
}

func TestMockMessageService(t *testing.T) {
	t.Run("send and list messages", func(t *testing.T) {
		mockService := httphandler.NewMockMessageService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		// Send messages
		for range 3 {
			msg := createTestMessage(t, chatID, userID, "Message content")
			mockService.AddMessage(msg)
		}

		// List messages
		result, err := mockService.ListMessages(context.Background(), messageapp.ListMessagesQuery{
			ChatID: chatID,
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, result.Value, 3)
	})

	t.Run("edit message", func(t *testing.T) {
		mockService := httphandler.NewMockMessageService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		msg := createTestMessage(t, chatID, userID, "Original content")
		mockService.AddMessage(msg)

		// Edit message
		result, err := mockService.EditMessage(context.Background(), messageapp.EditMessageCommand{
			MessageID: msg.ID(),
			Content:   "Updated content",
			EditorID:  userID,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated content", result.Value.Content())
	})

	t.Run("delete message", func(t *testing.T) {
		mockService := httphandler.NewMockMessageService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		msg := createTestMessage(t, chatID, userID, "To be deleted")
		mockService.AddMessage(msg)

		// Delete message
		result, err := mockService.DeleteMessage(context.Background(), messageapp.DeleteMessageCommand{
			MessageID: msg.ID(),
			DeletedBy: userID,
		})
		require.NoError(t, err)
		assert.True(t, result.Value.IsDeleted())
	})

	t.Run("get message", func(t *testing.T) {
		mockService := httphandler.NewMockMessageService()
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()

		msg := createTestMessage(t, chatID, userID, "Test message")
		mockService.AddMessage(msg)

		// Get message
		result, err := mockService.GetMessage(context.Background(), msg.ID())
		require.NoError(t, err)
		assert.Equal(t, msg.ID(), result.ID())
	})

	t.Run("get non-existent message", func(t *testing.T) {
		mockService := httphandler.NewMockMessageService()
		nonExistentID := uuid.NewUUID()

		_, err := mockService.GetMessage(context.Background(), nonExistentID)
		assert.Error(t, err)
	})
}

func TestMessageListPagination(t *testing.T) {
	t.Run("empty list returns empty array", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("negative offset treated as zero", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID)+"?offset=-5", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("invalid limit uses default", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		mockService := httphandler.NewMockMessageService()
		handler := httphandler.NewMessageHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, chatMessagesURL(chatID)+"?limit=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("chat_id")
		c.SetParamValues(chatID.String())

		setupMessageAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})
}
