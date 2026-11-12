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

const (
	ownerToken    = "owner-secret"
	sessionCookie = "flowra_session"
)

type mapValidator map[string]*middleware.TokenClaims

func (m mapValidator) ValidateToken(_ context.Context, token string) (*middleware.TokenClaims, error) {
	if claims, ok := m[token]; ok {
		return claims, nil
	}
	return nil, middleware.ErrInvalidToken
}

// newWSServer wires the handler like cmd/api/routes.go: /api/v1 group, Auth, then the same-origin check.
func newWSServer(t *testing.T) (*httptest.Server, *ws.Hub) {
	t.Helper()

	hub := ws.NewHub()
	go hub.Run(t.Context())

	handler := wshandler.NewHandler(hub)
	e := echo.New()
	api := e.Group("/api/v1", middleware.Auth(middleware.AuthConfig{
		TokenValidator:    mapValidator{ownerToken: {UserID: uuid.NewUUID(), Username: "owner"}},
		SessionCookieName: sessionCookie,
	}))
	api.GET("/ws", handler.HandleWebSocket, middleware.RequireSameOriginForCookie)

	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return server, hub
}

func dial(
	t *testing.T, server *httptest.Server, query string, header http.Header,
) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	url := "ws" + server.URL[len("http"):] + "/api/v1/ws" + query
	return websocket.DefaultDialer.Dial(url, header)
}

func cookieHeader(origin, referer string) http.Header {
	h := http.Header{}
	h.Set("Cookie", sessionCookie+"="+ownerToken)
	if origin != "" {
		h.Set("Origin", origin)
	}
	if referer != "" {
		h.Set("Referer", referer)
	}
	return h
}

func requireRefused(t *testing.T, conn *websocket.Conn, resp *http.Response, err error, status int) {
	t.Helper()
	if conn != nil {
		_ = conn.Close()
	}
	require.ErrorIs(t, err, websocket.ErrBadHandshake)
	require.NotNil(t, resp)
	assert.Equal(t, status, resp.StatusCode)
}

func TestHandler_Handshake(t *testing.T) {
	t.Run("cookie with matching Origin upgrades", func(t *testing.T) {
		server, hub := newWSServer(t)

		conn, resp, err := dial(t, server, "", cookieHeader(server.URL, ""))

		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, time.Second, 10*time.Millisecond)
	})

	t.Run("cookie with matching Referer and no Origin upgrades", func(t *testing.T) {
		server, _ := newWSServer(t)

		conn, resp, err := dial(t, server, "", cookieHeader("", server.URL+"/board"))

		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})

	t.Run("owner token in the query is refused", func(t *testing.T) {
		server, hub := newWSServer(t)

		conn, resp, err := dial(t, server, "?token="+ownerToken, nil)

		requireRefused(t, conn, resp, err, http.StatusUnauthorized)
		assert.Zero(t, hub.ClientCount())
	})

	t.Run("owner token in the query does not replace a missing cookie", func(t *testing.T) {
		server, _ := newWSServer(t)
		header := http.Header{}
		header.Set("Origin", server.URL)

		conn, resp, err := dial(t, server, "?token="+ownerToken, header)

		requireRefused(t, conn, resp, err, http.StatusUnauthorized)
	})

	t.Run("cookie with a foreign Origin is refused", func(t *testing.T) {
		server, hub := newWSServer(t)

		conn, resp, err := dial(t, server, "", cookieHeader("http://evil.example", ""))

		requireRefused(t, conn, resp, err, http.StatusForbidden)
		assert.Zero(t, hub.ClientCount())
	})

	t.Run("cookie without Origin and Referer is refused", func(t *testing.T) {
		server, hub := newWSServer(t)

		conn, resp, err := dial(t, server, "", cookieHeader("", ""))

		requireRefused(t, conn, resp, err, http.StatusForbidden)
		assert.Zero(t, hub.ClientCount())
	})

	t.Run("Bearer header without Origin upgrades", func(t *testing.T) {
		server, _ := newWSServer(t)
		header := http.Header{}
		header.Set("Authorization", "Bearer "+ownerToken)

		conn, resp, err := dial(t, server, "", header)

		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})

	t.Run("invalid Bearer token is refused", func(t *testing.T) {
		server, _ := newWSServer(t)
		header := http.Header{}
		header.Set("Authorization", "Bearer wrong")

		conn, resp, err := dial(t, server, "", header)

		requireRefused(t, conn, resp, err, http.StatusUnauthorized)
	})

	t.Run("no credentials is refused", func(t *testing.T) {
		server, _ := newWSServer(t)

		conn, resp, err := dial(t, server, "", nil)

		requireRefused(t, conn, resp, err, http.StatusUnauthorized)
	})
}

func TestHandler_ConnectionLifecycle(t *testing.T) {
	server, hub := newWSServer(t)
	header := http.Header{}
	header.Set("Authorization", "Bearer "+ownerToken)

	conn, _, err := dial(t, server, "", header)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, time.Second, 10*time.Millisecond)

	require.NoError(t, conn.WriteJSON(map[string]string{"type": "ping"}))
	var response map[string]any
	require.NoError(t, conn.ReadJSON(&response))
	assert.Equal(t, "pong", response["type"])

	_ = conn.Close()
	require.Eventually(t, func() bool { return hub.ClientCount() == 0 }, time.Second, 10*time.Millisecond)
}

func TestHandler_HandleWebSocket_NoUserInContext(t *testing.T) {
	handler := wshandler.NewHandler(ws.NewHub())
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+ownerToken, nil)
	rec := httptest.NewRecorder()

	err := handler.HandleWebSocket(echo.New().NewContext(req, rec))

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with defaults", func(t *testing.T) {
		assert.NotNil(t, wshandler.NewHandler(ws.NewHub()))
	})

	t.Run("creates handler with custom config", func(t *testing.T) {
		config := wshandler.HandlerConfig{
			ReadBufferSize:  2048,
			WriteBufferSize: 2048,
			CheckOrigin:     func(r *http.Request) bool { return r.Host == "example.com" },
		}

		assert.NotNil(t, wshandler.NewHandler(ws.NewHub(), wshandler.WithHandlerConfig(config)))
	})
}

func TestDefaultHandlerConfig(t *testing.T) {
	config := wshandler.DefaultHandlerConfig()

	assert.Equal(t, 1024, config.ReadBufferSize)
	assert.Equal(t, 1024, config.WriteBufferSize)
	assert.Nil(t, config.CheckOrigin)
	assert.NotNil(t, config.Logger)
}

func TestHandler_RegisterRoutes(t *testing.T) {
	t.Run("registers route on echo instance", func(t *testing.T) {
		e := echo.New()
		wshandler.NewHandler(ws.NewHub()).RegisterRoutes(e)

		assert.True(t, hasRoute(e, "/ws"), "expected /ws route to be registered")
	})

	t.Run("registers route on echo group", func(t *testing.T) {
		e := echo.New()
		wshandler.NewHandler(ws.NewHub()).RegisterRoutesWithGroup(e.Group("/api/v1"))

		assert.True(t, hasRoute(e, "/api/v1/ws"), "expected /api/v1/ws route to be registered")
	})
}

func hasRoute(e *echo.Echo, path string) bool {
	for _, r := range e.Routes() {
		if r.Path == path && r.Method == http.MethodGet {
			return true
		}
	}
	return false
}
