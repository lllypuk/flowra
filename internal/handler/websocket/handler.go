// Package websocket provides HTTP handlers for WebSocket connections.
package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	ws "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
)

// Handler configuration constants.
const (
	defaultHandlerReadBufferSize  = 1024
	defaultHandlerWriteBufferSize = 1024
)

// TokenValidator defines the interface for validating JWT tokens.
// Declared on the consumer side per project guidelines.
type TokenValidator interface {
	// ValidateToken validates a JWT token and returns the claims.
	ValidateToken(ctx context.Context, token string) (*middleware.TokenClaims, error)
}

// Handler handles WebSocket HTTP requests.
type Handler struct {
	hub            *ws.Hub
	upgrader       websocket.Upgrader
	tokenValidator TokenValidator
	logger         *slog.Logger
	clientConfig   ws.ClientConfig
}

// HandlerConfig holds configuration for the WebSocket handler.
type HandlerConfig struct {
	// ReadBufferSize is the size of the read buffer for WebSocket connections.
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer for WebSocket connections.
	WriteBufferSize int

	// CheckOrigin is a function that returns true if the request origin is acceptable.
	// If nil, a default function allowing all origins is used.
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

// WithTokenValidator sets the token validator for the handler.
func WithTokenValidator(validator TokenValidator) HandlerOption {
	return func(h *Handler) {
		h.tokenValidator = validator
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
			CheckOrigin: func(_ *http.Request) bool {
				// Allow all origins in development
				// In production, this should be configured properly
				return true
			},
		},
		logger:       slog.Default(),
		clientConfig: ws.DefaultClientConfig(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// HandleWebSocket handles WebSocket upgrade requests.
// It validates the JWT token from query parameter or header, upgrades the connection,
// and registers the client with the hub.
func (h *Handler) HandleWebSocket(c echo.Context) error {
	// Get user ID from context (set by auth middleware) or validate token
	userID := h.getUserID(c)
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

// getUserID extracts the user ID from the echo context or validates the token.
func (h *Handler) getUserID(c echo.Context) uuid.UUID {
	// First, try to get user ID from context (set by auth middleware)
	if userID := middleware.GetUserID(c); !userID.IsZero() {
		return userID
	}

	// If not in context, try to validate token from query parameter
	token := c.QueryParam("token")
	if token == "" {
		// Try to get from Authorization header
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader != "" {
			const bearerPrefix = "Bearer "
			if after, ok := strings.CutPrefix(authHeader, bearerPrefix); ok {
				token = after
			}
		}
	}

	if token == "" || h.tokenValidator == nil {
		return uuid.UUID("")
	}

	// Validate token
	claims, err := h.tokenValidator.ValidateToken(c.Request().Context(), token)
	if err != nil {
		h.logger.Debug("token validation failed",
			slog.String("error", err.Error()),
		)
		return uuid.UUID("")
	}

	return claims.UserID
}

// RegisterRoutes registers the WebSocket handler with the Echo router.
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws", h.HandleWebSocket)
}

// RegisterRoutesWithGroup registers the WebSocket handler with an Echo group.
func (h *Handler) RegisterRoutesWithGroup(g *echo.Group) {
	g.GET("/ws", h.HandleWebSocket)
}
