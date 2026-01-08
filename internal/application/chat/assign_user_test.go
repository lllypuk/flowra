package chat_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestAssignUserUseCase_Success_AssignUser tests assigning a user
func TestAssignUserUseCase_Success_AssignUser(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(
		t,
		eventStore,
		domainChat.TypeTask,
		"Test Task",
		workspaceID,
		creatorID,
	)

	assigneeID := generateUUID(t)
	assignUseCase := chat.NewAssignUserUseCase(eventStore)
	assignCmd := chat.AssignUserCommand{
		ChatID:     createdChat.ID(),
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
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(
		t,
		eventStore,
		domainChat.TypeTask,
		"Test Task",
		workspaceID,
		creatorID,
	)

	// First assign
	assigneeID := generateUUID(t)
	assignUseCase := chat.NewAssignUserUseCase(eventStore)
	assignCmd := chat.AssignUserCommand{
		ChatID:     createdChat.ID(),
		AssigneeID: &assigneeID,
		AssignedBy: creatorID,
	}
	_, err := assignUseCase.Execute(testContext(), assignCmd)
	require.NoError(t, err)

	// Then unassign
	unassignCmd := chat.AssignUserCommand{
		ChatID:     createdChat.ID(),
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
	assignUseCase := chat.NewAssignUserUseCase(eventStore)

	assigneeID := generateUUID(t)
	assignCmd := chat.AssignUserCommand{
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
	assignUseCase := chat.NewAssignUserUseCase(eventStore)

	assigneeID := generateUUID(t)
	assignCmd := chat.AssignUserCommand{
		ChatID:     generateUUID(t),
		AssigneeID: &assigneeID,
		AssignedBy: generateUUID(t),
	}
	result, err := assignUseCase.Execute(testContext(), assignCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
