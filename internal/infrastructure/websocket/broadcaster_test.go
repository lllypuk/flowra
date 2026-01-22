package websocket_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	ws "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventBus is a mock implementation of EventBus for testing.
type mockEventBus struct {
	handlers map[string][]eventbus.EventHandler
	mu       sync.RWMutex
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		handlers: make(map[string][]eventbus.EventHandler),
	}
}

func (m *mockEventBus) Subscribe(
	eventType string,
	handler eventbus.EventHandler,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[eventType] = append(m.handlers[eventType], handler)
	return nil
}

func (m *mockEventBus) Publish(ctx context.Context, evt event.DomainEvent) error {
	m.mu.RLock()
	handlers := m.handlers[evt.EventType()]
	m.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockEventBus) HandlerCount(eventType string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.handlers[eventType])
}

// testDomainEvent is a test implementation of DomainEvent.
type testDomainEvent struct {
	event.BaseEvent

	payload json.RawMessage
}

func newTestDomainEvent(eventType, aggregateID, aggregateType string) *testDomainEvent {
	return &testDomainEvent{
		BaseEvent: event.NewBaseEvent(
			eventType,
			aggregateID,
			aggregateType,
			1,
			event.NewMetadata("user-123", "correlation-1", "causation-1"),
		),
	}
}

func newTestDomainEventWithPayload(eventType, aggregateID, aggregateType string, payload any) *testDomainEvent {
	payloadBytes, _ := json.Marshal(payload)
	return &testDomainEvent{
		BaseEvent: event.NewBaseEvent(
			eventType,
			aggregateID,
			aggregateType,
			1,
			event.NewMetadata("user-123", "correlation-1", "causation-1"),
		),
		payload: payloadBytes,
	}
}

func (e *testDomainEvent) Payload() json.RawMessage {
	return e.payload
}

func TestNewBroadcaster(t *testing.T) {
	t.Run("creates broadcaster with defaults", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		broadcaster := ws.NewBroadcaster(hub, eventBus)

		assert.NotNil(t, broadcaster)
		assert.False(t, broadcaster.IsRunning())
	})

	t.Run("creates broadcaster with custom event types", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		customEventTypes := []string{"custom.event1", "custom.event2"}
		broadcaster := ws.NewBroadcaster(hub, eventBus,
			ws.WithEventTypes(customEventTypes),
		)

		assert.NotNil(t, broadcaster)
	})

	t.Run("creates broadcaster with logger", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		broadcaster := ws.NewBroadcaster(hub, eventBus,
			ws.WithBroadcasterLogger(nil),
		)

		assert.NotNil(t, broadcaster)
	})
}

func TestDefaultEventTypes(t *testing.T) {
	eventTypes := ws.DefaultEventTypes()

	expectedTypes := []string{
		"message.created",
		"message.edited",
		"message.deleted",
		"chat.created",
		"chat.updated",
		"chat.deleted",
		"chat.member_added",
		"chat.member_removed",
		"task.created",
		"task.updated",
		"task.status_changed",
		"task.assigned",
		"notification.created",
	}

	assert.Equal(t, expectedTypes, eventTypes)
}

func TestBroadcaster_Start(t *testing.T) {
	t.Run("subscribes to all event types", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(context.Background())
		require.NoError(t, err)

		assert.True(t, broadcaster.IsRunning())

		// Verify subscriptions
		for _, eventType := range ws.DefaultEventTypes() {
			assert.Equal(t, 1, eventBus.HandlerCount(eventType),
				"expected handler for event type %s", eventType)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(context.Background())
		require.NoError(t, err)

		// Second start should not error
		err = broadcaster.Start(context.Background())
		require.NoError(t, err)

		// Still should only have one handler per event type
		for _, eventType := range ws.DefaultEventTypes() {
			assert.Equal(t, 1, eventBus.HandlerCount(eventType))
		}
	})

	t.Run("subscribes to custom event types", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()

		customEventTypes := []string{"custom.event1", "custom.event2"}
		broadcaster := ws.NewBroadcaster(hub, eventBus,
			ws.WithEventTypes(customEventTypes),
		)

		err := broadcaster.Start(context.Background())
		require.NoError(t, err)

		assert.Equal(t, 1, eventBus.HandlerCount("custom.event1"))
		assert.Equal(t, 1, eventBus.HandlerCount("custom.event2"))
		assert.Equal(t, 0, eventBus.HandlerCount("message.sent")) // Default type not subscribed
	})
}

func TestBroadcaster_HandleEvent(t *testing.T) {
	t.Run("broadcasts message.created event to chat", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(ctx)
		require.NoError(t, err)

		// Create a client and subscribe to a chat
		chatID := uuid.NewUUID()
		client, receiveChan := createTestBroadcasterClient(t, hub, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(20 * time.Millisecond) // Wait for registration to complete
		hub.JoinChat(client, chatID)
		time.Sleep(20 * time.Millisecond) // Wait for join to complete

		// Publish event
		evt := newTestDomainEvent("message.created", chatID.String(), "chat")
		err = eventBus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for message delivery
		time.Sleep(50 * time.Millisecond)

		// Verify message was received
		select {
		case msg := <-receiveChan:
			var wsMsg map[string]any
			require.NoError(t, json.Unmarshal(msg, &wsMsg))
			assert.Equal(t, "chat.message.posted", wsMsg["type"])
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected message but did not receive")
		}
	})

	t.Run("broadcasts notification to specific user", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(ctx)
		require.NoError(t, err)

		// Create clients
		userID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		client1, receiveChan1 := createTestBroadcasterClient(t, hub, userID)
		client2, receiveChan2 := createTestBroadcasterClient(t, hub, otherUserID)
		hub.Register(client1)
		hub.Register(client2)
		time.Sleep(20 * time.Millisecond)

		// Publish notification event with user_id in payload
		payload := map[string]string{"user_id": userID.String()}
		evt := newTestDomainEventWithPayload("notification.created", uuid.NewUUID().String(), "notification", payload)
		err = eventBus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for message delivery
		time.Sleep(50 * time.Millisecond)

		// Only user1 should receive
		select {
		case msg := <-receiveChan1:
			var wsMsg map[string]any
			require.NoError(t, json.Unmarshal(msg, &wsMsg))
			assert.Equal(t, "notification.new", wsMsg["type"])
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected message for user1 but did not receive")
		}

		select {
		case <-receiveChan2:
			t.Fatal("user2 should not receive the notification")
		case <-time.After(50 * time.Millisecond):
			// Expected - no message
		}
	})

	t.Run("does not broadcast unregistered event types", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus,
			ws.WithEventTypes([]string{"message.created"}), // Only subscribe to message.created
		)

		err := broadcaster.Start(ctx)
		require.NoError(t, err)

		// Create a client
		chatID := uuid.NewUUID()
		client, receiveChan := createTestBroadcasterClient(t, hub, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(20 * time.Millisecond)
		hub.JoinChat(client, chatID)
		time.Sleep(20 * time.Millisecond)

		// Publish event that we're not subscribed to
		evt := newTestDomainEvent("task.updated", chatID.String(), "chat")
		err = eventBus.Publish(ctx, evt)
		require.NoError(t, err)

		// Should not receive anything
		select {
		case <-receiveChan:
			t.Fatal("should not receive message for unsubscribed event type")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})

	t.Run("handles chat.updated event", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(ctx)
		require.NoError(t, err)

		// Create a client and subscribe to a chat
		chatID := uuid.NewUUID()
		client, receiveChan := createTestBroadcasterClient(t, hub, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(20 * time.Millisecond)
		hub.JoinChat(client, chatID)
		time.Sleep(20 * time.Millisecond)

		// Publish event
		evt := newTestDomainEvent("chat.updated", chatID.String(), "chat")
		err = eventBus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for message delivery
		time.Sleep(50 * time.Millisecond)

		// Verify message was received
		select {
		case msg := <-receiveChan:
			var wsMsg map[string]any
			require.NoError(t, json.Unmarshal(msg, &wsMsg))
			assert.Equal(t, "chat.updated", wsMsg["type"])
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected message but did not receive")
		}
	})

	t.Run("handles task.status_changed event", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(ctx)
		require.NoError(t, err)

		// Create a client and subscribe to a chat
		chatID := uuid.NewUUID()
		client, receiveChan := createTestBroadcasterClient(t, hub, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(20 * time.Millisecond)
		hub.JoinChat(client, chatID)
		time.Sleep(20 * time.Millisecond)

		// Publish event with ChatID in payload (matches task domain event structure)
		payload := map[string]string{"ChatID": chatID.String()}
		evt := newTestDomainEventWithPayload("task.status_changed", uuid.NewUUID().String(), "task", payload)
		err = eventBus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for message delivery
		time.Sleep(50 * time.Millisecond)

		// Verify message was received (task.status_changed maps to task.updated)
		select {
		case msg := <-receiveChan:
			var wsMsg map[string]any
			require.NoError(t, json.Unmarshal(msg, &wsMsg))
			assert.Equal(t, "task.updated", wsMsg["type"])
		case <-time.After(100 * time.Millisecond):
			t.Fatal("expected message but did not receive")
		}
	})
}

func TestBroadcaster_IsRunning(t *testing.T) {
	t.Run("returns false before start", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		assert.False(t, broadcaster.IsRunning())
	})

	t.Run("returns true after start", func(t *testing.T) {
		hub := ws.NewHub()
		eventBus := newMockEventBus()
		broadcaster := ws.NewBroadcaster(hub, eventBus)

		err := broadcaster.Start(context.Background())
		require.NoError(t, err)

		assert.True(t, broadcaster.IsRunning())
	})
}

// Helper function to create a test client with a receive channel
func createTestBroadcasterClient(t *testing.T, hub *ws.Hub, userID uuid.UUID) (*ws.Client, chan []byte) {
	t.Helper()

	// Create WebSocket pair
	serverConn, clientConn, cleanup := createWSConnPair(t)

	client := ws.NewClient(hub, serverConn, userID)
	receiveChan := make(chan []byte, 10)

	// Start goroutine to read messages from the client connection
	go func() {
		for {
			_, msg, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case receiveChan <- msg:
			default:
			}
		}
	}()

	// Start write pump
	go client.WritePump()

	t.Cleanup(func() {
		client.Close()
		cleanup()
	})

	return client, receiveChan
}
