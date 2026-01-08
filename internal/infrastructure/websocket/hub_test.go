package websocket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	ws "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	t.Run("creates hub with defaults", func(t *testing.T) {
		hub := ws.NewHub()

		assert.NotNil(t, hub)
		assert.False(t, hub.IsRunning())
		assert.Equal(t, 0, hub.ClientCount())
		assert.Equal(t, 0, hub.ChatRoomCount())
	})

	t.Run("creates hub with logger option", func(t *testing.T) {
		hub := ws.NewHub(ws.WithHubLogger(nil))

		assert.NotNil(t, hub)
	})
}

func TestHub_Run(t *testing.T) {
	t.Run("starts and stops with context cancellation", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			hub.Run(ctx)
			close(done)
		}()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)
		assert.True(t, hub.IsRunning())

		// Cancel context
		cancel()

		// Wait for hub to stop
		select {
		case <-done:
			assert.False(t, hub.IsRunning())
		case <-time.After(time.Second):
			t.Fatal("hub did not stop in time")
		}
	})

	t.Run("stops with Stop method", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := context.Background()

		done := make(chan struct{})
		go func() {
			hub.Run(ctx)
			close(done)
		}()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)
		assert.True(t, hub.IsRunning())

		// Stop hub
		hub.Stop()

		// Wait for hub to stop
		select {
		case <-done:
			assert.False(t, hub.IsRunning())
		case <-time.After(time.Second):
			t.Fatal("hub did not stop in time")
		}
	})

	t.Run("does not start twice", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		done1 := make(chan struct{})
		go func() {
			hub.Run(ctx)
			close(done1)
		}()

		// Give hub time to start
		time.Sleep(10 * time.Millisecond)

		// Try to start again (should return immediately)
		done2 := make(chan struct{})
		go func() {
			hub.Run(ctx)
			close(done2)
		}()

		// Second Run should return immediately
		select {
		case <-done2:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("second Run call did not return immediately")
		}
	})
}

func TestHub_RegisterUnregister(t *testing.T) {
	t.Run("registers and counts client", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		client := createMockClient(t, hub, userID)

		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, hub.ClientCount())
		assert.Equal(t, 1, hub.UserConnectionCount(userID))
	})

	t.Run("unregisters client", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		client := createMockClient(t, hub, userID)

		hub.Register(client)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())

		hub.Unregister(client)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 0, hub.ClientCount())
		assert.Equal(t, 0, hub.UserConnectionCount(userID))
	})

	t.Run("handles multiple clients for same user", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		client1 := createMockClient(t, hub, userID)
		client2 := createMockClient(t, hub, userID)

		hub.Register(client1)
		hub.Register(client2)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 2, hub.ClientCount())
		assert.Equal(t, 2, hub.UserConnectionCount(userID))

		hub.Unregister(client1)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())
		assert.Equal(t, 1, hub.UserConnectionCount(userID))
	})
}

func TestHub_ChatRooms(t *testing.T) {
	t.Run("joins chat room", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		client := createMockClient(t, hub, userID)

		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		hub.JoinChat(client, chatID)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, hub.ChatRoomCount())
		assert.Equal(t, 1, hub.ClientsInChat(chatID))
		assert.True(t, client.HasChat(chatID))
	})

	t.Run("leaves chat room", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		client := createMockClient(t, hub, userID)

		hub.Register(client)
		hub.JoinChat(client, chatID)
		time.Sleep(10 * time.Millisecond)

		hub.LeaveChat(client, chatID)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 0, hub.ChatRoomCount())
		assert.Equal(t, 0, hub.ClientsInChat(chatID))
		assert.False(t, client.HasChat(chatID))
	})

	t.Run("multiple clients in same chat", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		chatID := uuid.NewUUID()
		client1 := createMockClient(t, hub, uuid.NewUUID())
		client2 := createMockClient(t, hub, uuid.NewUUID())

		hub.Register(client1)
		hub.Register(client2)
		hub.JoinChat(client1, chatID)
		hub.JoinChat(client2, chatID)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, hub.ChatRoomCount())
		assert.Equal(t, 2, hub.ClientsInChat(chatID))
	})

	t.Run("removes chat room when empty", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		chatID := uuid.NewUUID()
		client := createMockClient(t, hub, uuid.NewUUID())

		hub.Register(client)
		hub.JoinChat(client, chatID)
		time.Sleep(10 * time.Millisecond)

		hub.Unregister(client)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 0, hub.ChatRoomCount())
	})
}

func TestHub_BroadcastToChat(t *testing.T) {
	t.Run("broadcasts message to chat members", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		chatID := uuid.NewUUID()
		client1, sendChan1 := createTestClientWithChannel(t, hub, uuid.NewUUID())
		client2, sendChan2 := createTestClientWithChannel(t, hub, uuid.NewUUID())

		hub.Register(client1)
		hub.Register(client2)
		hub.JoinChat(client1, chatID)
		hub.JoinChat(client2, chatID)
		time.Sleep(10 * time.Millisecond)

		message := []byte(`{"type":"test","data":"hello"}`)
		hub.BroadcastToChat(chatID, message)
		time.Sleep(10 * time.Millisecond)

		// Both clients should receive the message
		assertReceived(t, sendChan1, message)
		assertReceived(t, sendChan2, message)
	})

	t.Run("does not broadcast to clients not in chat", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		chatID := uuid.NewUUID()
		otherChatID := uuid.NewUUID()
		client1, sendChan1 := createTestClientWithChannel(t, hub, uuid.NewUUID())
		client2, sendChan2 := createTestClientWithChannel(t, hub, uuid.NewUUID())

		hub.Register(client1)
		hub.Register(client2)
		hub.JoinChat(client1, chatID)
		hub.JoinChat(client2, otherChatID)
		time.Sleep(10 * time.Millisecond)

		message := []byte(`{"type":"test","data":"hello"}`)
		hub.BroadcastToChat(chatID, message)
		time.Sleep(10 * time.Millisecond)

		// Only client1 should receive the message
		assertReceived(t, sendChan1, message)
		assertNotReceived(t, sendChan2)
	})
}

func TestHub_SendToUser(t *testing.T) {
	t.Run("sends message to specific user", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		client1, sendChan1 := createTestClientWithChannel(t, hub, userID)
		client2, sendChan2 := createTestClientWithChannel(t, hub, otherUserID)

		hub.Register(client1)
		hub.Register(client2)
		time.Sleep(10 * time.Millisecond)

		message := []byte(`{"type":"notification","data":"hello"}`)
		hub.SendToUser(userID, message)
		time.Sleep(10 * time.Millisecond)

		assertReceived(t, sendChan1, message)
		assertNotReceived(t, sendChan2)
	})

	t.Run("sends message to all user connections", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		client1, sendChan1 := createTestClientWithChannel(t, hub, userID)
		client2, sendChan2 := createTestClientWithChannel(t, hub, userID)

		hub.Register(client1)
		hub.Register(client2)
		time.Sleep(10 * time.Millisecond)

		message := []byte(`{"type":"notification","data":"hello"}`)
		hub.SendToUser(userID, message)
		time.Sleep(10 * time.Millisecond)

		assertReceived(t, sendChan1, message)
		assertReceived(t, sendChan2, message)
	})
}

// Helper functions

func createMockClient(t *testing.T, hub *ws.Hub, userID uuid.UUID) *ws.Client {
	t.Helper()

	// Create a mock websocket.Conn using a pipe
	server, client, err := createWebSocketPair(t)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	return ws.NewClient(hub, server, userID)
}

func createTestClientWithChannel(t *testing.T, hub *ws.Hub, userID uuid.UUID) (*ws.Client, chan []byte) {
	t.Helper()

	// Create a mock websocket.Conn using a pipe
	server, clientConn, err := createWebSocketPair(t)
	require.NoError(t, err)

	client := ws.NewClient(hub, server, userID)
	sendChan := make(chan []byte, 10)

	// Start a goroutine to read from the client connection
	go func() {
		for {
			_, msg, readErr := clientConn.ReadMessage()
			if readErr != nil {
				return
			}
			select {
			case sendChan <- msg:
			default:
			}
		}
	}()

	// Start write pump to actually send messages
	go client.WritePump()

	t.Cleanup(func() {
		client.Close()
		_ = clientConn.Close()
	})

	return client, sendChan
}

func createWebSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, error) {
	t.Helper()

	// Create an HTTP test server
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	serverChan := make(chan *websocket.Conn, 1)

	server := newTestWSServer(t, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverChan <- conn
	})

	// Connect to the server
	clientConn, _, err := websocket.DefaultDialer.Dial(server.URL, nil)
	if err != nil {
		return nil, nil, err
	}

	// Wait for server connection
	select {
	case serverConn := <-serverChan:
		return serverConn, clientConn, nil
	case <-time.After(time.Second):
		clientConn.Close()
		return nil, nil, context.DeadlineExceeded
	}
}

func assertReceived(t *testing.T, ch chan []byte, expected []byte) {
	t.Helper()
	select {
	case received := <-ch:
		// Compare JSON to handle formatting differences
		var expectedJSON, receivedJSON any
		if unmarshalErr := json.Unmarshal(expected, &expectedJSON); unmarshalErr == nil {
			if unmarshalErr2 := json.Unmarshal(received, &receivedJSON); unmarshalErr2 == nil {
				assert.Equal(t, expectedJSON, receivedJSON)
				return
			}
		}
		assert.Equal(t, expected, received)
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive message but did not")
	}
}

func assertNotReceived(t *testing.T, ch chan []byte) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Errorf("expected no message but received: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
		// Expected - no message received
	}
}

// testWSServer is a helper for creating test WebSocket servers.
type testWSServer struct {
	*httptest.Server

	URL string
}

func newTestWSServer(t *testing.T, handler http.HandlerFunc) *testWSServer {
	t.Helper()
	server := httptest.NewServer(handler)
	wsURL := "ws" + server.URL[4:] // Convert http:// to ws://
	t.Cleanup(server.Close)
	return &testWSServer{Server: server, URL: wsURL}
}
