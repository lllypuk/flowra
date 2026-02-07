// Package websocket provides WebSocket server implementation for real-time updates.
package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Default client configuration constants.
const (
	defaultReadBufferSize  = 1024
	defaultWriteBufferSize = 1024
	defaultPingInterval    = 30 * time.Second
	defaultPongWait        = 60 * time.Second
	defaultWriteWait       = 10 * time.Second
	defaultMaxMessageSize  = 65536
	defaultSendBufferSize  = 256
)

// ClientConfig holds configuration for WebSocket clients.
type ClientConfig struct {
	// ReadBufferSize is the size of the read buffer.
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer.
	WriteBufferSize int

	// PingInterval is the interval for sending ping messages.
	PingInterval time.Duration

	// PongWait is the maximum time to wait for a pong response.
	PongWait time.Duration

	// WriteWait is the maximum time to wait for a write operation.
	WriteWait time.Duration

	// MaxMessageSize is the maximum allowed message size.
	MaxMessageSize int64
}

// DefaultClientConfig returns sensible default configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		ReadBufferSize:  defaultReadBufferSize,
		WriteBufferSize: defaultWriteBufferSize,
		PingInterval:    defaultPingInterval,
		PongWait:        defaultPongWait,
		WriteWait:       defaultWriteWait,
		MaxMessageSize:  defaultMaxMessageSize,
	}
}

// ClientMessage represents a message from client to server.
type ClientMessage struct {
	Type   string    `json:"type"`
	ChatID uuid.UUID `json:"chat_id,omitempty"`
}

// Client represents a single WebSocket connection.
type Client struct {
	// hub is the hub this client belongs to.
	hub *Hub

	// conn is the underlying WebSocket connection.
	conn *websocket.Conn

	// send is the channel for outgoing messages.
	send chan []byte

	// userID is the authenticated user ID.
	userID uuid.UUID

	// chatIDs are the chat rooms this client has subscribed to.
	chatIDs map[uuid.UUID]bool

	// mu protects concurrent access to chatIDs.
	mu sync.RWMutex

	// config holds client configuration.
	config ClientConfig

	// logger for structured logging.
	logger *slog.Logger

	// closed indicates if the client connection has been closed.
	closed bool

	// closedMu protects the closed flag.
	closedMu sync.RWMutex
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithClientConfig sets the client configuration.
func WithClientConfig(config ClientConfig) ClientOption {
	return func(c *Client) {
		c.config = config
	}
}

// WithClientLogger sets the logger for the client.
func WithClientLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, opts ...ClientOption) *Client {
	c := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, defaultSendBufferSize),
		userID:  userID,
		chatIDs: make(map[uuid.UUID]bool),
		config:  DefaultClientConfig(),
		logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// UserID returns the user ID associated with this client.
func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// GetChatIDs returns a copy of the chat IDs this client is subscribed to.
func (c *Client) GetChatIDs() []uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(c.chatIDs))
	for id := range c.chatIDs {
		ids = append(ids, id)
	}
	return ids
}

// AddChat adds a chat ID to the client's subscriptions.
func (c *Client) AddChat(chatID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chatIDs[chatID] = true
}

// RemoveChat removes a chat ID from the client's subscriptions.
func (c *Client) RemoveChat(chatID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.chatIDs, chatID)
}

// HasChat checks if the client is subscribed to a chat.
func (c *Client) HasChat(chatID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.chatIDs[chatID]
}

// IsClosed returns whether the client connection has been closed.
func (c *Client) IsClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}

// ReadPump reads messages from the WebSocket connection.
// It should be run as a goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
	}()

	c.conn.SetReadLimit(c.config.MaxMessageSize)

	if err := c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait)); err != nil {
		c.logger.Error("failed to set read deadline", slog.String("error", err.Error()))
		return
	}

	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("websocket read error",
					slog.String("user_id", c.userID.String()),
					slog.String("error", err.Error()),
				)
			}
			return
		}

		c.handleClientMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection.
// It should be run as a goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait)); err != nil {
				c.logger.Error("failed to set write deadline", slog.String("error", err.Error()))
				return
			}

			if !ok {
				// Hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.logger.Warn("websocket write error",
					slog.String("user_id", c.userID.String()),
					slog.String("error", err.Error()),
				)
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait)); err != nil {
				c.logger.Error("failed to set write deadline", slog.String("error", err.Error()))
				return
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage processes a message received from the client.
func (c *Client) handleClientMessage(message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Warn("invalid client message",
			slog.String("user_id", c.userID.String()),
			slog.String("error", err.Error()),
		)
		c.sendError("invalid message format")
		return
	}

	switch msg.Type {
	case "subscribe":
		if msg.ChatID.IsZero() {
			c.sendError("chat_id is required for subscribe")
			return
		}
		c.hub.JoinChat(c, msg.ChatID)
		c.sendAck("subscribed", msg.ChatID)

	case "unsubscribe":
		if msg.ChatID.IsZero() {
			c.sendError("chat_id is required for unsubscribe")
			return
		}
		c.hub.LeaveChat(c, msg.ChatID)
		c.sendAck("unsubscribed", msg.ChatID)

	case "chat.typing":
		if msg.ChatID.IsZero() {
			c.sendError("chat_id is required for chat.typing")
			return
		}
		c.hub.BroadcastTyping(msg.ChatID, c.userID)

	case "ping":
		c.sendPong()

	default:
		c.logger.Debug("unknown message type",
			slog.String("user_id", c.userID.String()),
			slog.String("type", msg.Type),
		)
		c.sendError("unknown message type: " + msg.Type)
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(message string) {
	response := map[string]any{
		"type":    "error",
		"message": message,
	}
	data, _ := json.Marshal(response)
	c.Send(data)
}

// sendAck sends an acknowledgment message to the client.
func (c *Client) sendAck(action string, chatID uuid.UUID) {
	response := map[string]any{
		"type":    "ack",
		"action":  action,
		"chat_id": chatID.String(),
	}
	data, _ := json.Marshal(response)
	c.Send(data)
}

// sendPong sends a pong response to the client.
func (c *Client) sendPong() {
	response := map[string]string{
		"type": "pong",
	}
	data, _ := json.Marshal(response)
	c.Send(data)
}

// Send sends a message to the client.
func (c *Client) Send(message []byte) {
	if c.IsClosed() {
		return
	}

	select {
	case c.send <- message:
	default:
		c.logger.Warn("client send buffer full",
			slog.String("user_id", c.userID.String()),
		)
	}
}

// Close closes the client connection.
func (c *Client) Close() {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	if c.closed {
		return
	}
	c.closed = true

	close(c.send)
	_ = c.conn.Close()

	c.logger.Debug("client connection closed",
		slog.String("user_id", c.userID.String()),
	)
}
