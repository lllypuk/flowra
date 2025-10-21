package chat_test

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestSetDueDateUseCase_Success_SetFutureDate tests setting a future due date
func TestSetDueDateUseCase_Success_SetFutureDate(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		Title:       "Test Task",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	futureDate := time.Now().AddDate(0, 0, 7) // 7 days in future
	setDueDateUseCase := chat.NewSetDueDateUseCase(eventStore)
	setDueDateCmd := chat.SetDueDateCommand{
		ChatID:  createResult.Value.ID(),
		DueDate: &futureDate,
		SetBy:   creatorID,
	}
	result, err := setDueDateUseCase.Execute(testContext(), setDueDateCmd)

	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value.DueDate())
}

// TestSetDueDateUseCase_Success_ClearDueDate tests clearing due date
func TestSetDueDateUseCase_Success_ClearDueDate(t *testing.T) {
	eventStore := newTestEventStore()
	creatorID := generateUUID(t)

	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		Title:       "Test Task",
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)

	// First set a due date
	futureDate := time.Now().AddDate(0, 0, 7)
	setDueDateUseCase := chat.NewSetDueDateUseCase(eventStore)
	setDueDateCmd := chat.SetDueDateCommand{
		ChatID:  createResult.Value.ID(),
		DueDate: &futureDate,
		SetBy:   creatorID,
	}
	_, err = setDueDateUseCase.Execute(testContext(), setDueDateCmd)
	require.NoError(t, err)

	// Then clear it
	clearCmd := chat.SetDueDateCommand{
		ChatID:  createResult.Value.ID(),
		DueDate: nil,
		SetBy:   creatorID,
	}
	result, err := setDueDateUseCase.Execute(testContext(), clearCmd)

	executeAndAssertSuccess(t, err)
	assert.Nil(t, result.Value.DueDate())
}

// TestSetDueDateUseCase_ValidationError_InvalidChatID tests validation error
func TestSetDueDateUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	setDueDateUseCase := chat.NewSetDueDateUseCase(eventStore)

	futureDate := time.Now().AddDate(0, 0, 7)
	setDueDateCmd := chat.SetDueDateCommand{
		ChatID:  "",
		DueDate: &futureDate,
		SetBy:   generateUUID(t),
	}
	result, err := setDueDateUseCase.Execute(testContext(), setDueDateCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestSetDueDateUseCase_Error_ChatNotFound tests error when chat not found
func TestSetDueDateUseCase_Error_ChatNotFound(t *testing.T) {
	eventStore := newTestEventStore()
	setDueDateUseCase := chat.NewSetDueDateUseCase(eventStore)

	futureDate := time.Now().AddDate(0, 0, 7)
	setDueDateCmd := chat.SetDueDateCommand{
		ChatID:  generateUUID(t),
		DueDate: &futureDate,
		SetBy:   generateUUID(t),
	}
	result, err := setDueDateUseCase.Execute(testContext(), setDueDateCmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
