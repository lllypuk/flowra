package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	notificationdomain "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// setupTestNotificationRepository creates test репозиторий уведомлений
func setupTestNotificationRepository(t *testing.T) *mongodb.MongoNotificationRepository {
	t.Helper()

	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("notifications")

	return mongodb.NewMongoNotificationRepository(coll)
}

// createTestNotification creates тестовое notification
func createTestNotification(
	t *testing.T,
	userID uuid.UUID,
	notifType notificationdomain.Type,
	title, message string,
) *notificationdomain.Notification {
	t.Helper()

	notif, err := notificationdomain.NewNotification(userID, notifType, title, message, "")
	require.NoError(t, err)
	return notif
}

// createTestNotificationWithResource creates тестовое notification с ресурсом
func createTestNotificationWithResource(
	t *testing.T,
	userID uuid.UUID,
	notifType notificationdomain.Type,
	title, message, resourceID string,
) *notificationdomain.Notification {
	t.Helper()

	notif, err := notificationdomain.NewNotification(userID, notifType, title, message, resourceID)
	require.NoError(t, err)
	return notif
}

// TestMongoNotificationRepository_Save_And_FindByID checks storage and search уведомления по ID
func TestMongoNotificationRepository_Save_And_FindByID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()
	resourceID := uuid.NewUUID().String()

	// Create and save notification
	notif := createTestNotificationWithResource(
		t,
		userID,
		notificationdomain.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to task #123",
		resourceID,
	)
	err := repo.Save(ctx, notif)
	require.NoError(t, err)

	// Find by ID
	loaded, err := repo.FindByID(ctx, notif.ID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all fields
	assert.Equal(t, notif.ID(), loaded.ID())
	assert.Equal(t, notif.UserID(), loaded.UserID())
	assert.Equal(t, notificationdomain.TypeTaskAssigned, loaded.Type())
	assert.Equal(t, "Task Assigned", loaded.Title())
	assert.Equal(t, "You have been assigned to task #123", loaded.Message())
	assert.Equal(t, resourceID, loaded.ResourceID())
	assert.False(t, loaded.IsRead())
	assert.nil(t, loaded.ReadAt())
	assert.WithinDuration(t, notif.CreatedAt(), loaded.CreatedAt(), time.Millisecond)
}

// TestMongoNotificationRepository_FindByID_NotFound checks search неexistingего уведомления
func TestMongoNotificationRepository_FindByID_NotFound(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Try to find non-existent notification
	_, err := repo.FindByID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoNotificationRepository_FindByUserID checks search уведомлений user
func TestMongoNotificationRepository_FindByUserID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create and save multiple notifications
	for i := range 5 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeChatMention,
			"Notification",
			"Message "+string(rune('A'+i)),
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
		// Small delay to ensure different created_at timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Find notifications with pagination
	notifications, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, notifications, 5)

	// Verify ordering (newest first)
	for i := range len(notifications) - 1 {
		assert.True(t, notifications[i].CreatedAt().After(notifications[i+1].CreatedAt()) ||
			notifications[i].CreatedAt().Equal(notifications[i+1].CreatedAt()))
	}

	// Test pagination - get only 2
	notifications, err = repo.FindByUserID(ctx, userID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, notifications, 2)

	// Test pagination - skip 2, get 2
	notifications, err = repo.FindByUserID(ctx, userID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, notifications, 2)
}

// TestMongoNotificationRepository_FindByUserID_Empty checks empty result
func TestMongoNotificationRepository_FindByUserID_Empty(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Find notifications for user with no notifications
	notifications, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, notifications)
	assert.Empty(t, notifications)
}

// TestMongoNotificationRepository_FindUnreadByUserID checks search unread уведомлений
func TestMongoNotificationRepository_FindUnreadByUserID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create 3 unread notifications
	for i := range 3 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeTaskCreated,
			"Task Created",
			"Task "+string(rune('1'+i)),
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create 2 read notifications
	for i := range 2 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeTaskCreated,
			"Task Created Read",
			"Read Task "+string(rune('1'+i)),
		)
		err := notif.MarkAsRead()
		require.NoError(t, err)
		err = repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Find unread notifications
	unread, err := repo.FindUnreadByUserID(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, unread, 3)

	// Verify all are unread
	for _, n := range unread {
		assert.False(t, n.IsRead())
	}
}

// TestMongoNotificationRepository_CountUnreadByUserID checks подсчет unread уведомлений
func TestMongoNotificationRepository_CountUnreadByUserID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create 4 unread notifications
	for range 4 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeChatMessage,
			"New Message",
			"You have a New message",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create 2 read notifications
	for range 2 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeChatMessage,
			"New Message Read",
			"Read message",
		)
		err := notif.MarkAsRead()
		require.NoError(t, err)
		err = repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Count unread
	count, err := repo.CountUnreadByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestMongoNotificationRepository_MarkAsRead checks отметку уведомления as read
func TestMongoNotificationRepository_MarkAsRead(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create notification
	notif := createTestNotification(
		t,
		userID,
		notificationdomain.TypeChatMention,
		"Mention",
		"You were mentioned",
	)
	err := repo.Save(ctx, notif)
	require.NoError(t, err)

	// Verify it's unread
	loaded, err := repo.FindByID(ctx, notif.ID())
	require.NoError(t, err)
	assert.False(t, loaded.IsRead())

	// Mark as read
	err = repo.MarkAsRead(ctx, notif.ID())
	require.NoError(t, err)

	// Verify it's read
	loaded, err = repo.FindByID(ctx, notif.ID())
	require.NoError(t, err)
	assert.True(t, loaded.IsRead())
	assert.NotNil(t, loaded.ReadAt())
}

// TestMongoNotificationRepository_MarkAsRead_AlreadyRead checks повторную отметку
func TestMongoNotificationRepository_MarkAsRead_AlreadyRead(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create and mark as read immediately
	notif := createTestNotification(
		t,
		userID,
		notificationdomain.TypeChatMention,
		"Mention",
		"You were mentioned",
	)
	err := notif.MarkAsRead()
	require.NoError(t, err)
	err = repo.Save(ctx, notif)
	require.NoError(t, err)

	// Mark as read again - should not error
	err = repo.MarkAsRead(ctx, notif.ID())
	require.NoError(t, err)
}

// TestMongoNotificationRepository_MarkAsRead_NotFound checks error for неexistingего уведомления
func TestMongoNotificationRepository_MarkAsRead_NotFound(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Mark non-existent notification as read
	err := repo.MarkAsRead(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoNotificationRepository_MarkAllAsRead checks отметку all уведомлений as прочитанных
func TestMongoNotificationRepository_MarkAllAsRead(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()

	// Create 3 notifications for user
	for range 3 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeTaskAssigned,
			"Task",
			"Task notification",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create 2 notifications for other user
	for range 2 {
		notif := createTestNotification(
			t,
			otherUserID,
			notificationdomain.TypeTaskAssigned,
			"Task",
			"Other user notification",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Mark all as read for first user
	err := repo.MarkAllAsRead(ctx, userID)
	require.NoError(t, err)

	// Verify all first user's notifications are read
	unread, err := repo.FindUnreadByUserID(ctx, userID, 10)
	require.NoError(t, err)
	assert.Empty(t, unread)

	// Verify other user's notifications are still unread
	unread, err = repo.FindUnreadByUserID(ctx, otherUserID, 10)
	require.NoError(t, err)
	assert.Len(t, unread, 2)
}

// TestMongoNotificationRepository_MarkManyAsRead checks отметку нескольких уведомлений
func TestMongoNotificationRepository_MarkManyAsRead(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create 5 notifications
	var notifIDs []uuid.UUID
	for range 5 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeChatMessage,
			"Message",
			"New message",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
		notifIDs = append(notifIDs, notif.ID())
	}

	// Mark first 3 as read
	err := repo.MarkManyAsRead(ctx, notifIDs[:3])
	require.NoError(t, err)

	// Verify unread count
	count, err := repo.CountUnreadByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// TestMongoNotificationRepository_MarkManyAsRead_Empty checks empty list
func TestMongoNotificationRepository_MarkManyAsRead_Empty(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Mark empty list - should not error
	err := repo.MarkManyAsRead(ctx, []uuid.UUID{})
	require.NoError(t, err)
}

// TestMongoNotificationRepository_Delete checks deletion уведомления
func TestMongoNotificationRepository_Delete(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create notification
	notif := createTestNotification(
		t,
		userID,
		notificationdomain.TypeSystem,
		"System",
		"System notification",
	)
	err := repo.Save(ctx, notif)
	require.NoError(t, err)

	// Delete notification
	err = repo.Delete(ctx, notif.ID())
	require.NoError(t, err)

	// Verify it's deleted
	_, err = repo.FindByID(ctx, notif.ID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoNotificationRepository_Delete_NotFound checks deletion неexistingего уведомления
func TestMongoNotificationRepository_Delete_NotFound(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Delete non-existent notification
	err := repo.Delete(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoNotificationRepository_DeleteByUserID checks deletion all уведомлений user
func TestMongoNotificationRepository_DeleteByUserID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()

	// Create notifications for user
	for range 3 {
		notif := createTestNotification(t, userID, notificationdomain.TypeTaskCreated, "Task", "Body")
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create notification for other user
	notif := createTestNotification(t, otherUserID, notificationdomain.TypeTaskCreated, "Task", "Body")
	err := repo.Save(ctx, notif)
	require.NoError(t, err)

	// Delete all user's notifications
	err = repo.DeleteByUserID(ctx, userID)
	require.NoError(t, err)

	// Verify user has no notifications
	notifications, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, notifications)

	// Verify other user still has notification
	notifications, err = repo.FindByUserID(ctx, otherUserID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
}

// TestMongoNotificationRepository_DeleteOlderThan checks deletion старых уведомлений
func TestMongoNotificationRepository_DeleteOlderThan(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("notifications")
	repo := mongodb.NewMongoNotificationRepository(coll)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Insert old notification directly with old date
	oldDoc := bson.M{
		"notification_id": uuid.NewUUID().String(),
		"user_id":         userID.String(),
		"type":            string(notificationdomain.TypeSystem),
		"title":           "Old notification",
		"message":         "Body",
		"created_at":      time.Now().Add(-30 * 24 * time.Hour), // 30 days ago
	}
	_, err := coll.InsertOne(ctx, oldDoc)
	require.NoError(t, err)

	// Create recent notification through repo
	recentNotif := createTestNotification(t, userID, notificationdomain.TypeSystem, "Recent", "Body")
	err = repo.Save(ctx, recentNotif)
	require.NoError(t, err)

	// Delete older than 7 days
	deleted, err := repo.DeleteOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify recent notification still exists
	_, err = repo.FindByID(ctx, recentNotif.ID())
	require.NoError(t, err)
}

// TestMongoNotificationRepository_DeleteReadOlderThan checks deletion прочитанных старых уведомлений
func TestMongoNotificationRepository_DeleteReadOlderThan(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("notifications")
	repo := mongodb.NewMongoNotificationRepository(coll)
	ctx := context.Background()

	userID := uuid.NewUUID()
	now := time.Now()
	oldDate := now.Add(-30 * 24 * time.Hour)

	// Insert old read notification
	oldReadDoc := bson.M{
		"notification_id": uuid.NewUUID().String(),
		"user_id":         userID.String(),
		"type":            string(notificationdomain.TypeSystem),
		"title":           "Old read notification",
		"message":         "Body",
		"read_at":         oldDate.Add(time.Hour),
		"created_at":      oldDate,
	}
	_, err := coll.InsertOne(ctx, oldReadDoc)
	require.NoError(t, err)

	// Insert old unread notification
	oldUnreadDoc := bson.M{
		"notification_id": uuid.NewUUID().String(),
		"user_id":         userID.String(),
		"type":            string(notificationdomain.TypeSystem),
		"title":           "Old unread notification",
		"message":         "Body",
		"created_at":      oldDate,
	}
	_, err = coll.InsertOne(ctx, oldUnreadDoc)
	require.NoError(t, err)

	// Delete read older than 7 days
	deleted, err := repo.DeleteReadOlderThan(ctx, now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, deleted) // Only the read one

	// Verify the unread old notification still exists
	notifications, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, "Old unread notification", notifications[0].Title())
}

// TestMongoNotificationRepository_SaveBatch checks пакетное storage уведомлений
func TestMongoNotificationRepository_SaveBatch(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create batch of notifications
	var notifications []*notificationdomain.Notification
	for i := range 5 {
		notif := createTestNotification(
			t,
			userID,
			notificationdomain.TypeTaskAssigned,
			"Task "+string(rune('A'+i)),
			"Batch notification",
		)
		notifications = append(notifications, notif)
	}

	// Save batch
	err := repo.SaveBatch(ctx, notifications)
	require.NoError(t, err)

	// Verify all were saved
	loaded, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, loaded, 5)
}

// TestMongoNotificationRepository_SaveBatch_Empty checks empty пакет
func TestMongoNotificationRepository_SaveBatch_Empty(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Save empty batch - should not error
	err := repo.SaveBatch(ctx, []*notificationdomain.Notification{})
	require.NoError(t, err)
}

// TestMongoNotificationRepository_SaveBatch_WithNil checks пакет с nil элементом
func TestMongoNotificationRepository_SaveBatch_WithNil(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()
	notif := createTestNotification(t, userID, notificationdomain.TypeSystem, "Title", "Message")

	// Batch with nil element
	notifications := []*notificationdomain.Notification{notif, nil}

	// Should return error
	err := repo.SaveBatch(ctx, notifications)
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

// TestMongoNotificationRepository_FindByType checks search по типу уведомления
func TestMongoNotificationRepository_FindByType(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create notifications of different types
	types := []notificationdomain.Type{
		notificationdomain.TypeChatMention,
		notificationdomain.TypeChatMention,
		notificationdomain.TypeTaskAssigned,
		notificationdomain.TypeTaskCreated,
		notificationdomain.TypeTaskCreated,
		notificationdomain.TypeTaskCreated,
	}

	for i, typ := range types {
		notif := createTestNotification(
			t,
			userID,
			typ,
			"Notification "+string(rune('A'+i)),
			"Body",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Find by type
	mentions, err := repo.FindByType(ctx, userID, notificationdomain.TypeChatMention, 0, 10)
	require.NoError(t, err)
	assert.Len(t, mentions, 2)

	taskAssigned, err := repo.FindByType(ctx, userID, notificationdomain.TypeTaskAssigned, 0, 10)
	require.NoError(t, err)
	assert.Len(t, taskAssigned, 1)

	taskCreated, err := repo.FindByType(ctx, userID, notificationdomain.TypeTaskCreated, 0, 10)
	require.NoError(t, err)
	assert.Len(t, taskCreated, 3)

	// Test with limit
	taskCreatedLimited, err := repo.FindByType(ctx, userID, notificationdomain.TypeTaskCreated, 0, 2)
	require.NoError(t, err)
	assert.Len(t, taskCreatedLimited, 2)
}

// TestMongoNotificationRepository_FindByResourceID checks search по ID ресурса
func TestMongoNotificationRepository_FindByResourceID(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()
	resourceID := uuid.NewUUID().String()
	otherResourceID := uuid.NewUUID().String()

	// Create notifications for resource
	for i := range 3 {
		notif := createTestNotificationWithResource(
			t,
			userID,
			notificationdomain.TypeTaskStatusChanged,
			"Status Changed "+string(rune('A'+i)),
			"Task status changed",
			resourceID,
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create notification for other resource
	otherNotif := createTestNotificationWithResource(
		t,
		userID,
		notificationdomain.TypeTaskStatusChanged,
		"Other Status",
		"Other task status changed",
		otherResourceID,
	)
	err := repo.Save(ctx, otherNotif)
	require.NoError(t, err)

	// Find by resource ID
	notifications, err := repo.FindByResourceID(ctx, resourceID)
	require.NoError(t, err)
	assert.Len(t, notifications, 3)

	for _, n := range notifications {
		assert.Equal(t, resourceID, n.ResourceID())
	}
}

// TestMongoNotificationRepository_FindByResourceID_Empty checks empty result
func TestMongoNotificationRepository_FindByResourceID_Empty(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	// Find by non-existent resource ID
	notifications, err := repo.FindByResourceID(ctx, uuid.NewUUID().String())
	require.NoError(t, err)
	assert.NotNil(t, notifications)
	assert.Empty(t, notifications)
}

// TestMongoNotificationRepository_CountByType checks подсчет по типам
func TestMongoNotificationRepository_CountByType(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create notifications of different types
	types := []notificationdomain.Type{
		notificationdomain.TypeChatMention,
		notificationdomain.TypeChatMention,
		notificationdomain.TypeTaskAssigned,
		notificationdomain.TypeTaskStatusChanged,
		notificationdomain.TypeTaskStatusChanged,
		notificationdomain.TypeTaskStatusChanged,
	}

	for i, typ := range types {
		notif := createTestNotification(
			t,
			userID,
			typ,
			"Notification "+string(rune('A'+i)),
			"Body",
		)
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Count by type
	counts, err := repo.CountByType(ctx, userID)
	require.NoError(t, err)

	assert.Equal(t, 2, counts[notificationdomain.TypeChatMention])
	assert.Equal(t, 1, counts[notificationdomain.TypeTaskAssigned])
	assert.Equal(t, 3, counts[notificationdomain.TypeTaskStatusChanged])
}

// TestMongoNotificationRepository_CountByType_Empty checks empty result
func TestMongoNotificationRepository_CountByType_Empty(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Count for user with no notifications
	counts, err := repo.CountByType(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, counts)
	assert.Empty(t, counts)
}

// TestMongoNotificationRepository_InputValidation checks validацию входных данных
func TestMongoNotificationRepository_InputValidation(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	t.Run("FindByID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByUserID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByUserID(ctx, uuid.UUID(""), 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindUnreadByUserID with zero UUID", func(t *testing.T) {
		_, err := repo.FindUnreadByUserID(ctx, uuid.UUID(""), 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountUnreadByUserID with zero UUID", func(t *testing.T) {
		_, err := repo.CountUnreadByUserID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountByType with zero UUID", func(t *testing.T) {
		_, err := repo.CountByType(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByType with zero UUID", func(t *testing.T) {
		_, err := repo.FindByType(ctx, uuid.UUID(""), notificationdomain.TypeChatMention, 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByResourceID with empty string", func(t *testing.T) {
		_, err := repo.FindByResourceID(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Save with nil notification", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Delete with zero UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("DeleteByUserID with zero UUID", func(t *testing.T) {
		err := repo.DeleteByUserID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("MarkAsRead with zero UUID", func(t *testing.T) {
		err := repo.MarkAsRead(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("MarkAllAsRead with zero UUID", func(t *testing.T) {
		err := repo.MarkAllAsRead(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("MarkManyAsRead with zero UUID in list", func(t *testing.T) {
		err := repo.MarkManyAsRead(ctx, []uuid.UUID{uuid.NewUUID(), uuid.UUID("")})
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

// TestMongoNotificationRepository_Save_Update checks update existingего уведомления
func TestMongoNotificationRepository_Save_Update(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	userID := uuid.NewUUID()

	// Create and save notification
	notif := createTestNotification(
		t,
		userID,
		notificationdomain.TypeTaskAssigned,
		"Original Title",
		"Original Message",
	)
	err := repo.Save(ctx, notif)
	require.NoError(t, err)

	// Mark as read and save again
	err = notif.MarkAsRead()
	require.NoError(t, err)
	err = repo.Save(ctx, notif)
	require.NoError(t, err)

	// Verify update
	loaded, err := repo.FindByID(ctx, notif.ID())
	require.NoError(t, err)
	assert.True(t, loaded.IsRead())
}

// TestMongoNotificationRepository_IsolationBetweenUsers checks изоляцию данных between userелями
func TestMongoNotificationRepository_IsolationBetweenUsers(t *testing.T) {
	repo := setupTestNotificationRepository(t)
	ctx := context.Background()

	user1ID := uuid.NewUUID()
	user2ID := uuid.NewUUID()

	// Create notifications for user1
	for range 3 {
		notif := createTestNotification(t, user1ID, notificationdomain.TypeChatMessage, "User1 Notif", "Body")
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Create notifications for user2
	for range 5 {
		notif := createTestNotification(t, user2ID, notificationdomain.TypeChatMessage, "User2 Notif", "Body")
		err := repo.Save(ctx, notif)
		require.NoError(t, err)
	}

	// Verify counts are separate
	count1, err := repo.CountUnreadByUserID(ctx, user1ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count1)

	count2, err := repo.CountUnreadByUserID(ctx, user2ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count2)

	// Mark all read for user1 should not affect user2
	err = repo.MarkAllAsRead(ctx, user1ID)
	require.NoError(t, err)

	count1, err = repo.CountUnreadByUserID(ctx, user1ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count1)

	count2, err = repo.CountUnreadByUserID(ctx, user2ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count2)
}
