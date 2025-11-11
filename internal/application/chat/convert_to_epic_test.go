package chat_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToEpicUseCase_Success tests converting Discussion to Epic
func TestConvertToEpicUseCase_Success(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)
	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      createResult.Value.ID(),
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

	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeEpic,
		Title:       "Existing Epic",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	convertUseCase := chat.NewConvertToEpicUseCase(eventStore)
	convertCmd := chat.ConvertToEpicCommand{
		ChatID:      createResult.Value.ID(),
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
