package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestAddParticipantUseCase_Success_AddMember tests adding a member participant
func TestAddParticipantUseCase_Success_AddMember(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := NewAddParticipantUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userID := generateUUID(t)

	cmd := AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assert.True(t, result.Value.HasParticipant(userID))
	assert.False(t, result.Value.IsParticipantAdmin(userID))
	assert.Equal(t, 2, len(result.Value.Participants())) // creator + new participant
}

// TestAddParticipantUseCase_Success_AddAdmin tests adding an admin participant
func TestAddParticipantUseCase_Success_AddAdmin(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := NewAddParticipantUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}
	createResult, err := createUseCase.Execute(testContext(), createCmd)
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userID := generateUUID(t)

	cmd := AddParticipantCommand{
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
	useCase := NewAddParticipantUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create and save a chat first
	createUseCase := NewCreateChatUseCase(eventStore)
	createCmd := CreateChatCommand{
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
	cmd1 := AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}
	_, err = useCase.Execute(testContext(), cmd1)
	require.NoError(t, err)

	// Try to add same participant again
	cmd2 := AddParticipantCommand{
		ChatID:  chatID,
		UserID:  userID,
		Role:    domainChat.RoleMember,
		AddedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd2)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
}

// TestAddParticipantUseCase_ValidationError_InvalidChatID tests validation error for invalid ChatID
func TestAddParticipantUseCase_ValidationError_InvalidChatID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := NewAddParticipantUseCase(eventStore)

	cmd := AddParticipantCommand{
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
	useCase := NewAddParticipantUseCase(eventStore)

	cmd := AddParticipantCommand{
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
	useCase := NewAddParticipantUseCase(eventStore)

	cmd := AddParticipantCommand{
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
