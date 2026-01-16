# Task 002: Unify Task Repository Pattern

## Status: Completed

## Priority: Critical

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Refactor Task domain to use Repository pattern consistently with Chat domain, instead of direct EventStore access from use cases.

## Completed Changes

All task use cases have been updated to use `CommandRepository` instead of direct `EventStore` access:

### 1. Updated Use Cases

- `create_task.go` - Now uses `taskRepo.Save()` instead of `eventStore.SaveEvents()`
- `change_status.go` - Updated to use `CommandRepository`
- `assign_task.go` - Updated to use `CommandRepository`
- `change_priority.go` - Updated via `BaseExecutor` which now uses `CommandRepository`
- `set_due_date.go` - Updated via `BaseExecutor` which now uses `CommandRepository`
- `base_executor.go` - Refactored to use `CommandRepository`

### 2. Container Wiring

Updated `cmd/api/container.go`:
- `boardTaskCreatorAdapter` now receives `CommandRepository`
- `fullTaskServiceAdapter` now receives `CommandRepository`
- Removed manual read model update logic (repository handles this)

### 3. Test Infrastructure

- Created `tests/mocks/task_repository.go` - Mock implementation for unit tests
- Updated all task use case tests to use `MockTaskRepository`
- Mock properly handles:
  - Clearing uncommitted events after save (simulates real behavior)
  - Returning `errs.ErrNotFound` for missing tasks

## MongoTaskRepository.Save()

**File**: `internal/infrastructure/repository/mongodb/task_repository.go`

The repository's `Save()` method:
- [x] Saves events to EventStore
- [x] Updates ReadModel
- [ ] Publishes to EventBus (Task 003)

## Acceptance Criteria

- [x] `CreateTaskUseCase` uses `taskRepo.Save()` instead of `eventStore.SaveEvents()`
- [x] All task use cases inject `CommandRepository`, not `EventStore`
- [x] Task ReadModel is updated on every save
- [x] Unit tests pass
- [x] Integration tests pass (via short test suite)

## Files Modified

| File | Change |
|------|--------|
| `internal/application/task/create_task.go` | Replaced EventStore with CommandRepository |
| `internal/application/task/change_status.go` | Replaced EventStore with CommandRepository |
| `internal/application/task/assign_task.go` | Replaced EventStore with CommandRepository |
| `internal/application/task/change_priority.go` | Uses BaseExecutor with CommandRepository |
| `internal/application/task/set_due_date.go` | Uses BaseExecutor with CommandRepository |
| `internal/application/task/base_executor.go` | Replaced EventStore with CommandRepository |
| `cmd/api/container.go` | Updated wiring to use TaskRepo |
| `tests/mocks/task_repository.go` | Created mock for testing |
| `*_test.go` files | Updated to use MockTaskRepository |

## Testing

```bash
# All tests pass
go test ./internal/application/task/... -v
go test ./... -short
```
