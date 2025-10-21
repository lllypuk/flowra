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
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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
	assertEventCount(t, result, 1)
	createdEvent := getEventByType(t, result, "ChatCreated")
	require.NotNil(t, createdEvent)
}

// TestCreateChatUseCase_Success_Task tests creating a Task chat with title
func TestCreateChatUseCase_Success_Task(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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

	// Check events: ChatCreated + TypeChanged
	assertEventCount(t, result, 2)
	_, isChatCreated := result.Events[0].(*domainChat.Created)
	assert.True(t, isChatCreated)
	_, isTypeChanged := result.Events[1].(*domainChat.TypeChanged)
	assert.True(t, isTypeChanged)
}

// TestCreateChatUseCase_Success_Bug tests creating a Bug chat
func TestCreateChatUseCase_Success_Bug(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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

	assertEventCount(t, result, 2)
}

// TestCreateChatUseCase_Success_Epic tests creating an Epic chat
func TestCreateChatUseCase_Success_Epic(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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

	assertEventCount(t, result, 2)
}

// TestCreateChatUseCase_ValidationError_InvalidWorkspaceID tests validation error for invalid workspace ID
func TestCreateChatUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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

	// EventStore should not be called
	assertEventStoreCallCount(t, eventStore, "SaveEvents", 0)
}

// TestCreateChatUseCase_ValidationError_InvalidCreatedBy tests validation error for invalid CreatedBy
func TestCreateChatUseCase_ValidationError_InvalidCreatedBy(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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
	assertEventStoreCallCount(t, eventStore, "SaveEvents", 0)
}

// TestCreateChatUseCase_ValidationError_MissingTitleForTypedChat tests validation error when title is missing for Task
func TestCreateChatUseCase_ValidationError_MissingTitleForTypedChat(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

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
	assertEventStoreCallCount(t, eventStore, "SaveEvents", 0)
}

// TestCreateChatUseCase_EventStoreError tests handling of EventStore error
func TestCreateChatUseCase_EventStoreError(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewCreateChatUseCase(eventStore)

	// Setup EventStore to fail
	storeError := errors.New("database error")
	setEventStoreError(eventStore, storeError)

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
