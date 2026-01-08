package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestChangeStatusUseCase_Success_TaskStatus tests changing Task status
func TestChangeStatusUseCase_Success_TaskStatus(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeTask, "Test Task", workspaceID, creatorID, true)

	changeUseCase := chat.NewChangeStatusUseCase(eventStore)
	changeCmd := chat.ChangeStatusCommand{
		ChatID:    createdChat.ID(),
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
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeBug, "Test Bug", workspaceID, creatorID, true)

	changeUseCase := chat.NewChangeStatusUseCase(eventStore)
	changeCmd := chat.ChangeStatusCommand{
		ChatID:    createdChat.ID(),
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
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeEpic, "Test Epic", workspaceID, creatorID, true)

	changeUseCase := chat.NewChangeStatusUseCase(eventStore)
	changeCmd := chat.ChangeStatusCommand{
		ChatID:    createdChat.ID(),
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
	changeUseCase := chat.NewChangeStatusUseCase(eventStore)

	changeCmd := chat.ChangeStatusCommand{
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
	changeUseCase := chat.NewChangeStatusUseCase(eventStore)

	changeCmd := chat.ChangeStatusCommand{
		ChatID:    "",
		Status:    "In Progress",
		ChangedBy: generateUUID(t),
	}
	result, err := changeUseCase.Execute(testContext(), changeCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
