//go:build integration

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Это пример integration теста
// Реальные integration тесты будут написаны когда будет реализован PostgresEventStore
func TestIntegrationExample(t *testing.T) {
	// db := testutil.SetupTestDatabase(t)
	// defer testutil.TeardownTestDatabase(t, db)

	// eventStore := eventstore.NewPostgresEventStore(db)
	// useCase := taskusecase.NewCreateTaskUseCase(eventStore)

	// Test...
	assert.True(t, true)
}
