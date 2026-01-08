package chat_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/chat"
	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	domainEvent "github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/tests/mocks"
	"github.com/stretchr/testify/require"
)

// TestContext creates a context for tests
func testContext() context.Context {
	return context.Background()
}

// NewTestEventStore creates a mock EventStore
func newTestEventStore() *mocks.MockEventStore {
	return mocks.NewMockEventStore()
}

// NewTestChatRepo creates a mock ChatRepository (implements CommandRepository)
func newTestChatRepo() *mocks.MockChatRepository {
	return mocks.NewMockChatRepository()
}

// createTestChat creates a chat using chatRepo and syncs events to eventStore for other use cases.
// This is needed because CreateChatUseCase uses chatRepo, but other use cases use eventStore.
func createTestChat(t *testing.T, eventStore *mocks.MockEventStore, chatType domainChat.Type, title string) *domainChat.Chat {
	t.Helper()
	return createTestChatWithParams(t, eventStore, chatType, title, generateUUID(t), generateUUID(t), true)
}

// createTestChatWithParams creates a chat with specific parameters and syncs events to eventStore.
func createTestChatWithParams(
	t *testing.T,
	eventStore *mocks.MockEventStore,
	chatType domainChat.Type,
	title string,
	workspaceID uuid.UUID,
	creatorID uuid.UUID,
	isPublic bool,
) *domainChat.Chat {
	t.Helper()

	chatRepo := newTestChatRepo()
	createUC := chat.NewCreateChatUseCase(chatRepo)

	cmd := chat.CreateChatCommand{
		WorkspaceID: workspaceID,
		Type:        chatType,
		Title:       title,
		IsPublic:    isPublic,
		CreatedBy:   creatorID,
	}

	result, err := createUC.Execute(testContext(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result.Value)

	// Sync events to eventStore so other use cases can load the chat
	createdChat := result.Value
	events := createdChat.GetUncommittedEvents()
	if len(events) == 0 {
		// Events already committed, re-apply from result
		for _, e := range result.Events {
			if evt, ok := e.(domainEvent.DomainEvent); ok {
				events = append(events, evt)
			}
		}
	}
	if len(events) > 0 {
		_ = eventStore.SaveEvents(testContext(), createdChat.ID().String(), events, 0)
	}

	return createdChat
}

// ExecuteAndAssertSuccess executes a use case and asserts no error
func executeAndAssertSuccess(t *testing.T, err error) {
	require.NoError(t, err, "Expected no error during use case execution")
}

// ExecuteAndAssertError executes a use case and asserts error occurs
func executeAndAssertError(t *testing.T, err error) {
	require.Error(t, err, "Expected error during use case execution")
}

// AssertEventCount asserts the number of events in result
func assertEventCount(t *testing.T, result chat.Result, expected int) {
	require.Len(t, result.Events, expected, "Expected %d events, got %d", expected, len(result.Events))
}

// AssertChatType asserts the chat type
func assertChatType(t *testing.T, chat *domainChat.Chat, expectedType domainChat.Type) {
	require.Equal(t, expectedType, chat.Type(), "Expected chat type %s, got %s", expectedType, chat.Type())
}

// AssertChatTitle asserts the chat title
func assertChatTitle(t *testing.T, chat *domainChat.Chat, expectedTitle string) {
	require.Equal(t, expectedTitle, chat.Title(), "Expected title %q, got %q", expectedTitle, chat.Title())
}

// AssertChatStatus asserts the chat status
func assertChatStatus(t *testing.T, chat *domainChat.Chat, expectedStatus string) {
	require.Equal(t, expectedStatus, chat.Status(), "Expected status %q, got %q", expectedStatus, chat.Status())
}

// AssertEventStoreCallCount asserts EventStore method call count
func assertEventStoreCallCount(t *testing.T, es *mocks.MockEventStore, method string, expected int) {
	actual := es.GetCallCount(method)
	require.Equal(t, expected, actual, "Expected %d calls to %s, got %d", expected, method, actual)
}

// GenerateUUID generates a new UUID for tests
func generateUUID(_ *testing.T) uuid.UUID {
	return uuid.NewUUID()
}

// SetEventStoreError sets error for next call
func setEventStoreError(es *mocks.MockEventStore, err error) {
	es.SetFailureNext(err)
}

// AssertChatRepoCallCount asserts ChatRepository method call count
func assertChatRepoCallCount(t *testing.T, repo *mocks.MockChatRepository, method string, expected int) {
	var actual int
	switch method {
	case "Save":
		actual = repo.SaveCallCount()
	case "Load":
		actual = repo.LoadCallCount()
	default:
		require.Fail(t, "Unknown method: "+method)
	}
	require.Equal(t, expected, actual, "Expected %d calls to %s, got %d", expected, method, actual)
}
