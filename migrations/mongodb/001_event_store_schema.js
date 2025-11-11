// Migration: Create Event Store schema for MongoDB
// This migration sets up the collections and indexes required for event sourcing

// Create events collection if it doesn't exist
db.createCollection("events");

// Unique index: (aggregate_id, version) - ensures no duplicate versions for same aggregate
db.events.createIndex(
    { aggregate_id: 1, version: 1 },
    { unique: true, name: "aggregate_version_unique" }
);

// Index: aggregate_id - for loading all events of an aggregate
db.events.createIndex(
    { aggregate_id: 1 },
    { name: "aggregate_id" }
);

// Index: aggregate_type + created_at - for querying aggregates by type
db.events.createIndex(
    { aggregate_type: 1, created_at: -1 },
    { name: "aggregate_type_created" }
);

// Index: correlation_id - for tracing related events
db.events.createIndex(
    { "metadata.correlation_id": 1 },
    { name: "correlation_id" }
);

// Index: created_at desc - for getting recent events
db.events.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);

// Index: occurred_at - for time-based queries
db.events.createIndex(
    { occurred_at: -1 },
    { name: "occurred_at_desc" }
);

// Index: event_type - for filtering by event type
db.events.createIndex(
    { event_type: 1 },
    { name: "event_type" }
);

// Collection schema validation (optional, for data integrity)
db.runCommand({
    collMod: "events",
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["aggregate_id", "aggregate_type", "event_type", "version", "data", "metadata", "occurred_at", "created_at"],
            properties: {
                _id: { bsonType: "objectId" },
                aggregate_id: { bsonType: "string", description: "Unique identifier of the aggregate" },
                aggregate_type: { bsonType: "string", description: "Type of the aggregate (Chat, Message, Task, etc.)" },
                event_type: { bsonType: "string", description: "Type of the domain event" },
                version: { bsonType: "int", description: "Version number of the event for optimistic locking" },
                data: { bsonType: "object", description: "Event-specific data" },
                metadata: {
                    bsonType: "object",
                    required: ["timestamp", "correlation_id"],
                    properties: {
                        timestamp: { bsonType: "date", description: "When the event occurred" },
                        user_id: { bsonType: "string", description: "User who triggered the event" },
                        correlation_id: { bsonType: "string", description: "Correlation ID for distributed tracing" },
                        causation_id: { bsonType: "string", description: "Event that caused this event" },
                        ip_address: { bsonType: "string", description: "IP address of the request" },
                        user_agent: { bsonType: "string", description: "User agent of the request" }
                    }
                },
                occurred_at: { bsonType: "date", description: "When the event occurred" },
                created_at: { bsonType: "date", description: "When the event was stored" }
            }
        }
    }
});

// Print confirmation
print("✓ Event Store schema created successfully");
print("✓ Indexes created:");
print("  - aggregate_version_unique (unique)");
print("  - aggregate_id");
print("  - aggregate_type_created");
print("  - correlation_id");
print("  - created_at_desc");
print("  - occurred_at_desc");
print("  - event_type");
