package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
)

const chatAggregateType = "chat"

// TaskReadModelProjector defines projection behavior required by TaskReadModelProjectionHandler.
// Interface is declared on consumer side.
type TaskReadModelProjector interface {
	ProcessEvent(ctx context.Context, event event.DomainEvent) error
}

// TaskReadModelProjectionHandler updates tasks_read_model from selected chat.* events.
type TaskReadModelProjectionHandler struct {
	projector  TaskReadModelProjector
	repairQ    repair.Queue
	logger     *slog.Logger
	eventTypes map[string]struct{}
}

// NewTaskReadModelProjectionHandler creates a new projection handler.
func NewTaskReadModelProjectionHandler(
	projector TaskReadModelProjector,
	repairQ repair.Queue,
	logger *slog.Logger,
) *TaskReadModelProjectionHandler {
	if logger == nil {
		logger = slog.Default()
	}

	eventTypes := make(map[string]struct{}, len(TaskReadModelProjectionEventTypes()))
	for _, eventType := range TaskReadModelProjectionEventTypes() {
		eventTypes[eventType] = struct{}{}
	}

	return &TaskReadModelProjectionHandler{
		projector:  projector,
		repairQ:    repairQ,
		logger:     logger,
		eventTypes: eventTypes,
	}
}

// Handle processes a chat event and updates tasks_read_model projection.
func (h *TaskReadModelProjectionHandler) Handle(ctx context.Context, evt event.DomainEvent) error {
	if h == nil || h.projector == nil || evt == nil {
		return nil
	}

	if !h.shouldProcess(evt) {
		return nil
	}

	if err := h.projector.ProcessEvent(ctx, evt); err != nil {
		h.queueRepair(ctx, evt, err)
		return fmt.Errorf("failed to project task read model: %w", err)
	}

	return nil
}

// AsEventHandler converts handler to event bus function signature.
func (h *TaskReadModelProjectionHandler) AsEventHandler() EventHandler {
	return h.Handle
}

func (h *TaskReadModelProjectionHandler) shouldProcess(evt event.DomainEvent) bool {
	if !strings.EqualFold(strings.TrimSpace(evt.AggregateType()), chatAggregateType) {
		return false
	}

	if _, ok := h.eventTypes[evt.EventType()]; ok {
		return true
	}

	return false
}

func (h *TaskReadModelProjectionHandler) queueRepair(ctx context.Context, evt event.DomainEvent, projectionErr error) {
	if h.repairQ == nil {
		return
	}

	err := h.repairQ.Add(ctx, repair.Task{
		AggregateID:   evt.AggregateID(),
		AggregateType: chatAggregateType,
		TaskType:      repair.TaskTypeReadModelSync,
		Error:         projectionErr.Error(),
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to queue task projection repair",
			slog.String("aggregate_id", evt.AggregateID()),
			slog.String("event_type", evt.EventType()),
			slog.String("error", err.Error()),
		)
	}
}

// TaskReadModelProjectionEventTypes returns chat events that must update tasks_read_model.
func TaskReadModelProjectionEventTypes() []string {
	return []string{
		chat.EventTypeChatTypeChanged,
		chat.EventTypeStatusChanged,
		chat.EventTypePrioritySet,
		chat.EventTypeUserAssigned,
		chat.EventTypeAssigneeRemoved,
		chat.EventTypeDueDateSet,
		chat.EventTypeDueDateRemoved,
		chat.EventTypeSeveritySet,
		chat.EventTypeAttachmentAdded,
		chat.EventTypeAttachmentRemoved,
		chat.EventTypeChatClosed,
		chat.EventTypeChatReopened,
		chat.EventTypeChatRenamed,
		chat.EventTypeChatDeleted,
	}
}

// RegisterTaskReadModelProjectionHandler registers task projection handler subscriptions.
func RegisterTaskReadModelProjectionHandler(
	bus *RedisEventBus,
	handler *TaskReadModelProjectionHandler,
	logger *slog.Logger,
) error {
	if handler == nil {
		return nil
	}
	registry := NewHandlerRegistry(bus, logger)
	return registry.Register(TaskReadModelProjectionEventTypes(), handler.AsEventHandler())
}
