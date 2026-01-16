# Task 002: Unify Task Repository Pattern

## Status: Pending

## Priority: Critical

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Refactor Task domain to use Repository pattern consistently with Chat domain, instead of direct EventStore access from use cases.

## Current State

### Problem: Direct EventStore Access in Use Cases

```go
// internal/application/task/create_task.go (current)
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
    // ...
    events := aggregate.UncommittedEvents()
    
    // Direct eventStore access - WRONG!
    if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
        return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
    }
    // No ReadModel update!
    // No EventBus publish!
}
```

### Contrast with Chat (correct pattern)

```go
// internal/application/chat/create_chat.go (reference)
func (uc *CreateChatUseCase) Execute(ctx context.Context, cmd CreateChatCommand) (Result, error) {
    // ...
    // Repository handles EventStore + ReadModel
    if err = uc.chatRepo.Save(ctx, chatAggregate); err != nil {
        return Result{}, fmt.Errorf("failed to save chat: %w", err)
    }
}
```

## Required Changes

### 1. Update CreateTaskUseCase

**File**: `internal/application/task/create_task.go`

Change from:
- Inject `appcore.EventStore` directly
- Call `eventStore.SaveEvents()` in use case

Change to:
- Inject `CommandRepository` (like Chat does)
- Call `taskRepo.Save()` which handles everything

### 2. Verify MongoTaskRepository.Save()

**File**: `internal/infrastructure/repository/mongodb/task_repository.go`

Ensure `Save()` method:
- [x] Saves events to EventStore
- [x] Updates ReadModel
- [ ] Publishes to EventBus (Task 003)

### 3. Update Other Task Use Cases

Review and update all task use cases that might use direct EventStore access:
- `change_status.go`
- `assign_task.go`
- `change_priority.go`
- `set_due_date.go`

## Acceptance Criteria

- [ ] `CreateTaskUseCase` uses `taskRepo.Save()` instead of `eventStore.SaveEvents()`
- [ ] All task use cases inject `CommandRepository`, not `EventStore`
- [ ] Task ReadModel is updated on every save
- [ ] Unit tests pass
- [ ] Integration tests pass

## Files to Modify

| File | Change |
|------|--------|
| `internal/application/task/create_task.go` | Replace EventStore with Repository |
| `internal/application/task/change_status.go` | Verify uses Repository |
| `internal/application/task/assign_task.go` | Verify uses Repository |
| `internal/application/task/change_priority.go` | Verify uses Repository |
| `internal/application/task/set_due_date.go` | Verify uses Repository |
| `internal/application/task/interfaces.go` | Verify CommandRepository interface |

## Testing

```bash
# Run task-related tests
go test ./internal/application/task/... -v
go test ./internal/infrastructure/repository/mongodb/task_repository_test.go -v
go test ./tests/integration/... -tags=integration -run Task -v
```
