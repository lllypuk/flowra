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

func TestAddAttachmentUseCase_Success(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.NoError(t, err)
	assert.NotNil(t, result.Value)
	assert.Len(t, result.Value.Attachments(), 1)

	attachment := result.Value.Attachments()[0]
	assert.Equal(t, cmd.FileName, attachment.FileName())
	assert.Equal(t, cmd.FileSize, attachment.FileSize())
	assert.Equal(t, cmd.MimeType, attachment.MimeType())

	// Check event was published
	assert.Len(t, eventBus.Published, 1)
}

func TestAddAttachmentUseCase_MultipleAttachments(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	// Add first attachment
	cmd1 := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    authorID,
	}
	result1, err := useCase.Execute(context.Background(), cmd1)
	require.NoError(t, err)
	assert.Len(t, result1.Value.Attachments(), 1)

	// Add second attachment
	cmd2 := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "image.png",
		FileSize:  2048,
		MimeType:  "image/png",
		UserID:    authorID,
	}
	result2, err := useCase.Execute(context.Background(), cmd2)
	require.NoError(t, err)
	assert.Len(t, result2.Value.Attachments(), 2)
}

func TestAddAttachmentUseCase_NotAuthor(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	// Create message
	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    otherUserID, // Different user
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrNotAuthor)
	assert.Nil(t, result.Value)
}

func TestAddAttachmentUseCase_MessageDeleted(t *testing.T) {
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

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageDeleted)
	assert.Nil(t, result.Value)
}

func TestAddAttachmentUseCase_MessageNotFound(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: uuid.NewUUID(), // Non-existent
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    uuid.NewUUID(),
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrMessageNotFound)
	assert.Nil(t, result.Value)
}

func TestAddAttachmentUseCase_FileSizeTooLarge(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "huge-file.zip",
		FileSize:  message.MaxFileSize + 1, // Exceeds limit
		MimeType:  "application/zip",
		UserID:    authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrInvalidFileSize)
	assert.Nil(t, result.Value)
}

func TestAddAttachmentUseCase_InvalidFileName(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "", // Empty
		FileSize:  1024,
		MimeType:  "application/pdf",
		UserID:    authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrInvalidFileName)
	assert.Nil(t, result.Value)
}

func TestAddAttachmentUseCase_InvalidMimeType(t *testing.T) {
	messageRepo := message.NewMockMessageRepository()
	eventBus := message.NewMockEventBus()

	authorID := uuid.NewUUID()
	chatID := uuid.NewUUID()

	msg, err := domain.NewMessage(chatID, authorID, "Test message", "")
	require.NoError(t, err)
	messageRepo.Messages[msg.ID()] = msg

	useCase := message.NewAddAttachmentUseCase(messageRepo, eventBus)

	cmd := message.AddAttachmentCommand{
		MessageID: msg.ID(),
		FileID:    uuid.NewUUID(),
		FileName:  "document.pdf",
		FileSize:  1024,
		MimeType:  "", // Empty
		UserID:    authorID,
	}

	result, err := useCase.Execute(context.Background(), cmd)

	require.Error(t, err)
	require.ErrorIs(t, err, message.ErrInvalidMimeType)
	assert.Nil(t, result.Value)
}
