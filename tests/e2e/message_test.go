//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_Send_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user
	testUser := suite.CreateTestUser("msgsendowner")
	client := suite.NewHTTPClient(testUser.Token)

	// Use a random chat ID (mock service doesn't validate chat existence)
	chatID := uuid.NewUUID()

	// Send message
	resp := client.Post("/chats/"+chatID.String()+"/messages", map[string]string{
		"content": "Hello, world!",
	})

	AssertStatus(t, resp, http.StatusCreated)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID        string `json:"id"`
			ChatID    string `json:"chat_id"`
			SenderID  string `json:"sender_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
			IsDeleted bool   `json:"is_deleted"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID        string `json:"id"`
			ChatID    string `json:"chat_id"`
			SenderID  string `json:"sender_id"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
			IsDeleted bool   `json:"is_deleted"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.ID)
	assert.Equal(t, chatID.String(), result.Data.ChatID)
	assert.Equal(t, testUser.ID.String(), result.Data.SenderID)
	assert.Equal(t, "Hello, world!", result.Data.Content)
	assert.False(t, result.Data.IsDeleted)
	assert.NotEmpty(t, result.Data.CreatedAt)
}

func TestMessage_Send_WithReply(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgreplyowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Create original message
	originalMsg, err := message.NewMessage(chatID, testUser.ID, "Original message", uuid.UUID(""))
	require.NoError(t, err)
	suite.MockMessageService.AddMessage(originalMsg)

	// Send reply
	resp := client.Post("/chats/"+chatID.String()+"/messages", map[string]interface{}{
		"content":     "This is a reply",
		"reply_to_id": originalMsg.ID().String(),
	})

	AssertStatus(t, resp, http.StatusCreated)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID        string  `json:"id"`
			Content   string  `json:"content"`
			ReplyToID *string `json:"reply_to_id"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID        string  `json:"id"`
			Content   string  `json:"content"`
			ReplyToID *string `json:"reply_to_id"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "This is a reply", result.Data.Content)
	assert.NotNil(t, result.Data.ReplyToID)
	assert.Equal(t, originalMsg.ID().String(), *result.Data.ReplyToID)
}

func TestMessage_Send_EmptyContent(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgemptyowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Send empty message
	resp := client.Post("/chats/"+chatID.String()+"/messages", map[string]string{
		"content": "",
	})

	AssertStatus(t, resp, http.StatusBadRequest)

	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}](t, resp)

	assert.False(t, result.Success)
	assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
}

func TestMessage_Send_ContentTooLong(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msglongowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Create very long content (>10000 chars)
	longContent := ""
	for i := 0; i < 10001; i++ {
		longContent += "a"
	}

	resp := client.Post("/chats/"+chatID.String()+"/messages", map[string]string{
		"content": longContent,
	})

	AssertStatus(t, resp, http.StatusBadRequest)
}

func TestMessage_Send_Unauthorized(t *testing.T) {
	suite := NewE2ETestSuite(t)

	client := suite.NewHTTPClient("")
	chatID := uuid.NewUUID()

	resp := client.Post("/chats/"+chatID.String()+"/messages", map[string]string{
		"content": "Unauthorized message",
	})

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestMessage_Send_InvalidChatID(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msginvalidchat")
	client := suite.NewHTTPClient(testUser.Token)

	resp := client.Post("/chats/invalid-uuid/messages", map[string]string{
		"content": "Test message",
	})

	AssertStatus(t, resp, http.StatusBadRequest)
}

func TestMessage_List_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msglistowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Create multiple messages
	for i := 0; i < 5; i++ {
		msg, err := message.NewMessage(chatID, testUser.ID, "Message "+string(rune('A'+i)), uuid.UUID(""))
		require.NoError(t, err)
		suite.MockMessageService.AddMessage(msg)
	}

	// List messages
	resp := client.Get("/chats/" + chatID.String() + "/messages")

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"messages"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"messages"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.GreaterOrEqual(t, len(result.Data.Messages), 5)
}

func TestMessage_List_WithPagination(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgpaginationowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Create messages
	for i := 0; i < 10; i++ {
		msg, err := message.NewMessage(chatID, testUser.ID, "Page Message "+string(rune('0'+i)), uuid.UUID(""))
		require.NoError(t, err)
		suite.MockMessageService.AddMessage(msg)
	}

	// List with limit
	resp := client.Get("/chats/" + chatID.String() + "/messages?limit=3&offset=0")

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
			HasMore bool `json:"has_more"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
			HasMore bool `json:"has_more"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Len(t, result.Data.Messages, 3)
	assert.True(t, result.Data.HasMore)
}

func TestMessage_Edit_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgeditowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	// Create message
	msg, err := message.NewMessage(chatID, testUser.ID, "Original content", uuid.UUID(""))
	require.NoError(t, err)
	suite.MockMessageService.AddMessage(msg)

	// Edit message
	resp := client.Put("/messages/"+msg.ID().String(), map[string]string{
		"content": "Edited content",
	})

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string  `json:"id"`
			Content  string  `json:"content"`
			EditedAt *string `json:"edited_at"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string  `json:"id"`
			Content  string  `json:"content"`
			EditedAt *string `json:"edited_at"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, "Edited content", result.Data.Content)
	assert.NotNil(t, result.Data.EditedAt)
}

func TestMessage_Edit_EmptyContent(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgeditemptyowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	msg, err := message.NewMessage(chatID, testUser.ID, "Original", uuid.UUID(""))
	require.NoError(t, err)
	suite.MockMessageService.AddMessage(msg)

	// Edit with empty content
	resp := client.Put("/messages/"+msg.ID().String(), map[string]string{
		"content": "",
	})

	AssertStatus(t, resp, http.StatusBadRequest)
}

func TestMessage_Edit_NotFound(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgeditnotfound")
	client := suite.NewHTTPClient(testUser.Token)

	nonExistentID := uuid.NewUUID()
	resp := client.Put("/messages/"+nonExistentID.String(), map[string]string{
		"content": "Edited content",
	})

	AssertStatus(t, resp, http.StatusNotFound)
}

func TestMessage_Delete_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgdeleteowner")
	client := suite.NewHTTPClient(testUser.Token)

	chatID := uuid.NewUUID()

	msg, err := message.NewMessage(chatID, testUser.ID, "To be deleted", uuid.UUID(""))
	require.NoError(t, err)
	suite.MockMessageService.AddMessage(msg)

	// Delete message
	resp := client.Delete("/messages/" + msg.ID().String())

	AssertStatus(t, resp, http.StatusNoContent)
}

func TestMessage_Delete_NotFound(t *testing.T) {
	suite := NewE2ETestSuite(t)

	testUser := suite.CreateTestUser("msgdeletenotfound")
	client := suite.NewHTTPClient(testUser.Token)

	nonExistentID := uuid.NewUUID()
	resp := client.Delete("/messages/" + nonExistentID.String())

	AssertStatus(t, resp, http.StatusNotFound)
}

func TestMessage_CompleteFlow(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create users
	user1 := suite.CreateTestUser("msgflowuser1")
	user2 := suite.CreateTestUser("msgflowuser2")

	user1Client := suite.NewHTTPClient(user1.Token)
	user2Client := suite.NewHTTPClient(user2.Token)

	// Use a random chat ID
	chatID := uuid.NewUUID()

	// 1. User1 sends a message
	sendResp1 := user1Client.Post("/chats/"+chatID.String()+"/messages", map[string]string{
		"content": "Hello from user1!",
	})
	AssertStatus(t, sendResp1, http.StatusCreated)

	var msg1Result struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	msg1Result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}](t, sendResp1)
	msg1ID := msg1Result.Data.ID

	// 2. User2 sends a reply
	sendResp2 := user2Client.Post("/chats/"+chatID.String()+"/messages", map[string]interface{}{
		"content":     "Hello from user2!",
		"reply_to_id": msg1ID,
	})
	AssertStatus(t, sendResp2, http.StatusCreated)

	// 3. User1 lists messages
	listResp := user1Client.Get("/chats/" + chatID.String() + "/messages")
	AssertStatus(t, listResp, http.StatusOK)

	var listResult struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		} `json:"data"`
	}
	listResult = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		} `json:"data"`
	}](t, listResp)

	assert.GreaterOrEqual(t, len(listResult.Data.Messages), 2)

	// 4. User1 edits their message
	editResp := user1Client.Put("/messages/"+msg1ID, map[string]string{
		"content": "Edited: Hello from user1!",
	})
	AssertStatus(t, editResp, http.StatusOK)

	// 5. User1 deletes their message
	deleteResp := user1Client.Delete("/messages/" + msg1ID)
	AssertStatus(t, deleteResp, http.StatusNoContent)
}
