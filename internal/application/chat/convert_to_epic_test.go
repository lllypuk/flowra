package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToEpicUseCase_Success tests converting Discussion to Epic
func TestConvertToEpicUseCase_Success(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeDiscussion, "", workspaceID, creatorID, true)

	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)
	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      createdChat.ID(),
		Title:       "Q4 Epic",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertSuccess(t, err)
	assertChatType(t, result.Value, domainChat.TypeEpic)
	assertChatTitle(t, result.Value, "Q4 Epic")
}

// TestConvertToEpicUseCase_Error_AlreadyEpic tests error when already Epic
func TestConvertToEpicUseCase_Error_AlreadyEpic(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeEpic, "Existing Epic", workspaceID, creatorID, true)

	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)
	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      createdChat.ID(),
		Title:       "Another Epic",
		ConvertedBy: creatorID,
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToEpicUseCase_ValidationError_EmptyTitle tests validation error
func TestConvertToEpicUseCase_ValidationError_EmptyTitle(t *testing.T) {
	eventStore := newTestEventStore()
	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)

	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      generateUUID(t),
		Title:       "",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestConvertToEpicUseCase_ValidationError_InvalidConvertedBy tests validation error for invalid ConvertedBy
func TestConvertToEpicUseCase_ValidationError_InvalidConvertedBy(t *testing.T) {
	eventStore := newTestEventStore()
	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)

	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      generateUUID(t),
		Title:       "Epic Title",
		ConvertedBy: "", // Invalid
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
