package chat

import (
	"context"
	"testing"
	"time"

	domainChat "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
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

// CreateTestChat creates a test chat aggregate
func createTestChat(t *testing.T, workspaceID, creatorID uuid.UUID, chatType domainChat.Type) *domainChat.Chat {
	chat, err := domainChat.NewChat(workspaceID, chatType, true, creatorID)
	require.NoError(t, err)
	return chat
}

// CreateTestChatWithTitle creates a test chat with title (for typed chats)
func createTestChatWithTitle(t *testing.T, workspaceID, creatorID uuid.UUID, chatType domainChat.Type, title string) *domainChat.Chat {
	chat := createTestChat(t, workspaceID, creatorID, chatType)

	// For typed chats, set type and title
	if chatType != domainChat.TypeDiscussion && title != "" {
		switch chatType {
		case domainChat.TypeTask:
			err := chat.ConvertToTask(title, creatorID)
			require.NoError(t, err)
		case domainChat.TypeBug:
			err := chat.ConvertToBug(title, creatorID)
			require.NoError(t, err)
		case domainChat.TypeEpic:
			err := chat.ConvertToEpic(title, creatorID)
			require.NoError(t, err)
		}
	}

	return chat
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
func assertEventCount(t *testing.T, result Result, expected int) {
	require.Len(t, result.Events, expected, "Expected %d events, got %d", expected, len(result.Events))
}

// AssertEventType asserts that event is of specific type
func assertEventType(t *testing.T, eventInterface interface{}, typeName string) {
	switch typeName {
	case "ChatCreated":
		_, ok := eventInterface.(*domainChat.Created)
		require.True(t, ok, "Expected ChatCreated event")
	case "ParticipantAdded":
		_, ok := eventInterface.(*domainChat.ParticipantAdded)
		require.True(t, ok, "Expected ParticipantAdded event")
	case "ParticipantRemoved":
		_, ok := eventInterface.(*domainChat.ParticipantRemoved)
		require.True(t, ok, "Expected ParticipantRemoved event")
	case "TypeChanged":
		_, ok := eventInterface.(*domainChat.TypeChanged)
		require.True(t, ok, "Expected TypeChanged event")
	case "StatusChanged":
		_, ok := eventInterface.(*domainChat.StatusChanged)
		require.True(t, ok, "Expected StatusChanged event")
	case "UserAssigned":
		_, ok := eventInterface.(*domainChat.UserAssigned)
		require.True(t, ok, "Expected UserAssigned event")
	case "AssigneeRemoved":
		_, ok := eventInterface.(*domainChat.AssigneeRemoved)
		require.True(t, ok, "Expected AssigneeRemoved event")
	case "PrioritySet":
		_, ok := eventInterface.(*domainChat.PrioritySet)
		require.True(t, ok, "Expected PrioritySet event")
	case "DueDateSet":
		_, ok := eventInterface.(*domainChat.DueDateSet)
		require.True(t, ok, "Expected DueDateSet event")
	case "DueDateRemoved":
		_, ok := eventInterface.(*domainChat.DueDateRemoved)
		require.True(t, ok, "Expected DueDateRemoved event")
	case "Renamed":
		_, ok := eventInterface.(*domainChat.Renamed)
		require.True(t, ok, "Expected Renamed event")
	case "SeveritySet":
		_, ok := eventInterface.(*domainChat.SeveritySet)
		require.True(t, ok, "Expected SeveritySet event")
	default:
		t.Fatalf("Unknown event type: %s", typeName)
	}
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
func generateUUID(t *testing.T) uuid.UUID {
	return uuid.NewUUID()
}

// SetEventStoreError sets error for next call
func setEventStoreError(es *mocks.MockEventStore, err error) {
	es.SetFailureNext(err)
}

// Helper to compare timestamps (allowing for small time differences due to timing)
func assertTimeNear(t *testing.T, actual, expected time.Time, tolerance time.Duration) {
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	require.LessOrEqual(t, diff, tolerance, "Times differ by more than %v: %v vs %v", tolerance, actual, expected)
}

// GetEventByType retrieves first event of specific type from result
func getEventByType(t *testing.T, result Result, typeName string) interface{} {
	for _, evt := range result.Events {
		switch typeName {
		case "ChatCreated":
			if _, ok := evt.(*domainChat.Created); ok {
				return evt
			}
		case "TypeChanged":
			if _, ok := evt.(*domainChat.TypeChanged); ok {
				return evt
			}
		case "StatusChanged":
			if _, ok := evt.(*domainChat.StatusChanged); ok {
				return evt
			}
		case "ParticipantAdded":
			if _, ok := evt.(*domainChat.ParticipantAdded); ok {
				return evt
			}
		case "ParticipantRemoved":
			if _, ok := evt.(*domainChat.ParticipantRemoved); ok {
				return evt
			}
		}
	}
	t.Fatalf("Event of type %s not found in result", typeName)
	return nil
}

// LoadChatFromEvents loads a chat aggregate from events
func loadChatFromEvents(t *testing.T, events []event.DomainEvent) *domainChat.Chat {
	if len(events) == 0 {
		t.Fatal("No events provided to load chat")
	}

	chat := &domainChat.Chat{}
	for _, evt := range events {
		err := chat.Apply(evt)
		require.NoError(t, err, "Failed to apply event")
	}
	return chat
}

// SaveAggregateAndLoad is a helper that simulates saving and loading an aggregate
func saveAggregateAndLoad(t *testing.T, es *mocks.MockEventStore, aggregate *domainChat.Chat) *domainChat.Chat {
	// Get the aggregate ID string
	aggregateID := aggregate.ID().String()

	// Save events
	events := aggregate.GetUncommittedEvents()
	err := es.SaveEvents(context.Background(), aggregateID, events, 0)
	require.NoError(t, err, "Failed to save events")

	// Load events back
	loadedEvents, err := es.LoadEvents(context.Background(), aggregateID)
	require.NoError(t, err, "Failed to load events")

	// Reconstruct aggregate from events
	return loadChatFromEvents(t, loadedEvents)
}
