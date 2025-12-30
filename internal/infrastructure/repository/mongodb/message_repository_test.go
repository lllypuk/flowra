package mongodb_test

import (
	"testing"
)

// TestMongoMessageRepository_Save_Find проверяет сохранение и поиск сообщения
func TestMongoMessageRepository_Save_Find(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// chatID := uuid.NewUUID()
	// authorID := uuid.NewUUID()
	// content := "Test message"
	// parentID := uuid.UUID{}

	// // Create and save
	// msg, _ := messagedomain.NewMessage(chatID, authorID, content, parentID)
	// err := repo.Save(ctx, msg)
	// require.NoError(t, err)

	// // Find
	// loaded, err := repo.FindByID(ctx, msg.ID())
	// require.NoError(t, err)
	// assert.Equal(t, msg.ID(), loaded.ID())
	// assert.Equal(t, content, loaded.Content())
}

// TestMongoMessageRepository_FindByChatID проверяет поиск сообщений в чате
func TestMongoMessageRepository_FindByChatID(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// chatID := uuid.NewUUID()
	// pagination := messagedomain.Pagination{Limit: 10, Offset: 0}

	// messages, err := repo.FindByChatID(ctx, chatID, pagination)
	// require.NoError(t, err)
	// assert.NotNil(t, messages)
}

// TestMongoMessageRepository_FindThread проверяет поиск треда
func TestMongoMessageRepository_FindThread(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// parentID := uuid.NewUUID()
	// messages, err := repo.FindThread(ctx, parentID)
	// require.NoError(t, err)
	// assert.NotNil(t, messages)
}

// TestMongoMessageRepository_CountByChatID проверяет подсчет сообщений
func TestMongoMessageRepository_CountByChatID(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// chatID := uuid.NewUUID()
	// count, err := repo.CountByChatID(ctx, chatID)
	// require.NoError(t, err)
	// assert.GreaterOrEqual(t, count, 0)
}

// TestMongoMessageRepository_Delete проверяет удаление сообщения
func TestMongoMessageRepository_Delete(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// chatID := uuid.NewUUID()
	// authorID := uuid.NewUUID()
	// msg, _ := messagedomain.NewMessage(chatID, authorID, "Delete me", uuid.UUID{})
	// repo.Save(ctx, msg)

	// // Delete
	// err := repo.Delete(ctx, msg.ID())
	// require.NoError(t, err)

	// // Verify deleted
	// _, err = repo.FindByID(ctx, msg.ID())
	// assert.Equal(t, errs.ErrNotFound, err)
}

// TestMongoMessageRepository_InputValidation проверяет валидацию входных данных
func TestMongoMessageRepository_InputValidation(t *testing.T) {
	t.Skip("Requires MongoDB integration test setup")

	// repo := setupTestMessageRepository()
	// ctx := context.Background()

	// // Zero UUID
	// _, err := repo.FindByID(ctx, uuid.UUID{})
	// assert.Equal(t, errs.ErrInvalidInput, err)

	// // Nil message
	// err = repo.Save(ctx, nil)
	// assert.Equal(t, errs.ErrInvalidInput, err)
}
