package testutil

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
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

// ==================== UUID Assertions ====================

// AssertUUIDEqual проверяет равенство двух UUID
func AssertUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.Equal(t, expected, actual, msgAndArgs...)
}

// RequireUUIDEqual проверяет равенство двух UUID и останавливает тест при ошибке
func RequireUUIDEqual(t *testing.T, expected, actual uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.Equal(t, expected, actual, msgAndArgs...)
}

// AssertNotZeroUUID проверяет, что UUID не пустой
func AssertNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.NotEmpty(t, id, msgAndArgs...)
	assert.NotEqual(t, uuid.UUID(""), id, msgAndArgs...)
}

// RequireNotZeroUUID проверяет, что UUID не пустой и останавливает тест при ошибке
func RequireNotZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.NotEmpty(t, id, msgAndArgs...)
	require.NotEqual(t, uuid.UUID(""), id, msgAndArgs...)
}

// AssertZeroUUID проверяет, что UUID пустой
func AssertZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	assert.Empty(t, id, msgAndArgs...)
}

// RequireZeroUUID проверяет, что UUID пустой и останавливает тест при ошибке
func RequireZeroUUID(t *testing.T, id uuid.UUID, msgAndArgs ...any) {
	t.Helper()

	require.Empty(t, id, msgAndArgs...)
}

// ==================== Time Assertions ====================

// AssertTimeApproximatelyEqual проверяет, что два времени приблизительно равны
// с допустимой погрешностью delta (обычно time.Second или time.Millisecond)
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

// RequireTimeApproximatelyEqual проверяет, что два времени приблизительно равны
// и останавливает тест при ошибке
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

// AssertTimeNotZero проверяет, что время не нулевое
func AssertTimeNotZero(t *testing.T, tm time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.False(t, tm.IsZero(), msgAndArgs...)
}

// RequireTimeNotZero проверяет, что время не нулевое и останавливает тест при ошибке
func RequireTimeNotZero(t *testing.T, tm time.Time, msgAndArgs ...any) {
	t.Helper()

	require.False(t, tm.IsZero(), msgAndArgs...)
}

// AssertTimeAfter проверяет, что actual время после expected
func AssertTimeAfter(t *testing.T, actual, expected time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.True(t, actual.After(expected), append([]any{
		"expected %v to be after %v", actual, expected,
	}, msgAndArgs...)...)
}

// AssertTimeBefore проверяет, что actual время до expected
func AssertTimeBefore(t *testing.T, actual, expected time.Time, msgAndArgs ...any) {
	t.Helper()

	assert.True(t, actual.Before(expected), append([]any{
		"expected %v to be before %v", actual, expected,
	}, msgAndArgs...)...)
}
