// Migration: Create Chat Read Model collection
// This migration sets up the denormalized read model for fast chat queries

// Create chat_read_model collection
db.createCollection("chat_read_model");

// Unique index on chat_id
db.chat_read_model.createIndex(
    { chat_id: 1 },
    { unique: true, name: "chat_id_unique" }
);

// Composite index for workspace queries
db.chat_read_model.createIndex(
    { workspace_id: 1, type: 1 },
    { name: "workspace_type" }
);

// Index for participant queries
db.chat_read_model.createIndex(
    { participants: 1 },
    { name: "participants" }
);

// Index for sorting by creation time
db.chat_read_model.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index for chat creation by workspace and time
db.chat_read_model.createIndex(
    { workspace_id: 1, created_at: -1 },
    { name: "workspace_created_desc" }
);

// Index for public chats
db.chat_read_model.createIndex(
    { is_public: 1, workspace_id: 1 },
    { name: "public_workspace" }
);

// Index for finding chats by creator
db.chat_read_model.createIndex(
    { created_by: 1 },
    { name: "created_by" }
);

// Index for task/bug status queries
db.chat_read_model.createIndex(
    { type: 1, status: 1 },
    { name: "type_status", sparse: true }
);

// Index for assigned tasks
db.chat_read_model.createIndex(
    { assigned_to: 1 },
    { name: "assigned_to", sparse: true }
);

// Index for due date queries
db.chat_read_model.createIndex(
    { due_date: 1 },
    { name: "due_date", sparse: true }
);

print("✓ Chat Read Model collection created successfully");
print("✓ Indexes created:");
print("  - chat_id_unique");
print("  - workspace_type");
print("  - participants");
print("  - created_at_desc");
print("  - workspace_created_desc");
print("  - public_workspace");
print("  - created_by");
print("  - type_status");
print("  - assigned_to");
print("  - due_date");
