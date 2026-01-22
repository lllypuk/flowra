package projector_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/projector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventStore is a simple in-memory event store for testing.
type mockEventStore struct {
	events map[string][]event.DomainEvent
}

func newMockEventStore() *mockEventStore {
	return &mockEventStore{
		events: make(map[string][]event.DomainEvent),
	}
}

func (m *mockEventStore) SaveEvents(_ context.Context, aggregateID string, events []event.DomainEvent, _ int) error {
	m.events[aggregateID] = append(m.events[aggregateID], events...)
	return nil
}

func (m *mockEventStore) LoadEvents(_ context.Context, aggregateID string) ([]event.DomainEvent, error) {
	events, ok := m.events[aggregateID]
	if !ok || len(events) == 0 {
		return nil, appcore.ErrAggregateNotFound
	}
	return events, nil
}

func (m *mockEventStore) GetVersion(_ context.Context, aggregateID string) (int, error) {
	events, ok := m.events[aggregateID]
	if !ok {
		return 0, nil
	}
	return len(events), nil
}

func TestChatProjector_RebuildOne_AggregateNotFound(t *testing.T) {
	// Arrange
	eventStore := newMockEventStore()
	logger := slog.Default()

	// Use NewChatProjector (public API)
	chatProj := projector.NewChatProjector(eventStore, nil, logger)

	chatID := uuid.NewUUID()
	ctx := context.Background()

	// Act
	err := chatProj.RebuildOne(ctx, chatID)

	// Assert
	assert.ErrorIs(t, err, appcore.ErrAggregateNotFound)
}

func TestChatProjector_ProcessEvent_InvalidAggregateType(t *testing.T) {
	// Arrange
	eventStore := newMockEventStore()
	logger := slog.Default()

	chatProj := projector.NewChatProjector(eventStore, nil, logger)

	// Create a task event (wrong type)
	evt := &mockDomainEvent{
		aggregateID:   uuid.NewUUID().String(),
		aggregateType: "task",
		eventType:     "TaskCreated",
	}

	ctx := context.Background()

	// Act
	err := chatProj.ProcessEvent(ctx, evt)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid aggregate type")
}

func TestChatProjector_VerifyConsistency_NoEvents(t *testing.T) {
	// Arrange
	eventStore := newMockEventStore()
	logger := slog.Default()

	chatProj := projector.NewChatProjector(eventStore, nil, logger)

	chatID := uuid.NewUUID()
	ctx := context.Background()

	// Act
	consistent, err := chatProj.VerifyConsistency(ctx, chatID)

	// Assert - with no events, LoadEvents returns ErrAggregateNotFound
	// This is expected behavior - the method handles this case
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aggregate not found")
	_ = consistent
}

// mockDomainEvent implements event.DomainEvent for testing.
type mockDomainEvent struct {
	aggregateID   string
	aggregateType string
	eventType     string
	occurredAt    time.Time
	version       int
	metadata      event.Metadata
}

func (m *mockDomainEvent) EventType() string        { return m.eventType }
func (m *mockDomainEvent) AggregateID() string      { return m.aggregateID }
func (m *mockDomainEvent) AggregateType() string    { return m.aggregateType }
func (m *mockDomainEvent) OccurredAt() time.Time    { return m.occurredAt }
func (m *mockDomainEvent) Version() int             { return m.version }
func (m *mockDomainEvent) Metadata() event.Metadata { return m.metadata }

func TestNewChatProjector(t *testing.T) {
	eventStore := newMockEventStore()
	logger := slog.Default()

	chatProj := projector.NewChatProjector(eventStore, nil, logger)

	assert.NotNil(t, chatProj)
}

func TestNewChatProjector_NilLogger(t *testing.T) {
	eventStore := newMockEventStore()

	chatProj := projector.NewChatProjector(eventStore, nil, nil)

	assert.NotNil(t, chatProj)
	assert.NotNil(t, chatProj) // Should use default logger
}
