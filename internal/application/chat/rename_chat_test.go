package chat_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestRenameChatUseCase_Success tests successful chat rename
func TestRenameChatUseCase_Success(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		Title:       "Old Title",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	renameUseCase := chat.NewRenameChatUseCase(eventStore)
	renameCmd := chat.RenameChatCommand{
		ChatID:    createResult.Value.ID(),
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
