package testutil

import (
	"testing"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/tests/mocks"
)

// TestSuite represents full set for integration testing
type TestSuite struct {
	t *testing.T

	// Mocks
	ChatRepo         *mocks.MockChatRepository
	MessageRepo      *mocks.MockMessageRepository
	UserRepo         *mocks.MockUserRepository
	WorkspaceRepo    *mocks.MockWorkspaceRepository
	NotificationRepo *mocks.MockNotificationRepository
	EventBus         *mocks.MockEventBus
	EventStore       *mocks.MockEventStore

	// Use Cases - Chat
	CreateChat     *chatapp.CreateChatUseCase
	AddParticipant *chatapp.AddParticipantUseCase
	ConvertToTask  *chatapp.ConvertToTaskUseCase
	ChangeStatus   *chatapp.ChangeStatusUseCase
	AssignUser     *chatapp.AssignUserUseCase
}

// NewTestSuite creates New test suite with all initialized components
func NewTestSuite(t *testing.T) *TestSuite {
	suite := &TestSuite{
		t:                t,
		ChatRepo:         mocks.NewMockChatRepository(),
		MessageRepo:      mocks.NewMockMessageRepository(),
		UserRepo:         mocks.NewMockUserRepository(),
		WorkspaceRepo:    mocks.NewMockWorkspaceRepository(),
		NotificationRepo: mocks.NewMockNotificationRepository(),
		EventBus:         mocks.NewMockEventBus(),
		EventStore:       mocks.NewMockEventStore(),
	}

	// Initialize Chat Use Cases
	// CreateChat uses ChatRepo which updates both event store and read model
	suite.CreateChat = chatapp.NewCreateChatUseCase(suite.ChatRepo)
	suite.AddParticipant = chatapp.NewAddParticipantUseCase(suite.EventStore)
	suite.ConvertToTask = chatapp.NewConvertToTaskUseCase(suite.EventStore)
	suite.ChangeStatus = chatapp.NewChangeStatusUseCase(suite.EventStore)
	suite.AssignUser = chatapp.NewAssignUserUseCase(suite.EventStore)

	return suite
}

// Reset clears all repositories (helpful between tests)
func (s *TestSuite) Reset() {
	s.ChatRepo.Reset()
	s.MessageRepo.Reset()
	s.UserRepo.Reset()
	s.WorkspaceRepo.Reset()
	s.NotificationRepo.Reset()
	s.EventBus.Reset()
	s.EventStore.Reset()
}
