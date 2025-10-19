package message_test

import (
	"context"
	"testing"

	"github.com/lllypuk/teams-up/internal/application/message"
	domainMessage "github.com/lllypuk/teams-up/internal/domain/message"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessageUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	// Setup chat with participant
	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{authorID})

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "Hello, world!",
		AuthorID:        authorID,
		ParentMessageID: "", // zero UUID
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Equal(t, cmd.Content, result.Value.Content())
	assert.Equal(t, cmd.ChatID, result.Value.ChatID())
	assert.Equal(t, cmd.AuthorID, result.Value.AuthorID())
	assert.True(t, result.Value.ParentMessageID().IsZero())

	// Check message was saved
	assert.Len(t, messageRepo.Messages, 1)

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestSendMessageUseCase_WithParentMessage(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{authorID})

	// Create parent message
	parentMsg, err := domainMessage.NewMessage(chatID, authorID, "Parent message", "")
	require.NoError(t, err)
	messageRepo.Messages[parentMsg.ID()] = parentMsg

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "Reply message",
		AuthorID:        authorID,
		ParentMessageID: parentMsg.ID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Equal(t, parentMsg.ID(), result.Value.ParentMessageID())
	assert.True(t, result.Value.IsReply())
}

func TestSendMessageUseCase_NotParticipant(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{otherUserID}) // Другой пользователь

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "Hello",
		AuthorID:        uuid.NewUUID(), // Не участник
		ParentMessageID: "",
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrNotChatParticipant)
	assert.Nil(t, result.Value)
}

func TestSendMessageUseCase_ChatNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          uuid.NewUUID(), // Несуществующий чат
		Content:         "Hello",
		AuthorID:        uuid.NewUUID(),
		ParentMessageID: "",
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrChatNotFound)
	assert.Nil(t, result.Value)
}

func TestSendMessageUseCase_EmptyContent(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{authorID})

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "", // Пустой контент
		AuthorID:        authorID,
		ParentMessageID: "",
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrEmptyContent)
	assert.Nil(t, result.Value)
}

func TestSendMessageUseCase_ContentTooLong(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{authorID})

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	// Создаем контент > message.MaxContentLength
	longContent := make([]byte, message.MaxContentLength+1)
	for i := range longContent {
		longContent[i] = 'a'
	}

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         string(longContent),
		AuthorID:        authorID,
		ParentMessageID: "",
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrContentTooLong)
	assert.Nil(t, result.Value)
}

func TestSendMessageUseCase_ParentNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID, []uuid.UUID{authorID})

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID,
		Content:         "Reply",
		AuthorID:        authorID,
		ParentMessageID: uuid.NewUUID(), // Несуществующий parent
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrParentNotFound)
	assert.Nil(t, result.Value)
}

func TestSendMessageUseCase_ParentInDifferentChat(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	chatRepo := message.NewMockChatRepository()
	eventBus := message.NewMockEventBus()

	chatID1 := uuid.NewUUID()
	chatID2 := uuid.NewUUID()
	authorID := uuid.NewUUID()
	chatRepo.AddChat(chatID1, []uuid.UUID{authorID})
	chatRepo.AddChat(chatID2, []uuid.UUID{authorID})

	// Create parent message in different chat
	parentMsg, err := domainMessage.NewMessage(chatID1, authorID, "Parent in chat1", "")
	require.NoError(t, err)
	messageRepo.Messages[parentMsg.ID()] = parentMsg

	useCase := message.NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

	cmd := message.SendMessageCommand{
		ChatID:          chatID2, // Другой чат
		Content:         "Reply",
		AuthorID:        authorID,
		ParentMessageID: parentMsg.ID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrParentInDifferentChat)
	assert.Nil(t, result.Value)
}
