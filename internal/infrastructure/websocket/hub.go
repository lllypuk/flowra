// Package websocket provides WebSocket server implementation for real-time updates.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Hub configuration constants.
const (
	defaultBroadcastBufferSize = 256
)

// Message represents a WebSocket message.
type Message struct {
	Type   string          `json:"type"`
	ChatID *uuid.UUID      `json:"chat_id,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Hub manages all WebSocket connections and chat room subscriptions.
type Hub struct {
	// clients holds all connected clients.
	clients map[*Client]bool

	// chatRooms maps chat IDs to their subscribed clients.
	chatRooms map[uuid.UUID]map[*Client]bool

	// userClients maps user IDs to their connected clients (one user can have multiple connections).
	userClients map[uuid.UUID]map[*Client]bool

	// register channel for new client connections.
	register chan *Client

	// unregister channel for client disconnections.
	unregister chan *Client

	// broadcast channel for messages to be broadcast.
	broadcast chan *broadcastMessage

	// mu protects concurrent access to maps.
	mu sync.RWMutex

	// logger for structured logging.
	logger *slog.Logger

	// done signals when the hub should stop.
	done chan struct{}

	// running indicates if the hub is currently running.
	running bool

	// runningMu protects the running flag.
	runningMu sync.RWMutex
}

// broadcastMessage represents a message to be broadcast to a specific target.
type broadcastMessage struct {
	// chatID is the target chat (nil for user-specific messages).
	chatID *uuid.UUID

	// userID is the target user (nil for chat-wide messages).
	userID *uuid.UUID

	// message is the raw message bytes.
	message []byte
}

// HubOption configures the Hub.
type HubOption func(*Hub)

// WithHubLogger sets the logger for the hub.
func WithHubLogger(logger *slog.Logger) HubOption {
	return func(h *Hub) {
		h.logger = logger
	}
}

// NewHub creates a new Hub with the given options.
func NewHub(opts ...HubOption) *Hub {
	h := &Hub{
		clients:     make(map[*Client]bool),
		chatRooms:   make(map[uuid.UUID]map[*Client]bool),
		userClients: make(map[uuid.UUID]map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *broadcastMessage, defaultBroadcastBufferSize),
		logger:      slog.Default(),
		done:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Run starts the hub's main event loop.
// It should be run as a goroutine.
func (h *Hub) Run(ctx context.Context) {
	h.runningMu.Lock()
	if h.running {
		h.runningMu.Unlock()
		return
	}
	h.running = true
	h.runningMu.Unlock()

	h.logger.InfoContext(ctx, "websocket hub started")

	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return

		case <-h.done:
			h.shutdown()
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.handleBroadcast(msg)
		}
	}
}

// Stop signals the hub to stop.
func (h *Hub) Stop() {
	h.runningMu.Lock()
	defer h.runningMu.Unlock()

	if !h.running {
		return
	}

	close(h.done)
}

// shutdown performs graceful shutdown of all connections.
func (h *Hub) shutdown() {
	h.runningMu.Lock()
	h.running = false
	h.runningMu.Unlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all client connections
	for client := range h.clients {
		client.Close()
	}

	// Clear all maps
	h.clients = make(map[*Client]bool)
	h.chatRooms = make(map[uuid.UUID]map[*Client]bool)
	h.userClients = make(map[uuid.UUID]map[*Client]bool)

	h.logger.Info("websocket hub stopped")
}

// Register registers a new client with the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// registerClient adds a client to the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Add to user clients map
	if !client.userID.IsZero() {
		if h.userClients[client.userID] == nil {
			h.userClients[client.userID] = make(map[*Client]bool)
		}
		h.userClients[client.userID][client] = true
	}

	h.logger.Debug("client registered",
		slog.String("user_id", client.userID.String()),
		slog.Int("total_clients", len(h.clients)),
	)
}

// unregisterClient removes a client from the hub.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	// Remove from all chat rooms
	for _, chatID := range client.GetChatIDs() {
		if room, ok := h.chatRooms[chatID]; ok {
			delete(room, client)
			if len(room) == 0 {
				delete(h.chatRooms, chatID)
			}
		}
	}

	// Remove from user clients map
	if !client.userID.IsZero() {
		if userClients, ok := h.userClients[client.userID]; ok {
			delete(userClients, client)
			if len(userClients) == 0 {
				delete(h.userClients, client.userID)
			}
		}
	}

	delete(h.clients, client)
	client.Close()

	h.logger.Debug("client unregistered",
		slog.String("user_id", client.userID.String()),
		slog.Int("total_clients", len(h.clients)),
	)
}

// JoinChat adds a client to a chat room.
func (h *Hub) JoinChat(client *Client, chatID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	if h.chatRooms[chatID] == nil {
		h.chatRooms[chatID] = make(map[*Client]bool)
	}
	h.chatRooms[chatID][client] = true
	client.AddChat(chatID)

	h.logger.Debug("client joined chat",
		slog.String("user_id", client.userID.String()),
		slog.String("chat_id", chatID.String()),
	)
}

// LeaveChat removes a client from a chat room.
func (h *Hub) LeaveChat(client *Client, chatID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.chatRooms[chatID]; ok {
		delete(room, client)
		if len(room) == 0 {
			delete(h.chatRooms, chatID)
		}
	}
	client.RemoveChat(chatID)

	h.logger.Debug("client left chat",
		slog.String("user_id", client.userID.String()),
		slog.String("chat_id", chatID.String()),
	)
}

// BroadcastToChat sends a message to all clients in a chat room.
func (h *Hub) BroadcastToChat(chatID uuid.UUID, message []byte) {
	h.broadcast <- &broadcastMessage{
		chatID:  &chatID,
		message: message,
	}
}

// SendToUser sends a message to all connections of a specific user.
func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.broadcast <- &broadcastMessage{
		userID:  &userID,
		message: message,
	}
}

// handleBroadcast processes a broadcast message.
func (h *Hub) handleBroadcast(msg *broadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if msg.chatID != nil {
		// Broadcast to chat room
		if room, ok := h.chatRooms[*msg.chatID]; ok {
			for client := range room {
				select {
				case client.send <- msg.message:
				default:
					// Client's send buffer is full, skip this message
					h.logger.Warn("client send buffer full, dropping message",
						slog.String("user_id", client.userID.String()),
						slog.String("chat_id", msg.chatID.String()),
					)
				}
			}
		}
	} else if msg.userID != nil {
		// Send to specific user
		if userClients, ok := h.userClients[*msg.userID]; ok {
			for client := range userClients {
				select {
				case client.send <- msg.message:
				default:
					h.logger.Warn("client send buffer full, dropping message",
						slog.String("user_id", msg.userID.String()),
					)
				}
			}
		}
	}
}

// ClientCount returns the total number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ChatRoomCount returns the number of active chat rooms.
func (h *Hub) ChatRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.chatRooms)
}

// ClientsInChat returns the number of clients in a specific chat room.
func (h *Hub) ClientsInChat(chatID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, ok := h.chatRooms[chatID]; ok {
		return len(room)
	}
	return 0
}

// IsRunning returns whether the hub is currently running.
func (h *Hub) IsRunning() bool {
	h.runningMu.RLock()
	defer h.runningMu.RUnlock()
	return h.running
}

// UserConnectionCount returns the number of connections for a specific user.
func (h *Hub) UserConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.userClients[userID]; ok {
		return len(clients)
	}
	return 0
}
