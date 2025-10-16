package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const contextTimeout = 30 * time.Second

// NewTestContext создает context с таймаутом для тестов
func NewTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	t.Cleanup(cancel)
	return ctx
}

// AssertNoError проверяет отсутствие ошибки и останавливает тест
func AssertNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}
