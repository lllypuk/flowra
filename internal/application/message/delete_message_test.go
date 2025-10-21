package message_test

import (
	"context"
	"testing"

	"github.com/flowra/flowra/internal/application/message"
	domain "github.com/flowra/flowra/internal/domain/message"
	"github.com/flowra/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteMessageUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewDeleteMessageUseCase(messageRepo, eventBus)

	cmd := message.DeleteMessageCommand{
		MessageID: msg.ID(),
		DeletedBy: authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.True(t, result.Value.IsDeleted())
	assert.NotNil(t, result.Value.DeletedAt())

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestDeleteMessageUseCase_NotAuthor(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewDeleteMessageUseCase(messageRepo, eventBus)

	cmd := message.DeleteMessageCommand{
		MessageID: msg.ID(),
		DeletedBy: otherUserID, // Different user
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestDeleteMessageUseCase_MessageNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewDeleteMessageUseCase(messageRepo, eventBus)

	cmd := message.DeleteMessageCommand{
		MessageID: uuid.NewUUID(), // Non-existent
		DeletedBy: uuid.NewUUID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}

func TestDeleteMessageUseCase_AlreadyDeleted(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create and delete message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	err = msg.Delete(authorID)
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewDeleteMessageUseCase(messageRepo, eventBus)

	cmd := message.DeleteMessageCommand{
		MessageID: msg.ID(),
		DeletedBy: authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}
