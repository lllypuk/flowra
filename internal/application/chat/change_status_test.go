package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestChangeStatusUseCase_Success_TaskStatus tests changing Task status
func TestChangeStatusUseCase_Success_TaskStatus(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		Title:       "Test Task",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	changeUseCase := NewChangeStatusUseCase(eventStore)
	changeCmd := ChangeStatusCommand{
		ChatID:    createResult.Value.ID(),
		Status:    "In Progress",
		ChangedBy: creatorID,
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertSuccess(t, err)
	assertChatStatus(t, result.Value, "In Progress")
}

// TestChangeStatusUseCase_Success_BugStatus tests changing Bug status
func TestChangeStatusUseCase_Success_BugStatus(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeBug,
		Title:       "Test Bug",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	changeUseCase := NewChangeStatusUseCase(eventStore)
	changeCmd := ChangeStatusCommand{
		ChatID:    createResult.Value.ID(),
		Status:    "Fixed",
		ChangedBy: creatorID,
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertSuccess(t, err)
	assertChatStatus(t, result.Value, "Fixed")
}

// TestChangeStatusUseCase_Success_EpicStatus tests changing Epic status
func TestChangeStatusUseCase_Success_EpicStatus(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeEpic,
		Title:       "Test Epic",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	changeUseCase := NewChangeStatusUseCase(eventStore)
	changeCmd := ChangeStatusCommand{
		ChatID:    createResult.Value.ID(),
		Status:    "In Progress",
		ChangedBy: creatorID,
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertSuccess(t, err)
	assertChatStatus(t, result.Value, "In Progress")
}

// TestChangeStatusUseCase_ValidationError_EmptyStatus tests validation error
func TestChangeStatusUseCase_ValidationError_EmptyStatus(t *testing.T) {
	eventStore := newTestEventStore()
	changeUseCase := NewChangeStatusUseCase(eventStore)

	changeCmd := ChangeStatusCommand{
		ChatID:    generateUUID(t),
		Status:    "",
		ChangedBy: generateUUID(t),
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestChangeStatusUseCase_ValidationError_InvalidChatID tests validation error
func TestChangeStatusUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	changeUseCase := NewChangeStatusUseCase(eventStore)

	changeCmd := ChangeStatusCommand{
		ChatID:    "",
		Status:    "In Progress",
		ChangedBy: generateUUID(t),
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
