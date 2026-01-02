// Package eventbus provides event bus implementations for asynchronous event delivery.
package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/redis/go-redis/v9"
)

// Default retry configuration constants.
const (
	defaultMaxRetries     = 3
	defaultInitialBackoff = 100 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	defaultBackoffFactor  = 2.0
	defaultChannelPrefix  = "events:"
)

// EventHandler is a function that handles domain events.
type EventHandler func(ctx context.Context, event event.DomainEvent) error

// eventEnvelope wraps a domain event with metadata for serialization.
type eventEnvelope struct {
	ID            string          `json:"id"`
	EventType     string          `json:"event_type"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Version       int             `json:"version"`
	Metadata      metadataJSON    `json:"metadata"`
	Payload       json.RawMessage `json:"payload"`
}

// metadataJSON is a JSON-serializable version of event.Metadata.
type metadataJSON struct {
	UserID        string    `json:"user_id"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	Timestamp     time.Time `json:"timestamp"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
}

func toMetadataJSON(m event.Metadata) metadataJSON {
	return metadataJSON{
		UserID:        m.UserID,
		CorrelationID: m.CorrelationID,
		CausationID:   m.CausationID,
		Timestamp:     m.Timestamp,
		IPAddress:     m.IPAddress,
		UserAgent:     m.UserAgent,
	}
}

func (m metadataJSON) toMetadata() event.Metadata {
	return event.Metadata{
		UserID:        m.UserID,
		CorrelationID: m.CorrelationID,
		CausationID:   m.CausationID,
		Timestamp:     m.Timestamp,
		IPAddress:     m.IPAddress,
		UserAgent:     m.UserAgent,
	}
}

// deserializedEvent implements DomainEvent for events reconstructed from Redis.
type deserializedEvent struct {
	envelope eventEnvelope
}

func (e *deserializedEvent) EventType() string     { return e.envelope.EventType }
func (e *deserializedEvent) AggregateID() string   { return e.envelope.AggregateID }
func (e *deserializedEvent) AggregateType() string { return e.envelope.AggregateType }
func (e *deserializedEvent) OccurredAt() time.Time { return e.envelope.OccurredAt }
func (e *deserializedEvent) Version() int          { return e.envelope.Version }

func (e *deserializedEvent) Metadata() event.Metadata {
	return e.envelope.Metadata.toMetadata()
}

// Payload returns the raw JSON payload of the event.
func (e *deserializedEvent) Payload() json.RawMessage { return e.envelope.Payload }

// RetryConfig configures retry behavior for event handling.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     defaultMaxRetries,
		InitialBackoff: defaultInitialBackoff,
		MaxBackoff:     defaultMaxBackoff,
		BackoffFactor:  defaultBackoffFactor,
	}
}

// RedisEventBus implements event.Bus using Redis Pub/Sub.
type RedisEventBus struct {
	client        *redis.Client
	pubsub        *redis.PubSub
	pubsubMu      sync.RWMutex
	handlers      map[string][]EventHandler
	handlersMu    sync.RWMutex
	running       bool
	runningMu     sync.RWMutex
	shutdown      chan struct{}
	wg            sync.WaitGroup
	logger        *slog.Logger
	retryConfig   RetryConfig
	channelPrefix string
}

// Option configures a RedisEventBus.
type Option func(*RedisEventBus)

// WithLogger sets the logger for the event bus.
func WithLogger(logger *slog.Logger) Option {
	return func(b *RedisEventBus) {
		b.logger = logger
	}
}

// WithRetryConfig sets the retry configuration for event handling.
func WithRetryConfig(config RetryConfig) Option {
	return func(b *RedisEventBus) {
		b.retryConfig = config
	}
}

// WithChannelPrefix sets a prefix for Redis channel names.
func WithChannelPrefix(prefix string) Option {
	return func(b *RedisEventBus) {
		b.channelPrefix = prefix
	}
}

// NewRedisEventBus creates a new Redis-based event bus.
func NewRedisEventBus(client *redis.Client, opts ...Option) *RedisEventBus {
	b := &RedisEventBus{
		client:        client,
		handlers:      make(map[string][]EventHandler),
		shutdown:      make(chan struct{}),
		logger:        slog.Default(),
		retryConfig:   DefaultRetryConfig(),
		channelPrefix: defaultChannelPrefix,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Publish publishes a domain event to Redis Pub/Sub.
func (b *RedisEventBus) Publish(ctx context.Context, evt event.DomainEvent) error {
	if evt == nil {
		return errors.New("event cannot be nil")
	}

	envelope, err := b.createEnvelope(evt)
	if err != nil {
		return fmt.Errorf("failed to create event envelope: %w", err)
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	channel := b.channelName(evt.EventType())

	if publishErr := b.client.Publish(ctx, channel, data).Err(); publishErr != nil {
		return fmt.Errorf("failed to publish event to Redis: %w", publishErr)
	}

	b.logger.DebugContext(ctx, "event published",
		slog.String("event_id", envelope.ID),
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
		slog.String("channel", channel),
	)

	return nil
}

// Subscribe registers an event handler for a specific event type.
// Handlers are called concurrently when events are received.
func (b *RedisEventBus) Subscribe(eventType string, handler EventHandler) error {
	if eventType == "" {
		return errors.New("event type cannot be empty")
	}
	if handler == nil {
		return errors.New("handler cannot be nil")
	}

	b.handlersMu.Lock()
	defer b.handlersMu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}

// Start begins listening for events on subscribed channels.
// This method blocks until Shutdown is called or the context is cancelled.
func (b *RedisEventBus) Start(ctx context.Context) error {
	b.runningMu.Lock()
	if b.running {
		b.runningMu.Unlock()
		return errors.New("event bus is already running")
	}
	b.running = true
	b.runningMu.Unlock()

	channels := b.subscribedChannels()
	if len(channels) == 0 {
		b.logger.WarnContext(ctx, "starting event bus with no subscriptions")
		// Still start but just wait for shutdown
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.shutdown:
			return nil
		}
	}

	pubsub := b.client.Subscribe(ctx, channels...)

	// Wait for subscription confirmation
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return fmt.Errorf("failed to subscribe to channels: %w", err)
	}

	b.pubsubMu.Lock()
	b.pubsub = pubsub
	b.pubsubMu.Unlock()

	b.logger.InfoContext(ctx, "event bus started",
		slog.Int("channel_count", len(channels)),
		slog.Any("channels", channels),
	)

	msgCh := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			b.logger.InfoContext(ctx, "event bus stopping due to context cancellation")
			return ctx.Err()

		case <-b.shutdown:
			b.logger.InfoContext(ctx, "event bus stopping due to shutdown signal")
			return nil

		case msg, ok := <-msgCh:
			if !ok {
				b.logger.WarnContext(ctx, "message channel closed")
				return nil
			}
			b.handleMessage(ctx, msg)
		}
	}
}

// Shutdown gracefully stops the event bus.
// It waits for all pending event handlers to complete.
func (b *RedisEventBus) Shutdown() error {
	b.runningMu.Lock()
	if !b.running {
		b.runningMu.Unlock()
		return nil
	}
	b.running = false
	b.runningMu.Unlock()

	close(b.shutdown)

	// Wait for all handlers to complete
	b.wg.Wait()

	b.pubsubMu.Lock()
	pubsub := b.pubsub
	b.pubsub = nil
	b.pubsubMu.Unlock()

	if pubsub != nil {
		if err := pubsub.Close(); err != nil {
			return fmt.Errorf("failed to close pubsub: %w", err)
		}
	}

	return nil
}

// IsRunning returns true if the event bus is currently running.
func (b *RedisEventBus) IsRunning() bool {
	b.runningMu.RLock()
	defer b.runningMu.RUnlock()
	return b.running
}

// HandlerCount returns the number of handlers registered for an event type.
func (b *RedisEventBus) HandlerCount(eventType string) int {
	b.handlersMu.RLock()
	defer b.handlersMu.RUnlock()
	return len(b.handlers[eventType])
}

// createEnvelope wraps a domain event in an envelope for serialization.
func (b *RedisEventBus) createEnvelope(evt event.DomainEvent) (eventEnvelope, error) {
	// Try to serialize the event payload
	payload, err := json.Marshal(evt)
	if err != nil {
		return eventEnvelope{}, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	return eventEnvelope{
		ID:            uuid.New().String(),
		EventType:     evt.EventType(),
		AggregateID:   evt.AggregateID(),
		AggregateType: evt.AggregateType(),
		OccurredAt:    evt.OccurredAt(),
		Version:       evt.Version(),
		Metadata:      toMetadataJSON(evt.Metadata()),
		Payload:       payload,
	}, nil
}

// channelName returns the Redis channel name for an event type.
func (b *RedisEventBus) channelName(eventType string) string {
	return b.channelPrefix + eventType
}

// subscribedChannels returns all Redis channel names for subscribed event types.
func (b *RedisEventBus) subscribedChannels() []string {
	b.handlersMu.RLock()
	defer b.handlersMu.RUnlock()

	channels := make([]string, 0, len(b.handlers))
	for eventType := range b.handlers {
		channels = append(channels, b.channelName(eventType))
	}
	return channels
}

// handleMessage processes a message received from Redis.
func (b *RedisEventBus) handleMessage(ctx context.Context, msg *redis.Message) {
	var envelope eventEnvelope
	if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
		b.logger.ErrorContext(ctx, "failed to unmarshal event",
			slog.String("channel", msg.Channel),
			slog.String("error", err.Error()),
		)
		return
	}

	evt := &deserializedEvent{envelope: envelope}

	b.handlersMu.RLock()
	handlers := b.handlers[envelope.EventType]
	b.handlersMu.RUnlock()

	for i, handler := range handlers {
		b.wg.Add(1)
		go b.executeHandler(ctx, handler, evt, i)
	}
}

// executeHandler runs a single event handler with retry logic.
func (b *RedisEventBus) executeHandler(
	ctx context.Context,
	handler EventHandler,
	evt event.DomainEvent,
	handlerIndex int,
) {
	defer b.wg.Done()

	var lastErr error
	backoff := b.retryConfig.InitialBackoff

	for attempt := 0; attempt <= b.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			b.logger.DebugContext(ctx, "retrying event handler",
				slog.String("event_type", evt.EventType()),
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
			)

			select {
			case <-ctx.Done():
				b.logger.WarnContext(ctx, "handler retry cancelled",
					slog.String("event_type", evt.EventType()),
					slog.String("error", ctx.Err().Error()),
				)
				return
			case <-time.After(backoff):
			}

			// Calculate next backoff with exponential growth
			backoff = time.Duration(float64(backoff) * b.retryConfig.BackoffFactor)
			if backoff > b.retryConfig.MaxBackoff {
				backoff = b.retryConfig.MaxBackoff
			}
		}

		if err := handler(ctx, evt); err != nil {
			lastErr = err
			b.logger.WarnContext(ctx, "event handler failed",
				slog.String("event_type", evt.EventType()),
				slog.String("aggregate_id", evt.AggregateID()),
				slog.Int("handler_index", handlerIndex),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()),
			)
			continue
		}

		// Success
		b.logger.DebugContext(ctx, "event handler completed",
			slog.String("event_type", evt.EventType()),
			slog.String("aggregate_id", evt.AggregateID()),
			slog.Int("handler_index", handlerIndex),
		)
		return
	}

	// All retries exhausted
	b.logger.ErrorContext(ctx, "event handler failed after all retries",
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
		slog.Int("handler_index", handlerIndex),
		slog.Int("max_retries", b.retryConfig.MaxRetries),
		slog.String("error", lastErr.Error()),
	)
}

// Ensure RedisEventBus implements event.Bus
var _ event.Bus = (*RedisEventBus)(nil)
