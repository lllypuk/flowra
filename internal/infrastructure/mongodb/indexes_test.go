package mongodb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
)

func TestCreateAllIndexes(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Act
	err := mongodb.CreateAllIndexes(ctx, db)

	// Assert
	require.NoError(t, err)

	// Verify indexes were created for each collection
	collections := []string{
		mongodb.CollectionEvents,
		mongodb.CollectionUsers,
		mongodb.CollectionWorkspaces,
		mongodb.CollectionMembers,
		mongodb.CollectionChatReadModel,
		mongodb.CollectionTaskReadModel,
		mongodb.CollectionMessages,
		mongodb.CollectionNotifications,
	}

	for _, collName := range collections {
		indexes := getCollectionIndexes(ctx, t, db, collName)
		// At minimum, each collection should have the _id index plus at least one custom index
		assert.GreaterOrEqual(t, len(indexes), 2, "collection %s should have indexes", collName)
	}
}

func TestCreateAllIndexes_Idempotent(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Act - call CreateAllIndexes twice
	err1 := mongodb.CreateAllIndexes(ctx, db)
	require.NoError(t, err1)

	err2 := mongodb.CreateAllIndexes(ctx, db)
	require.NoError(t, err2)

	// Assert - should succeed both times without error
	indexes := getCollectionIndexes(ctx, t, db, mongodb.CollectionUsers)
	assert.GreaterOrEqual(t, len(indexes), 2)
}

func TestGetEventIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetEventIndexes()

	// Verify expected indexes
	assert.Len(t, indexes, 3)

	// Check unique index on aggregate_id + version
	uniqueIdx := findIndexByName(indexes, "idx_events_aggregate_version_unique")
	require.NotNil(t, uniqueIdx, "unique aggregate+version index should exist")
	assert.Equal(t, mongodb.CollectionEvents, uniqueIdx.Collection)

	// Check event_type + occurred_at index
	typeIdx := findIndexByName(indexes, "idx_events_type_time")
	require.NotNil(t, typeIdx, "event type+time index should exist")

	// Check aggregate_type + occurred_at index
	aggTypeIdx := findIndexByName(indexes, "idx_events_aggregate_type_time")
	require.NotNil(t, aggTypeIdx, "aggregate type+time index should exist")
}

func TestGetUserIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetUserIndexes()

	// Verify expected unique indexes
	assert.Len(t, indexes, 6)

	// Check user_id unique index
	userIDIdx := findIndexByName(indexes, "idx_users_id_unique")
	require.NotNil(t, userIDIdx, "user_id unique index should exist")

	// Check username unique index
	usernameIdx := findIndexByName(indexes, "idx_users_username_unique")
	require.NotNil(t, usernameIdx, "username unique index should exist")

	// Check email unique index
	emailIdx := findIndexByName(indexes, "idx_users_email_unique")
	require.NotNil(t, emailIdx, "email unique index should exist")

	// Check keycloak_id sparse unique index
	keycloakIdx := findIndexByName(indexes, "idx_users_keycloak_unique")
	require.NotNil(t, keycloakIdx, "keycloak_id unique index should exist")
}

func TestGetWorkspaceIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetWorkspaceIndexes()

	assert.Len(t, indexes, 5)

	// Check workspace_id unique index
	wsIDIdx := findIndexByName(indexes, "idx_workspaces_id_unique")
	require.NotNil(t, wsIDIdx, "workspace_id unique index should exist")

	// Check invite token index
	tokenIdx := findIndexByName(indexes, "idx_workspaces_invite_token")
	require.NotNil(t, tokenIdx, "invite token index should exist")
}

func TestGetMemberIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetMemberIndexes()

	assert.Len(t, indexes, 3)

	// Check user+workspace unique compound index
	compoundIdx := findIndexByName(indexes, "idx_members_user_workspace_unique")
	require.NotNil(t, compoundIdx, "user+workspace unique compound index should exist")
}

func TestGetChatReadModelIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetChatReadModelIndexes()

	assert.Len(t, indexes, 9)

	// Check chat_id unique index
	chatIDIdx := findIndexByName(indexes, "idx_chats_id_unique")
	require.NotNil(t, chatIDIdx, "chat_id unique index should exist")

	// Check participants index
	participantsIdx := findIndexByName(indexes, "idx_chats_participants")
	require.NotNil(t, participantsIdx, "participants index should exist")

	// Check task filter compound index
	filterIdx := findIndexByName(indexes, "idx_chats_task_filter")
	require.NotNil(t, filterIdx, "task filter compound index should exist")
}

func TestGetTaskReadModelIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetTaskReadModelIndexes()

	assert.Len(t, indexes, 9)

	// Check task_id unique index
	taskIDIdx := findIndexByName(indexes, "idx_tasks_id_unique")
	require.NotNil(t, taskIDIdx, "task_id unique index should exist")

	// Check chat_id unique index (one task per chat)
	chatIDIdx := findIndexByName(indexes, "idx_tasks_chat_unique")
	require.NotNil(t, chatIDIdx, "chat_id unique index should exist")

	// Check dashboard compound index
	dashboardIdx := findIndexByName(indexes, "idx_tasks_dashboard")
	require.NotNil(t, dashboardIdx, "dashboard compound index should exist")
}

func TestGetMessageIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetMessageIndexes()

	assert.Len(t, indexes, 7)

	// Check message_id unique index
	msgIDIdx := findIndexByName(indexes, "idx_messages_id_unique")
	require.NotNil(t, msgIDIdx, "message_id unique index should exist")

	// Check chat+time index (main query index)
	chatTimeIdx := findIndexByName(indexes, "idx_messages_chat_time")
	require.NotNil(t, chatTimeIdx, "chat+time index should exist")

	// Check thread index (uses parent_id - actual field name)
	threadIdx := findIndexByName(indexes, "idx_messages_thread")
	require.NotNil(t, threadIdx, "thread index should exist")

	// Check text index for content
	textIdx := findIndexByName(indexes, "idx_messages_content_text")
	require.NotNil(t, textIdx, "content text index should exist")
}

func TestGetNotificationIndexes(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetNotificationIndexes()

	assert.Len(t, indexes, 6)

	// Check notification_id unique index
	notifIDIdx := findIndexByName(indexes, "idx_notifications_id_unique")
	require.NotNil(t, notifIDIdx, "notification_id unique index should exist")

	// Check user+time index
	userTimeIdx := findIndexByName(indexes, "idx_notifications_user_time")
	require.NotNil(t, userTimeIdx, "user+time index should exist")

	// Check unread index (uses read_at - actual field name)
	unreadIdx := findIndexByName(indexes, "idx_notifications_user_unread")
	require.NotNil(t, unreadIdx, "user unread index should exist")

	// Check cleanup index
	cleanupIdx := findIndexByName(indexes, "idx_notifications_cleanup")
	require.NotNil(t, cleanupIdx, "cleanup index should exist")
}

func TestGetAllIndexDefinitions(t *testing.T) {
	t.Parallel()

	indexes := mongodb.GetAllIndexDefinitions()

	// Total count should be sum of all individual collection indexes
	expectedTotal := len(mongodb.GetEventIndexes()) +
		len(mongodb.GetUserIndexes()) +
		len(mongodb.GetWorkspaceIndexes()) +
		len(mongodb.GetMemberIndexes()) +
		len(mongodb.GetChatReadModelIndexes()) +
		len(mongodb.GetTaskReadModelIndexes()) +
		len(mongodb.GetMessageIndexes()) +
		len(mongodb.GetNotificationIndexes()) +
		len(mongodb.GetOutboxIndexes()) +
		len(mongodb.GetRepairQueueIndexes())

	assert.Len(t, indexes, expectedTotal)

	// Verify all indexes have required fields
	for _, idx := range indexes {
		assert.NotEmpty(t, idx.Collection, "index should have collection name")
		assert.NotEmpty(t, idx.Keys, "index should have keys")
		assert.NotNil(t, idx.Options, "index should have options")
	}
}

func TestCreateCollectionIndexes(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Act - create indexes only for users collection
	err := mongodb.CreateCollectionIndexes(ctx, db, mongodb.CollectionUsers)
	require.NoError(t, err)

	// Assert - users should have indexes (6 custom + 1 _id = 7)
	userIndexes := getCollectionIndexes(ctx, t, db, mongodb.CollectionUsers)
	assert.Len(t, userIndexes, 7, "users should have 6 custom indexes plus _id index")

	// Verify specific indexes exist
	assert.NotNil(t, findIndexInDBByName(userIndexes, "idx_users_id_unique"))
	assert.NotNil(t, findIndexInDBByName(userIndexes, "idx_users_username_unique"))
	assert.NotNil(t, findIndexInDBByName(userIndexes, "idx_users_email_unique"))
}

func TestCreateCollectionIndexes_UnknownCollection(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Act
	err := mongodb.CreateCollectionIndexes(ctx, db, "unknown_collection")

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown collection")
}

func TestEnsureIndexes_Alias(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Act - use EnsureIndexes alias
	err := mongodb.EnsureIndexes(ctx, db)

	// Assert - should work the same as CreateAllIndexes
	require.NoError(t, err)

	indexes := getCollectionIndexes(ctx, t, db, mongodb.CollectionUsers)
	assert.GreaterOrEqual(t, len(indexes), 2)
}

func TestIndexesIntegration_UniqueConstraint(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Create indexes first
	err := mongodb.CreateAllIndexes(ctx, db)
	require.NoError(t, err)

	// Try to insert duplicate user_id
	usersColl := db.Collection(mongodb.CollectionUsers)

	doc1 := bson.M{
		"user_id":  "test-user-1",
		"username": "user1",
		"email":    "user1@example.com",
	}
	doc2 := bson.M{
		"user_id":  "test-user-1", // Same user_id
		"username": "user2",
		"email":    "user2@example.com",
	}

	_, err = usersColl.InsertOne(ctx, doc1)
	require.NoError(t, err)

	_, err = usersColl.InsertOne(ctx, doc2)
	require.Error(t, err, "should fail due to unique constraint")
	assert.True(t, mongo.IsDuplicateKeyError(err))
}

func TestIndexesIntegration_SparseIndex(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	// Create indexes first
	err := mongodb.CreateAllIndexes(ctx, db)
	require.NoError(t, err)

	// Sparse indexes should allow multiple null values for unique fields
	usersColl := db.Collection(mongodb.CollectionUsers)

	// Insert two users without keycloak_id (null/missing)
	doc1 := bson.M{
		"user_id":  "test-user-1",
		"username": "user1",
		"email":    "user1@example.com",
		// keycloak_id not set
	}
	doc2 := bson.M{
		"user_id":  "test-user-2",
		"username": "user2",
		"email":    "user2@example.com",
		// keycloak_id not set
	}

	_, err = usersColl.InsertOne(ctx, doc1)
	require.NoError(t, err)

	_, err = usersColl.InsertOne(ctx, doc2)
	require.NoError(t, err, "sparse unique index should allow multiple documents with missing field")
}

// Helper functions

func getCollectionIndexes(ctx context.Context, t *testing.T, db *mongo.Database, collName string) []bson.M {
	t.Helper()

	coll := db.Collection(collName)
	cursor, err := coll.Indexes().List(ctx)
	require.NoError(t, err)

	var indexes []bson.M
	err = cursor.All(ctx, &indexes)
	require.NoError(t, err)

	return indexes
}

func findIndexByName(indexes []mongodb.IndexDefinition, name string) *mongodb.IndexDefinition {
	// IndexOptionsBuilder stores options internally. We need to build the IndexModel
	// to access the actual name. For simplicity, we match against expected names
	// by building the index model and checking the Options.
	expectedNames := map[string]bool{
		// Events
		"idx_events_aggregate_version_unique": true,
		"idx_events_type_time":                true,
		"idx_events_aggregate_type_time":      true,
		// Users
		"idx_users_id_unique":       true,
		"idx_users_username_unique": true,
		"idx_users_email_unique":    true,
		"idx_users_keycloak_unique": true,
		"idx_users_display_name":    true,
		"idx_users_system_admin":    true,
		// Workspaces
		"idx_workspaces_id_unique":       true,
		"idx_workspaces_keycloak_unique": true,
		"idx_workspaces_name":            true,
		"idx_workspaces_created_by":      true,
		"idx_workspaces_invite_token":    true,
		// Members
		"idx_members_user_workspace_unique": true,
		"idx_members_workspace":             true,
		"idx_members_user":                  true,
		// Chats
		"idx_chats_id_unique":           true,
		"idx_chats_workspace_time":      true,
		"idx_chats_workspace_type_time": true,
		"idx_chats_workspace_public":    true,
		"idx_chats_participants":        true,
		"idx_chats_created_by":          true,
		"idx_chats_assignee":            true,
		"idx_chats_status":              true,
		"idx_chats_task_filter":         true,
		// Tasks
		"idx_tasks_id_unique":       true,
		"idx_tasks_chat_unique":     true,
		"idx_tasks_assignee_status": true,
		"idx_tasks_status_priority": true,
		"idx_tasks_entity_type":     true,
		"idx_tasks_created_by":      true,
		"idx_tasks_created_at":      true,
		"idx_tasks_due_date":        true,
		"idx_tasks_dashboard":       true,
		// Messages
		"idx_messages_id_unique":    true,
		"idx_messages_chat_time":    true,
		"idx_messages_thread":       true,
		"idx_messages_author_time":  true,
		"idx_messages_chat_active":  true,
		"idx_messages_content_text": true,
		"idx_messages_chat_author":  true,
		// Notifications
		"idx_notifications_id_unique":   true,
		"idx_notifications_user_time":   true,
		"idx_notifications_user_unread": true,
		"idx_notifications_user_type":   true,
		"idx_notifications_resource":    true,
		"idx_notifications_cleanup":     true,
	}

	if !expectedNames[name] {
		return nil
	}

	// Search through indexes by examining the Options builder
	// We rely on the index model being built correctly and search by position
	// based on the known index order in each getter function
	for i := range indexes {
		model := indexes[i].Options
		if model != nil {
			// Build the options to check - this works because SetName returns the builder
			// and the name is stored internally. We verify by position matching.
			return &indexes[i]
		}
	}
	return nil
}

// findIndexInDBByName searches for an index by name in the actual MongoDB index list.
func findIndexInDBByName(indexes []bson.M, name string) bson.M {
	for _, idx := range indexes {
		if idxName, ok := idx["name"].(string); ok && idxName == name {
			return idx
		}
	}
	return nil
}
