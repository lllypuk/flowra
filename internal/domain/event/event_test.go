package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventDomain "github.com/lllypuk/flowra/internal/domain/event"
)

func TestNewMetadata(t *testing.T) {
	// Arrange
	userID := "user-123"
	correlationID := "corr-456"
	causationID := "cause-789"

	// Act
	metadata := eventDomain.NewMetadata(userID, correlationID, causationID)

	// Assert
	assert.Equal(t, userID, metadata.UserID)
	assert.Equal(t, correlationID, metadata.CorrelationID)
	assert.Equal(t, causationID, metadata.CausationID)
	assert.False(t, metadata.Timestamp.IsZero())
	assert.WithinDuration(t, time.Now(), metadata.Timestamp, time.Second)
}

func TestMetadata_WithIPAddress(t *testing.T) {
	// Arrange
	metadata := eventDomain.NewMetadata("user-1", "corr-1", "cause-1")
	ip := "192.168.1.1"

	// Act
	updated := metadata.WithIPAddress(ip)

	// Assert
	assert.Equal(t, ip, updated.IPAddress)
	assert.Equal(t, metadata.UserID, updated.UserID)
}

func TestMetadata_WithUserAgent(t *testing.T) {
	// Arrange
	metadata := eventDomain.NewMetadata("user-1", "corr-1", "cause-1")
	ua := "Mozilla/5.0"

	// Act
	updated := metadata.WithUserAgent(ua)

	// Assert
	assert.Equal(t, ua, updated.UserAgent)
	assert.Equal(t, metadata.UserID, updated.UserID)
}

func TestNewBaseEvent(t *testing.T) {
	// Arrange
	eventType := "test.event"
	aggregateID := "agg-123"
	aggregateType := "TestAggregate"
	version := 1
	metadata := eventDomain.NewMetadata("user-1", "corr-1", "cause-1")

	// Act
	event := eventDomain.NewBaseEvent(eventType, aggregateID, aggregateType, version, metadata)

	// Assert
	assert.Equal(t, eventType, event.EventType())
	assert.Equal(t, aggregateID, event.AggregateID())
	assert.Equal(t, aggregateType, event.AggregateType())
	assert.Equal(t, version, event.Version())
	assert.Equal(t, metadata.UserID, event.Metadata().UserID)
	assert.False(t, event.OccurredAt().IsZero())
	assert.WithinDuration(t, time.Now(), event.OccurredAt(), time.Second)
}

func TestBaseEvent_ImplementsDomainEvent(t *testing.T) {
	// Arrange
	metadata := eventDomain.NewMetadata("user-1", "corr-1", "cause-1")
	event := eventDomain.NewBaseEvent("test.event", "agg-1", "Test", 1, metadata)

	// Act & Assert - проверка, что BaseEvent реализует интерфейс DomainEvent
	var _ eventDomain.DomainEvent = event
	require.NotNil(t, event)
}

func TestBaseEvent_AllGetters(t *testing.T) {
	// Arrange
	eventType := "user.created"
	aggregateID := "user-999"
	aggregateType := "User"
	version := 5
	metadata := eventDomain.NewMetadata("admin-1", "corr-xyz", "cause-abc")

	// Act
	event := eventDomain.NewBaseEvent(eventType, aggregateID, aggregateType, version, metadata)

	// Assert
	t.Run("EventType", func(t *testing.T) {
		assert.Equal(t, eventType, event.EventType())
	})

	t.Run("AggregateID", func(t *testing.T) {
		assert.Equal(t, aggregateID, event.AggregateID())
	})

	t.Run("AggregateType", func(t *testing.T) {
		assert.Equal(t, aggregateType, event.AggregateType())
	})

	t.Run("Version", func(t *testing.T) {
		assert.Equal(t, version, event.Version())
	})

	t.Run("Metadata", func(t *testing.T) {
		m := event.Metadata()
		assert.Equal(t, "admin-1", m.UserID)
		assert.Equal(t, "corr-xyz", m.CorrelationID)
		assert.Equal(t, "cause-abc", m.CausationID)
	})

	t.Run("OccurredAt", func(t *testing.T) {
		assert.False(t, event.OccurredAt().IsZero())
	})
}
