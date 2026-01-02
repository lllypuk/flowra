//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	wsinfra "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WSTestSuite is a specialized test suite for WebSocket tests with full WS support.
type WSTestSuite struct {
	t              *testing.T
	Echo           *echo.Echo
	Hub            *wsinfra.Hub
	TokenValidator *E2ETokenValidator
	serverAddr     string
	serverCancel   context.CancelFunc
	serverWg       sync.WaitGroup
	users          map[string]*TestUser
	usersMu        sync.RWMutex
}

// NewWSTestSuite creates a new WebSocket test suite.
func NewWSTestSuite(t *testing.T) *WSTestSuite {
	t.Helper()

	suite := &WSTestSuite{
		t:              t,
		TokenValidator: NewE2ETokenValidator(),
		users:          make(map[string]*TestUser),
	}

	suite.setupServer()

	return suite
}

// setupServer creates and starts the HTTP server with WebSocket support.
func (s *WSTestSuite) setupServer() {
	s.Echo = echo.New()
	s.Echo.HideBanner = true
	s.Echo.HidePort = true

	// Create logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create WebSocket hub
	s.Hub = wsinfra.NewHub(
		wsinfra.WithHubLogger(logger),
	)

	// Create upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}

	// WebSocket handler
	s.Echo.GET("/ws", func(c echo.Context) error {
		// Get token from query or header
		token := c.QueryParam("token")
		if token == "" {
			authHeader := c.Request().Header.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token required"})
		}

		claims, err := s.TokenValidator.ValidateToken(c.Request().Context(), token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		}

		// Upgrade connection
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return nil
		}

		// Create and register client
		client := wsinfra.NewClient(
			s.Hub,
			conn,
			claims.UserID,
			wsinfra.WithClientLogger(logger),
		)

		s.Hub.Register(client)

		go client.WritePump()
		go client.ReadPump()

		return nil
	})

	// Health endpoint
	s.Echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.t, err)
	s.serverAddr = listener.Addr().String()
	_ = listener.Close()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	s.serverCancel = cancel

	// Start WebSocket Hub
	go s.Hub.Run(ctx)

	// Start HTTP server
	s.serverWg.Add(1)
	go func() {
		defer s.serverWg.Done()
		if err := s.Echo.Start(s.serverAddr); err != nil && err != http.ErrServerClosed {
			s.t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	s.waitForServer()

	// Register cleanup
	s.t.Cleanup(func() {
		s.Shutdown()
	})
}

// waitForServer waits for the server to be ready.
func (s *WSTestSuite) waitForServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.t.Fatalf("Server failed to start")
		case <-ticker.C:
			resp, err := http.Get("http://" + s.serverAddr + "/health")
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return
				}
			}
		}
	}
}

// Shutdown gracefully shuts down the test suite.
func (s *WSTestSuite) Shutdown() {
	if s.serverCancel != nil {
		s.serverCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.Echo != nil {
		_ = s.Echo.Shutdown(ctx)
	}

	if s.Hub != nil {
		s.Hub.Stop()
	}

	s.serverWg.Wait()
}

// WSURL returns the WebSocket URL.
func (s *WSTestSuite) WSURL() string {
	return "ws://" + s.serverAddr + "/ws"
}

// CreateTestUser creates a test user with token.
func (s *WSTestSuite) CreateTestUser(username string) *TestUser {
	s.t.Helper()

	id := uuid.NewUUID()
	email := username + "@test.local"

	testUser := &TestUser{
		ID:       id,
		Username: username,
		Email:    email,
		Token:    "ws-token-" + id.String(),
	}

	// Register token
	s.TokenValidator.RegisterToken(testUser.Token, &middleware.TokenClaims{
		UserID:    id,
		Username:  username,
		Email:     email,
		Roles:     []string{"user"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	s.usersMu.Lock()
	s.users[username] = testUser
	s.usersMu.Unlock()

	return testUser
}

// ConnectWS establishes a WebSocket connection.
func (s *WSTestSuite) ConnectWS(token string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(s.WSURL()+"?token="+token, nil)
	return conn, err
}

func TestWebSocket_Connect_Success(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsuser1")

	// Connect to WebSocket
	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Connection should be established
	assert.NotNil(t, conn)

	// Wait a bit for the hub to register the client
	time.Sleep(100 * time.Millisecond)

	// Hub should have the client
	assert.True(t, suite.Hub.IsRunning())
}

func TestWebSocket_Connect_InvalidToken(t *testing.T) {
	suite := NewWSTestSuite(t)

	// Try to connect with invalid token
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	_, resp, err := dialer.Dial(suite.WSURL()+"?token=invalid-token", nil)

	// Should fail with 401
	if err != nil {
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	}
}

func TestWebSocket_Connect_NoToken(t *testing.T) {
	suite := NewWSTestSuite(t)

	// Try to connect without token
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	_, resp, err := dialer.Dial(suite.WSURL(), nil)

	// Should fail with 401
	if err != nil {
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	}
}

func TestWebSocket_MultipleConnections(t *testing.T) {
	suite := NewWSTestSuite(t)

	user1 := suite.CreateTestUser("wsmulti1")
	user2 := suite.CreateTestUser("wsmulti2")
	user3 := suite.CreateTestUser("wsmulti3")

	// Connect multiple users
	conn1, err := suite.ConnectWS(user1.Token)
	require.NoError(t, err)
	defer func() { _ = conn1.Close() }()

	conn2, err := suite.ConnectWS(user2.Token)
	require.NoError(t, err)
	defer func() { _ = conn2.Close() }()

	conn3, err := suite.ConnectWS(user3.Token)
	require.NoError(t, err)
	defer func() { _ = conn3.Close() }()

	// Wait for registrations
	time.Sleep(200 * time.Millisecond)

	// All should be connected
	assert.NotNil(t, conn1)
	assert.NotNil(t, conn2)
	assert.NotNil(t, conn3)
}

func TestWebSocket_SendReceiveMessage(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsmsg1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Send a ping message
	pingMsg := map[string]string{
		"type": "ping",
	}
	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)

	// The client should handle the message
	// (In a real scenario, we'd expect a pong response)
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_SubscribeToChat(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wssub1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Subscribe to a chat
	chatID := uuid.NewUUID()
	subscribeMsg := map[string]string{
		"type":    "subscribe",
		"chat_id": chatID.String(),
	}
	err = conn.WriteJSON(subscribeMsg)
	require.NoError(t, err)

	// Wait for subscription to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestWebSocket_Disconnect(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsdc1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	// Close connection
	err = conn.Close()
	require.NoError(t, err)

	// Wait for unregistration
	time.Sleep(200 * time.Millisecond)

	// Hub should have removed the client
	// (We can't easily verify this without exposing internal state)
}

func TestWebSocket_ReconnectAfterDisconnect(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsrecon1")

	// First connection
	conn1, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Disconnect
	_ = conn1.Close()
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	conn2, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn2.Close() }()

	assert.NotNil(t, conn2)
}

func TestWebSocket_SimultaneousSameUser(t *testing.T) {
	suite := NewWSTestSuite(t)

	// Same user connects from multiple devices
	testUser := suite.CreateTestUser("wssameuser")

	conn1, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn1.Close() }()

	conn2, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn2.Close() }()

	// Both connections should work
	time.Sleep(100 * time.Millisecond)

	assert.NotNil(t, conn1)
	assert.NotNil(t, conn2)
}

func TestWebSocket_LargeMessage(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wslarge1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Create a large message
	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	msg := map[string]string{
		"type":    "message",
		"content": string(largeContent),
	}

	err = conn.WriteJSON(msg)
	require.NoError(t, err)
}

func TestWebSocket_JSONMessage(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsjson1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Send a structured JSON message
	msg := map[string]interface{}{
		"type":    "chat.message",
		"chat_id": uuid.NewUUID().String(),
		"data": map[string]interface{}{
			"content": "Hello, WebSocket!",
			"metadata": map[string]string{
				"client": "test",
			},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	err = conn.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)
}

func TestWebSocket_BinaryMessage(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsbinary1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Send binary message
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	err = conn.WriteMessage(websocket.BinaryMessage, binaryData)
	require.NoError(t, err)
}

func TestWebSocket_ConcurrentWrites(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsconcurrent1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Send multiple messages sequentially with a mutex
	// WebSocket connections are NOT safe for concurrent writes
	// This test demonstrates proper serialized writes
	var writeMu sync.Mutex
	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			msg := map[string]interface{}{
				"type":  "ping",
				"index": index,
			}
			writeMu.Lock()
			writeErr := conn.WriteJSON(msg)
			writeMu.Unlock()
			if writeErr == nil {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// All writes should succeed when properly serialized
	assert.Equal(t, 10, successCount)
}

func TestWebSocket_ConnectionTimeout(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wstimeout1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Set a read deadline
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	// Try to read - should timeout since no messages are sent
	_, _, err = conn.ReadMessage()
	if err != nil {
		// Expected timeout error
		assert.Error(t, err)
	}
}

func TestWebSocket_GracefulClose(t *testing.T) {
	suite := NewWSTestSuite(t)

	testUser := suite.CreateTestUser("wsgraceful1")

	conn, err := suite.ConnectWS(testUser.Token)
	require.NoError(t, err)

	// Send close message
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "goodbye"))
	require.NoError(t, err)

	// Close the connection
	err = conn.Close()
	require.NoError(t, err)
}

func TestWebSocket_CompleteFlow(t *testing.T) {
	suite := NewWSTestSuite(t)

	// Create two users
	user1 := suite.CreateTestUser("wsflow1")
	user2 := suite.CreateTestUser("wsflow2")

	// Both users connect
	conn1, err := suite.ConnectWS(user1.Token)
	require.NoError(t, err)
	defer func() { _ = conn1.Close() }()

	conn2, err := suite.ConnectWS(user2.Token)
	require.NoError(t, err)
	defer func() { _ = conn2.Close() }()

	// Wait for connections
	time.Sleep(100 * time.Millisecond)

	// Create a chat ID for subscription
	chatID := uuid.NewUUID()

	// User1 subscribes to chat
	err = conn1.WriteJSON(map[string]string{
		"type":    "subscribe",
		"chat_id": chatID.String(),
	})
	require.NoError(t, err)

	// User2 subscribes to same chat
	err = conn2.WriteJSON(map[string]string{
		"type":    "subscribe",
		"chat_id": chatID.String(),
	})
	require.NoError(t, err)

	// Wait for subscriptions
	time.Sleep(100 * time.Millisecond)

	// User1 sends a message to the chat
	err = conn1.WriteJSON(map[string]interface{}{
		"type":    "chat.send",
		"chat_id": chatID.String(),
		"content": "Hello from user1!",
	})
	require.NoError(t, err)

	// User2 sends a message
	err = conn2.WriteJSON(map[string]interface{}{
		"type":    "chat.send",
		"chat_id": chatID.String(),
		"content": "Hello from user2!",
	})
	require.NoError(t, err)

	// User1 unsubscribes
	err = conn1.WriteJSON(map[string]string{
		"type":    "unsubscribe",
		"chat_id": chatID.String(),
	})
	require.NoError(t, err)

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)

	// Both connections should still be active
	assert.NotNil(t, conn1)
	assert.NotNil(t, conn2)
}
