// Migration: Create Notifications collection
// This migration sets up the notifications collection for user notifications

// Create notifications collection
db.createCollection("notifications");

// Unique index on notification_id
db.notifications.createIndex(
    { notification_id: 1 },
    { unique: true, name: "notification_id_unique" }
);

// Composite index for finding unread notifications for a user (most common query)
db.notifications.createIndex(
    { user_id: 1, read_at: 1, created_at: -1 },
    { name: "user_unread_created", sparse: true }
);

// Index for finding all notifications for a user
db.notifications.createIndex(
    { user_id: 1, created_at: -1 },
    { name: "user_created_desc" }
);

// Index for global recent notifications
db.notifications.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index for notification type queries
db.notifications.createIndex(
    { type: 1 },
    { name: "type" }
);

// Composite index for user notifications by type
db.notifications.createIndex(
    { user_id: 1, type: 1 },
    { name: "user_type" }
);

// Index for finding notifications related to a resource
db.notifications.createIndex(
    { resource_id: 1 },
    { name: "resource_id" }
);

// Index for cleanup of read notifications
db.notifications.createIndex(
    { read_at: 1 },
    { name: "read_at", sparse: true }
);

// Index for read_at ordering (for pagination of read notifications)
db.notifications.createIndex(
    { user_id: 1, read_at: -1 },
    { name: "user_read_desc", sparse: true }
);

print("✓ Notifications collection created successfully");
print("✓ Indexes created:");
print("  - notification_id_unique");
print("  - user_unread_created");
print("  - user_created_desc");
print("  - created_at_desc");
print("  - type");
print("  - user_type");
print("  - resource_id");
print("  - read_at");
print("  - user_read_desc");
