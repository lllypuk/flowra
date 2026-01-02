// Package eventbus provides event bus implementations for asynchronous event delivery.
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
	domainNotif "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/redis/go-redis/v9"
)

// Default dead letter queue configuration.
const (
	deadLetterQueueKey     = "events:dead_letter"
	defaultMaxDeadLetters  = 1000
	mentionPatternTemplate = `@([a-zA-Z0-9_-]+)`
	minMentionMatchGroups  = 2
	maxTaskTitleLength     = 50
	maxPayloadLogLength    = 500
)

var mentionRegex = regexp.MustCompile(mentionPatternTemplate)

// PayloadEvent is an interface for events that carry raw JSON payload.
// This is implemented by deserializedEvent for events received from Redis.
type PayloadEvent interface {
	event.DomainEvent
	Payload() json.RawMessage
}

// NotificationHandler handles domain events and creates notifications for users.
// It processes various events like chat creation, message sending, and task assignments.
type NotificationHandler struct {
	createNotifUC *notification.CreateNotificationUseCase
	logger        *slog.Logger
	// userResolver is used to resolve usernames from mentions to user IDs.
	// If nil, mention resolution will be skipped.
	userResolver UserResolver
}

// UserResolver resolves usernames to user IDs.
// This interface is declared on the consumer side (this handler).
type UserResolver interface {
	// ResolveUsername returns the user ID for a given username.
	// Returns empty UUID if the user is not found.
	ResolveUsername(ctx context.Context, username string) (uuid.UUID, error)
}

// NotificationHandlerOption configures NotificationHandler.
type NotificationHandlerOption func(*NotificationHandler)

// WithNotificationLogger sets the logger for NotificationHandler.
func WithNotificationLogger(logger *slog.Logger) NotificationHandlerOption {
	return func(h *NotificationHandler) {
		h.logger = logger
	}
}

// WithUserResolver sets the user resolver for mention processing.
func WithUserResolver(resolver UserResolver) NotificationHandlerOption {
	return func(h *NotificationHandler) {
		h.userResolver = resolver
	}
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(
	createNotifUC *notification.CreateNotificationUseCase,
	opts ...NotificationHandlerOption,
) *NotificationHandler {
	h := &NotificationHandler{
		createNotifUC: createNotifUC,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Handle processes a domain event and creates appropriate notifications.
func (h *NotificationHandler) Handle(ctx context.Context, evt event.DomainEvent) error {
	switch evt.EventType() {
	case chat.EventTypeChatCreated:
		return h.handleChatCreated(ctx, evt)
	case chat.EventTypeParticipantAdded:
		return h.handleParticipantAdded(ctx, evt)
	case message.EventTypeMessageCreated:
		return h.handleMessageCreated(ctx, evt)
	case task.EventTypeTaskCreated:
		return h.handleTaskCreated(ctx, evt)
	case task.EventTypeStatusChanged:
		return h.handleTaskStatusChanged(ctx, evt)
	case task.EventTypeAssigneeChanged:
		return h.handleTaskAssigneeChanged(ctx, evt)
	default:
		// Ignore unknown event types
		return nil
	}
}

// handleChatCreated processes chat.created events and notifies participants.
func (h *NotificationHandler) handleChatCreated(ctx context.Context, evt event.DomainEvent) error {
	// For chat created events, we need the creator ID from metadata
	// Participants are added separately via ParticipantAdded events
	h.logger.DebugContext(ctx, "processing chat.created event",
		slog.String("chat_id", evt.AggregateID()),
	)
	return nil
}

// handleParticipantAdded notifies users when they are added to a chat.
func (h *NotificationHandler) handleParticipantAdded(ctx context.Context, evt event.DomainEvent) error {
	payload, extractErr := h.extractPayload(evt)
	if extractErr != nil {
		h.logger.WarnContext(ctx, "failed to extract payload for participant_added",
			slog.String("error", extractErr.Error()),
		)
		return nil // Don't retry for payload extraction failures
	}

	var data struct {
		UserID string `json:"UserID"`
	}
	if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal participant_added payload",
			slog.String("error", unmarshalErr.Error()),
		)
		return nil
	}

	if data.UserID == "" {
		return nil
	}

	userID, err := uuid.ParseUUID(data.UserID)
	if err != nil {
		h.logger.WarnContext(ctx, "invalid user ID in participant_added",
			slog.String("user_id", data.UserID),
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Don't notify if the user added themselves (they know they joined)
	if evt.Metadata().UserID == data.UserID {
		return nil
	}

	cmd := notification.CreateNotificationCommand{
		UserID:     userID,
		Type:       domainNotif.TypeChatMessage,
		Title:      "Added to chat",
		Message:    "You have been added to a new chat",
		ResourceID: evt.AggregateID(),
	}

	if _, execErr := h.createNotifUC.Execute(ctx, cmd); execErr != nil {
		return fmt.Errorf("failed to create notification for participant added: %w", execErr)
	}

	return nil
}

// handleMessageCreated processes message.created events and notifies mentioned users.
func (h *NotificationHandler) handleMessageCreated(ctx context.Context, evt event.DomainEvent) error {
	payload, extractErr := h.extractPayload(evt)
	if extractErr != nil {
		h.logger.WarnContext(ctx, "failed to extract payload for message.created",
			slog.String("error", extractErr.Error()),
		)
		return nil
	}

	var data struct {
		ChatID   string `json:"ChatID"`
		AuthorID string `json:"AuthorID"`
		Content  string `json:"Content"`
	}
	if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal message.created payload",
			slog.String("error", unmarshalErr.Error()),
		)
		return nil
	}

	// Extract mentions from content
	mentions := h.extractMentions(data.Content)
	if len(mentions) == 0 {
		return nil
	}

	// Resolve usernames to user IDs and create notifications
	for _, username := range mentions {
		if notifyErr := h.notifyMentionedUser(ctx, username, data.AuthorID, evt.AggregateID()); notifyErr != nil {
			h.logger.WarnContext(ctx, "failed to notify mentioned user",
				slog.String("username", username),
				slog.String("error", notifyErr.Error()),
			)
			// Continue with other mentions even if one fails
		}
	}

	return nil
}

// extractMentions extracts @mentions from message content.
func (h *NotificationHandler) extractMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var mentions []string
	for _, match := range matches {
		if len(match) >= minMentionMatchGroups {
			username := match[1]
			if _, ok := seen[username]; !ok {
				seen[username] = struct{}{}
				mentions = append(mentions, username)
			}
		}
	}
	return mentions
}

// notifyMentionedUser creates a notification for a mentioned user.
func (h *NotificationHandler) notifyMentionedUser(
	ctx context.Context,
	username, authorID, messageID string,
) error {
	if h.userResolver == nil {
		h.logger.DebugContext(ctx, "user resolver not configured, skipping mention notification",
			slog.String("username", username),
		)
		return nil
	}

	userID, resolveErr := h.userResolver.ResolveUsername(ctx, username)
	if resolveErr != nil {
		return fmt.Errorf("failed to resolve username %s: %w", username, resolveErr)
	}

	if userID.IsZero() {
		h.logger.DebugContext(ctx, "mentioned user not found",
			slog.String("username", username),
		)
		return nil
	}

	// Don't notify if user mentioned themselves
	if userID.String() == authorID {
		return nil
	}

	cmd := notification.CreateNotificationCommand{
		UserID:     userID,
		Type:       domainNotif.TypeChatMention,
		Title:      "You were mentioned",
		Message:    fmt.Sprintf("@%s mentioned you in a chat", username),
		ResourceID: messageID,
	}

	if _, execErr := h.createNotifUC.Execute(ctx, cmd); execErr != nil {
		return fmt.Errorf("failed to create mention notification: %w", execErr)
	}

	return nil
}

// handleTaskCreated processes task.created events.
func (h *NotificationHandler) handleTaskCreated(ctx context.Context, evt event.DomainEvent) error {
	payload, extractErr := h.extractPayload(evt)
	if extractErr != nil {
		h.logger.WarnContext(ctx, "failed to extract payload for task.created",
			slog.String("error", extractErr.Error()),
		)
		return nil
	}

	var data struct {
		Title      string  `json:"Title"`
		AssigneeID *string `json:"AssigneeID"`
		CreatedBy  string  `json:"CreatedBy"`
	}
	if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal task.created payload",
			slog.String("error", unmarshalErr.Error()),
		)
		return nil
	}

	// Notify assignee if assigned and different from creator
	if data.AssigneeID != nil && *data.AssigneeID != "" && *data.AssigneeID != data.CreatedBy {
		assigneeID, parseErr := uuid.ParseUUID(*data.AssigneeID)
		if parseErr != nil {
			h.logger.WarnContext(ctx, "invalid assignee ID in task.created",
				slog.String("assignee_id", *data.AssigneeID),
				slog.String("error", parseErr.Error()),
			)
			return nil
		}

		cmd := notification.CreateNotificationCommand{
			UserID: assigneeID,
			Type:   domainNotif.TypeTaskAssigned,
			Title:  "Task assigned to you",
			Message: fmt.Sprintf(
				"You have been assigned to task: %s",
				truncateString(data.Title, maxTaskTitleLength),
			),
			ResourceID: evt.AggregateID(),
		}

		if _, execErr := h.createNotifUC.Execute(ctx, cmd); execErr != nil {
			return fmt.Errorf("failed to create task assignment notification: %w", execErr)
		}
	}

	return nil
}

// handleTaskStatusChanged processes task.status_changed events.
func (h *NotificationHandler) handleTaskStatusChanged(ctx context.Context, evt event.DomainEvent) error {
	payload, extractErr := h.extractPayload(evt)
	if extractErr != nil {
		h.logger.WarnContext(ctx, "failed to extract payload for task.status_changed",
			slog.String("error", extractErr.Error()),
		)
		return nil
	}

	var data struct {
		OldStatus string `json:"OldStatus"`
		NewStatus string `json:"NewStatus"`
		ChangedBy string `json:"ChangedBy"`
	}
	if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal task.status_changed payload",
			slog.String("error", unmarshalErr.Error()),
		)
		return nil
	}

	h.logger.DebugContext(ctx, "task status changed",
		slog.String("task_id", evt.AggregateID()),
		slog.String("old_status", data.OldStatus),
		slog.String("new_status", data.NewStatus),
	)

	// TODO: Notify watchers and reporter when we have that information
	// For now, we just log the event

	return nil
}

// handleTaskAssigneeChanged processes task.assignee_changed events.
func (h *NotificationHandler) handleTaskAssigneeChanged(ctx context.Context, evt event.DomainEvent) error {
	payload, extractErr := h.extractPayload(evt)
	if extractErr != nil {
		h.logger.WarnContext(ctx, "failed to extract payload for task.assignee_changed",
			slog.String("error", extractErr.Error()),
		)
		return nil
	}

	var data struct {
		OldAssignee *string `json:"OldAssignee"`
		NewAssignee *string `json:"NewAssignee"`
		ChangedBy   string  `json:"ChangedBy"`
	}
	if unmarshalErr := json.Unmarshal(payload, &data); unmarshalErr != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal task.assignee_changed payload",
			slog.String("error", unmarshalErr.Error()),
		)
		return nil
	}

	// Notify new assignee if different from who made the change
	if data.NewAssignee != nil && *data.NewAssignee != "" && *data.NewAssignee != data.ChangedBy {
		assigneeID, parseErr := uuid.ParseUUID(*data.NewAssignee)
		if parseErr != nil {
			h.logger.WarnContext(ctx, "invalid new assignee ID",
				slog.String("assignee_id", *data.NewAssignee),
				slog.String("error", parseErr.Error()),
			)
			return nil
		}

		cmd := notification.CreateNotificationCommand{
			UserID:     assigneeID,
			Type:       domainNotif.TypeTaskAssigned,
			Title:      "Task assigned to you",
			Message:    "A task has been assigned to you",
			ResourceID: evt.AggregateID(),
		}

		if _, execErr := h.createNotifUC.Execute(ctx, cmd); execErr != nil {
			return fmt.Errorf("failed to create assignee notification: %w", execErr)
		}
	}

	return nil
}

// extractPayload extracts raw JSON payload from an event.
func (h *NotificationHandler) extractPayload(evt event.DomainEvent) (json.RawMessage, error) {
	if pe, ok := evt.(PayloadEvent); ok {
		return pe.Payload(), nil
	}

	// For non-PayloadEvent, try to marshal the event itself
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return data, nil
}

// AsEventHandler converts NotificationHandler to EventHandler function type.
func (h *NotificationHandler) AsEventHandler() EventHandler {
	return h.Handle
}

// LoggingHandler logs all domain events for audit trail purposes.
type LoggingHandler struct {
	logger *slog.Logger
}

// NewLoggingHandler creates a new LoggingHandler.
func NewLoggingHandler(logger *slog.Logger) *LoggingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingHandler{
		logger: logger,
	}
}

// Handle logs the domain event.
func (h *LoggingHandler) Handle(ctx context.Context, evt event.DomainEvent) error {
	attrs := []any{
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
		slog.String("aggregate_type", evt.AggregateType()),
		slog.Time("occurred_at", evt.OccurredAt()),
		slog.Int("version", evt.Version()),
	}

	// Add metadata if available
	metadata := evt.Metadata()
	if metadata.UserID != "" {
		attrs = append(attrs, slog.String("user_id", metadata.UserID))
	}
	if metadata.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", metadata.CorrelationID))
	}

	// Add payload if available
	if pe, ok := evt.(PayloadEvent); ok {
		// Truncate large payloads for logging
		payload := string(pe.Payload())
		if len(payload) > maxPayloadLogLength {
			payload = payload[:maxPayloadLogLength] + "..."
		}
		attrs = append(attrs, slog.String("payload", payload))
	}

	h.logger.InfoContext(ctx, "domain event", attrs...)

	return nil
}

// AsEventHandler converts LoggingHandler to EventHandler function type.
func (h *LoggingHandler) AsEventHandler() EventHandler {
	return h.Handle
}

// DeadLetterHandler stores failed events in Redis for later analysis.
type DeadLetterHandler struct {
	client        *redis.Client
	logger        *slog.Logger
	queueKey      string
	maxDeadLetter int64
}

// DeadLetterEntry represents a failed event stored in the dead letter queue.
type DeadLetterEntry struct {
	EventType     string          `json:"event_type"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Error         string          `json:"error"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	Timestamp     int64           `json:"timestamp"`
}

// DeadLetterHandlerOption configures DeadLetterHandler.
type DeadLetterHandlerOption func(*DeadLetterHandler)

// WithDeadLetterQueueKey sets a custom key for the dead letter queue.
func WithDeadLetterQueueKey(key string) DeadLetterHandlerOption {
	return func(h *DeadLetterHandler) {
		h.queueKey = key
	}
}

// WithDeadLetterLogger sets the logger for DeadLetterHandler.
func WithDeadLetterLogger(logger *slog.Logger) DeadLetterHandlerOption {
	return func(h *DeadLetterHandler) {
		h.logger = logger
	}
}

// WithMaxDeadLetters sets the maximum number of entries to keep in the queue.
func WithMaxDeadLetters(maxEntries int64) DeadLetterHandlerOption {
	return func(h *DeadLetterHandler) {
		h.maxDeadLetter = maxEntries
	}
}

// NewDeadLetterHandler creates a new DeadLetterHandler.
func NewDeadLetterHandler(client *redis.Client, opts ...DeadLetterHandlerOption) *DeadLetterHandler {
	h := &DeadLetterHandler{
		client:        client,
		logger:        slog.Default(),
		queueKey:      deadLetterQueueKey,
		maxDeadLetter: defaultMaxDeadLetters,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Handle stores a failed event in the dead letter queue.
func (h *DeadLetterHandler) Handle(ctx context.Context, evt event.DomainEvent, err error) {
	entry := DeadLetterEntry{
		EventType:     evt.EventType(),
		AggregateID:   evt.AggregateID(),
		AggregateType: evt.AggregateType(),
		Error:         err.Error(),
		Timestamp:     evt.OccurredAt().Unix(),
	}

	// Extract payload if available
	if pe, ok := evt.(PayloadEvent); ok {
		entry.Payload = pe.Payload()
	}

	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		h.logger.ErrorContext(ctx, "failed to marshal dead letter entry",
			slog.String("event_type", evt.EventType()),
			slog.String("error", marshalErr.Error()),
		)
		return
	}

	// Push to dead letter queue
	if pushErr := h.client.LPush(ctx, h.queueKey, string(data)).Err(); pushErr != nil {
		h.logger.ErrorContext(ctx, "failed to push to dead letter queue",
			slog.String("event_type", evt.EventType()),
			slog.String("error", pushErr.Error()),
		)
		return
	}

	// Trim queue to max size
	if trimErr := h.client.LTrim(ctx, h.queueKey, 0, h.maxDeadLetter-1).Err(); trimErr != nil {
		h.logger.WarnContext(ctx, "failed to trim dead letter queue",
			slog.String("error", trimErr.Error()),
		)
	}

	h.logger.ErrorContext(ctx, "event moved to dead letter queue",
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
		slog.String("original_error", err.Error()),
	)
}

// GetDeadLetters retrieves entries from the dead letter queue.
func (h *DeadLetterHandler) GetDeadLetters(ctx context.Context, count int64) ([]DeadLetterEntry, error) {
	requestedCount := count
	if requestedCount <= 0 {
		requestedCount = 10
	}

	data, rangeErr := h.client.LRange(ctx, h.queueKey, 0, requestedCount-1).Result()
	if rangeErr != nil {
		return nil, fmt.Errorf("failed to get dead letters: %w", rangeErr)
	}

	entries := make([]DeadLetterEntry, 0, len(data))
	for _, d := range data {
		var entry DeadLetterEntry
		if unmarshalErr := json.Unmarshal([]byte(d), &entry); unmarshalErr != nil {
			h.logger.WarnContext(ctx, "failed to unmarshal dead letter entry",
				slog.String("error", unmarshalErr.Error()),
			)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ClearDeadLetters removes all entries from the dead letter queue.
func (h *DeadLetterHandler) ClearDeadLetters(ctx context.Context) error {
	return h.client.Del(ctx, h.queueKey).Err()
}

// QueueLength returns the number of entries in the dead letter queue.
func (h *DeadLetterHandler) QueueLength(ctx context.Context) (int64, error) {
	return h.client.LLen(ctx, h.queueKey).Result()
}

// HandlerRegistry manages event handler registration.
type HandlerRegistry struct {
	bus        *RedisEventBus
	logger     *slog.Logger
	dlqHandler *DeadLetterHandler
}

// NewHandlerRegistry creates a new HandlerRegistry.
func NewHandlerRegistry(bus *RedisEventBus, logger *slog.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		bus:    bus,
		logger: logger,
	}
}

// SetDeadLetterHandler sets the dead letter handler for failed events.
func (r *HandlerRegistry) SetDeadLetterHandler(dlq *DeadLetterHandler) {
	r.dlqHandler = dlq
}

// Register registers an event handler for specific event types.
func (r *HandlerRegistry) Register(eventTypes []string, handler EventHandler) error {
	for _, eventType := range eventTypes {
		if err := r.bus.Subscribe(eventType, handler); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", eventType, err)
		}
		r.logger.Debug("registered handler for event",
			slog.String("event_type", eventType),
		)
	}
	return nil
}

// RegisterNotificationHandler registers the notification handler for relevant events.
func (r *HandlerRegistry) RegisterNotificationHandler(handler *NotificationHandler) error {
	eventTypes := []string{
		chat.EventTypeChatCreated,
		chat.EventTypeParticipantAdded,
		message.EventTypeMessageCreated,
		task.EventTypeTaskCreated,
		task.EventTypeStatusChanged,
		task.EventTypeAssigneeChanged,
	}

	return r.Register(eventTypes, handler.AsEventHandler())
}

// RegisterLoggingHandler registers the logging handler for specified event types.
// Note: Redis Pub/Sub doesn't support wildcards natively, so you need to specify
// all event types explicitly.
func (r *HandlerRegistry) RegisterLoggingHandler(handler *LoggingHandler, eventTypes []string) error {
	return r.Register(eventTypes, handler.AsEventHandler())
}

// RegisterAllHandlers is a convenience function that registers all standard handlers.
func RegisterAllHandlers(
	bus *RedisEventBus,
	notifHandler *NotificationHandler,
	logHandler *LoggingHandler,
	logger *slog.Logger,
) error {
	registry := NewHandlerRegistry(bus, logger)

	// Register notification handler
	if notifHandler != nil {
		if err := registry.RegisterNotificationHandler(notifHandler); err != nil {
			return fmt.Errorf("failed to register notification handler: %w", err)
		}
	}

	// Register logging handler for all notification-relevant events
	if logHandler != nil {
		eventTypes := []string{
			chat.EventTypeChatCreated,
			chat.EventTypeParticipantAdded,
			chat.EventTypeParticipantRemoved,
			chat.EventTypeChatTypeChanged,
			chat.EventTypeStatusChanged,
			chat.EventTypeUserAssigned,
			chat.EventTypeChatRenamed,
			message.EventTypeMessageCreated,
			message.EventTypeMessageEdited,
			message.EventTypeMessageDeleted,
			task.EventTypeTaskCreated,
			task.EventTypeTaskUpdated,
			task.EventTypeTaskDeleted,
			task.EventTypeStatusChanged,
			task.EventTypeAssigneeChanged,
			task.EventTypePriorityChanged,
			task.EventTypeDueDateChanged,
		}
		if err := registry.RegisterLoggingHandler(logHandler, eventTypes); err != nil {
			return fmt.Errorf("failed to register logging handler: %w", err)
		}
	}

	return nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen]) + "..."
}
