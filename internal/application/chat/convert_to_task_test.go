package chat_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToTaskUseCase_Success_FromDiscussion tests converting Discussion to Task
func TestConvertToTaskUseCase_Success_FromDiscussion(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	// Create Discussion chat using helper
	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeDiscussion, "", workspaceID, creatorID)
	chatID := createdChat.ID()

	// Act
	convertUseCase := chat.NewConvertToTaskUseCase(eventStore)
	convertCmd := chat.ConvertToTaskCommand{
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
	workspaceID := generateUUID(t)

	// Create Task chat using helper
	createdChat := createTestChatWithParams(
		t,
		eventStore,
		domainChat.TypeTask,
		"Existing Task",
		workspaceID,
		creatorID,
	)
	chatID := createdChat.ID()

	// Try to convert to Task again
	convertUseCase := chat.NewConvertToTaskUseCase(eventStore)
	convertCmd := chat.ConvertToTaskCommand{
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
	convertUseCase := chat.NewConvertToTaskUseCase(eventStore)

	convertCmd := chat.ConvertToTaskCommand{
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
	convertUseCase := chat.NewConvertToTaskUseCase(eventStore)

	convertCmd := chat.ConvertToTaskCommand{
		ChatID:      "",
		Title:       "Task Title",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
