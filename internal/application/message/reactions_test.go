package message_test

import (
	"context"
	"testing"

	"github.com/lllypuk/teams-up/internal/application/message"
	domain "github.com/lllypuk/teams-up/internal/domain/message"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddReactionUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	cmd := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.True(t, result.Value.HasReaction(userID, "👍"))
	assert.Equal(t, 1, result.Value.GetReactionCount("👍"))

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestAddReactionUseCase_MultipleUsersReact(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	// User 1 adds reaction
	cmd1 := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    user1ID,
	}
	result1, err := useCase.Execute(context.Background(), cmd1)
	require.NoError(t, err)
	assert.Equal(t, 1, result1.Value.GetReactionCount("👍"))

	// User 2 adds same reaction
	cmd2 := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    user2ID,
	}
	result2, err := useCase.Execute(context.Background(), cmd2)
	require.NoError(t, err)
	assert.Equal(t, 2, result2.Value.GetReactionCount("👍"))
}

func TestAddReactionUseCase_DuplicateReaction(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)

	// Add reaction first time
	err = msg.AddReaction(userID, "👍")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	// Try to add same reaction again
	cmd := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestAddReactionUseCase_MessageDeleted(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create and delete message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	err = msg.Delete(authorID)
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	cmd := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageDeleted)
	assert.Nil(t, result.Value)
}

func TestAddReactionUseCase_MessageNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	cmd := message.AddReactionCommand{
		MessageID: uuid.NewUUID(), // Non-existent
		Emoji:     "👍",
		UserID:    uuid.NewUUID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}

func TestAddReactionUseCase_EmptyEmoji(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddReactionUseCase(messageRepo, eventBus)

	cmd := message.AddReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "", // Empty
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrInvalidEmoji)
	assert.Nil(t, result.Value)
}

// RemoveReaction tests

func TestRemoveReactionUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message and add reaction
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	err = msg.AddReaction(userID, "👍")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewRemoveReactionUseCase(messageRepo, eventBus)

	cmd := message.RemoveReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.False(t, result.Value.HasReaction(userID, "👍"))
	assert.Equal(t, 0, result.Value.GetReactionCount("👍"))

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestRemoveReactionUseCase_ReactionNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	userID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message without reactions
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewRemoveReactionUseCase(messageRepo, eventBus)

	cmd := message.RemoveReactionCommand{
		MessageID: msg.ID(),
		Emoji:     "👍",
		UserID:    userID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestRemoveReactionUseCase_MessageNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewRemoveReactionUseCase(messageRepo, eventBus)

	cmd := message.RemoveReactionCommand{
		MessageID: uuid.NewUUID(), // Non-existent
		Emoji:     "👍",
		UserID:    uuid.NewUUID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}
