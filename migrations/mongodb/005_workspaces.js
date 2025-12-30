// Migration: Create Workspaces and Workspace Members collections
// This migration sets up collections for workspace management

// Create workspaces collection
db.createCollection("workspaces");

// Unique index on workspace_id
db.workspaces.createIndex(
    { workspace_id: 1 },
    { unique: true, name: "workspace_id_unique" }
);

// Unique index on keycloak_group_id (sparse because not all workspaces are Keycloak-synced)
db.workspaces.createIndex(
    { keycloak_group_id: 1 },
    { unique: true, sparse: true, name: "keycloak_group_id_unique" }
);

// Index for finding workspaces by creator
db.workspaces.createIndex(
    { created_by: 1 },
    { name: "created_by" }
);

// Index for sorting by creation time
db.workspaces.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index for sorting by update time
db.workspaces.createIndex(
    { updated_at: -1 },
    { name: "updated_at_desc" }
);

// Create workspace_members collection
db.createCollection("workspace_members");

// Composite unique index: workspace_id + user_id (prevents duplicate membership)
db.workspace_members.createIndex(
    { workspace_id: 1, user_id: 1 },
    { unique: true, name: "workspace_user_unique" }
);

// Index for finding all members in a workspace
db.workspace_members.createIndex(
    { workspace_id: 1 },
    { name: "workspace_id" }
);

// Index for finding all workspaces a user belongs to
db.workspace_members.createIndex(
    { user_id: 1 },
    { name: "user_id" }
);

// Index for finding members by role
db.workspace_members.createIndex(
    { workspace_id: 1, role: 1 },
    { name: "workspace_role" }
);

// Index for finding workspace admins
db.workspace_members.createIndex(
    { workspace_id: 1, role: 1 },
    { name: "workspace_admin" }
);

// Index for sorting members by join date
db.workspace_members.createIndex(
    { workspace_id: 1, joined_at: -1 },
    { name: "workspace_joined_desc" }
);

print("✓ Workspaces collection created successfully");
print("✓ Workspace Members collection created successfully");
print("✓ Indexes created:");
print("  - workspace_id_unique");
print("  - keycloak_group_id_unique");
print("  - created_by");
print("  - created_at_desc");
print("  - updated_at_desc");
print("  - workspace_user_unique");
print("  - workspace_id");
print("  - user_id");
print("  - workspace_role");
print("  - workspace_admin");
print("  - workspace_joined_desc");
