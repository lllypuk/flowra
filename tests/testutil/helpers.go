package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// NewTestContext создает context с таймаутом для тестов
func NewTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// AssertNoError проверяет отсутствие ошибки и останавливает тест
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}
