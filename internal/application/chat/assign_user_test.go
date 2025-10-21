package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestAssignUserUseCase_Success_AssignUser tests assigning a user
func TestAssignUserUseCase_Success_AssignUser(t *testing.T) {
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

	assigneeID := generateUUID(t)
	assignUseCase := NewAssignUserUseCase(eventStore)
	assignCmd := AssignUserCommand{
		ChatID:     createResult.Value.ID(),
		AssigneeID: &assigneeID,
		AssignedBy: creatorID,
	}
	result, err := assignUseCase.Execute(testContext(), assignCmd)

	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value.AssigneeID())
	assert.Equal(t, assigneeID, *result.Value.AssigneeID())
}

// TestAssignUserUseCase_Success_UnassignUser tests removing assignment
func TestAssignUserUseCase_Success_UnassignUser(t *testing.T) {
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

	// First assign
	assigneeID := generateUUID(t)
	assignUseCase := NewAssignUserUseCase(eventStore)
	assignCmd := AssignUserCommand{
		ChatID:     createResult.Value.ID(),
		AssigneeID: &assigneeID,
		AssignedBy: creatorID,
	}
	_, err = assignUseCase.Execute(testContext(), assignCmd)
	require.NoError(t, err)

	// Then unassign
	unassignCmd := AssignUserCommand{
		ChatID:     createResult.Value.ID(),
		AssigneeID: nil,
		AssignedBy: creatorID,
	}
	result, err := assignUseCase.Execute(testContext(), unassignCmd)

	executeAndAssertSuccess(t, err)
	assert.Nil(t, result.Value.AssigneeID())
}

// TestAssignUserUseCase_ValidationError_InvalidChatID tests validation error
func TestAssignUserUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	assignUseCase := NewAssignUserUseCase(eventStore)

	assigneeID := generateUUID(t)
	assignCmd := AssignUserCommand{
		ChatID:     "",
		AssigneeID: &assigneeID,
		AssignedBy: generateUUID(t),
	}
	result, err := assignUseCase.Execute(testContext(), assignCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAssignUserUseCase_Error_ChatNotFound tests error when chat not found
func TestAssignUserUseCase_Error_ChatNotFound(t *testing.T) {
	eventStore := newTestEventStore()
	assignUseCase := NewAssignUserUseCase(eventStore)

	assigneeID := generateUUID(t)
	assignCmd := AssignUserCommand{
		ChatID:     generateUUID(t),
		AssigneeID: &assigneeID,
		AssignedBy: generateUUID(t),
	}
	result, err := assignUseCase.Execute(testContext(), assignCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
