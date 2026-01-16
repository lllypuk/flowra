// Package mongodb provides MongoDB infrastructure components including index management.
package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection names as constants for consistency.
const (
	CollectionEvents        = "events"
	CollectionUsers         = "users"
	CollectionWorkspaces    = "workspaces"
	CollectionMembers       = "workspace_members"
	CollectionChatReadModel = "chat_read_model"
	CollectionTaskReadModel = "task_read_model"
	CollectionMessages      = "messages"
	CollectionNotifications = "notifications"
	CollectionOutbox        = "outbox"
	CollectionRepairQueue   = "repair_queue"
)

// IndexDefinition describes a MongoDB index to be created.
type IndexDefinition struct {
	Collection string
	Keys       bson.D
	Options    *options.IndexOptionsBuilder
}

// CreateAllIndexes creates all necessary indexes for the application.
// This function is idempotent - calling it multiple times is safe.
func CreateAllIndexes(ctx context.Context, db *mongo.Database) error {
	indexes := GetAllIndexDefinitions()

	for _, idx := range indexes {
		coll := db.Collection(idx.Collection)
		model := mongo.IndexModel{
			Keys:    idx.Keys,
			Options: idx.Options,
		}

		_, err := coll.Indexes().CreateOne(ctx, model)
		if err != nil {
			return fmt.Errorf("failed to create index %s on collection %s: %w",
				getIndexName(idx.Options), idx.Collection, err)
		}
	}

	return nil
}

// getIndexName extracts the index name from options for error messages.
func getIndexName(opts *options.IndexOptionsBuilder) string {
	if opts == nil {
		return "<unnamed>"
	}
	// Options builder doesn't expose the name directly,
	// so we return a placeholder
	return "<index>"
}

// GetAllIndexDefinitions returns all index definitions for all collections.
func GetAllIndexDefinitions() []IndexDefinition {
	var indexes []IndexDefinition

	indexes = append(indexes, GetEventIndexes()...)
	indexes = append(indexes, GetUserIndexes()...)
	indexes = append(indexes, GetWorkspaceIndexes()...)
	indexes = append(indexes, GetMemberIndexes()...)
	indexes = append(indexes, GetChatReadModelIndexes()...)
	indexes = append(indexes, GetTaskReadModelIndexes()...)
	indexes = append(indexes, GetMessageIndexes()...)
	indexes = append(indexes, GetNotificationIndexes()...)
	indexes = append(indexes, GetOutboxIndexes()...)
	indexes = append(indexes, GetRepairQueueIndexes()...)

	return indexes
}

// GetEventIndexes returns index definitions for the events collection (Event Store).
func GetEventIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Unique index for optimistic locking - prevents duplicate events for same aggregate+version
			Collection: CollectionEvents,
			Keys:       bson.D{{Key: "aggregate_id", Value: 1}, {Key: "version", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_events_aggregate_version_unique"),
		},
		{
			// Index for filtering events by type
			Collection: CollectionEvents,
			Keys:       bson.D{{Key: "event_type", Value: 1}, {Key: "occurred_at", Value: -1}},
			Options:    options.Index().SetName("idx_events_type_time"),
		},
		{
			// Index for filtering events by aggregate type
			Collection: CollectionEvents,
			Keys:       bson.D{{Key: "aggregate_type", Value: 1}, {Key: "occurred_at", Value: -1}},
			Options:    options.Index().SetName("idx_events_aggregate_type_time"),
		},
	}
}

// GetUserIndexes returns index definitions for the users collection.
func GetUserIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique user ID
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_users_id_unique"),
		},
		{
			// Unique index for username
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "username", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_users_username_unique"),
		},
		{
			// Unique index for email
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "email", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_users_email_unique"),
		},
		{
			// Unique sparse index for Keycloak ID (sparse because not all users have it)
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "keycloak_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetSparse(true).SetName("idx_users_keycloak_unique"),
		},
		{
			// Index for display name search
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "display_name", Value: 1}},
			Options:    options.Index().SetName("idx_users_display_name"),
		},
		{
			// Index for system admin filtering
			Collection: CollectionUsers,
			Keys:       bson.D{{Key: "is_system_admin", Value: 1}},
			Options:    options.Index().SetName("idx_users_system_admin"),
		},
	}
}

// GetWorkspaceIndexes returns index definitions for the workspaces collection.
func GetWorkspaceIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique workspace ID
			Collection: CollectionWorkspaces,
			Keys:       bson.D{{Key: "workspace_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_workspaces_id_unique"),
		},
		{
			// Unique sparse index for Keycloak group ID
			Collection: CollectionWorkspaces,
			Keys:       bson.D{{Key: "keycloak_group_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetSparse(true).SetName("idx_workspaces_keycloak_unique"),
		},
		{
			// Index for workspace name search
			Collection: CollectionWorkspaces,
			Keys:       bson.D{{Key: "name", Value: 1}},
			Options:    options.Index().SetName("idx_workspaces_name"),
		},
		{
			// Index for finding workspaces by creator
			Collection: CollectionWorkspaces,
			Keys:       bson.D{{Key: "created_by", Value: 1}},
			Options:    options.Index().SetName("idx_workspaces_created_by"),
		},
		{
			// Index for finding invites by token (embedded documents)
			Collection: CollectionWorkspaces,
			Keys:       bson.D{{Key: "invites.token", Value: 1}},
			Options:    options.Index().SetName("idx_workspaces_invite_token"),
		},
	}
}

// GetMemberIndexes returns index definitions for the workspace_members collection.
func GetMemberIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Unique compound index for user membership in workspace
			Collection: CollectionMembers,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "workspace_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_members_user_workspace_unique"),
		},
		{
			// Index for finding all members of a workspace
			Collection: CollectionMembers,
			Keys:       bson.D{{Key: "workspace_id", Value: 1}},
			Options:    options.Index().SetName("idx_members_workspace"),
		},
		{
			// Index for finding all workspaces of a user
			Collection: CollectionMembers,
			Keys:       bson.D{{Key: "user_id", Value: 1}},
			Options:    options.Index().SetName("idx_members_user"),
		},
	}
}

// GetChatReadModelIndexes returns index definitions for the chat_read_model collection.
func GetChatReadModelIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique chat ID
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "chat_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_chats_id_unique"),
		},
		{
			// Index for workspace chats with time ordering
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "workspace_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_chats_workspace_time"),
		},
		{
			// Index for filtering by workspace, type, and time
			Collection: CollectionChatReadModel,
			Keys: bson.D{
				{Key: "workspace_id", Value: 1},
				{Key: "type", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_chats_workspace_type_time"),
		},
		{
			// Index for public chats within workspace
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "workspace_id", Value: 1}, {Key: "is_public", Value: 1}},
			Options:    options.Index().SetName("idx_chats_workspace_public"),
		},
		{
			// Index for finding chats by participant (array field)
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "participants", Value: 1}},
			Options:    options.Index().SetName("idx_chats_participants"),
		},
		{
			// Index for finding chats by creator
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "created_by", Value: 1}},
			Options:    options.Index().SetName("idx_chats_created_by"),
		},
		{
			// Sparse index for assignee (only for task/bug chats)
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "assigned_to", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_chats_assignee"),
		},
		{
			// Sparse index for status (only for task/bug chats)
			Collection: CollectionChatReadModel,
			Keys:       bson.D{{Key: "status", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_chats_status"),
		},
		{
			// Compound index for task/ticket filtering
			Collection: CollectionChatReadModel,
			Keys: bson.D{
				{Key: "workspace_id", Value: 1},
				{Key: "type", Value: 1},
				{Key: "status", Value: 1},
				{Key: "assigned_to", Value: 1},
			},
			Options: options.Index().SetName("idx_chats_task_filter"),
		},
	}
}

// GetTaskReadModelIndexes returns index definitions for the task_read_model collection.
func GetTaskReadModelIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique task ID
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "task_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_tasks_id_unique"),
		},
		{
			// Unique index for chat_id (one task per chat)
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "chat_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_tasks_chat_unique"),
		},
		{
			// Index for assigned tasks by status (sparse for unassigned tasks)
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "assigned_to", Value: 1}, {Key: "status", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_tasks_assignee_status"),
		},
		{
			// Index for status and priority ordering
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "status", Value: 1}, {Key: "priority", Value: 1}},
			Options:    options.Index().SetName("idx_tasks_status_priority"),
		},
		{
			// Index for entity type filtering
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "entity_type", Value: 1}},
			Options:    options.Index().SetName("idx_tasks_entity_type"),
		},
		{
			// Index for creator
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "created_by", Value: 1}},
			Options:    options.Index().SetName("idx_tasks_created_by"),
		},
		{
			// Index for time-based sorting
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_tasks_created_at"),
		},
		{
			// Sparse index for due date (not all tasks have due dates)
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "due_date", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_tasks_due_date"),
		},
		{
			// Compound index for dashboard queries
			Collection: CollectionTaskReadModel,
			Keys:       bson.D{{Key: "assigned_to", Value: 1}, {Key: "status", Value: 1}, {Key: "due_date", Value: 1}},
			Options:    options.Index().SetName("idx_tasks_dashboard"),
		},
	}
}

// GetMessageIndexes returns index definitions for the messages collection.
// Note: Uses actual field names from messageDocument struct (sent_by, parent_id).
func GetMessageIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique message ID
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "message_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_messages_id_unique"),
		},
		{
			// Main index for loading chat messages (most common query)
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_messages_chat_time"),
		},
		{
			// Sparse index for thread replies (parent_id - actual field name)
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "parent_id", Value: 1}, {Key: "created_at", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_messages_thread"),
		},
		{
			// Index for messages by author (sent_by - actual field name)
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "sent_by", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_messages_author_time"),
		},
		{
			// Compound index for filtering non-deleted messages in chat
			Collection: CollectionMessages,
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "is_deleted", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_messages_chat_active"),
		},
		{
			// Text index for full-text search
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "content", Value: "text"}},
			Options:    options.Index().SetName("idx_messages_content_text").SetDefaultLanguage("russian"),
		},
		{
			// Compound index for author's messages in a chat
			Collection: CollectionMessages,
			Keys:       bson.D{{Key: "chat_id", Value: 1}, {Key: "sent_by", Value: 1}},
			Options:    options.Index().SetName("idx_messages_chat_author"),
		},
	}
}

// GetNotificationIndexes returns index definitions for the notifications collection.
// Note: Uses read_at (nullable timestamp) instead of is_read boolean.
func GetNotificationIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary key - unique notification ID
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "notification_id", Value: 1}},
			Options:    options.Index().SetUnique(true).SetName("idx_notifications_id_unique"),
		},
		{
			// Main index for loading user's notifications
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_notifications_user_time"),
		},
		{
			// Index for unread notifications (read_at is null for unread)
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "read_at", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_notifications_user_unread"),
		},
		{
			// Index for filtering by notification type
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_notifications_user_type"),
		},
		{
			// Sparse index for finding notifications by resource
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "resource_id", Value: 1}},
			Options:    options.Index().SetSparse(true).SetName("idx_notifications_resource"),
		},
		{
			// Index for cleanup operations (deleting old notifications)
			Collection: CollectionNotifications,
			Keys:       bson.D{{Key: "created_at", Value: 1}},
			Options:    options.Index().SetName("idx_notifications_cleanup"),
		},
	}
}

// GetOutboxIndexes returns index definitions for the outbox collection.
func GetOutboxIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary index for polling unprocessed entries ordered by time
			Collection: CollectionOutbox,
			Keys:       bson.D{{Key: "processed_at", Value: 1}, {Key: "created_at", Value: 1}},
			Options:    options.Index().SetName("idx_outbox_poll"),
		},
		{
			// Index for cleanup operations (deleting old processed entries)
			Collection: CollectionOutbox,
			Keys:       bson.D{{Key: "processed_at", Value: 1}},
			Options:    options.Index().SetName("idx_outbox_cleanup"),
		},
		{
			// Index for monitoring by event type
			Collection: CollectionOutbox,
			Keys:       bson.D{{Key: "event_type", Value: 1}, {Key: "created_at", Value: -1}},
			Options:    options.Index().SetName("idx_outbox_event_type"),
		},
		{
			// Index for filtering by aggregate
			Collection: CollectionOutbox,
			Keys:       bson.D{{Key: "aggregate_id", Value: 1}},
			Options:    options.Index().SetName("idx_outbox_aggregate"),
		},
	}
}

// GetRepairQueueIndexes returns index definitions for the repair queue collection.
func GetRepairQueueIndexes() []IndexDefinition {
	return []IndexDefinition{
		{
			// Primary index for polling pending tasks ordered by creation time
			Collection: CollectionRepairQueue,
			Keys:       bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}},
			Options:    options.Index().SetName("idx_repair_queue_poll"),
		},
		{
			// Index for finding tasks by aggregate
			Collection: CollectionRepairQueue,
			Keys:       bson.D{{Key: "aggregate_id", Value: 1}, {Key: "aggregate_type", Value: 1}},
			Options:    options.Index().SetName("idx_repair_queue_aggregate"),
		},
		{
			// Index for monitoring by task type
			Collection: CollectionRepairQueue,
			Keys:       bson.D{{Key: "task_type", Value: 1}, {Key: "status", Value: 1}},
			Options:    options.Index().SetName("idx_repair_queue_task_type"),
		},
		{
			// Index for retry tracking
			Collection: CollectionRepairQueue,
			Keys:       bson.D{{Key: "retry_count", Value: 1}, {Key: "status", Value: 1}},
			Options:    options.Index().SetName("idx_repair_queue_retry"),
		},
	}
}

// EnsureIndexes is an alias for CreateAllIndexes for semantic clarity.
// Use this when you want to ensure indexes exist without caring about creation.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	return CreateAllIndexes(ctx, db)
}

// CreateCollectionIndexes creates indexes for a specific collection only.
// Useful for targeted index creation or testing.
func CreateCollectionIndexes(ctx context.Context, db *mongo.Database, collectionName string) error {
	var indexes []IndexDefinition

	switch collectionName {
	case CollectionEvents:
		indexes = GetEventIndexes()
	case CollectionUsers:
		indexes = GetUserIndexes()
	case CollectionWorkspaces:
		indexes = GetWorkspaceIndexes()
	case CollectionMembers:
		indexes = GetMemberIndexes()
	case CollectionChatReadModel:
		indexes = GetChatReadModelIndexes()
	case CollectionTaskReadModel:
		indexes = GetTaskReadModelIndexes()
	case CollectionMessages:
		indexes = GetMessageIndexes()
	case CollectionNotifications:
		indexes = GetNotificationIndexes()
	case CollectionOutbox:
		indexes = GetOutboxIndexes()
	case CollectionRepairQueue:
		indexes = GetRepairQueueIndexes()
	default:
		return fmt.Errorf("unknown collection: %s", collectionName)
	}

	for _, idx := range indexes {
		coll := db.Collection(idx.Collection)
		model := mongo.IndexModel{
			Keys:    idx.Keys,
			Options: idx.Options,
		}

		_, err := coll.Indexes().CreateOne(ctx, model)
		if err != nil {
			return fmt.Errorf("failed to create index on %s: %w", idx.Collection, err)
		}
	}

	return nil
}
