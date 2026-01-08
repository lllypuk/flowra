package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/errs"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestMessageRepository creates test репозиторий сообщений
func setupTestMessageRepository(t *testing.T) *mongodb.MongoMessageRepository {
	t.Helper()

	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("messages")

	return mongodb.NewMongoMessageRepository(coll)
}

// createTestMessage creates тестовое message
func createTestMessage(t *testing.T, chatID, authorID uuid.UUID, content string) *messagedomain.Message {
	t.Helper()

	msg, err := messagedomain.NewMessage(chatID, authorID, content, uuid.UUID(""))
	require.NoError(t, err)
	return msg
}

// createTestThreadReply creates response in треде
func createTestThreadReply(t *testing.T, chatID, authorID, parentID uuid.UUID, content string) *messagedomain.Message {
	t.Helper()

	msg, err := messagedomain.NewMessage(chatID, authorID, content, parentID)
	require.NoError(t, err)
	return msg
}

// TestMongoMessageRepository_Save_And_FindByID checks storage and search messages по ID
func TestMongoMessageRepository_Save_And_FindByID(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	content := "Test message content"

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, content)
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Find by ID
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all fields
	assert.Equal(t, msg.ID(), loaded.ID())
	assert.Equal(t, msg.ChatID(), loaded.ChatID())
	assert.Equal(t, msg.AuthorID(), loaded.AuthorID())
	assert.Equal(t, msg.Content(), loaded.Content())
	assert.False(t, loaded.IsEdited())
	assert.False(t, loaded.IsDeleted())
	assert.WithinDuration(t, msg.CreatedAt(), loaded.CreatedAt(), time.Millisecond)
}

// TestMongoMessageRepository_FindByID_NotFound checks search неexistingего messages
func TestMongoMessageRepository_FindByID_NotFound(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	// Try to find non-existent message
	_, err := repo.FindByID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoMessageRepository_FindByChatID checks search сообщений in чате
func TestMongoMessageRepository_FindByChatID(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save multiple messages
	for i := range 5 {
		content := "Message " + string(rune('A'+i))
		msg := createTestMessage(t, chatID, authorID, content)
		err := repo.Save(ctx, msg)
		require.NoError(t, err)
		// Small delay to ensure different created_at timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Find all messages in chat
	pagination := message.Pagination{Limit: 10, Offset: 0}
	messages, err := repo.FindByChatID(ctx, chatID, pagination)
	require.NoError(t, err)
	assert.Len(t, messages, 5)

	// Test pagination - limit
	pagination = message.Pagination{Limit: 2, Offset: 0}
	messages, err = repo.FindByChatID(ctx, chatID, pagination)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	// Test pagination - offset
	pagination = message.Pagination{Limit: 10, Offset: 3}
	messages, err = repo.FindByChatID(ctx, chatID, pagination)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	// Test empty result for non-existent chat
	pagination = message.Pagination{Limit: 10, Offset: 0}
	messages, err = repo.FindByChatID(ctx, uuid.NewUUID(), pagination)
	require.NoError(t, err)
	assert.Empty(t, messages)
}

// TestMongoMessageRepository_FindThread checks search треда
func TestMongoMessageRepository_FindThread(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create parent message
	parent := createTestMessage(t, chatID, authorID, "Parent message")
	err := repo.Save(ctx, parent)
	require.NoError(t, err)

	// Create thread replies
	for i := range 3 {
		content := "Reply " + string(rune('1'+i))
		reply := createTestThreadReply(t, chatID, authorID, parent.ID(), content)
		saveErr := repo.Save(ctx, reply)
		require.NoError(t, saveErr)
		time.Sleep(10 * time.Millisecond)
	}

	// Find thread
	replies, err := repo.FindThread(ctx, parent.ID())
	require.NoError(t, err)
	assert.Len(t, replies, 3)

	// Verify all are replies to parent
	for _, reply := range replies {
		assert.Equal(t, parent.ID(), reply.ParentMessageID())
	}

	// Empty thread
	replies, err = repo.FindThread(ctx, uuid.NewUUID())
	require.NoError(t, err)
	assert.Empty(t, replies)
}

// TestMongoMessageRepository_CountThreadReplies checks подсчет responseов in треде
func TestMongoMessageRepository_CountThreadReplies(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create parent message
	parent := createTestMessage(t, chatID, authorID, "Parent message")
	err := repo.Save(ctx, parent)
	require.NoError(t, err)

	// Initial count should be 0
	count, err := repo.CountThreadReplies(ctx, parent.ID())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Create thread replies
	for i := range 4 {
		content := "Reply " + string(rune('A'+i))
		reply := createTestThreadReply(t, chatID, authorID, parent.ID(), content)
		saveErr := repo.Save(ctx, reply)
		require.NoError(t, saveErr)
	}

	// Count should be 4
	count, err = repo.CountThreadReplies(ctx, parent.ID())
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestMongoMessageRepository_CountByChatID checks подсчет сообщений in чате
func TestMongoMessageRepository_CountByChatID(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Initial count should be 0
	count, err := repo.CountByChatID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add messages
	for i := range 3 {
		content := "Message " + string(rune('X'+i))
		msg := createTestMessage(t, chatID, authorID, content)
		saveErr := repo.Save(ctx, msg)
		require.NoError(t, saveErr)
	}

	count, err = repo.CountByChatID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestMongoMessageRepository_Delete checks deletion messages
func TestMongoMessageRepository_Delete(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Message to delete")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Verify message exists
	_, err = repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)

	// Delete message
	err = repo.Delete(ctx, msg.ID())
	require.NoError(t, err)

	// Verify message no longer exists
	_, err = repo.FindByID(ctx, msg.ID())
	require.ErrorIs(t, err, errs.ErrNotFound)

	// Delete non-existent message should return error
	err = repo.Delete(ctx, uuid.NewUUID())
	require.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoMessageRepository_AddReaction checks adding реакции
func TestMongoMessageRepository_AddReaction(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Message for reactions")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Add first reaction
	reactorID1 := uuid.NewUUID()
	err = repo.AddReaction(ctx, msg.ID(), "👍", reactorID1)
	require.NoError(t, err)

	// Add second reaction from different user
	reactorID2 := uuid.NewUUID()
	err = repo.AddReaction(ctx, msg.ID(), "👍", reactorID2)
	require.NoError(t, err)

	// Add different reaction
	err = repo.AddReaction(ctx, msg.ID(), "❤️", reactorID1)
	require.NoError(t, err)

	// Verify reactions are saved
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Reactions(), 3)

	// Add reaction to non-existent message should return error
	err = repo.AddReaction(ctx, uuid.NewUUID(), "👍", reactorID1)
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoMessageRepository_RemoveReaction checks deletion реакции
func TestMongoMessageRepository_RemoveReaction(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Message for removing reactions")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Add reactions
	reactorID := uuid.NewUUID()
	err = repo.AddReaction(ctx, msg.ID(), "👍", reactorID)
	require.NoError(t, err)

	err = repo.AddReaction(ctx, msg.ID(), "❤️", reactorID)
	require.NoError(t, err)

	// Verify 2 reactions
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Reactions(), 2)

	// Remove one reaction
	err = repo.RemoveReaction(ctx, msg.ID(), "👍", reactorID)
	require.NoError(t, err)

	// Verify 1 reaction left
	loaded, err = repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Reactions(), 1)
	assert.Equal(t, "❤️", loaded.Reactions()[0].EmojiCode())

	// Remove reaction from non-existent message should return error
	err = repo.RemoveReaction(ctx, uuid.NewUUID(), "👍", reactorID)
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoMessageRepository_GetReactionUsers checks retrieval users по реакции
func TestMongoMessageRepository_GetReactionUsers(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Message for reaction users")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Add reactions from multiple users
	userIDs := []uuid.UUID{uuid.NewUUID(), uuid.NewUUID(), uuid.NewUUID()}
	for _, userID := range userIDs {
		addErr := repo.AddReaction(ctx, msg.ID(), "👍", userID)
		require.NoError(t, addErr)
	}

	// Add different reaction
	err = repo.AddReaction(ctx, msg.ID(), "❤️", userIDs[0])
	require.NoError(t, err)

	// Get users for 👍 reaction
	thumbsUpUsers, err := repo.GetReactionUsers(ctx, msg.ID(), "👍")
	require.NoError(t, err)
	assert.Len(t, thumbsUpUsers, 3)

	// Get users for ❤️ reaction
	heartUsers, err := repo.GetReactionUsers(ctx, msg.ID(), "❤️")
	require.NoError(t, err)
	assert.Len(t, heartUsers, 1)

	// Get users for non-existent reaction
	noUsers, err := repo.GetReactionUsers(ctx, msg.ID(), "🎉")
	require.NoError(t, err)
	assert.Empty(t, noUsers)

	// Get users from non-existent message should return error
	_, err = repo.GetReactionUsers(ctx, uuid.NewUUID(), "👍")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoMessageRepository_SearchInChat checks search сообщений in чате
func TestMongoMessageRepository_SearchInChat(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create messages with searchable content
	contents := []string{
		"Hello world",
		"World is beautiful",
		"Something completely different",
		"Hello there",
		"Another message",
	}

	for _, content := range contents {
		msg := createTestMessage(t, chatID, authorID, content)
		err := repo.Save(ctx, msg)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Search for "world" (case-insensitive)
	results, err := repo.SearchInChat(ctx, chatID, "world", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Search for "Hello"
	results, err = repo.SearchInChat(ctx, chatID, "Hello", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Search for "Something"
	results, err = repo.SearchInChat(ctx, chatID, "Something", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Search with no matches
	results, err = repo.SearchInChat(ctx, chatID, "xyz123", 0, 10)
	require.NoError(t, err)
	assert.Empty(t, results)

	// Search with pagination
	results, err = repo.SearchInChat(ctx, chatID, "e", 0, 2) // matches most messages
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// TestMongoMessageRepository_SearchInChat_SpecialCharacters checks search with спецсимволами
func TestMongoMessageRepository_SearchInChat_SpecialCharacters(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create message with regex special characters
	msg := createTestMessage(t, chatID, authorID, "Price is $100.00 (discounted)")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Search for string with special regex characters - should be escaped
	results, err := repo.SearchInChat(ctx, chatID, "$100.00", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Search for parentheses
	results, err = repo.SearchInChat(ctx, chatID, "(discounted)", 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// TestMongoMessageRepository_FindByAuthor checks search сообщений по автору
func TestMongoMessageRepository_FindByAuthor(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	author1 := uuid.NewUUID()
	author2 := uuid.NewUUID()

	// Create messages from author1
	for i := range 3 {
		content := "Author1 message " + string(rune('A'+i))
		msg := createTestMessage(t, chatID, author1, content)
		err := repo.Save(ctx, msg)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Create messages from author2
	for i := range 2 {
		content := "Author2 message " + string(rune('X'+i))
		msg := createTestMessage(t, chatID, author2, content)
		err := repo.Save(ctx, msg)
		require.NoError(t, err)
	}

	// Find messages by author1
	results, err := repo.FindByAuthor(ctx, chatID, author1, 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	for _, msg := range results {
		assert.Equal(t, author1, msg.AuthorID())
	}

	// Find messages by author2
	results, err = repo.FindByAuthor(ctx, chatID, author2, 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Find messages by non-existent author
	results, err = repo.FindByAuthor(ctx, chatID, uuid.NewUUID(), 0, 10)
	require.NoError(t, err)
	assert.Empty(t, results)

	// Test pagination
	results, err = repo.FindByAuthor(ctx, chatID, author1, 0, 2)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	results, err = repo.FindByAuthor(ctx, chatID, author1, 2, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// TestMongoMessageRepository_InputValidation checks validацию входных данных
func TestMongoMessageRepository_InputValidation(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	t.Run("FindByID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByChatID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByChatID(ctx, uuid.UUID(""), message.Pagination{Limit: 10})
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindThread with zero UUID", func(t *testing.T) {
		_, err := repo.FindThread(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountByChatID with zero UUID", func(t *testing.T) {
		_, err := repo.CountByChatID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountThreadReplies with zero UUID", func(t *testing.T) {
		_, err := repo.CountThreadReplies(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Save with nil message", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Delete with zero UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("AddReaction with zero messageID", func(t *testing.T) {
		err := repo.AddReaction(ctx, uuid.UUID(""), "👍", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("AddReaction with empty emoji", func(t *testing.T) {
		err := repo.AddReaction(ctx, uuid.NewUUID(), "", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("AddReaction with zero userID", func(t *testing.T) {
		err := repo.AddReaction(ctx, uuid.NewUUID(), "👍", uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("RemoveReaction with zero messageID", func(t *testing.T) {
		err := repo.RemoveReaction(ctx, uuid.UUID(""), "👍", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("RemoveReaction with empty emoji", func(t *testing.T) {
		err := repo.RemoveReaction(ctx, uuid.NewUUID(), "", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("RemoveReaction with zero userID", func(t *testing.T) {
		err := repo.RemoveReaction(ctx, uuid.NewUUID(), "👍", uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("GetReactionUsers with zero messageID", func(t *testing.T) {
		_, err := repo.GetReactionUsers(ctx, uuid.UUID(""), "👍")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("GetReactionUsers with empty emoji", func(t *testing.T) {
		_, err := repo.GetReactionUsers(ctx, uuid.NewUUID(), "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("SearchInChat with zero chatID", func(t *testing.T) {
		_, err := repo.SearchInChat(ctx, uuid.UUID(""), "test", 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("SearchInChat with empty query", func(t *testing.T) {
		_, err := repo.SearchInChat(ctx, uuid.NewUUID(), "", 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByAuthor with zero chatID", func(t *testing.T) {
		_, err := repo.FindByAuthor(ctx, uuid.UUID(""), uuid.NewUUID(), 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByAuthor with zero authorID", func(t *testing.T) {
		_, err := repo.FindByAuthor(ctx, uuid.NewUUID(), uuid.UUID(""), 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

// TestMongoMessageRepository_UpdateMessage checks update messages
func TestMongoMessageRepository_UpdateMessage(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Original content")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Edit message
	err = msg.EditContent("Updated content", authorID)
	require.NoError(t, err)

	// Save updated message
	err = repo.Save(ctx, msg)
	require.NoError(t, err)

	// Load and verify
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated content", loaded.Content())
	assert.True(t, loaded.IsEdited())
	assert.NotNil(t, loaded.EditedAt())
}

// TestMongoMessageRepository_SoftDelete checks мягкое deletion via domain
func TestMongoMessageRepository_SoftDelete(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create and save message
	msg := createTestMessage(t, chatID, authorID, "Message to soft delete")
	err := repo.Save(ctx, msg)
	require.NoError(t, err)

	// Soft delete through domain
	err = msg.Delete(authorID)
	require.NoError(t, err)

	// Save updated message
	err = repo.Save(ctx, msg)
	require.NoError(t, err)

	// Load and verify soft delete
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.True(t, loaded.IsDeleted())
	assert.NotNil(t, loaded.DeletedAt())
}

// TestMongoMessageRepository_WithAttachments checks storage messages с вложениями
func TestMongoMessageRepository_WithAttachments(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create message
	msg := createTestMessage(t, chatID, authorID, "Message with attachments")

	// Add attachments
	err := msg.AddAttachment(uuid.NewUUID(), "document.pdf", 1024, "application/pdf")
	require.NoError(t, err)

	err = msg.AddAttachment(uuid.NewUUID(), "image.png", 2048, "image/png")
	require.NoError(t, err)

	// Save message
	err = repo.Save(ctx, msg)
	require.NoError(t, err)

	// Load and verify attachments
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Attachments(), 2)

	// Verify attachment fields
	attachments := loaded.Attachments()
	assert.Equal(t, "document.pdf", attachments[0].FileName())
	assert.Equal(t, int64(1024), attachments[0].FileSize())
	assert.Equal(t, "application/pdf", attachments[0].MimeType())
}

// TestMongoMessageRepository_WithReactions checks storage messages с реакциями via domain
func TestMongoMessageRepository_WithReactions(t *testing.T) {
	repo := setupTestMessageRepository(t)
	ctx := context.Background()

	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	// Create message
	msg := createTestMessage(t, chatID, authorID, "Message with reactions")

	// Add reactions through domain
	userID1 := uuid.NewUUID()
	userID2 := uuid.NewUUID()

	err := msg.AddReaction(userID1, "👍")
	require.NoError(t, err)

	err = msg.AddReaction(userID2, "❤️")
	require.NoError(t, err)

	// Save message
	err = repo.Save(ctx, msg)
	require.NoError(t, err)

	// Load and verify reactions
	loaded, err := repo.FindByID(ctx, msg.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Reactions(), 2)
}
