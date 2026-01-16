package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToBugUseCase_Success tests converting Discussion to Bug
func TestConvertToBugUseCase_Success(t *testing.T) {
	chatRepo := newTestChatRepo()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithRepo(t, chatRepo, domainChat.TypeDiscussion, "", workspaceID, creatorID)

	convertUseCase := chat.NewConvertToBugUseCase(chatRepo)
	convertCmd := chat.ConvertToBugCommand{
		ChatID:      createdChat.ID(),
		Title:       "Critical Bug",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertSuccess(t, err)
	assertChatType(t, result.Value, domainChat.TypeBug)
	assertChatTitle(t, result.Value, "Critical Bug")
}

// TestConvertToBugUseCase_Error_AlreadyBug tests error when already Bug
func TestConvertToBugUseCase_Error_AlreadyBug(t *testing.T) {
	chatRepo := newTestChatRepo()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithRepo(
		t,
		chatRepo,
		domainChat.TypeBug,
		"Existing Bug",
		workspaceID,
		creatorID,
	)

	convertUseCase := chat.NewConvertToBugUseCase(chatRepo)
	convertCmd := chat.ConvertToBugCommand{
		ChatID:      createdChat.ID(),
		Title:       "Another Bug",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToBugUseCase_ValidationError_EmptyTitle tests validation error
func TestConvertToBugUseCase_ValidationError_EmptyTitle(t *testing.T) {
	chatRepo := newTestChatRepo()
	convertUseCase := chat.NewConvertToBugUseCase(chatRepo)

	convertCmd := chat.ConvertToBugCommand{
		ChatID:      generateUUID(t),
		Title:       "",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToBugUseCase_Error_ChatNotFound tests error when chat not found
func TestConvertToBugUseCase_Error_ChatNotFound(t *testing.T) {
	chatRepo := newTestChatRepo()
	convertUseCase := chat.NewConvertToBugUseCase(chatRepo)

	convertCmd := chat.ConvertToBugCommand{
		ChatID:      generateUUID(t),
		Title:       "Bug Title",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
