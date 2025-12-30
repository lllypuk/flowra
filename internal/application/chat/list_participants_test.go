package chat_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestListParticipantsUseCase_Success tests listing all participants
func TestListParticipantsUseCase_Success(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	member1ID := generateUUID(t)
	member2ID := generateUUID(t)

	// Create chat with multiple participants
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, testChat.AddParticipant(member1ID, domainChat.RoleMember))
	require.NoError(t, testChat.AddParticipant(member2ID, domainChat.RoleMember))

	// Save chat with participants to event store
	saveTestChat(t, eventStore, testChat, creatorID)

	query := chat.ListParticipantsQuery{
		ChatID:      testChat.ID(),
		RequestedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Participants, 3) // Creator + 2 members
}

// TestListParticipantsUseCase_Error_ChatNotFound tests error when chat doesn't exist
func TestListParticipantsUseCase_Error_ChatNotFound(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	nonExistentChatID := generateUUID(t)
	requestedBy := generateUUID(t)

	query := chat.ListParticipantsQuery{
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

// TestListParticipantsUseCase_Error_NotParticipant tests access denial for non-participant of private chat
func TestListParticipantsUseCase_Error_NotParticipant(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t) // Not a participant

	// Create a private chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)

	// Save chat to event store
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

	query := chat.ListParticipantsQuery{
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

// TestListParticipantsUseCase_Success_IncludesRoles tests that roles are included in results
func TestListParticipantsUseCase_Success_IncludesRoles(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	memberID := generateUUID(t)

	// Create chat with mixed roles
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, testChat.AddParticipant(memberID, domainChat.RoleMember))

	// Save chat
	saveTestChat(t, eventStore, testChat, creatorID)

	query := chat.ListParticipantsQuery{
		ChatID:      testChat.ID(),
		RequestedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Participants, 2)

	// Check roles
	adminFound := false
	memberFound := false
	for _, p := range result.Participants {
		if p.UserID == creatorID {
			assert.Equal(t, domainChat.RoleAdmin, p.Role)
			adminFound = true
		}
		if p.UserID == memberID {
			assert.Equal(t, domainChat.RoleMember, p.Role)
			memberFound = true
		}
	}
	assert.True(t, adminFound, "Admin participant not found")
	assert.True(t, memberFound, "Member participant not found")
}

// TestListParticipantsUseCase_Success_SortedByJoinDate tests that participants are sorted by join date
func TestListParticipantsUseCase_Success_SortedByJoinDate(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	member1ID := generateUUID(t)
	member2ID := generateUUID(t)
	member3ID := generateUUID(t)

	// Create chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)

	// Add members with slight time delays to ensure different join times
	require.NoError(t, testChat.AddParticipant(member1ID, domainChat.RoleMember))
	time.Sleep(1 * time.Millisecond) // Ensure measurable time difference
	require.NoError(t, testChat.AddParticipant(member2ID, domainChat.RoleMember))
	time.Sleep(1 * time.Millisecond)
	require.NoError(t, testChat.AddParticipant(member3ID, domainChat.RoleMember))

	// Save chat
	saveTestChat(t, eventStore, testChat, creatorID)

	query := chat.ListParticipantsQuery{
		ChatID:      testChat.ID(),
		RequestedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Participants, 4) // Creator + 3 members

	// Verify sorted by join date (ascending)
	for i := 1; i < len(result.Participants); i++ {
		assert.True(t, result.Participants[i-1].JoinedAt.Before(result.Participants[i].JoinedAt) ||
			result.Participants[i-1].JoinedAt.Equal(result.Participants[i].JoinedAt),
			"Participants should be sorted by join date in ascending order")
	}
}

// TestListParticipantsUseCase_Success_PublicChatNonParticipant tests that non-participants can view participants in public chats
func TestListParticipantsUseCase_Success_PublicChatNonParticipant(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	memberID := generateUUID(t)
	requestedBy := generateUUID(t) // Not a participant

	// Create public chat
	testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, testChat.AddParticipant(memberID, domainChat.RoleMember))

	// Save chat
	saveTestChat(t, eventStore, testChat, creatorID)

	query := chat.ListParticipantsQuery{
		ChatID:      testChat.ID(),
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Participants, 2) // Creator + member visible to non-participant
}

// TestListParticipantsUseCase_ValidationError_InvalidChatID tests validation for invalid chat ID
func TestListParticipantsUseCase_ValidationError_InvalidChatID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	requestedBy := generateUUID(t)

	query := chat.ListParticipantsQuery{
		ChatID:      "", // Invalid
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestListParticipantsUseCase_ValidationError_InvalidRequestedBy tests validation for invalid requester ID
func TestListParticipantsUseCase_ValidationError_InvalidRequestedBy(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	useCase := chat.NewListParticipantsUseCase(eventStore)

	chatID := generateUUID(t)

	query := chat.ListParticipantsQuery{
		ChatID:      chatID,
		RequestedBy: "", // Invalid
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "validation failed")
}

// Helper function to save test chat with events to event store
func saveTestChat(t *testing.T, eventStore interface {
	SaveEvents(context.Context, string, []event.DomainEvent, int) error
}, testChat *domainChat.Chat, creatorID uuid.UUID) {
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		testChat.WorkspaceID(),
		testChat.Type(),
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

	// Save participant events - including the creator
	version := 1
	for _, participant := range testChat.Participants() {
		version++
		participantAddedEvent := domainChat.NewParticipantAdded(
			testChat.ID(),
			participant.UserID(),
			participant.Role(),
			participant.JoinedAt(),
			version,
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
				version-1,
			),
		)
	}
}
