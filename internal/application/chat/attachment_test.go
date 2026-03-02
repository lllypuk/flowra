package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	domainchat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestAddAttachmentUseCase_Success(t *testing.T) {
	chatRepo := newTestChatRepo()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithRepo(t, chatRepo, domainchat.TypeTask, "Task", workspaceID, creatorID)

	useCase := chatapp.NewAddAttachmentUseCase(chatRepo)
	cmd := chatapp.AddAttachmentCommand{
		ChatID:   createdChat.ID(),
		FileID:   uuid.NewUUID(),
		FileName: "report.pdf",
		FileSize: 1024,
		MimeType: "application/pdf",
		AddedBy:  creatorID,
	}

	result, err := useCase.Execute(testContext(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result.Value)
	require.Len(t, result.Value.Attachments(), 1)
	assert.Equal(t, "report.pdf", result.Value.Attachments()[0].FileName())
}

func TestAddAttachmentUseCase_Error_DiscussionChat(t *testing.T) {
	chatRepo := newTestChatRepo()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithRepo(t, chatRepo, domainchat.TypeDiscussion, "", workspaceID, creatorID)

	useCase := chatapp.NewAddAttachmentUseCase(chatRepo)
	cmd := chatapp.AddAttachmentCommand{
		ChatID:   createdChat.ID(),
		FileID:   uuid.NewUUID(),
		FileName: "report.pdf",
		FileSize: 1024,
		MimeType: "application/pdf",
		AddedBy:  creatorID,
	}

	result, err := useCase.Execute(testContext(), cmd)
	require.Error(t, err)
	assert.Nil(t, result.Value)
}

func TestRemoveAttachmentUseCase_Success(t *testing.T) {
	chatRepo := newTestChatRepo()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithRepo(t, chatRepo, domainchat.TypeTask, "Task", workspaceID, creatorID)
	fileID := uuid.NewUUID()

	addUseCase := chatapp.NewAddAttachmentUseCase(chatRepo)
	_, addErr := addUseCase.Execute(testContext(), chatapp.AddAttachmentCommand{
		ChatID:   createdChat.ID(),
		FileID:   fileID,
		FileName: "report.pdf",
		FileSize: 1024,
		MimeType: "application/pdf",
		AddedBy:  creatorID,
	})
	require.NoError(t, addErr)

	removeUseCase := chatapp.NewRemoveAttachmentUseCase(chatRepo)
	result, err := removeUseCase.Execute(testContext(), chatapp.RemoveAttachmentCommand{
		ChatID:    createdChat.ID(),
		FileID:    fileID,
		RemovedBy: creatorID,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Value)
	assert.Empty(t, result.Value.Attachments())
}

func TestRemoveAttachmentUseCase_ValidationError(t *testing.T) {
	chatRepo := newTestChatRepo()
	useCase := chatapp.NewRemoveAttachmentUseCase(chatRepo)

	result, err := useCase.Execute(testContext(), chatapp.RemoveAttachmentCommand{
		ChatID:    "",
		FileID:    uuid.NewUUID(),
		RemovedBy: uuid.NewUUID(),
	})

	require.Error(t, err)
	assert.Nil(t, result.Value)
}
