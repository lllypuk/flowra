package testutil

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Event Assertions ====================

// AssertEventPublished checks that event of specific type was published
func AssertEventPublished(t *testing.T, events []event.DomainEvent, eventType string) event.DomainEvent {
	t.Helper()

	for _, evt := range events {
		if evt.EventType() == eventType {
			return evt
		}
	}

	t.Fatalf("Expected event of type %q, but it was not found. Got %d events", eventType, len(events))
	return nil
}

// AssertEventCount checks count of published events
func AssertEventCount(t *testing.T, events []event.DomainEvent, expected int) {
	t.Helper()

	if len(events) != expected {
		t.Fatalf("Expected %d events, but got %d", expected, len(events))
	}
}

// AssertEventType checks event type
func AssertEventType(t *testing.T, evt event.DomainEvent, expectedType string) {
	t.Helper()

	require.Equal(t, expectedType, evt.EventType())
}

// AssertAggregateID checks aggregate ID in the event
func AssertAggregateID(t *testing.T, evt event.DomainEvent, expectedID string) {
	t.Helper()

	require.Equal(t, expectedID, evt.AggregateID())
}

// AssertVersion checks aggregate version in the event
func AssertVersion(t *testing.T, evt event.DomainEvent, expectedVersion int) {
	t.Helper()

	assert.Equal(t, expectedVersion, evt.Version())
}

// ==================== UUID Assertions ====================

// AssertUUIDEqual checks equality of two UUIDs
func AssertUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.Equal(t, expected, actual, msgAndArgs...)
}

// RequireUUIDEqual checks equality of two UUIDs and stops test at error
func RequireUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.Equal(t, expected, actual, msgAndArgs...)
}

// AssertNotZeroUUID checks, that UUID not empty
func AssertNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.NotEmpty(t, id, msgAndArgs...)
	assert.NotEqual(t, uuid.UUID(""), id, msgAndArgs...)
}

// RequireNotZeroUUID checks, that UUID not empty and stops test at error
func RequireNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.NotEmpty(t, id, msgAndArgs...)
	require.NotEqual(t, uuid.UUID(""), id, msgAndArgs...)
}

// AssertZeroUUID checks, that UUID empty
func AssertZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.Empty(t, id, msgAndArgs...)
}

// RequireZeroUUID checks, that UUID empty and stops test at error
func RequireZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.Empty(t, id, msgAndArgs...)
}

// ==================== Time Assertions ====================

// AssertTimeApproximatelyEqual checks, that two time approximately equal
// with acceptable tolerance delta (usually time.Second or time.Millisecond)
func AssertTimeApproximatelyEqual(t *testing.T, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	t.Helper()

	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}

	assert.LessOrEqual(t, diff, delta, append([]any{
		"expected time %v to be within %v of %v, but difference was %v",
		actual, delta, expected, diff,
	}, msgAndArgs...)...)
}

// RequireTimeApproximatelyEqual checks, that two time approximately equal
// and stops test at error
func RequireTimeApproximatelyEqual(t *testing.T, expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	t.Helper()

	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}

	require.LessOrEqual(t, diff, delta, append([]any{
		"expected time %v to be within %v of %v, but difference was %v",
		actual, delta, expected, diff,
	}, msgAndArgs...)...)
}

// AssertTimeNotZero checks, that time not nil
func AssertTimeNotZero(t *testing.T, tm time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.False(t, tm.IsZero(), msgAndArgs...)
}

// RequireTimeNotZero checks, that time not nil and stops test at error
func RequireTimeNotZero(t *testing.T, tm time.Time, msgAndArgs ...any) {
	t.Helper()

	require.False(t, tm.IsZero(), msgAndArgs...)
}

// AssertTimeAfter checks, that actual time after expected
func AssertTimeAfter(t *testing.T, actual, expected time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.True(t, actual.After(expected), append([]any{
		"expected %v to be after %v", actual, expected,
	}, msgAndArgs...)...)
}

// AssertTimeBefore checks, that actual time before expected
func AssertTimeBefore(t *testing.T, actual, expected time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.True(t, actual.Before(expected), append([]any{
		"expected %v to be before %v", actual, expected,
	}, msgAndArgs...)...)
}
