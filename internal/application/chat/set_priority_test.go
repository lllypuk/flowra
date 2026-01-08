package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestSetPriorityUseCase_Success_Low tests setting Low priority
func TestSetPriorityUseCase_Success_Low(t *testing.T) {
	testSetPrioritySuccess(t, "Low")
}

// TestSetPriorityUseCase_Success_Medium tests setting Medium priority
func TestSetPriorityUseCase_Success_Medium(t *testing.T) {
	testSetPrioritySuccess(t, "Medium")
}

// TestSetPriorityUseCase_Success_High tests setting High priority
func TestSetPriorityUseCase_Success_High(t *testing.T) {
	testSetPrioritySuccess(t, "High")
}

// TestSetPriorityUseCase_Success_Critical tests setting Critical priority
func TestSetPriorityUseCase_Success_Critical(t *testing.T) {
	testSetPrioritySuccess(t, "Critical")
}

func testSetPrioritySuccess(t *testing.T, priority string) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeTask, "Test Task", workspaceID, creatorID, true)

	setPriorityUseCase := chat.NewSetPriorityUseCase(eventStore)
	setPriorityCmd := chat.SetPriorityCommand{
		ChatID:   createdChat.ID(),
		Priority: priority,
		SetBy:    creatorID,
	}
	result, err := setPriorityUseCase.Execute(testContext(), setPriorityCmd)

	executeAndAssertSuccess(t, err)
	assert.Equal(t, priority, result.Value.Priority())
}

// TestSetPriorityUseCase_ValidationError_InvalidPriority tests validation error
func TestSetPriorityUseCase_ValidationError_InvalidPriority(t *testing.T) {
	eventStore := newTestEventStore()
	setPriorityUseCase := chat.NewSetPriorityUseCase(eventStore)

	setPriorityCmd := chat.SetPriorityCommand{
		ChatID:   generateUUID(t),
		Priority: "InvalidPriority",
		SetBy:    generateUUID(t),
	}
	result, err := setPriorityUseCase.Execute(testContext(), setPriorityCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestSetPriorityUseCase_ValidationError_InvalidChatID tests validation error
func TestSetPriorityUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	setPriorityUseCase := chat.NewSetPriorityUseCase(eventStore)

	setPriorityCmd := chat.SetPriorityCommand{
		ChatID:   "",
		Priority: "High",
		SetBy:    generateUUID(t),
	}
	result, err := setPriorityUseCase.Execute(testContext(), setPriorityCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
