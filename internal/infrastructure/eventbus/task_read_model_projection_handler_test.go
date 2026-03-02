package eventbus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskReadModelProjectionHandler_Handle_IgnoresUnsupportedEvent(t *testing.T) {
	projector := &mockTaskProjectionProjector{}
	handler := eventbus.NewTaskReadModelProjectionHandler(projector, nil, nil)

	evt := &projectionTestEvent{BaseEvent: event.NewBaseEvent(
		chat.EventTypeParticipantAdded,
		uuid.NewUUID().String(),
		"Chat",
		1,
		event.Metadata{},
	)}

	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 0, projector.calls)
}

func TestTaskReadModelProjectionHandler_Handle_ProjectsSupportedEvent(t *testing.T) {
	projector := &mockTaskProjectionProjector{}
	handler := eventbus.NewTaskReadModelProjectionHandler(projector, nil, nil)

	evt := &projectionTestEvent{BaseEvent: event.NewBaseEvent(
		chat.EventTypeStatusChanged,
		uuid.NewUUID().String(),
		"Chat",
		1,
		event.Metadata{},
	)}

	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 1, projector.calls)
}

func TestTaskReadModelProjectionHandler_Handle_QueuesRepairOnFailure(t *testing.T) {
	projector := &mockTaskProjectionProjector{err: errors.New("boom")}
	queue := &mockRepairQueue{}
	handler := eventbus.NewTaskReadModelProjectionHandler(projector, queue, nil)

	chatID := uuid.NewUUID()
	evt := &projectionTestEvent{BaseEvent: event.NewBaseEvent(
		chat.EventTypeDueDateSet,
		chatID.String(),
		"chat",
		1,
		event.Metadata{},
	)}

	err := handler.Handle(context.Background(), evt)
	require.Error(t, err)
	assert.Equal(t, 1, projector.calls)
	require.Len(t, queue.added, 1)
	assert.Equal(t, chatID.String(), queue.added[0].AggregateID)
	assert.Equal(t, "chat", queue.added[0].AggregateType)
	assert.Equal(t, repair.TaskTypeReadModelSync, queue.added[0].TaskType)
}

func TestTaskReadModelProjectionEventTypes(t *testing.T) {
	eventTypes := eventbus.TaskReadModelProjectionEventTypes()
	assert.Contains(t, eventTypes, chat.EventTypeChatTypeChanged)
	assert.Contains(t, eventTypes, chat.EventTypeStatusChanged)
	assert.Contains(t, eventTypes, chat.EventTypePrioritySet)
	assert.Contains(t, eventTypes, chat.EventTypeUserAssigned)
	assert.Contains(t, eventTypes, chat.EventTypeAssigneeRemoved)
	assert.Contains(t, eventTypes, chat.EventTypeDueDateSet)
	assert.Contains(t, eventTypes, chat.EventTypeDueDateRemoved)
	assert.Contains(t, eventTypes, chat.EventTypeSeveritySet)
	assert.Contains(t, eventTypes, chat.EventTypeAttachmentAdded)
	assert.Contains(t, eventTypes, chat.EventTypeAttachmentRemoved)
	assert.Contains(t, eventTypes, chat.EventTypeChatClosed)
	assert.Contains(t, eventTypes, chat.EventTypeChatReopened)
}

type projectionTestEvent struct {
	event.BaseEvent
}

type mockTaskProjectionProjector struct {
	calls int
	err   error
}

func (m *mockTaskProjectionProjector) ProcessEvent(_ context.Context, _ event.DomainEvent) error {
	m.calls++
	return m.err
}

type mockRepairQueue struct {
	added []repair.Task
}

func (m *mockRepairQueue) Add(_ context.Context, task repair.Task) error {
	m.added = append(m.added, task)
	return nil
}

func (m *mockRepairQueue) Poll(context.Context, int) ([]repair.Task, error) {
	return nil, nil
}

func (m *mockRepairQueue) MarkCompleted(context.Context, string) error {
	return nil
}

func (m *mockRepairQueue) MarkFailed(context.Context, string, error) error {
	return nil
}

func (m *mockRepairQueue) GetStats(context.Context) (*repair.QueueStats, error) {
	return &repair.QueueStats{}, nil
}
