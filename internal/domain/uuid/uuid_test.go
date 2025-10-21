package uuid_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowra/flowra/internal/domain/uuid"
)

func TestNewUUID(t *testing.T) {
	// Act
	id := uuid.NewUUID()

	// Assert
	assert.NotEmpty(t, id)
	assert.False(t, id.IsZero())
	assert.Len(t, id.String(), 36) // UUID v4 length
}

func TestNewUUID_Uniqueness(t *testing.T) {
	// Act
	id1 := uuid.NewUUID()
	id2 := uuid.NewUUID()

	// Assert
	assert.NotEqual(t, id1, id2, "Generated UUIDs should be unique")
}

func TestParseUUID_ValidUUID(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act
	id, err := uuid.ParseUUID(validUUID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
	assert.False(t, id.IsZero())
}

func TestParseUUID_InvalidUUID(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid format", "not-a-uuid"},
		{"too short", "550e8400"},
		{"invalid characters", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			id, err := uuid.ParseUUID(tc.input)

			// Assert
			require.Error(t, err)
			assert.Empty(t, id)
			assert.True(t, id.IsZero())
		})
	}
}

func TestMustParseUUID_ValidUUID(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act & Assert - should not panic
	assert.NotPanics(t, func() {
		id := uuid.MustParseUUID(validUUID)
		assert.Equal(t, validUUID, id.String())
	})
}

func TestMustParseUUID_InvalidUUID_Panics(t *testing.T) {
	// Arrange
	invalidUUID := "not-a-uuid"

	// Act & Assert - should panic
	assert.Panics(t, func() {
		uuid.MustParseUUID(invalidUUID)
	})
}

func TestUUID_String(t *testing.T) {
	// Arrange
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"
	id := uuid.UUID(uuidStr)

	// Act
	result := id.String()

	// Assert
	assert.Equal(t, uuidStr, result)
}

func TestUUID_IsZero(t *testing.T) {
	testCases := []struct {
		name     string
		uuid     uuid.UUID
		expected bool
	}{
		{"empty UUID", uuid.UUID(""), true},
		{"non-empty UUID", uuid.UUID("550e8400-e29b-41d4-a716-446655440000"), false},
		{"new UUID", uuid.NewUUID(), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			result := tc.uuid.IsZero()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}
