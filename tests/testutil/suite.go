package testutil

import (
	"testing"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/tests/mocks"
)

// TestSuite представляет полный набор для интеграционного тестирования
type TestSuite struct {
	t *testing.T

	// Mocks
	ChatRepo         *mocks.MockChatRepository
	MessageRepo      *mocks.MessageRepository
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

// NewTestSuite создает новый test suite со всеми инициализированными компонентами
func NewTestSuite(t *testing.T) *TestSuite {
	suite := &TestSuite{
		t:                t,
		ChatRepo:         mocks.NewMockChatRepository(),
		MessageRepo:      mocks.NewMessageRepository(),
		UserRepo:         mocks.NewMockUserRepository(),
		WorkspaceRepo:    mocks.NewMockWorkspaceRepository(),
		NotificationRepo: mocks.NewMockNotificationRepository(),
		EventBus:         mocks.NewMockEventBus(),
		EventStore:       mocks.NewMockEventStore(),
	}

	// Initialize Chat Use Cases (they take EventStore, not repositories)
	suite.CreateChat = chatapp.NewCreateChatUseCase(suite.EventStore)
	suite.AddParticipant = chatapp.NewAddParticipantUseCase(suite.EventStore)
	suite.ConvertToTask = chatapp.NewConvertToTaskUseCase(suite.EventStore)
	suite.ChangeStatus = chatapp.NewChangeStatusUseCase(suite.EventStore)
	suite.AssignUser = chatapp.NewAssignUserUseCase(suite.EventStore)

	return suite
}

// Reset очищает все repositories (полезно между тестами)
func (s *TestSuite) Reset() {
	s.ChatRepo.Reset()
	s.MessageRepo.Reset()
	s.UserRepo.Reset()
	s.WorkspaceRepo.Reset()
	s.NotificationRepo.Reset()
	s.EventBus.Reset()
	s.EventStore.Reset()
}
