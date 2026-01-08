package websocket_test

import (
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

func TestNewClient(t *testing.T) {
	t.Run("creates client with defaults", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		userID := uuid.NewUUID()
		client := ws.NewClient(hub, serverConn, userID)

		assert.NotNil(t, client)
		assert.Equal(t, userID, client.UserID())
		assert.Empty(t, client.GetChatIDs())
		assert.False(t, client.IsClosed())
	})

	t.Run("creates client with custom config", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		config := ws.ClientConfig{
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			PingInterval:    15 * time.Second,
			PongWait:        30 * time.Second,
			WriteWait:       5 * time.Second,
			MaxMessageSize:  32768,
		}

		userID := uuid.NewUUID()
		client := ws.NewClient(hub, serverConn, userID,
			ws.WithClientConfig(config),
		)

		assert.NotNil(t, client)
	})
}

func TestClient_ChatIDs(t *testing.T) {
	t.Run("adds chat ID", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		chatID := uuid.NewUUID()

		client.AddChat(chatID)

		assert.True(t, client.HasChat(chatID))
		assert.Contains(t, client.GetChatIDs(), chatID)
	})

	t.Run("removes chat ID", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		chatID := uuid.NewUUID()

		client.AddChat(chatID)
		assert.True(t, client.HasChat(chatID))

		client.RemoveChat(chatID)
		assert.False(t, client.HasChat(chatID))
	})

	t.Run("handles multiple chat IDs", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		chatID1 := uuid.NewUUID()
		chatID2 := uuid.NewUUID()
		chatID3 := uuid.NewUUID()

		client.AddChat(chatID1)
		client.AddChat(chatID2)
		client.AddChat(chatID3)

		assert.Len(t, client.GetChatIDs(), 3)
		assert.True(t, client.HasChat(chatID1))
		assert.True(t, client.HasChat(chatID2))
		assert.True(t, client.HasChat(chatID3))
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("closes connection", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())

		assert.False(t, client.IsClosed())
		client.Close()
		assert.True(t, client.IsClosed())
	})

	t.Run("close is idempotent", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())

		client.Close()
		// Should not panic on second close
		client.Close()
		assert.True(t, client.IsClosed())
	})
}

func TestClient_Send(t *testing.T) {
	t.Run("sends message to client", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())

		// Start write pump
		go client.WritePump()

		// Send message
		message := []byte(`{"type":"test"}`)
		client.Send(message)

		// Read from client connection
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, received, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var expectedJSON, receivedJSON any
		require.NoError(t, json.Unmarshal(message, &expectedJSON))
		require.NoError(t, json.Unmarshal(received, &receivedJSON))
		assert.Equal(t, expectedJSON, receivedJSON)
	})

	t.Run("does not send to closed client", func(t *testing.T) {
		hub := ws.NewHub()
		serverConn, _, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		client.Close()

		// Should not panic
		client.Send([]byte(`{"type":"test"}`))
	})
}

func TestClient_HandleClientMessage(t *testing.T) {
	t.Run("handles subscribe message", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		client := ws.NewClient(hub, serverConn, userID)
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send subscribe message from client
		subscribeMsg := map[string]any{
			"type":    "subscribe",
			"chat_id": chatID.String(),
		}
		msgBytes, _ := json.Marshal(subscribeMsg)
		err := clientConn.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(50 * time.Millisecond)

		// Client should be in chat room
		assert.True(t, client.HasChat(chatID))
		assert.Equal(t, 1, hub.ClientsInChat(chatID))

		// Read ack response
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, response, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var ack map[string]any
		require.NoError(t, json.Unmarshal(response, &ack))
		assert.Equal(t, "ack", ack["type"])
		assert.Equal(t, "subscribed", ack["action"])
	})

	t.Run("handles unsubscribe message", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		userID := uuid.NewUUID()
		chatID := uuid.NewUUID()
		client := ws.NewClient(hub, serverConn, userID)
		hub.Register(client)
		hub.JoinChat(client, chatID)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send unsubscribe message from client
		unsubscribeMsg := map[string]any{
			"type":    "unsubscribe",
			"chat_id": chatID.String(),
		}
		msgBytes, _ := json.Marshal(unsubscribeMsg)
		err := clientConn.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(50 * time.Millisecond)

		// Client should not be in chat room
		assert.False(t, client.HasChat(chatID))
		assert.Equal(t, 0, hub.ClientsInChat(chatID))
	})

	t.Run("handles ping message", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send ping message from client
		pingMsg := map[string]string{"type": "ping"}
		msgBytes, _ := json.Marshal(pingMsg)
		err := clientConn.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Read pong response
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, response, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var pong map[string]any
		require.NoError(t, json.Unmarshal(response, &pong))
		assert.Equal(t, "pong", pong["type"])
	})

	t.Run("handles unknown message type", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send unknown message type from client
		unknownMsg := map[string]string{"type": "unknown_type"}
		msgBytes, _ := json.Marshal(unknownMsg)
		err := clientConn.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Read error response
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, response, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var errorResp map[string]any
		require.NoError(t, json.Unmarshal(response, &errorResp))
		assert.Equal(t, "error", errorResp["type"])
		assert.Contains(t, errorResp["message"], "unknown message type")
	})

	t.Run("handles invalid JSON message", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send invalid JSON
		err := clientConn.WriteMessage(websocket.TextMessage, []byte(`{invalid json`))
		require.NoError(t, err)

		// Read error response
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, response, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var errorResp map[string]any
		require.NoError(t, json.Unmarshal(response, &errorResp))
		assert.Equal(t, "error", errorResp["type"])
	})

	t.Run("subscribe without chat_id returns error", func(t *testing.T) {
		hub := ws.NewHub()
		ctx := t.Context()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		serverConn, clientConn, cleanup := createWSConnPair(t)
		defer cleanup()

		client := ws.NewClient(hub, serverConn, uuid.NewUUID())
		hub.Register(client)
		time.Sleep(10 * time.Millisecond)

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Send subscribe without chat_id
		subscribeMsg := map[string]string{"type": "subscribe"}
		msgBytes, _ := json.Marshal(subscribeMsg)
		err := clientConn.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Read error response
		clientConn.SetReadDeadline(time.Now().Add(time.Second))
		_, response, err := clientConn.ReadMessage()
		require.NoError(t, err)

		var errorResp map[string]any
		require.NoError(t, json.Unmarshal(response, &errorResp))
		assert.Equal(t, "error", errorResp["type"])
		assert.Contains(t, errorResp["message"], "chat_id")
	})
}

func TestDefaultClientConfig(t *testing.T) {
	config := ws.DefaultClientConfig()

	assert.Equal(t, 1024, config.ReadBufferSize)
	assert.Equal(t, 1024, config.WriteBufferSize)
	assert.Equal(t, 30*time.Second, config.PingInterval)
	assert.Equal(t, 60*time.Second, config.PongWait)
	assert.Equal(t, 10*time.Second, config.WriteWait)
	assert.Equal(t, int64(65536), config.MaxMessageSize)
}

// Helper functions

func createWSConnPair(t *testing.T) (*websocket.Conn, *websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	serverChan := make(chan *websocket.Conn, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverChan <- conn
	}))

	wsURL := "ws" + server.URL[4:]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	select {
	case serverConn := <-serverChan:
		cleanup := func() {
			serverConn.Close()
			clientConn.Close()
			server.Close()
		}
		return serverConn, clientConn, cleanup
	case <-time.After(time.Second):
		clientConn.Close()
		server.Close()
		t.Fatal("timeout waiting for server connection")
		return nil, nil, nil
	}
}
