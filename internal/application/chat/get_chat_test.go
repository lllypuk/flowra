package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// TestGetChatUseCase_Success tests retrieving a chat successfully
func TestGetChatUseCase_Success(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t)

	// Create a public chat with the creator as admin and requester as member
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, testChat.AddParticipant(requestedBy, domainChat.RoleMember))

	// Save chat events to mock store
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		workspaceID,
		domainChat.TypeDiscussion,
		true,
		creatorID,
		testChat.CreatedAt(),
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(context.Background(), testChat.ID().String(), []event.DomainEvent{chatCreatedEvent}, 0),
	)

	// Add participant event
	participantAddedEvent := domainChat.NewParticipantAdded(
		testChat.ID(),
		requestedBy,
		domainChat.RoleMember,
		testChat.CreatedAt(),
		2,
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(
			context.Background(),
			testChat.ID().String(),
			[]event.DomainEvent{participantAddedEvent},
			1,
		),
	)

	query := chat.GetChatQuery{
		ChatID:      testChat.ID(),
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Chat)
	assert.Equal(t, testChat.ID(), result.Chat.ID)
	assert.Equal(t, workspaceID, result.Chat.WorkspaceID)
	assert.Equal(t, domainChat.TypeDiscussion, result.Chat.Type)
	assert.True(t, result.Chat.IsPublic)
	assert.True(t, result.Permissions.CanRead)
	assert.True(t, result.Permissions.CanWrite)
}

// TestGetChatUseCase_Error_ChatNotFound tests retrieving a non-existent chat
func TestGetChatUseCase_Error_ChatNotFound(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	nonExistentChatID := generateUUID(t)
	requestedBy := generateUUID(t)

	query := chat.GetChatQuery{
		ChatID:      nonExistentChatID,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to load chat")
}

// TestGetChatUseCase_Error_AccessDenied tests access denial for private chat without participation
func TestGetChatUseCase_Error_AccessDenied(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t) // Different user, not a participant

	// Create a private chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)

	// Save chat events to mock store
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		workspaceID,
		domainChat.TypeDiscussion,
		false,
		creatorID,
		testChat.CreatedAt(),
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(context.Background(), testChat.ID().String(), []event.DomainEvent{chatCreatedEvent}, 0),
	)

	query := chat.GetChatQuery{
		ChatID:      testChat.ID(),
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "access denied")
}

// TestGetChatUseCase_Success_PublicChat tests accessing public chat as non-participant
func TestGetChatUseCase_Success_PublicChat(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t) // Different user, not a participant

	// Create a public chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)

	// Save chat events
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		workspaceID,
		domainChat.TypeDiscussion,
		true,
		creatorID,
		testChat.CreatedAt(),
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(context.Background(), testChat.ID().String(), []event.DomainEvent{chatCreatedEvent}, 0),
	)

	query := chat.GetChatQuery{
		ChatID:      testChat.ID(),
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Chat)
	assert.Equal(t, testChat.ID(), result.Chat.ID)
	assert.True(t, result.Permissions.CanRead)
	assert.False(t, result.Permissions.CanWrite) // Non-participant can't write
	assert.False(t, result.Permissions.CanManage)
}

// TestGetChatUseCase_Success_CreatorHasManagePermissions tests that creator has manage permissions
func TestGetChatUseCase_Success_CreatorHasManagePermissions(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)

	// Create a chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)

	// Save chat events (including creator as participant)
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		workspaceID,
		domainChat.TypeDiscussion,
		false,
		creatorID,
		testChat.CreatedAt(),
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(context.Background(), testChat.ID().String(), []event.DomainEvent{chatCreatedEvent}, 0),
	)

	// Creator is automatically added as admin participant
	creatorAddedEvent := domainChat.NewParticipantAdded(
		testChat.ID(),
		creatorID,
		domainChat.RoleAdmin,
		testChat.CreatedAt(),
		2,
		event.Metadata{
			CorrelationID: testChat.ID().String(),
			CausationID:   testChat.ID().String(),
			UserID:        creatorID.String(),
		},
	)
	require.NoError(
		t,
		eventStore.SaveEvents(context.Background(), testChat.ID().String(), []event.DomainEvent{creatorAddedEvent}, 1),
	)

	query := chat.GetChatQuery{
		ChatID:      testChat.ID(),
		RequestedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Permissions.CanRead)
	assert.True(t, result.Permissions.CanWrite)
	assert.True(t, result.Permissions.CanManage)
}

// TestGetChatUseCase_ValidationError_InvalidChatID tests validation for invalid chat ID
func TestGetChatUseCase_ValidationError_InvalidChatID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	requestedBy := generateUUID(t)

	query := chat.GetChatQuery{
		ChatID:      "", // Invalid (zero)
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestGetChatUseCase_ValidationError_InvalidRequestedBy tests validation for invalid requester ID
func TestGetChatUseCase_ValidationError_InvalidRequestedBy(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewGetChatUseCase(eventStore)

	chatID := generateUUID(t)

	query := chat.GetChatQuery{
		ChatID:      chatID,
		RequestedBy: "", // Invalid (zero)
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}
