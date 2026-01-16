# Task 001: Event Architecture Standardization

## Status: Pending

## Priority: Critical

## Summary

Standardize event handling patterns across all domains (Chat, Task, Message) to eliminate inconsistencies that cause bugs and unpredictable behavior.

## Problem Statement

Currently, the project uses **three incompatible patterns** for event handling:

| Domain | Pattern | Issues |
|--------|---------|--------|
| **Chat** | `chatRepo.Save()` → EventStore + ReadModel | No EventBus publish |
| **Message** | `messageRepo.Save()` → MongoDB directly + `eventBus.Publish()` | No event sourcing |
| **Task** | `eventStore.SaveEvents()` directly from use case | No ReadModel update, no EventBus |

This inconsistency leads to:
- Events not reaching handlers (Chat, Task don't publish to EventBus)
- ReadModel desynchronization with EventStore
- Silent failures in ReadModel updates
- Unpredictable notification delivery

## Root Causes

1. **Mixed persistence strategies**: Event Sourcing (Chat/Task) vs State-based (Message)
2. **ReadModel as side effect**: Synchronous update inside Save() with no recovery mechanism
3. **No Outbox Pattern**: Events published to Redis after MongoDB save without delivery guarantee
4. **No transactional boundary**: Gap between EventStore save, ReadModel update, and EventBus publish

## Solution Overview

Implement unified repository pattern where `Repository.Save()` handles:
1. EventStore persistence (in transaction)
2. ReadModel update (in same transaction or via outbox)
3. EventBus publish (via outbox for guaranteed delivery)

## Sub-tasks

- [ ] [Task 002](./002-unify-task-repository.md) - Unify Task Repository Pattern
- [ ] [Task 003](./003-add-eventbus-to-repositories.md) - Add EventBus Publishing to Repositories
- [ ] [Task 004](./004-implement-outbox-pattern.md) - Implement Outbox Pattern
- [ ] [Task 005](./005-add-readmodel-rebuild.md) - Add ReadModel Rebuild Capability
- [ ] [Task 006](./006-add-consistency-healthcheck.md) - Add Consistency Health Checks

## Success Criteria

- [ ] All domains use same Repository pattern
- [ ] All events are published to EventBus
- [ ] ReadModel failures are recoverable
- [ ] Event delivery is guaranteed via Outbox
- [ ] Health checks detect EventStore ↔ ReadModel desync

## References

- `internal/infrastructure/repository/mongodb/chat_repository.go` - Reference implementation
- `internal/infrastructure/repository/mongodb/task_repository.go` - Needs update
- `internal/application/task/create_task.go` - Direct eventStore usage to remove
- `internal/infrastructure/eventbus/` - EventBus implementation
