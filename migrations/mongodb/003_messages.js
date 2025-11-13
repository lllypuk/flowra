// Migration: Create Messages collection
// This migration sets up the messages collection for chat message storage

// Create messages collection
db.createCollection("messages");

// Unique index on message_id
db.messages.createIndex(
    { message_id: 1 },
    { unique: true, name: "message_id_unique" }
);

// Composite index for retrieving messages by chat (most common query)
db.messages.createIndex(
    { chat_id: 1, created_at: -1 },
    { name: "chat_created_desc" }
);

// Index for finding message threads
db.messages.createIndex(
    { parent_id: 1, created_at: 1 },
    { name: "parent_created", sparse: true }
);

// Index for finding messages by author
db.messages.createIndex(
    { sent_by: 1 },
    { name: "sent_by" }
);

// Index for finding messages by author in specific chat
db.messages.createIndex(
    { chat_id: 1, sent_by: 1 },
    { name: "chat_sent_by" }
);

// Index for time-based queries
db.messages.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index for finding edited messages
db.messages.createIndex(
    { edited_at: 1 },
    { name: "edited_at", sparse: true }
);

// Index for finding deleted messages
db.messages.createIndex(
    { is_deleted: 1, deleted_at: -1 },
    { name: "deleted_at_desc" }
);

// Index for message search (if needed in future)
db.messages.createIndex(
    { content: "text" },
    { name: "content_text", sparse: true }
);

print("✓ Messages collection created successfully");
print("✓ Indexes created:");
print("  - message_id_unique");
print("  - chat_created_desc");
print("  - parent_created");
print("  - sent_by");
print("  - chat_sent_by");
print("  - created_at_desc");
print("  - edited_at");
print("  - deleted_at_desc");
print("  - content_text");
