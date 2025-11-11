package chat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestAddParticipantUseCase_Success_AddAdmin tests adding an admin participant
func TestAddParticipantUseCase_Success_AddAdmin(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userID := generateUUID(t)

	cmd := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleAdmin,
		AddedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	assert.True(t, result.Value.IsParticipantAdmin(userID))
}

// TestAddParticipantUseCase_Error_AlreadyParticipant tests error when participant already exists
func TestAddParticipantUseCase_Error_AlreadyParticipant(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userID := generateUUID(t)

	// Add participant first time
	addUseCase := chat.NewAddParticipantUseCase(eventStore)
	cmd1 := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}
	result1, err := addUseCase.Execute(testContext(), cmd1)
	require.NoError(t, err)
	require.NotNil(t, result1.Value)
	assert.True(t, result1.Value.HasParticipant(userID))

	// Try to add same participant again with fresh UseCase instance
	addUseCase2 := chat.NewAddParticipantUseCase(eventStore)
	cmd2 := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}

	// Act
	result, err := addUseCase2.Execute(testContext(), cmd2)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAddParticipantUseCase_ValidationError_InvalidChatID tests validation error for invalid ChatID
func TestAddParticipantUseCase_ValidationError_InvalidChatID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	cmd := chat.AddParticipantCommand{
		ChatID:  uuid.UUID(""),
		UserID:  generateUUID(t),
		Role:    domainChat.RoleMember,
		AddedBy: generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAddParticipantUseCase_ValidationError_InvalidUserID tests validation error for invalid UserID
func TestAddParticipantUseCase_ValidationError_InvalidUserID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	cmd := chat.AddParticipantCommand{
		ChatID:  generateUUID(t),
		UserID:  uuid.UUID(""),
		Role:    domainChat.RoleMember,
		AddedBy: generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAddParticipantUseCase_EventStoreError_LoadFails tests handling of EventStore load error
func TestAddParticipantUseCase_EventStoreError_LoadFails(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	cmd := chat.AddParticipantCommand{
		ChatID:  generateUUID(t),
		UserID:  generateUUID(t),
		Role:    domainChat.RoleMember,
		AddedBy: generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAddParticipantUseCase_Success_AddMember tests adding a member participant
func TestAddParticipantUseCase_Success_AddMember(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userID := generateUUID(t)

	cmd := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	assert.True(t, result.Value.HasParticipant(userID))
	assert.False(t, result.Value.IsParticipantAdmin(userID))
}

// TestAddParticipantUseCase_Success_MultipleParticipants tests adding multiple participants sequentially
func TestAddParticipantUseCase_Success_MultipleParticipants(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create chat
	createUseCase := chat.NewCreateChatUseCase(eventStore)
	createCmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	addUseCase := chat.NewAddParticipantUseCase(eventStore)

	// Act & Assert: Add first participant
	user1ID := generateUUID(t)
	cmd1 := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  user1ID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}
	result1, err := addUseCase.Execute(testContext(), cmd1)
	require.NoError(t, err)
	assert.True(t, result1.Value.HasParticipant(user1ID))

	// Add second participant with fresh UseCase
	addUseCase2 := chat.NewAddParticipantUseCase(eventStore)
	user2ID := generateUUID(t)
	cmd2 := chat.AddParticipantCommand{
		ChatID:  chatID,
		UserID:  user2ID,
		Role:    domainChat.RoleAdmin,
		AddedBy: creatorID,
	}
	result2, err := addUseCase2.Execute(testContext(), cmd2)
	require.NoError(t, err)
	assert.True(t, result2.Value.HasParticipant(user1ID))
	assert.True(t, result2.Value.HasParticipant(user2ID))
	assert.True(t, result2.Value.IsParticipantAdmin(user2ID))
}

// TestAddParticipantUseCase_ValidationError_InvalidAddedBy tests validation error for invalid AddedBy
func TestAddParticipantUseCase_ValidationError_InvalidAddedBy(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewAddParticipantUseCase(eventStore)

	cmd := chat.AddParticipantCommand{
		ChatID:  generateUUID(t),
		UserID:  generateUUID(t),
		Role:    domainChat.RoleMember,
		AddedBy: "", // Invalid
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}
