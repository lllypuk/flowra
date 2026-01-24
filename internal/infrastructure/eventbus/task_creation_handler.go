package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TaskCreationHandler handles chat type change events and creates corresponding tasks.
// When a chat is converted to task/bug/epic type, this handler creates a task entity.
type TaskCreationHandler struct {
	createTaskUC CreateTaskUseCase
	logger       *slog.Logger
}

// CreateTaskUseCase defines the interface for creating tasks.
// This interface is declared on the consumer side (this handler).
type CreateTaskUseCase interface {
	Execute(ctx context.Context, cmd taskapp.CreateTaskCommand) (taskapp.TaskResult, error)
}

// NewTaskCreationHandler creates a new TaskCreationHandler.
func NewTaskCreationHandler(
	createTaskUC CreateTaskUseCase,
	logger *slog.Logger,
) *TaskCreationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskCreationHandler{
		createTaskUC: createTaskUC,
		logger:       logger,
	}
}

// Handle processes chat.type_changed events and creates tasks when appropriate.
func (h *TaskCreationHandler) Handle(ctx context.Context, evt event.DomainEvent) error {
	// Only handle chat.type_changed events
	if evt.EventType() != chatdomain.EventTypeChatTypeChanged {
		return nil
	}

	h.logger.DebugContext(ctx, "processing chat.type_changed event",
		slog.String("chat_id", evt.AggregateID()),
	)

	// Extract payload
	payload, err := h.extractPayload(evt)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to extract payload from chat.type_changed",
			slog.String("error", err.Error()),
		)
		return nil // Don't retry for payload extraction failures
	}

	// Parse event data
	var data struct {
		NewType string `json:"new_type"`
		Title   string `json:"title"`
	}
	err = json.Unmarshal(payload, &data)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to unmarshal chat.type_changed payload",
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Only create tasks for task/bug/epic types
	if !isTaskType(data.NewType) {
		h.logger.DebugContext(ctx, "chat type is not task/bug/epic, skipping task creation",
			slog.String("chat_type", data.NewType),
		)
		return nil
	}

	// Parse chat ID
	chatID, err := uuid.ParseUUID(evt.AggregateID())
	if err != nil {
		h.logger.WarnContext(ctx, "invalid chat ID in chat.type_changed event",
			slog.String("chat_id", evt.AggregateID()),
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Extract creator from event metadata
	creatorID, err := h.extractCreatorID(evt)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to extract creator ID from event",
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Map chat type to task entity type
	entityType := mapChatTypeToEntityType(data.NewType)

	// Create task
	cmd := taskapp.CreateTaskCommand{
		ChatID:     chatID,
		Title:      data.Title,
		EntityType: entityType,
		Priority:   taskdomain.PriorityMedium, // Default priority
		CreatedBy:  creatorID,
	}

	result, err := h.createTaskUC.Execute(ctx, cmd)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to create task for chat type change",
			slog.String("chat_id", chatID.String()),
			slog.String("entity_type", string(entityType)),
			slog.String("error", err.Error()),
		)
		// Return error to trigger retry
		return fmt.Errorf("failed to create task: %w", err)
	}

	h.logger.InfoContext(ctx, "task created for chat type change",
		slog.String("chat_id", chatID.String()),
		slog.String("task_id", result.TaskID.String()),
		slog.String("entity_type", string(entityType)),
		slog.String("title", data.Title),
	)

	return nil
}

// AsEventHandler returns this handler as an EventHandler.
func (h *TaskCreationHandler) AsEventHandler() EventHandler {
	return EventHandler(h.Handle)
}

// extractPayload extracts the JSON payload from an event.
func (h *TaskCreationHandler) extractPayload(evt event.DomainEvent) (json.RawMessage, error) {
	// Check if event has Payload() method (PayloadEvent interface)
	if payloadEvt, ok := evt.(PayloadEvent); ok {
		return payloadEvt.Payload(), nil
	}

	// Fallback: marshal the entire event
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}
	return data, nil
}

// extractCreatorID extracts the creator/user ID from event metadata or payload.
func (h *TaskCreationHandler) extractCreatorID(evt event.DomainEvent) (uuid.UUID, error) {
	// Try to get from metadata first
	if metadata := evt.Metadata(); metadata.UserID != "" {
		if userID, err := uuid.ParseUUID(metadata.UserID); err == nil {
			return userID, nil
		}
	}

	// Try to extract from payload
	payload, err := h.extractPayload(evt)
	if err != nil {
		var zeroUUID uuid.UUID
		return zeroUUID, err
	}

	var data struct {
		RenamedBy   string `json:"renamed_by"`
		ChangedBy   string `json:"changed_by"`
		ConvertedBy string `json:"converted_by"`
	}
	err = json.Unmarshal(payload, &data)
	if err != nil {
		var zeroUUID uuid.UUID
		return zeroUUID, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Try different field names
	var userIDStr string
	switch {
	case data.ConvertedBy != "":
		userIDStr = data.ConvertedBy
	case data.ChangedBy != "":
		userIDStr = data.ChangedBy
	case data.RenamedBy != "":
		userIDStr = data.RenamedBy
	default:
		var zeroUUID uuid.UUID
		return zeroUUID, errors.New("no user ID found in event")
	}

	userID, err := uuid.ParseUUID(userIDStr)
	if err != nil {
		var zeroUUID uuid.UUID
		return zeroUUID, fmt.Errorf("invalid user ID: %w", err)
	}

	return userID, nil
}

// isTaskType checks if the chat type is task/bug/epic.
func isTaskType(chatType string) bool {
	return chatType == string(chatdomain.TypeTask) ||
		chatType == string(chatdomain.TypeBug) ||
		chatType == string(chatdomain.TypeEpic)
}

// mapChatTypeToEntityType maps chat type to task entity type.
func mapChatTypeToEntityType(chatType string) taskdomain.EntityType {
	switch chatType {
	case string(chatdomain.TypeTask):
		return taskdomain.TypeTask
	case string(chatdomain.TypeBug):
		return taskdomain.TypeBug
	case string(chatdomain.TypeEpic):
		return taskdomain.TypeEpic
	default:
		return taskdomain.TypeTask // Default fallback
	}
}
