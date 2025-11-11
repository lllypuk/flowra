package chat_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
)

// TestRemoveParticipantUseCase_Success tests successful participant removal
func TestRemoveParticipantUseCase_Success(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()

	// Create chat and add participant
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   generateUUID(t),
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()
	creatorID := createResult.Value.CreatedBy()

	// Add participant
	userID := generateUUID(t)
	addUseCase := chat.NewAddParticipantUseCase(eventStore)
	addCmd := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}
	_, err = addUseCase.Execute(testContext(), addCmd)
	require.NoError(t, err)

	// Act
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)
	removeCmd := chat.RemoveParticipantCommand{
		ChatID:    chatID,
		UserID:    userID,
		RemovedBy: creatorID,
	}
	result, err := removeUseCase.Execute(testContext(), removeCmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assert.False(t, result.Value.HasParticipant(userID))
}

// TestRemoveParticipantUseCase_ValidationError_InvalidChatID tests validation error
func TestRemoveParticipantUseCase_ValidationError_InvalidChatID(t *testing.T) {
	eventStore := newTestEventStore()
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)

	cmd := chat.RemoveParticipantCommand{
		ChatID:    "",
		UserID:    generateUUID(t),
		RemovedBy: generateUUID(t),
	}

	result, err := removeUseCase.Execute(testContext(), cmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRemoveParticipantUseCase_ValidationError_InvalidUserID tests validation error
func TestRemoveParticipantUseCase_ValidationError_InvalidUserID(t *testing.T) {
	eventStore := newTestEventStore()
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)

	cmd := chat.RemoveParticipantCommand{
		ChatID:    generateUUID(t),
		UserID:    "",
		RemovedBy: generateUUID(t),
	}

	result, err := removeUseCase.Execute(testContext(), cmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRemoveParticipantUseCase_Error_NotParticipant tests error for non-existent participant
func TestRemoveParticipantUseCase_Error_NotParticipant(t *testing.T) {
	eventStore := newTestEventStore()

	// Create chat
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   generateUUID(t),
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)
	cmd := chat.RemoveParticipantCommand{
		ChatID:    chatID,
		UserID:    generateUUID(t),
		RemovedBy: createResult.Value.CreatedBy(),
	}

	result, err := removeUseCase.Execute(testContext(), cmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRemoveParticipantUseCase_Success_SelfRemove tests user removing themselves
func TestRemoveParticipantUseCase_Success_SelfRemove(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()

	// Create chat and add participant
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   generateUUID(t),
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()
	creatorID := createResult.Value.CreatedBy()

	// Add participant
	userID := generateUUID(t)
	addUseCase := chat.NewAddParticipantUseCase(eventStore)
	addCmd := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}
	_, err = addUseCase.Execute(testContext(), addCmd)
	require.NoError(t, err)

	// Act
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)
	removeCmd := chat.RemoveParticipantCommand{
		ChatID:    chatID,
		UserID:    userID,
		RemovedBy: userID, // User removes themselves
	}
	result, err := removeUseCase.Execute(testContext(), removeCmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assert.False(t, result.Value.HasParticipant(userID))
}

// TestRemoveParticipantUseCase_Error_CannotRemoveCreator tests error when trying to remove chat creator
func TestRemoveParticipantUseCase_Error_CannotRemoveCreator(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()

	// Create chat
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   generateUUID(t),
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()
	creatorID := createResult.Value.CreatedBy()

	// Add another participant
	userID := generateUUID(t)
	addUseCase := chat.NewAddParticipantUseCase(eventStore)
	addCmd := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleAdmin,
		AddedBy: creatorID,
	}
	_, err = addUseCase.Execute(testContext(), addCmd)
	require.NoError(t, err)

	// Act: Try to remove creator as admin
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)
	removeCmd := chat.RemoveParticipantCommand{
		ChatID:    chatID,
		UserID:    creatorID,
		RemovedBy: userID,
	}
	result, err := removeUseCase.Execute(testContext(), removeCmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestRemoveParticipantUseCase_ValidationError_InvalidRemovedBy tests validation error for invalid RemovedBy
func TestRemoveParticipantUseCase_ValidationError_InvalidRemovedBy(t *testing.T) {
	eventStore := newTestEventStore()
	removeUseCase := chat.NewRemoveParticipantUseCase(eventStore)

	cmd := chat.RemoveParticipantCommand{
		ChatID:    generateUUID(t),
		UserID:    generateUUID(t),
		RemovedBy: "",
	}

	result, err := removeUseCase.Execute(testContext(), cmd)

	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
