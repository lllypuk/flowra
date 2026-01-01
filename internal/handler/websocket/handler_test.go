package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	wshandler "github.com/lllypuk/flowra/internal/handler/websocket"
	ws "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenValidator is a mock implementation of TokenValidator for testing.
type mockTokenValidator struct {
	claims *middleware.TokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateToken(_ context.Context, _ string) (*middleware.TokenClaims, error) {
	return m.claims, m.err
}

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with defaults", func(t *testing.T) {
		hub := ws.NewHub()
		handler := wshandler.NewHandler(hub)

		assert.NotNil(t, handler)
	})

	t.Run("creates handler with options", func(t *testing.T) {
		hub := ws.NewHub()
		validator := &mockTokenValidator{}

		handler := wshandler.NewHandler(hub,
			wshandler.WithTokenValidator(validator),
		)

		assert.NotNil(t, handler)
	})

	t.Run("creates handler with custom config", func(t *testing.T) {
		hub := ws.NewHub()
		config := wshandler.HandlerConfig{
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			CheckOrigin: func(r *http.Request) bool {
				return r.Host == "example.com"
			},
		}

		handler := wshandler.NewHandler(hub,
			wshandler.WithHandlerConfig(config),
		)

		assert.NotNil(t, handler)
	})
}

func TestDefaultHandlerConfig(t *testing.T) {
	config := wshandler.DefaultHandlerConfig()

	assert.Equal(t, 1024, config.ReadBufferSize)
	assert.Equal(t, 1024, config.WriteBufferSize)
	assert.Nil(t, config.CheckOrigin)
	assert.NotNil(t, config.Logger)
}

func TestHandler_HandleWebSocket(t *testing.T) {
	t.Run("rejects unauthenticated request", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		handler := wshandler.NewHandler(hub)

		// Create echo context without user ID
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.HandleWebSocket(c)

		require.NoError(t, err) // Error is returned as JSON response
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("accepts authenticated request from context", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		handler := wshandler.NewHandler(hub)

		userID := uuid.NewUUID()

		// Create test server that uses the handler
		e := echo.New()
		e.GET("/ws", func(c echo.Context) error {
			// Simulate auth middleware setting user ID
			c.Set(string(middleware.ContextKeyUserID), userID)
			return handler.HandleWebSocket(c)
		})

		server := httptest.NewServer(e)
		defer server.Close()

		// Connect via WebSocket
		wsURL := "ws" + server.URL[4:] + "/ws"
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		conn.Close()

		// Wait for hub to process registration
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("accepts request with token in query param", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		validator := &mockTokenValidator{
			claims: &middleware.TokenClaims{
				UserID:   userID,
				Username: "testuser",
			},
		}

		handler := wshandler.NewHandler(hub,
			wshandler.WithTokenValidator(validator),
		)

		// Create test server
		e := echo.New()
		e.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(e)
		defer server.Close()

		// Connect via WebSocket with token in query param
		wsURL := "ws" + server.URL[4:] + "/ws?token=valid-token"
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		conn.Close()
	})

	t.Run("accepts request with token in Authorization header", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		validator := &mockTokenValidator{
			claims: &middleware.TokenClaims{
				UserID:   userID,
				Username: "testuser",
			},
		}

		handler := wshandler.NewHandler(hub,
			wshandler.WithTokenValidator(validator),
		)

		// Create test server
		e := echo.New()
		e.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(e)
		defer server.Close()

		// Connect via WebSocket with Authorization header
		wsURL := "ws" + server.URL[4:] + "/ws"
		headers := http.Header{}
		headers.Set("Authorization", "Bearer valid-token")
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		conn.Close()
	})

	t.Run("rejects request with invalid token", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		validator := &mockTokenValidator{
			claims: nil,
			err:    middleware.ErrInvalidToken,
		}

		handler := wshandler.NewHandler(hub,
			wshandler.WithTokenValidator(validator),
		)

		// Create echo context
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ws?token=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.HandleWebSocket(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_RegisterRoutes(t *testing.T) {
	t.Run("registers route on echo instance", func(t *testing.T) {
		hub := ws.NewHub()
		handler := wshandler.NewHandler(hub)

		e := echo.New()
		handler.RegisterRoutes(e)

		routes := e.Routes()
		found := false
		for _, r := range routes {
			if r.Path == "/ws" && r.Method == http.MethodGet {
				found = true
				break
			}
		}
		assert.True(t, found, "expected /ws route to be registered")
	})

	t.Run("registers route on echo group", func(t *testing.T) {
		hub := ws.NewHub()
		handler := wshandler.NewHandler(hub)

		e := echo.New()
		g := e.Group("/api/v1")
		handler.RegisterRoutesWithGroup(g)

		routes := e.Routes()
		found := false
		for _, r := range routes {
			if r.Path == "/api/v1/ws" && r.Method == http.MethodGet {
				found = true
				break
			}
		}
		assert.True(t, found, "expected /api/v1/ws route to be registered")
	})
}

func TestHandler_Integration(t *testing.T) {
	t.Run("full connection lifecycle", func(t *testing.T) {
		hub := ws.NewHub()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go hub.Run(ctx)
		time.Sleep(10 * time.Millisecond)

		userID := uuid.NewUUID()
		validator := &mockTokenValidator{
			claims: &middleware.TokenClaims{
				UserID:   userID,
				Username: "testuser",
			},
		}

		handler := wshandler.NewHandler(hub,
			wshandler.WithTokenValidator(validator),
		)

		// Create test server
		e := echo.New()
		e.GET("/ws", handler.HandleWebSocket)

		server := httptest.NewServer(e)
		defer server.Close()

		// Connect
		wsURL := "ws" + server.URL[4:] + "/ws?token=valid-token"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		// Wait for registration
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 1, hub.ClientCount())

		// Send ping
		writeErr := conn.WriteJSON(map[string]string{"type": "ping"})
		require.NoError(t, writeErr)

		// Receive pong
		var response map[string]interface{}
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, "pong", response["type"])

		// Close connection
		conn.Close()

		// Wait for unregistration
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, 0, hub.ClientCount())
	})
}
