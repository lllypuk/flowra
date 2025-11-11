package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MockChatQueryRepository is a test implementation of ChatQueryRepository
type MockChatQueryRepository struct {
	chatIDs map[string][]uuid.UUID // key: "workspace:type"
	counts  map[string]int         // key: "workspace:type"
}

// NewMockChatQueryRepository creates a new mock query repository
func NewMockChatQueryRepository() *MockChatQueryRepository {
	return &MockChatQueryRepository{
		chatIDs: make(map[string][]uuid.UUID),
		counts:  make(map[string]int),
	}
}

// FindByWorkspace returns chat IDs for workspace with optional type filter
func (m *MockChatQueryRepository) FindByWorkspace(
	_ context.Context,
	workspaceID uuid.UUID,
	chatType *domainChat.Type,
	limit int,
	offset int,
) ([]uuid.UUID, error) {
	key := m.makeKey(workspaceID.String(), chatType)
	chatIDs := m.chatIDs[key]

	if offset > len(chatIDs) {
		return []uuid.UUID{}, nil
	}

	end := min(offset+limit, len(chatIDs))

	return chatIDs[offset:end], nil
}

// CountByWorkspace returns total count of chats for workspace with optional type filter
func (m *MockChatQueryRepository) CountByWorkspace(
	_ context.Context,
	workspaceID uuid.UUID,
	chatType *domainChat.Type,
) (int, error) {
	key := m.makeKey(workspaceID.String(), chatType)
	return m.counts[key], nil
}

// SetupChatsForWorkspace sets up test data
func (m *MockChatQueryRepository) SetupChatsForWorkspace(
	workspaceID uuid.UUID,
	chatType *domainChat.Type,
	chatIDs []uuid.UUID,
) {
	key := m.makeKey(workspaceID.String(), chatType)
	m.chatIDs[key] = chatIDs
	m.counts[key] = len(chatIDs)
}

func (m *MockChatQueryRepository) makeKey(workspaceID string, chatType *domainChat.Type) string {
	if chatType == nil {
		return workspaceID + ":all"
	}
	return workspaceID + ":" + string(*chatType)
}

// TestListChatsUseCase_Success_AllChats tests listing all chats in workspace
func TestListChatsUseCase_Success_AllChats(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t)

	// Create test chats
	chat1, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, chat1.AddParticipant(requestedBy, domainChat.RoleMember))

	chat2, err := domainChat.NewChat(workspaceID, domainChat.TypeTask, false, creatorID)
	require.NoError(t, err)
	require.NoError(t, chat2.AddParticipant(requestedBy, domainChat.RoleMember))

	// Save chats to event store
	saveTestChat(t, eventStore, chat1, creatorID, true)
	saveTestChat(t, eventStore, chat2, creatorID, false)

	// Setup query repo
	queryRepo.SetupChatsForWorkspace(workspaceID, nil, []uuid.UUID{chat1.ID(), chat2.ID()})

	query := chat.ListChatsQuery{
		WorkspaceID: workspaceID,
		Limit:       20,
		Offset:      0,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Chats, 2)
	assert.Equal(t, 2, result.Total)
	assert.False(t, result.HasMore)
}

// TestListChatsUseCase_Success_FilterByType_Task tests filtering only Task type chats
func TestListChatsUseCase_Success_FilterByType_Task(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t)

	// Create test chats of different types
	chatDiscussion, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, chatDiscussion.AddParticipant(requestedBy, domainChat.RoleMember))

	chatTask, err := domainChat.NewChat(workspaceID, domainChat.TypeTask, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, chatTask.AddParticipant(requestedBy, domainChat.RoleMember))

	// Save chats
	saveTestChat(t, eventStore, chatDiscussion, creatorID, true)
	saveTestChat(t, eventStore, chatTask, creatorID, true)

	// Setup query repo with only Task type
	typeFilter := domainChat.TypeTask
	queryRepo.SetupChatsForWorkspace(workspaceID, &typeFilter, []uuid.UUID{chatTask.ID()})

	query := chat.ListChatsQuery{
		WorkspaceID: workspaceID,
		Type:        &typeFilter,
		Limit:       20,
		Offset:      0,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Chats, 1)
	assert.Equal(t, domainChat.TypeTask, result.Chats[0].Type)
}

// TestListChatsUseCase_Success_Pagination tests pagination works correctly
func TestListChatsUseCase_Success_Pagination(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t)

	// Create 5 test chats
	var chatIDs []uuid.UUID
	for range 5 {
		testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
		require.NoError(t, err)
		require.NoError(t, testChat.AddParticipant(requestedBy, domainChat.RoleMember))
		saveTestChat(t, eventStore, testChat, creatorID, true)
		chatIDs = append(chatIDs, testChat.ID())
	}

	// Setup query repo with all chats
	queryRepo.SetupChatsForWorkspace(workspaceID, nil, chatIDs)

	query := chat.ListChatsQuery{
		WorkspaceID: workspaceID,
		Limit:       2,
		Offset:      0,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Chats, 2) // Limited to 2
	assert.True(t, result.HasMore) // More chats available
}

// TestListChatsUseCase_Success_OnlyUserChats tests only returns chats where user is participant or public
func TestListChatsUseCase_Success_OnlyUserChats(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t)
	otherUser := generateUUID(t)

	// Create public chat where requester is participant
	publicChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	require.NoError(t, publicChat.AddParticipant(requestedBy, domainChat.RoleMember))

	// Create private chat where requester is NOT participant
	privateChat, err := domainChat.NewChat(workspaceID, domainChat.TypeTask, false, otherUser)
	require.NoError(t, err)

	// Save chats
	saveTestChat(t, eventStore, publicChat, creatorID, true)
	saveTestChat(t, eventStore, privateChat, otherUser, false)

	// Setup query repo with both chats
	queryRepo.SetupChatsForWorkspace(workspaceID, nil, []uuid.UUID{publicChat.ID(), privateChat.ID()})

	query := chat.ListChatsQuery{
		WorkspaceID: workspaceID,
		Limit:       20,
		Offset:      0,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Chats, 1) // Only public chat accessible to requester
}

// TestListChatsUseCase_Success_IncludesPublicChats tests public chats are included even if not participant
func TestListChatsUseCase_Success_IncludesPublicChats(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	workspaceID := generateUUID(t)
	creatorID := generateUUID(t)
	requestedBy := generateUUID(t) // Not a participant

	// Create public chat where requester is not a participant
	publicChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
	require.NoError(t, err)
	// Don't add requestedBy as participant

	// Save chat
	saveTestChat(t, eventStore, publicChat, creatorID, true)

	// Setup query repo
	queryRepo.SetupChatsForWorkspace(workspaceID, nil, []uuid.UUID{publicChat.ID()})

	query := chat.ListChatsQuery{
		WorkspaceID: workspaceID,
		Limit:       20,
		Offset:      0,
		RequestedBy: requestedBy,
	}

	// Act
	result, err := useCase.Execute(testContext(), query)

	// Assert
	executeAndAssertSuccess(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Chats, 1) // Public chat is accessible
	assert.True(t, result.Chats[0].IsPublic)
}

// TestListChatsUseCase_ValidationError_InvalidWorkspaceID tests validation for invalid workspace ID
func TestListChatsUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	eventStore := newTestEventStore()
	queryRepo := NewMockChatQueryRepository()
	useCase := chat.NewListChatsUseCase(queryRepo, eventStore)

	requestedBy := generateUUID(t)

	query := chat.ListChatsQuery{
		WorkspaceID: "", // Invalid
		Limit:       20,
		Offset:      0,
		RequestedBy: requestedBy,
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
}, testChat *domainChat.Chat, creatorID uuid.UUID, isPublic bool) {
	chatCreatedEvent := domainChat.NewChatCreated(
		testChat.ID(),
		testChat.WorkspaceID(),
		testChat.Type(),
		isPublic,
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
		participantAddedEvent := domainChat.NewParticipantAdded(
			testChat.ID(),
			participant.UserID(),
			participant.Role(),
			participant.JoinedAt(),
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
				version,
			),
		)
		version++
	}
}
