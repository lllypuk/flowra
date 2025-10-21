package message_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/message"
	domain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetMessage tests

func TestGetMessageUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewGetMessageUseCase(messageRepo)

	query := message.GetMessageQuery{
		MessageID: msg.ID(),
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Equal(t, msg.ID(), result.Value.ID())
	assert.Equal(t, "Test message", result.Value.Content())
}

func TestGetMessageUseCase_NotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewGetMessageUseCase(messageRepo)

	query := message.GetMessageQuery{
		MessageID: uuid.NewUUID(), // Non-existent
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}

func TestGetMessageUseCase_InvalidID(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewGetMessageUseCase(messageRepo)

	query := message.GetMessageQuery{
		MessageID: "", // Zero UUID
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

// ListMessages tests

func TestListMessagesUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create multiple messages
	for range 5 {
		msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
		require.NoError(t, err)
		messageRepo.Messages[msg.ID()] = msg
	}

	useCase := message.NewListMessagesUseCase(messageRepo)

	query := message.ListMessagesQuery{
		ChatID: chatID,
		Limit:  10,
		Offset: 0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Len(t, result.Value, 5)
}

func TestListMessagesUseCase_EmptyChat(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewListMessagesUseCase(messageRepo)

	query := message.ListMessagesQuery{
		ChatID: uuid.NewUUID(), // Chat with no messages
		Limit:  10,
		Offset: 0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Empty(t, result.Value)
}

func TestListMessagesUseCase_Pagination(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create 25 messages
	for range 25 {
		msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
		require.NoError(t, err)
		messageRepo.Messages[msg.ID()] = msg
	}

	useCase := message.NewListMessagesUseCase(messageRepo)

	// First page
	query1 := message.ListMessagesQuery{
		ChatID: chatID,
		Limit:  10,
		Offset: 0,
	}
	result1, err := useCase.Execute(context.Background(), query1)
	require.NoError(t, err)
	assert.Len(t, result1.Value, 10)

	// Second page
	query2 := message.ListMessagesQuery{
		ChatID: chatID,
		Limit:  10,
		Offset: 10,
	}
	result2, err := useCase.Execute(context.Background(), query2)
	require.NoError(t, err)
	assert.Len(t, result2.Value, 10)

	// Third page
	query3 := message.ListMessagesQuery{
		ChatID: chatID,
		Limit:  10,
		Offset: 20,
	}
	result3, err := useCase.Execute(context.Background(), query3)
	require.NoError(t, err)
	assert.Len(t, result3.Value, 5)
}

func TestListMessagesUseCase_DefaultLimit(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewListMessagesUseCase(messageRepo)

	query := message.ListMessagesQuery{
		ChatID: uuid.NewUUID(),
		Limit:  0, // Should use default
		Offset: 0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	// Limit should be set to message.DefaultLimit (50)
}

func TestListMessagesUseCase_MaxLimit(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create 150 messages
	for range 150 {
		msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
		require.NoError(t, err)
		messageRepo.Messages[msg.ID()] = msg
	}

	useCase := message.NewListMessagesUseCase(messageRepo)

	query := message.ListMessagesQuery{
		ChatID: chatID,
		Limit:  200, // Exceeds message.MaxLimit, should be capped at 100
		Offset: 0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	// Should return at most message.MaxLimit (100) messages
	assert.LessOrEqual(t, len(result.Value), message.MaxLimit)
}

func TestListMessagesUseCase_InvalidChatID(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewListMessagesUseCase(messageRepo)

	query := message.ListMessagesQuery{
		ChatID: "", // Zero UUID
		Limit:  10,
		Offset: 0,
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

// GetThread tests

func TestGetThreadUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create parent message
	parentMsg, err := domain.NewMessage(chatID, authorID, "Parent message", "")
	require.NoError(t, err)
	messageRepo.Messages[parentMsg.ID()] = parentMsg

	// Create thread replies
	for range 3 {
		reply, replyErr := domain.NewMessage(chatID, authorID, "Reply message", parentMsg.ID())
		require.NoError(t, replyErr)
		messageRepo.Messages[reply.ID()] = reply
	}

	useCase := message.NewGetThreadUseCase(messageRepo)

	query := message.GetThreadQuery{
		ParentMessageID: parentMsg.ID(),
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Len(t, result.Value, 3)

	// Verify all messages are replies to the parent
	for _, msg := range result.Value {
		assert.Equal(t, parentMsg.ID(), msg.ParentMessageID())
		assert.True(t, msg.IsReply())
	}
}

func TestGetThreadUseCase_EmptyThread(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create parent message with no replies
	parentMsg, err := domain.NewMessage(chatID, authorID, "Parent message", "")
	require.NoError(t, err)
	messageRepo.Messages[parentMsg.ID()] = parentMsg

	useCase := message.NewGetThreadUseCase(messageRepo)

	query := message.GetThreadQuery{
		ParentMessageID: parentMsg.ID(),
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Empty(t, result.Value)
}

func TestGetThreadUseCase_ParentNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewGetThreadUseCase(messageRepo)

	query := message.GetThreadQuery{
		ParentMessageID: uuid.NewUUID(), // Non-existent
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrParentNotFound)
	assert.Nil(t, result.Value)
}

func TestGetThreadUseCase_InvalidParentID(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	useCase := message.NewGetThreadUseCase(messageRepo)

	query := message.GetThreadQuery{
		ParentMessageID: "", // Zero UUID
	}

	result, err := useCase.Execute(context.Background(), query)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestGetThreadUseCase_NestedThreads(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create parent message
	parentMsg, err := domain.NewMessage(chatID, authorID, "Parent message", "")
	require.NoError(t, err)
	messageRepo.Messages[parentMsg.ID()] = parentMsg

	// Create first level replies
	reply1, err := domain.NewMessage(chatID, authorID, "Reply 1", parentMsg.ID())
	require.NoError(t, err)
	messageRepo.Messages[reply1.ID()] = reply1

	// Create nested reply (reply to reply1)
	nestedReply, err := domain.NewMessage(chatID, authorID, "Nested reply", reply1.ID())
	require.NoError(t, err)
	messageRepo.Messages[nestedReply.ID()] = nestedReply

	useCase := message.NewGetThreadUseCase(messageRepo)

	// Get direct replies to parent
	query := message.GetThreadQuery{
		ParentMessageID: parentMsg.ID(),
	}

	result, err := useCase.Execute(context.Background(), query)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Len(t, result.Value, 1) // Should only return direct replies

	// Get nested thread
	queryNested := message.GetThreadQuery{
		ParentMessageID: reply1.ID(),
	}

	resultNested, err := useCase.Execute(context.Background(), queryNested)

	require.NoError(t, err)
	assert.NotNil(t, resultNested.Value)
	assert.Len(t, resultNested.Value, 1) // Should only return the nested reply
}
