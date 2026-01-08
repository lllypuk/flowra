package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const contextTimeout = 30 * time.Second

// NewTestContext creates context with timeout for tests
func NewTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	t.Cleanup(cancel)
	return ctx
}

// AssertNoError checks absence of error and stops test
func AssertNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}
