package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MockChatQueryRepository is a test implementation of ChatQueryRepository
type MockChatQueryRepository struct {
	readModels map[uuid.UUID]*chat.ReadModel
}

// NewMockChatQueryRepository creates a New mock query repository
func NewMockChatQueryRepository() *MockChatQueryRepository {
	return &MockChatQueryRepository{
		readModels: make(map[uuid.UUID]*chat.ReadModel),
	}
}

// FindByID finds chat by ID (from read model)
func (m *MockChatQueryRepository) FindByID(_ context.Context, chatID uuid.UUID) (*chat.ReadModel, error) {
	if rm, ok := m.readModels[chatID]; ok {
		return rm, nil
	}
	return nil, chat.ErrChatNotFound
}

// FindByWorkspace finds chats in workspace with filters
func (m *MockChatQueryRepository) FindByWorkspace(
	_ context.Context,
	workspaceID uuid.UUID,
	filters chat.Filters,
) ([]*chat.ReadModel, error) {
	result := m.filterChatsByWorkspace(workspaceID, filters)
	return m.applyPagination(result, filters.Offset, filters.Limit), nil
}

// filterChatsByWorkspace filters chats by workspace and applies all filters
func (m *MockChatQueryRepository) filterChatsByWorkspace(
	workspaceID uuid.UUID,
	filters chat.Filters,
) []*chat.ReadModel {
	var result []*chat.ReadModel

	for _, rm := range m.readModels {
		if m.chatMatchesFilters(rm, workspaceID, filters) {
			result = append(result, rm)
		}
	}

	return result
}

// chatMatchesFilters checks if a chat matches all filter criteria
func (m *MockChatQueryRepository) chatMatchesFilters(
	rm *chat.ReadModel,
	workspaceID uuid.UUID,
	filters chat.Filters,
) bool {
	if rm.WorkspaceID != workspaceID {
		return false
	}

	if !m.matchesTypeFilter(rm, filters.Type) {
		return false
	}

	if !m.matchesPublicFilter(rm, filters.IsPublic) {
		return false
	}

	if !m.matchesUserFilter(rm, filters.UserID) {
		return false
	}

	return true
}

// matchesTypeFilter checks if chat matches type filter
func (m *MockChatQueryRepository) matchesTypeFilter(rm *chat.ReadModel, typeFilter *domainChat.Type) bool {
	if typeFilter == nil {
		return true
	}
	return rm.Type == *typeFilter
}

// matchesPublicFilter checks if chat matches public filter
func (m *MockChatQueryRepository) matchesPublicFilter(rm *chat.ReadModel, isPublicFilter *bool) bool {
	if isPublicFilter == nil {
		return true
	}
	return rm.IsPublic == *isPublicFilter
}

// matchesUserFilter checks if chat matches user participant filter
func (m *MockChatQueryRepository) matchesUserFilter(rm *chat.ReadModel, userID *uuid.UUID) bool {
	if userID == nil {
		return true
	}

	for _, p := range rm.Participants {
		if p.UserID() == *userID {
			return true
		}
	}
	return false
}

// applyPagination applies offset and limit to the result set
func (m *MockChatQueryRepository) applyPagination(result []*chat.ReadModel, offset, limit int) []*chat.ReadModel {
	if offset >= len(result) {
		return []*chat.ReadModel{}
	}

	end := min(offset+limit, len(result))
	return result[offset:end]
}

// FindByParticipant finds chats for user
func (m *MockChatQueryRepository) FindByParticipant(
	_ context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*chat.ReadModel, error) {
	var result []*chat.ReadModel

	for _, rm := range m.readModels {
		for _, p := range rm.Participants {
			if p.UserID() == userID {
				result = append(result, rm)
				break
			}
		}
	}

	// Apply pagination
	if offset >= len(result) {
		return []*chat.ReadModel{}, nil
	}

	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

// Count returns total count of chats in workspace
func (m *MockChatQueryRepository) Count(_ context.Context, workspaceID uuid.UUID) (int, error) {
	count := 0
	for _, rm := range m.readModels {
		if rm.WorkspaceID == workspaceID {
			count++
		}
	}
	return count, nil
}

// SetupReadModel sets up test data
func (m *MockChatQueryRepository) SetupReadModel(rm *chat.ReadModel) {
	m.readModels[rm.ID] = rm
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

	// Setup read models in query repo
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           chat1.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeDiscussion,
		IsPublic:     true,
		CreatedBy:    creatorID,
		CreatedAt:    chat1.CreatedAt(),
		Participants: chat1.Participants(),
	})
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           chat2.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeTask,
		IsPublic:     false,
		CreatedBy:    creatorID,
		CreatedAt:    chat2.CreatedAt(),
		Participants: chat2.Participants(),
	})

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

	// Setup read models
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           chatDiscussion.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeDiscussion,
		IsPublic:     true,
		CreatedBy:    creatorID,
		CreatedAt:    chatDiscussion.CreatedAt(),
		Participants: chatDiscussion.Participants(),
	})
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           chatTask.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeTask,
		IsPublic:     true,
		CreatedBy:    creatorID,
		CreatedAt:    chatTask.CreatedAt(),
		Participants: chatTask.Participants(),
	})

	typeFilter := domainChat.TypeTask
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
	for range 5 {
		testChat, err := domainChat.NewChat(workspaceID, domainChat.TypeDiscussion, true, creatorID)
		require.NoError(t, err)
		require.NoError(t, testChat.AddParticipant(requestedBy, domainChat.RoleMember))

		queryRepo.SetupReadModel(&chat.ReadModel{
			ID:           testChat.ID(),
			WorkspaceID:  workspaceID,
			Type:         domainChat.TypeDiscussion,
			IsPublic:     true,
			CreatedBy:    creatorID,
			CreatedAt:    testChat.CreatedAt(),
			Participants: testChat.Participants(),
		})
	}

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

	// Setup read models
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           publicChat.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeDiscussion,
		IsPublic:     true,
		CreatedBy:    creatorID,
		CreatedAt:    publicChat.CreatedAt(),
		Participants: publicChat.Participants(),
	})
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           privateChat.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeTask,
		IsPublic:     false,
		CreatedBy:    otherUser,
		CreatedAt:    privateChat.CreatedAt(),
		Participants: privateChat.Participants(),
	})

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

	// Setup read model
	queryRepo.SetupReadModel(&chat.ReadModel{
		ID:           publicChat.ID(),
		WorkspaceID:  workspaceID,
		Type:         domainChat.TypeDiscussion,
		IsPublic:     true,
		CreatedBy:    creatorID,
		CreatedAt:    publicChat.CreatedAt(),
		Participants: publicChat.Participants(),
	})

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
