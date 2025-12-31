package testutil

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TestAssertUUIDEqual tests AssertUUIDEqual function
func TestAssertUUIDEqual(t *testing.T) {
	id := uuid.NewUUID()

	// Should pass for equal UUIDs
	AssertUUIDEqual(t, id, id)

	// Different UUIDs would fail, but we can't test that directly
}

// TestRequireUUIDEqual tests RequireUUIDEqual function
func TestRequireUUIDEqual(t *testing.T) {
	id := uuid.NewUUID()

	// Should pass for equal UUIDs
	RequireUUIDEqual(t, id, id)
}

// TestAssertNotZeroUUID tests AssertNotZeroUUID function
func TestAssertNotZeroUUID(t *testing.T) {
	id := uuid.NewUUID()

	// Should pass for non-zero UUID
	AssertNotZeroUUID(t, id)
}

// TestRequireNotZeroUUID tests RequireNotZeroUUID function
func TestRequireNotZeroUUID(t *testing.T) {
	id := uuid.NewUUID()

	// Should pass for non-zero UUID
	RequireNotZeroUUID(t, id)
}

// TestAssertZeroUUID tests AssertZeroUUID function
func TestAssertZeroUUID(t *testing.T) {
	// Should pass for zero UUID
	AssertZeroUUID(t, uuid.UUID(""))
}

// TestRequireZeroUUID tests RequireZeroUUID function
func TestRequireZeroUUID(t *testing.T) {
	// Should pass for zero UUID
	RequireZeroUUID(t, uuid.UUID(""))
}

// TestAssertTimeApproximatelyEqual tests AssertTimeApproximatelyEqual function
func TestAssertTimeApproximatelyEqual(t *testing.T) {
	now := time.Now()

	// Should pass for times within delta
	AssertTimeApproximatelyEqual(t, now, now.Add(100*time.Millisecond), time.Second)
	AssertTimeApproximatelyEqual(t, now, now.Add(-100*time.Millisecond), time.Second)

	// Should pass for exact times
	AssertTimeApproximatelyEqual(t, now, now, time.Nanosecond)
}

// TestRequireTimeApproximatelyEqual tests RequireTimeApproximatelyEqual function
func TestRequireTimeApproximatelyEqual(t *testing.T) {
	now := time.Now()

	// Should pass for times within delta
	RequireTimeApproximatelyEqual(t, now, now.Add(50*time.Millisecond), 100*time.Millisecond)
}

// TestAssertTimeNotZero tests AssertTimeNotZero function
func TestAssertTimeNotZero(t *testing.T) {
	// Should pass for non-zero time
	AssertTimeNotZero(t, time.Now())
}

// TestRequireTimeNotZero tests RequireTimeNotZero function
func TestRequireTimeNotZero(t *testing.T) {
	// Should pass for non-zero time
	RequireTimeNotZero(t, time.Now())
}

// TestAssertTimeAfter tests AssertTimeAfter function
func TestAssertTimeAfter(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	// Should pass when actual is after expected
	AssertTimeAfter(t, later, now)
}

// TestAssertTimeBefore tests AssertTimeBefore function
func TestAssertTimeBefore(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	// Should pass when actual is before expected
	AssertTimeBefore(t, earlier, now)
}
