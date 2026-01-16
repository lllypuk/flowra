# Task 005: Add ReadModel Rebuild Capability

## Status: Complete

## Priority: High

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Add the ability to rebuild ReadModels from EventStore events. Currently, if ReadModel update fails, there's no recovery mechanism despite comments claiming "read model can be recalculated".

## Problem Statement

### Current Code (Silent Failure)

```go
// internal/infrastructure/repository/mongodb/chat_repository.go:133-140
err = r.updateReadModel(ctx, chat)
if err != nil {
    r.logger.ErrorContext(ctx, "failed to update chat read model"...)
    // Don't fail - read model can be recalculated  ← LIE! No such mechanism exists
}
```

### Issues

1. **No rebuild mechanism**: If ReadModel update fails, data is lost forever
2. **No consistency check**: No way to verify ReadModel matches EventStore
3. **No migration support**: Cannot update ReadModel schema without downtime
4. **Silent desync**: Users see stale/incorrect data without any indication

## Design

### Projector Interface

```go
// internal/application/appcore/projector.go
type ReadModelProjector interface {
    // RebuildOne rebuilds ReadModel for a single aggregate
    RebuildOne(ctx context.Context, aggregateID uuid.UUID) error
    
    // RebuildAll rebuilds all ReadModels (for migrations)
    RebuildAll(ctx context.Context) error
    
    // ProcessEvent applies a single event to ReadModel
    ProcessEvent(ctx context.Context, event event.DomainEvent) error
    
    // VerifyConsistency checks if ReadModel matches EventStore
    VerifyConsistency(ctx context.Context, aggregateID uuid.UUID) (bool, error)
}
```

### Chat Projector Implementation

```go
// internal/infrastructure/projector/chat_projector.go
type ChatProjector struct {
    eventStore    appcore.EventStore
    readModelColl *mongo.Collection
    logger        *slog.Logger
}

func (p *ChatProjector) RebuildOne(ctx context.Context, chatID uuid.UUID) error {
    // 1. Load all events from EventStore
    events, err := p.eventStore.LoadEvents(ctx, chatID.String())
    if err != nil {
        return fmt.Errorf("failed to load events: %w", err)
    }
    
    // 2. Reconstruct aggregate from events
    chat := &chatdomain.Chat{}
    for _, evt := range events {
        if err := chat.Apply(evt); err != nil {
            return fmt.Errorf("failed to apply event: %w", err)
        }
    }
    
    // 3. Update ReadModel with full state
    return p.updateReadModel(ctx, chat)
}

func (p *ChatProjector) RebuildAll(ctx context.Context) error {
    // Get all unique aggregate IDs from events collection
    aggregateIDs, err := p.getAllAggregateIDs(ctx)
    if err != nil {
        return err
    }
    
    for _, id := range aggregateIDs {
        if err := p.RebuildOne(ctx, id); err != nil {
            p.logger.Error("failed to rebuild", "aggregate_id", id, "error", err)
            // Continue with others
        }
    }
    
    return nil
}
```

### Repair Queue for Failed Updates

```go
// internal/infrastructure/repair/repair_queue.go
type RepairTask struct {
    AggregateID   string    `bson:"aggregate_id"`
    AggregateType string    `bson:"aggregate_type"`
    TaskType      string    `bson:"task_type"` // "readmodel_sync"
    Error         string    `bson:"error"`
    CreatedAt     time.Time `bson:"created_at"`
    RetryCount    int       `bson:"retry_count"`
}

type RepairQueue interface {
    Add(ctx context.Context, task RepairTask) error
    Poll(ctx context.Context, batchSize int) ([]RepairTask, error)
    MarkCompleted(ctx context.Context, taskID string) error
    MarkFailed(ctx context.Context, taskID string, err error) error
}
```

### Repository Update

```go
// internal/infrastructure/repository/mongodb/chat_repository.go
func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
    // ... save events ...
    
    // Update ReadModel with repair fallback
    if err := r.updateReadModel(ctx, chat); err != nil {
        r.logger.ErrorContext(ctx, "failed to update read model, queuing repair",
            slog.String("chat_id", chat.ID().String()),
            slog.String("error", err.Error()),
        )
        // Queue for repair instead of silent failure
        r.repairQueue.Add(ctx, RepairTask{
            AggregateID:   chat.ID().String(),
            AggregateType: "chat",
            TaskType:      "readmodel_sync",
            Error:         err.Error(),
        })
    }
    
    // ... publish events ...
}
```

## Implementation Steps

### Phase 1: Projector Infrastructure

1. Create `internal/application/appcore/projector.go` interface
2. Create `internal/infrastructure/projector/chat_projector.go`
3. Create `internal/infrastructure/projector/task_projector.go`
4. Write unit tests

### Phase 2: Repair Queue

1. Create `internal/infrastructure/repair/repair_queue.go`
2. Create MongoDB collection and indexes
3. Add repair worker to `cmd/worker/`

### Phase 3: Repository Integration

1. Add RepairQueue to repositories
2. Update Save() to queue repairs on failure
3. Add repair worker startup

### Phase 4: CLI Tools

1. Add `rebuild-readmodel` CLI command
2. Add `verify-consistency` CLI command
3. Add metrics for repair queue depth

## Files to Create

| File | Purpose |
|------|---------|
| `internal/application/appcore/projector.go` | Interface definition |
| `internal/infrastructure/projector/chat_projector.go` | Chat ReadModel projector |
| `internal/infrastructure/projector/task_projector.go` | Task ReadModel projector |
| `internal/infrastructure/repair/repair_queue.go` | Repair queue implementation |
| `internal/worker/repair_worker.go` | Repair worker process |
| `cmd/tools/rebuild_readmodel.go` | CLI tool for manual rebuild |

## Files to Modify

| File | Change |
|------|--------|
| `internal/infrastructure/repository/mongodb/chat_repository.go` | Add repair queue integration |
| `internal/infrastructure/repository/mongodb/task_repository.go` | Add repair queue integration |
| `internal/infrastructure/mongodb/indexes.go` | Add repair queue indexes |
| `cmd/worker/main.go` | Add repair worker |

## CLI Usage

```bash
# Rebuild single aggregate
./flowra-tools rebuild-readmodel --type chat --id <uuid>

# Rebuild all of a type
./flowra-tools rebuild-readmodel --type chat --all

# Verify consistency
./flowra-tools verify-consistency --type chat --id <uuid>

# Verify all and report discrepancies
./flowra-tools verify-consistency --type chat --all --report
```

## Acceptance Criteria

- [x] Projector can rebuild ReadModel from EventStore events
- [x] RebuildAll can process all aggregates of a type
- [x] Repair queue captures failed ReadModel updates
- [x] Repair worker processes queue and rebuilds ReadModels
- [x] CLI tools available for manual operations
- [x] Unit tests for projector components

## Testing

```bash
# Test projector rebuild
go test ./internal/infrastructure/projector/... -v

# Integration test: create aggregate, corrupt readmodel, rebuild
go test ./tests/integration/... -tags=integration -run "Rebuild" -v
```

## Notes

- RebuildAll should be run during off-peak hours
- Consider rate limiting to avoid overwhelming the database
- Repair worker should have exponential backoff for retries
- Keep repair queue bounded to prevent unbounded growth
