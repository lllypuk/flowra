// Package websocket provides HTTP handlers for WebSocket connections.
package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	ws "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Handler configuration constants.
const (
	defaultHandlerReadBufferSize  = 1024
	defaultHandlerWriteBufferSize = 1024
)

// Handler handles WebSocket HTTP requests.
type Handler struct {
	hub          *ws.Hub
	upgrader     websocket.Upgrader
	logger       *slog.Logger
	clientConfig ws.ClientConfig
}

// HandlerConfig holds configuration for the WebSocket handler.
type HandlerConfig struct {
	// ReadBufferSize is the size of the read buffer for WebSocket connections.
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer for WebSocket connections.
	WriteBufferSize int

	// CheckOrigin reports whether the request Origin is acceptable.
	// If nil, the upgrader's default applies: an Origin header must match the request Host.
	CheckOrigin func(r *http.Request) bool

	// Logger is the structured logger for the handler.
	Logger *slog.Logger

	// ClientConfig is the configuration for WebSocket clients.
	ClientConfig ws.ClientConfig
}

// DefaultHandlerConfig returns a default configuration.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		ReadBufferSize:  defaultHandlerReadBufferSize,
		WriteBufferSize: defaultHandlerWriteBufferSize,
		CheckOrigin:     nil,
		Logger:          slog.Default(),
		ClientConfig:    ws.DefaultClientConfig(),
	}
}

// HandlerOption configures the Handler.
type HandlerOption func(*Handler)

// WithHandlerLogger sets the logger for the handler.
func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

// WithHandlerConfig sets the handler configuration.
func WithHandlerConfig(config HandlerConfig) HandlerOption {
	return func(h *Handler) {
		h.upgrader.ReadBufferSize = config.ReadBufferSize
		h.upgrader.WriteBufferSize = config.WriteBufferSize
		if config.CheckOrigin != nil {
			h.upgrader.CheckOrigin = config.CheckOrigin
		}
		if config.Logger != nil {
			h.logger = config.Logger
		}
		h.clientConfig = config.ClientConfig
	}
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *ws.Hub, opts ...HandlerOption) *Handler {
	h := &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  defaultHandlerReadBufferSize,
			WriteBufferSize: defaultHandlerWriteBufferSize,
		},
		logger:       slog.Default(),
		clientConfig: ws.DefaultClientConfig(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// HandleWebSocket upgrades an authenticated request and registers the client with the hub.
// The user comes from middleware.Auth (session cookie or Bearer header); the URL is never read for credentials.
func (h *Handler) HandleWebSocket(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		h.logger.Warn("websocket connection rejected: authentication required",
			slog.String("remote_ip", c.RealIP()),
		)
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil // Upgrade already sent an error response
	}

	// Create client with configuration
	client := ws.NewClient(
		h.hub,
		conn,
		userID,
		ws.WithClientConfig(h.clientConfig),
		ws.WithClientLogger(h.logger),
	)

	// Register client with hub
	h.hub.Register(client)

	h.logger.Info("websocket connection established",
		slog.String("user_id", userID.String()),
		slog.String("remote_ip", c.RealIP()),
	)

	// Start client pumps in goroutines
	go client.WritePump()
	go client.ReadPump()

	return nil
}

// RegisterRoutes registers the WebSocket handler with the Echo router.
// The caller must put the authentication middleware in front of it.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws", h.HandleWebSocket, middleware.RequireSameOriginForCookie)
}

// RegisterRoutesWithGroup registers the WebSocket handler with an Echo group.
func (h *Handler) RegisterRoutesWithGroup(g *echo.Group) {
	g.GET("/ws", h.HandleWebSocket, middleware.RequireSameOriginForCookie)
}
