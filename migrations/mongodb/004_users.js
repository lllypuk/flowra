// Migration: Create Users collection
// This migration sets up the users collection for user profile storage

// Create users collection
db.createCollection("users");

// Unique index on user_id
db.users.createIndex(
    { user_id: 1 },
    { unique: true, name: "user_id_unique" }
);

// Unique index on username
db.users.createIndex(
    { username: 1 },
    { unique: true, name: "username_unique" }
);

// Unique index on email
db.users.createIndex(
    { email: 1 },
    { unique: true, name: "email_unique" }
);

// Unique index on keycloak_id (sparse because not all users come from Keycloak)
db.users.createIndex(
    { keycloak_id: 1 },
    { unique: true, sparse: true, name: "keycloak_id_unique" }
);

// Index for finding users by display name (case-insensitive)
db.users.createIndex(
    { username: 1 },
    { name: "username_search" }
);

// Index for system admin queries
db.users.createIndex(
    { is_system_admin: 1 },
    { name: "is_system_admin" }
);

// Index for sorting by creation time
db.users.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index for finding recently updated users
db.users.createIndex(
    { updated_at: -1 },
    { name: "updated_at_desc" }
);

print("✓ Users collection created successfully");
print("✓ Indexes created:");
print("  - user_id_unique");
print("  - username_unique");
print("  - email_unique");
print("  - keycloak_id_unique");
print("  - username_search");
print("  - is_system_admin");
print("  - created_at_desc");
print("  - updated_at_desc");
