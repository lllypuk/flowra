package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestConvertToEpicUseCase_Success tests converting Discussion to Epic
func TestConvertToEpicUseCase_Success(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	convertUseCase := NewConvertToEpicUseCase(eventStore)
	convertCmd := ConvertToEpicCommand{
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

	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeEpic,
		Title:       "Existing Epic",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	convertUseCase := NewConvertToEpicUseCase(eventStore)
	convertCmd := ConvertToEpicCommand{
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
	convertUseCase := NewConvertToEpicUseCase(eventStore)

	convertCmd := ConvertToEpicCommand{
		ChatID:      generateUUID(t),
		Title:       "",
		ConvertedBy: generateUUID(t),
	}
	result, err := convertUseCase.Execute(testContext(), convertCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
