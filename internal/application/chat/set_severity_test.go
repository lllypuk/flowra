package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestSetSeverityUseCase_Success_Minor tests setting Minor severity
func TestSetSeverityUseCase_Success_Minor(t *testing.T) {
	testSetSeveritySuccess(t, "Minor")
}

// TestSetSeverityUseCase_Success_Major tests setting Major severity
func TestSetSeverityUseCase_Success_Major(t *testing.T) {
	testSetSeveritySuccess(t, "Major")
}

// TestSetSeverityUseCase_Success_Critical tests setting Critical severity
func TestSetSeverityUseCase_Success_Critical(t *testing.T) {
	testSetSeveritySuccess(t, "Critical")
}

// TestSetSeverityUseCase_Success_Blocker tests setting Blocker severity
func TestSetSeverityUseCase_Success_Blocker(t *testing.T) {
	testSetSeveritySuccess(t, "Blocker")
}

func testSetSeveritySuccess(t *testing.T, severity string) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeBug, "Test Bug", workspaceID, creatorID, true)

	setSeverityUseCase := chat.NewSetSeverityUseCase(eventStore)
	setSeverityCmd := chat.SetSeverityCommand{
		ChatID:   createdChat.ID(),
		Severity: severity,
		SetBy:    creatorID,
	}
	result, err := setSeverityUseCase.Execute(testContext(), setSeverityCmd)

	executeAndAssertSuccess(t, err)
	assert.Equal(t, severity, result.Value.Severity())
}

// TestSetSeverityUseCase_Error_OnlyForBugs tests error when used on non-Bug chat
func TestSetSeverityUseCase_Error_OnlyForBugs(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)
	workspaceID := generateUUID(t)

	createdChat := createTestChatWithParams(t, eventStore, domainChat.TypeTask, "Test Task", workspaceID, creatorID, true)

	setSeverityUseCase := chat.NewSetSeverityUseCase(eventStore)
	setSeverityCmd := chat.SetSeverityCommand{
		ChatID:   createdChat.ID(),
		Severity: "Critical",
		SetBy:    creatorID,
	}
	result, err := setSeverityUseCase.Execute(testContext(), setSeverityCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestSetSeverityUseCase_ValidationError_InvalidSeverity tests validation error
func TestSetSeverityUseCase_ValidationError_InvalidSeverity(t *testing.T) {
	eventStore := newTestEventStore()
	setSeverityUseCase := chat.NewSetSeverityUseCase(eventStore)

	setSeverityCmd := chat.SetSeverityCommand{
		ChatID:   generateUUID(t),
		Severity: "InvalidSeverity",
		SetBy:    generateUUID(t),
	}
	result, err := setSeverityUseCase.Execute(testContext(), setSeverityCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
