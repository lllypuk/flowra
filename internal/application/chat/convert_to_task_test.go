package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToTaskUseCase_Success_FromDiscussion tests converting Discussion to Task
func TestConvertToTaskUseCase_Success_FromDiscussion(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	// Create Discussion chat
	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	// Act
	convertUseCase := NewConvertToTaskUseCase(eventStore)
	convertCmd := ConvertToTaskCommand{
		ChatID:      chatID,
		Title:       "New Task Title",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeTask)
	assertChatTitle(t, result.Value, "New Task Title")
}

// TestConvertToTaskUseCase_Error_AlreadyTask tests error when chat is already Task
func TestConvertToTaskUseCase_Error_AlreadyTask(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	// Create Task chat
	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		Title:       "Existing Task",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	// Try to convert to Task again
	convertUseCase := NewConvertToTaskUseCase(eventStore)
	convertCmd := ConvertToTaskCommand{
		ChatID:      chatID,
		Title:       "Another Title",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToTaskUseCase_ValidationError_EmptyTitle tests validation error
func TestConvertToTaskUseCase_ValidationError_EmptyTitle(t *testing.T) {
	eventStore := newTestEventStore()
	convertUseCase := NewConvertToTaskUseCase(eventStore)

	convertCmd := ConvertToTaskCommand{
		ChatID:      generateUUID(t),
		Title:       "",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToTaskUseCase_ValidationError_InvalidChatID tests validation error
func TestConvertToTaskUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	convertUseCase := NewConvertToTaskUseCase(eventStore)

	convertCmd := ConvertToTaskCommand{
		ChatID:      "",
		Title:       "Task Title",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
