package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	// Arrange
	input := 2

	// Act
	result := input + 2

	// Assert
	assert.Equal(t, 4, result)
	require.NotZero(t, result)
}
