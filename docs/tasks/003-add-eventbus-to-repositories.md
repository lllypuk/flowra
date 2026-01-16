# Task 003: Add EventBus Publishing to Repositories

## Status: Pending

## Priority: Critical

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Add EventBus publishing to Chat and Task repositories so that event handlers (notifications, WebSocket broadcasts, logging) receive all domain events.

## Current State

### Chat Repository - No EventBus

```go
// internal/infrastructure/repository/mongodb/chat_repository.go
func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
    // 1. Save events to event store ✅
    err := r.eventStore.SaveEvents(ctx, chat.ID().String(), uncommittedEvents, expectedVersion)
    
    // 2. Update read model ✅
    err = r.updateReadModel(ctx, chat)
    
    // 3. Publish to EventBus ❌ MISSING!
    
    chat.MarkEventsAsCommitted()
    return nil
}
```

### Task Repository - Same Issue

```go
// internal/infrastructure/repository/mongodb/task_repository.go
func (r *MongoTaskRepository) Save(ctx context.Context, task *taskdomain.Aggregate) error {
    // 1. Save events ✅
    // 2. Update read model ✅
    // 3. EventBus publish ❌ MISSING!
}
```

### Message Repository - Has EventBus (in use case)

```go
// internal/application/message/send_message.go
// Message publishes events, but from use case, not repository
_ = uc.eventBus.Publish(ctx, evt)
```

## Required Changes

### 1. Add EventBus to MongoChatRepository

**File**: `internal/infrastructure/repository/mongodb/chat_repository.go`

```go
type MongoChatRepository struct {
    eventStore    appcore.EventStore
    readModelColl *mongo.Collection
    eventBus      event.Bus  // ADD THIS
    logger        *slog.Logger
}

func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
    // ... existing code ...
    
    // 3. Publish events to EventBus
    for _, evt := range uncommittedEvents {
        if pubErr := r.eventBus.Publish(ctx, evt); pubErr != nil {
            r.logger.WarnContext(ctx, "failed to publish event to bus",
                slog.String("event_type", evt.EventType()),
                slog.String("error", pubErr.Error()),
            )
            // Don't fail - event is already persisted
        }
    }
    
    chat.MarkEventsAsCommitted()
    return nil
}
```

### 2. Add EventBus to MongoTaskRepository

**File**: `internal/infrastructure/repository/mongodb/task_repository.go`

Same pattern as Chat repository.

### 3. Update Repository Constructors

Add EventBus parameter to constructors:
- `NewMongoChatRepository(eventStore, readModelColl, eventBus, opts...)`
- `NewMongoTaskRepository(eventStore, readModelColl, eventBus, opts...)`

### 4. Update Dependency Injection

**File**: `cmd/api/container.go`

Pass EventBus when creating repositories.

## Event Types to Publish

### Chat Events
- `chat.created`
- `chat.participant_added`
- `chat.participant_removed`
- `chat.type_changed`
- `chat.status_changed`
- `chat.user_assigned`
- `chat.renamed`
- `chat.deleted`

### Task Events
- `task.created`
- `task.status_changed`
- `task.assignee_changed`
- `task.priority_changed`
- `task.due_date_changed`

## Acceptance Criteria

- [ ] Chat repository publishes all events to EventBus
- [ ] Task repository publishes all events to EventBus
- [ ] NotificationHandler receives Chat events
- [ ] NotificationHandler receives Task events
- [ ] LoggingHandler logs all events
- [ ] No event loss (events persisted before publish)

## Files to Modify

| File | Change |
|------|--------|
| `internal/infrastructure/repository/mongodb/chat_repository.go` | Add eventBus field and publish logic |
| `internal/infrastructure/repository/mongodb/task_repository.go` | Add eventBus field and publish logic |
| `cmd/api/container.go` | Pass eventBus to repository constructors |

## Testing

```bash
# Verify handlers receive events
go test ./internal/infrastructure/eventbus/handlers_test.go -v

# Integration test for event flow
go test ./tests/integration/... -tags=integration -run "Event" -v
```

## Notes

- Events are published **after** successful persistence (at-least-once semantics)
- Failed publishes are logged but don't fail the operation
- For guaranteed delivery, see [Task 004: Implement Outbox Pattern](./004-implement-outbox-pattern.md)
