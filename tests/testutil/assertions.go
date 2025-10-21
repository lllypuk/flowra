package testutil

import (
	"testing"

	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertEventPublished проверяет, что событие определенного типа было опубликовано
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

// AssertEventCount проверяет количество опубликованных событий
func AssertEventCount(t *testing.T, events []event.DomainEvent, expected int) {
	t.Helper()

	if len(events) != expected {
		t.Fatalf("Expected %d events, but got %d", expected, len(events))
	}
}

// AssertEventType проверяет тип события
func AssertEventType(t *testing.T, evt event.DomainEvent, expectedType string) {
	t.Helper()

	require.Equal(t, expectedType, evt.EventType())
}

// AssertAggregateID проверяет ID агрегата в событии
func AssertAggregateID(t *testing.T, evt event.DomainEvent, expectedID string) {
	t.Helper()

	require.Equal(t, expectedID, evt.AggregateID())
}

// AssertVersion проверяет версию агрегата в событии
func AssertVersion(t *testing.T, evt event.DomainEvent, expectedVersion int) {
	t.Helper()

	assert.Equal(t, expectedVersion, evt.Version())
}

// AssertError проверяет наличие ошибки
func AssertError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()

	require.Error(t, err, msgAndArgs...)
}

// AssertErrorIs проверяет, что ошибка является конкретным типом
func AssertErrorIs(t *testing.T, err, target error, msgAndArgs ...any) {
	t.Helper()

	require.ErrorIs(t, err, target, msgAndArgs...)
}

// AssertEqual проверяет равенство
func AssertEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()

	require.Equal(t, expected, actual, msgAndArgs...)
}

// AssertNotNil проверяет, что значение не nil
func AssertNotNil(t *testing.T, value any, msgAndArgs ...any) {
	t.Helper()

	require.NotNil(t, value, msgAndArgs...)
}

// AssertNil проверяет, что значение является nil
func AssertNil(t *testing.T, value any, msgAndArgs ...any) {
	t.Helper()

	require.Nil(t, value, msgAndArgs...)
}

// AssertLen проверяет длину слайса/массива
func AssertLen(t *testing.T, collection any, length int, msgAndArgs ...any) {
	t.Helper()

	require.Len(t, collection, length, msgAndArgs...)
}

// AssertGreater проверяет, что значение больше
func AssertGreater(t *testing.T, value1, value2 any, msgAndArgs ...any) {
	t.Helper()

	require.Greater(t, value1, value2, msgAndArgs...)
}

// AssertGreaterOrEqual проверяет, что значение больше или равно
func AssertGreaterOrEqual(t *testing.T, value1, value2 any, msgAndArgs ...any) {
	t.Helper()

	require.GreaterOrEqual(t, value1, value2, msgAndArgs...)
}

// AssertTrue проверяет, что значение true
func AssertTrue(t *testing.T, value bool, msgAndArgs ...any) {
	t.Helper()

	require.True(t, value, msgAndArgs...)
}

// AssertFalse проверяет, что значение false
func AssertFalse(t *testing.T, value bool, msgAndArgs ...any) {
	t.Helper()

	require.False(t, value, msgAndArgs...)
}
