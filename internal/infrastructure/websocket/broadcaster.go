// Package websocket provides WebSocket server implementation for real-time updates.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// EventBus defines the interface for subscribing to domain events.
// Declared on the consumer side per project guidelines.
type EventBus interface {
	// Subscribe registers an event handler for a specific event type.
	Subscribe(eventType string, handler func(ctx context.Context, event event.DomainEvent) error) error
}

// OutboundMessage represents a message to be sent over WebSocket.
type OutboundMessage struct {
	Type   string  `json:"type"`
	ChatID *string `json:"chat_id,omitempty"`
	Data   any     `json:"data,omitempty"`
}

// Broadcaster listens to the event bus and broadcasts events via WebSocket.
type Broadcaster struct {
	hub      *Hub
	eventBus EventBus
	logger   *slog.Logger

	// eventTypes lists which event types to subscribe to.
	eventTypes []string

	// running indicates if the broadcaster is active.
	running bool

	// runningMu protects the running flag.
	runningMu sync.RWMutex
}

// BroadcasterOption configures a Broadcaster.
type BroadcasterOption func(*Broadcaster)

// WithBroadcasterLogger sets the logger for the broadcaster.
func WithBroadcasterLogger(logger *slog.Logger) BroadcasterOption {
	return func(b *Broadcaster) {
		b.logger = logger
	}
}

// WithEventTypes sets which event types to subscribe to.
func WithEventTypes(eventTypes []string) BroadcasterOption {
	return func(b *Broadcaster) {
		b.eventTypes = eventTypes
	}
}

// DefaultEventTypes returns the default event types to broadcast.
func DefaultEventTypes() []string {
	return []string{
		"message.sent",
		"message.updated",
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
}

// NewBroadcaster creates a new Broadcaster.
func NewBroadcaster(hub *Hub, eventBus EventBus, opts ...BroadcasterOption) *Broadcaster {
	b := &Broadcaster{
		hub:        hub,
		eventBus:   eventBus,
		logger:     slog.Default(),
		eventTypes: DefaultEventTypes(),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Start subscribes to event bus and starts broadcasting events.
// This method registers handlers but doesn't block.
func (b *Broadcaster) Start(ctx context.Context) error {
	b.runningMu.Lock()
	if b.running {
		b.runningMu.Unlock()
		return nil
	}
	b.running = true
	b.runningMu.Unlock()

	for _, eventType := range b.eventTypes {
		et := eventType // capture for closure
		if err := b.eventBus.Subscribe(et, func(handlerCtx context.Context, evt event.DomainEvent) error {
			return b.handleEvent(handlerCtx, evt)
		}); err != nil {
			b.logger.ErrorContext(ctx, "failed to subscribe to event",
				slog.String("event_type", et),
				slog.String("error", err.Error()),
			)
			return err
		}
		b.logger.DebugContext(ctx, "subscribed to event", slog.String("event_type", et))
	}

	b.logger.InfoContext(ctx, "websocket broadcaster started",
		slog.Int("event_types", len(b.eventTypes)),
	)

	return nil
}

// IsRunning returns whether the broadcaster is running.
func (b *Broadcaster) IsRunning() bool {
	b.runningMu.RLock()
	defer b.runningMu.RUnlock()
	return b.running
}

// handleEvent processes a domain event and broadcasts it via WebSocket.
func (b *Broadcaster) handleEvent(ctx context.Context, evt event.DomainEvent) error {
	b.logger.DebugContext(ctx, "handling event",
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
		slog.String("aggregate_type", evt.AggregateType()),
	)

	wsMessage := b.transformEvent(evt)
	if wsMessage == nil {
		b.logger.DebugContext(ctx, "event transformed to nil message", slog.String("event_type", evt.EventType()))
		return nil
	}

	messageBytes, err := json.Marshal(wsMessage)
	if err != nil {
		b.logger.ErrorContext(ctx, "failed to marshal websocket message",
			slog.String("event_type", evt.EventType()),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Route message based on event type
	switch {
	case b.isUserSpecificEvent(evt.EventType()):
		// Send to specific user
		userID := b.extractUserID(evt)
		if !userID.IsZero() {
			b.hub.SendToUser(userID, messageBytes)
			b.logger.DebugContext(ctx, "sent message to user",
				slog.String("event_type", evt.EventType()),
				slog.String("user_id", userID.String()),
			)
		}

	case b.isChatEvent(evt.EventType()):
		// Broadcast to chat room
		chatID := b.extractChatID(evt)
		b.logger.DebugContext(ctx, "extracted chat_id for broadcast",
			slog.String("event_type", evt.EventType()),
			slog.String("chat_id", chatID.String()),
			slog.Bool("is_zero", chatID.IsZero()),
		)
		if !chatID.IsZero() {
			b.hub.BroadcastToChat(chatID, messageBytes)
			b.logger.DebugContext(ctx, "broadcast message to chat",
				slog.String("event_type", evt.EventType()),
				slog.String("chat_id", chatID.String()),
			)
		}

	default:
		b.logger.DebugContext(ctx, "event not routable",
			slog.String("event_type", evt.EventType()),
		)
	}

	return nil
}

// transformEvent converts a domain event to a WebSocket message.
func (b *Broadcaster) transformEvent(evt event.DomainEvent) *OutboundMessage {
	wsType := b.mapEventTypeToWSType(evt.EventType())
	if wsType == "" {
		return nil
	}

	// Extract payload from event if it has one
	var data any
	if payloadEvent, ok := evt.(PayloadProvider); ok {
		data = payloadEvent.Payload()
	} else {
		// Create basic payload from event metadata
		data = map[string]any{
			"aggregate_id":   evt.AggregateID(),
			"aggregate_type": evt.AggregateType(),
			"occurred_at":    evt.OccurredAt(),
			"version":        evt.Version(),
		}
	}

	msg := &OutboundMessage{
		Type: wsType,
		Data: data,
	}

	// Add chat_id if this is a chat-related event
	if b.isChatEvent(evt.EventType()) {
		chatID := evt.AggregateID()
		msg.ChatID = &chatID
	}

	return msg
}

// PayloadProvider is an interface for events that can provide their payload.
type PayloadProvider interface {
	Payload() json.RawMessage
}

// mapEventTypeToWSType maps domain event types to WebSocket message types.
func (b *Broadcaster) mapEventTypeToWSType(eventType string) string {
	mapping := map[string]string{
		"message.sent":         "message.new",
		"message.updated":      "message.updated",
		"message.deleted":      "message.deleted",
		"chat.created":         "chat.created",
		"chat.updated":         "chat.updated",
		"chat.deleted":         "chat.deleted",
		"chat.member_added":    "chat.member_added",
		"chat.member_removed":  "chat.member_removed",
		"task.created":         "task.created",
		"task.updated":         "task.updated",
		"task.status_changed":  "task.updated",
		"task.assigned":        "task.updated",
		"notification.created": "notification.new",
	}

	if wsType, ok := mapping[eventType]; ok {
		return wsType
	}
	return ""
}

// isUserSpecificEvent returns true if the event should be sent to a specific user.
func (b *Broadcaster) isUserSpecificEvent(eventType string) bool {
	userEvents := map[string]bool{
		"notification.created": true,
	}
	return userEvents[eventType]
}

// isChatEvent returns true if the event should be broadcast to a chat room.
func (b *Broadcaster) isChatEvent(eventType string) bool {
	chatEvents := map[string]bool{
		"message.sent":        true,
		"message.updated":     true,
		"message.deleted":     true,
		"chat.created":        true,
		"chat.updated":        true,
		"chat.deleted":        true,
		"chat.member_added":   true,
		"chat.member_removed": true,
		"task.created":        true,
		"task.updated":        true,
		"task.status_changed": true,
		"task.assigned":       true,
	}
	return chatEvents[eventType]
}

// extractChatID extracts the chat ID from an event.
func (b *Broadcaster) extractChatID(evt event.DomainEvent) uuid.UUID {
	// For chat and message events, the aggregate ID is typically the chat ID
	if evt.AggregateType() == "chat" || evt.AggregateType() == "message" {
		id, err := uuid.ParseUUID(evt.AggregateID())
		if err == nil {
			return id
		}
	}

	// Try to get from payload if available
	if payloadEvent, ok := evt.(PayloadProvider); ok {
		payload := payloadEvent.Payload()
		var data struct {
			ChatID string `json:"chat_id"`
		}
		if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr == nil && data.ChatID != "" {
			if parsedID, parseErr := uuid.ParseUUID(data.ChatID); parseErr == nil {
				return parsedID
			}
		}
	}

	return uuid.UUID("")
}

// extractUserID extracts the target user ID from an event.
func (b *Broadcaster) extractUserID(evt event.DomainEvent) uuid.UUID {
	// For notification events, try to get user ID from metadata or payload
	metadata := evt.Metadata()
	if metadata.UserID != "" {
		id, err := uuid.ParseUUID(metadata.UserID)
		if err == nil {
			return id
		}
	}

	// Try to get from payload
	if payloadEvent, ok := evt.(PayloadProvider); ok {
		payload := payloadEvent.Payload()
		var data struct {
			UserID string `json:"user_id"`
		}
		if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr == nil && data.UserID != "" {
			if parsedID, parseErr := uuid.ParseUUID(data.UserID); parseErr == nil {
				return parsedID
			}
		}
	}

	return uuid.UUID("")
}
