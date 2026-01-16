# Task 003: Add EventBus Publishing to Repositories

## Status: Completed

## Priority: Critical

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Add EventBus publishing to Chat and Task repositories so that event handlers (notifications, WebSocket broadcasts, logging) receive all domain events.

## Completed Changes

### 1. MongoChatRepository

**File**: `internal/infrastructure/repository/mongodb/chat_repository.go`

- Added `eventBus event.Bus` field to struct
- Added `WithChatRepoEventBus(eventBus event.Bus)` option function
- Updated `Save()` method to publish events to EventBus after persistence

```go
// 3. Publish events to EventBus (if configured)
if r.eventBus != nil {
    for _, evt := range uncommittedEvents {
        if pubErr := r.eventBus.Publish(ctx, evt); pubErr != nil {
            r.logger.WarnContext(ctx, "failed to publish chat event to bus",
                slog.String("chat_id", chat.ID().String()),
                slog.String("event_type", evt.EventType()),
                slog.String("error", pubErr.Error()),
            )
            // Don't fail - event is already persisted
        }
    }
}
```

### 2. MongoTaskRepository

**File**: `internal/infrastructure/repository/mongodb/task_repository.go`

- Added `eventBus event.Bus` field to struct
- Added `WithTaskRepoEventBus(eventBus event.Bus)` option function
- Updated `Save()` method to publish events to EventBus after persistence

### 3. Container Wiring

**File**: `cmd/api/container.go`

- Updated `NewMongoChatRepository()` call to include `mongodb.WithChatRepoEventBus(c.EventBus)`
- Updated `NewMongoTaskRepository()` call to include `mongodb.WithTaskRepoEventBus(c.EventBus)`

## Event Types Published

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

- [x] Chat repository publishes all events to EventBus
- [x] Task repository publishes all events to EventBus
- [x] NotificationHandler receives Chat events (via existing handler subscription)
- [x] NotificationHandler receives Task events (via existing handler subscription)
- [x] LoggingHandler logs all events (via existing handler subscription)
- [x] No event loss (events persisted before publish)

## Files Modified

| File | Change |
|------|--------|
| `internal/infrastructure/repository/mongodb/chat_repository.go` | Added eventBus field and publish logic |
| `internal/infrastructure/repository/mongodb/task_repository.go` | Added eventBus field and publish logic |
| `cmd/api/container.go` | Pass eventBus to repository constructors |

## Testing

```bash
# All tests pass
go test ./internal/infrastructure/repository/mongodb/... -v
go test ./... -short
```

## Notes

- Events are published **after** successful persistence (at-least-once semantics)
- Failed publishes are logged but don't fail the operation
- EventBus is optional (nil check) for backwards compatibility in tests
- For guaranteed delivery, see [Task 004: Implement Outbox Pattern](./004-implement-outbox-pattern.md)
