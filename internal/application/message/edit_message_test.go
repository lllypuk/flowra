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

func TestEditMessageUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create and save message
	msg, err := domain.NewMessage(chatID, authorID, "Original content", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	cmd := message.EditMessageCommand{
		MessageID: msg.ID(),
		Content:   "Updated content",
		EditorID:  authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Equal(t, "Updated content", result.Value.Content())
	assert.True(t, result.Value.IsEdited())
	assert.NotNil(t, result.Value.EditedAt())

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestEditMessageUseCase_NotAuthor(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Original content", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	cmd := message.EditMessageCommand{
		MessageID: msg.ID(),
		Content:   "Updated content",
		EditorID:  otherUserID, // Different user
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestEditMessageUseCase_MessageNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	cmd := message.EditMessageCommand{
		MessageID: uuid.NewUUID(), // Non-existent
		Content:   "Updated content",
		EditorID:  uuid.NewUUID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}

func TestEditMessageUseCase_MessageDeleted(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create and delete message
	msg, err := domain.NewMessage(chatID, authorID, "Original content", "")
	require.NoError(t, err)
	err = msg.Delete(authorID)
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	cmd := message.EditMessageCommand{
		MessageID: msg.ID(),
		Content:   "Updated content",
		EditorID:  authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageDeleted)
	assert.Nil(t, result.Value)
}

func TestEditMessageUseCase_EmptyContent(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Original content", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	cmd := message.EditMessageCommand{
		MessageID: msg.ID(),
		Content:   "", // Empty
		EditorID:  authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrEmptyContent)
	assert.Nil(t, result.Value)
}

func TestEditMessageUseCase_ContentTooLong(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Original content", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewEditMessageUseCase(messageRepo, eventBus)

	// Create content > message.MaxContentLength
	longContent := make([]byte, message.MaxContentLength+1)
	for i := range longContent {
		longContent[i] = 'a'
	}

	cmd := message.EditMessageCommand{
		MessageID: msg.ID(),
		Content:   string(longContent),
		EditorID:  authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrContentTooLong)
	assert.Nil(t, result.Value)
}
