package chat_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestCreateChatUseCase_Success_Discussion tests creating a Discussion chat
func TestCreateChatUseCase_Success_Discussion(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeDiscussion)
	assert.Equal(t, workspaceID, result.Value.WorkspaceID())
	assert.True(t, result.Value.IsPublic())
	assert.Equal(t, creatorID, result.Value.CreatedBy())

	// Check that events are created for Discussion chats
	// NewChat() generates: ChatCreated + ParticipantAdded
	assertEventCount(t, result, 2)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated, "First event should be ChatCreated")
	_, isParticipantAdded := result.Events[1].(*domainChat.ParticipantAdded)
	assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
}

// TestCreateChatUseCase_Success_DiscussionWithTitle tests creating a Discussion chat with a title
func TestCreateChatUseCase_Success_DiscussionWithTitle(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	title := "General Discussion"

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		Title:       title,
		CreatedBy:   creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeDiscussion)
	assertChatTitle(t, result.Value, title)
	assert.Equal(t, workspaceID, result.Value.WorkspaceID())
	assert.True(t, result.Value.IsPublic())
	assert.Equal(t, creatorID, result.Value.CreatedBy())

	// Check that events include Renamed event for title
	// NewChat() generates: ChatCreated + ParticipantAdded + Renamed
	assertEventCount(t, result, 3)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated, "First event should be ChatCreated")
	_, isParticipantAdded := result.Events[1].(*domainChat.ParticipantAdded)
	assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
	_, isRenamed := result.Events[2].(*domainChat.Renamed)
	assert.True(t, isRenamed, "Third event should be Renamed")
}

// TestCreateChatUseCase_Success_Task tests creating a Task chat with title
func TestCreateChatUseCase_Success_Task(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	title := "Test Task Title"

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeTask,
		IsPublic:    true,
		Title:       title,
		CreatedBy:   creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeTask)
	assertChatTitle(t, result.Value, title)

	// Check events: ChatCreated + ParticipantAdded + TypeChanged
	assertEventCount(t, result, 3)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated, "First event should be ChatCreated")
	_, isParticipantAdded := result.Events[1].(*domainChat.ParticipantAdded)
	assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
	_, isTypeChanged := result.Events[2].(*domainChat.TypeChanged)
	assert.True(t, isTypeChanged, "Third event should be TypeChanged")
}

// TestCreateChatUseCase_Success_Bug tests creating a Bug chat
func TestCreateChatUseCase_Success_Bug(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	title := "Critical Bug"

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeBug,
		IsPublic:    false,
		Title:       title,
		CreatedBy:   creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeBug)
	assertChatTitle(t, result.Value, title)
	assert.False(t, result.Value.IsPublic())

	// Check events: ChatCreated + ParticipantAdded + TypeChanged
	assertEventCount(t, result, 3)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated, "First event should be ChatCreated")
	_, isParticipantAdded := result.Events[1].(*domainChat.ParticipantAdded)
	assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
	_, isTypeChanged := result.Events[2].(*domainChat.TypeChanged)
	assert.True(t, isTypeChanged, "Third event should be TypeChanged")
}

// TestCreateChatUseCase_Success_Epic tests creating an Epic chat
func TestCreateChatUseCase_Success_Epic(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	title := "Q4 Release Epic"

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        domainChat.TypeEpic,
		IsPublic:    true,
		Title:       title,
		CreatedBy:   creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result.Value)
	assertChatType(t, result.Value, domainChat.TypeEpic)
	assertChatTitle(t, result.Value, title)

	// Check events: ChatCreated + ParticipantAdded + TypeChanged
	assertEventCount(t, result, 3)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated, "First event should be ChatCreated")
	_, isParticipantAdded := result.Events[1].(*domainChat.ParticipantAdded)
	assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
	_, isTypeChanged := result.Events[2].(*domainChat.TypeChanged)
	assert.True(t, isTypeChanged, "Third event should be TypeChanged")
}

// TestCreateChatUseCase_ValidationError_InvalidWorkspaceID tests validation error for invalid workspace ID
func TestCreateChatUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	cmd := chat.CreateChatCommand{
		WorkspaceID: uuid.UUID(""),
		Type:        domainChat.TypeDiscussion,
		CreatedBy:   generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
	require.ErrorContains(t, err, "validation failed")

	// ChatRepo should not be called (validation fails before save)
	assertChatRepoCallCount(t, chatRepo, "Save", 0)
}

// TestCreateChatUseCase_ValidationError_InvalidCreatedBy tests validation error for invalid CreatedBy
func TestCreateChatUseCase_ValidationError_InvalidCreatedBy(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	cmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		CreatedBy:   uuid.UUID(""),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
	assertChatRepoCallCount(t, chatRepo, "Save", 0)
}

// TestCreateChatUseCase_ValidationError_MissingTitleForTypedChat tests validation error when title is missing for Task
func TestCreateChatUseCase_ValidationError_MissingTitleForTypedChat(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	cmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeTask,
		IsPublic:    true,
		Title:       "", // Missing title for Task
		CreatedBy:   generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	assert.Nil(t, result.Value)
	require.ErrorContains(t, err, "validation failed")
	assertChatRepoCallCount(t, chatRepo, "Save", 0)
}

// TestCreateChatUseCase_RepoError tests handling of repository error
func TestCreateChatUseCase_RepoError(t *testing.T) {
	// Arrange
	chatRepo := newTestChatRepo()
	useCase := chat.NewCreateChatUseCase(chatRepo)

	// Setup ChatRepo to fail
	repoError := errors.New("database error")
	chatRepo.SetFailureNext(repoError)

	cmd := chat.CreateChatCommand{
		WorkspaceID: generateUUID(t),
		Type:        domainChat.TypeDiscussion,
		IsPublic:    true,
		CreatedBy:   generateUUID(t),
	}

	// Act
	result, err := useCase.Execute(testContext(), cmd)

	// Assert
	executeAndAssertError(t, err)
	require.ErrorContains(t, err, "database error")
	assert.Nil(t, result.Value)
}
