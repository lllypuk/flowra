package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestRenameChatUseCase_Success tests successful chat rename
func TestRenameChatUseCase_Success(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeTask, "Old Title", workspaceID, creatorID, true)

	renameUseCase := chat.NewRenameChatUseCase(eventStore)
	renameCmd := chat.RenameChatCommand{
		ChatID:    createdChat.ID(),
		NewTitle:  "New Title",
		RenamedBy: creatorID,
	}
	result, err := renameUseCase.Execute(testContext(), renameCmd)

	executeAndAssertSuccess(t, err)
	assertChatTitle(t, result.Value, "New Title")
}

// TestRenameChatUseCase_ValidationError_EmptyTitle tests validation error
func TestRenameChatUseCase_ValidationError_EmptyTitle(t *testing.T) {
	eventStore := newTestEventStore()
	renameUseCase := chat.NewRenameChatUseCase(eventStore)

	renameCmd := chat.RenameChatCommand{
		ChatID:    generateUUID(t),
		NewTitle:  "",
		RenamedBy: generateUUID(t),
	}
	result, err := renameUseCase.Execute(testContext(), renameCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRenameChatUseCase_ValidationError_InvalidChatID tests validation error
func TestRenameChatUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	renameUseCase := chat.NewRenameChatUseCase(eventStore)

	renameCmd := chat.RenameChatCommand{
		ChatID:    "",
		NewTitle:  "New Title",
		RenamedBy: generateUUID(t),
	}
	result, err := renameUseCase.Execute(testContext(), renameCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRenameChatUseCase_Error_ChatNotFound tests error when chat not found
func TestRenameChatUseCase_Error_ChatNotFound(t *testing.T) {
	eventStore := newTestEventStore()
	renameUseCase := chat.NewRenameChatUseCase(eventStore)

	renameCmd := chat.RenameChatCommand{
		ChatID:    generateUUID(t),
		NewTitle:  "New Title",
		RenamedBy: generateUUID(t),
	}
	result, err := renameUseCase.Execute(testContext(), renameCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
