# Task 004: Implement Outbox Pattern

## Status: Complete

## Priority: High

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Implement the Outbox Pattern to guarantee event delivery to EventBus. Currently, events can be lost if Redis publish fails after MongoDB save.

## Problem Statement

### Current Flow (Unreliable)

```
MongoDB Transaction              Redis (separate operation)
┌─────────────────┐              ┌─────────────────┐
│ SaveEvents()   │──── GAP ────→│ Publish()       │
│ ✅ Committed    │              │ ❌ May fail      │
└─────────────────┘              └─────────────────┘
```

If `Publish()` fails:
- Event is saved to EventStore ✅
- Event is never delivered to handlers ❌
- Notifications not sent, WebSocket not updated, etc.

### Target Flow (Reliable with Outbox)

```
MongoDB Transaction (atomic)
┌─────────────────────────────┐
│ 1. SaveEvents()             │
│ 2. Insert into Outbox       │
│ ✅ Both committed together   │
└─────────────────────────────┘
        │
        ▼
Outbox Worker (separate process)
┌─────────────────────────────┐
│ 1. Poll Outbox              │
│ 2. Publish to Redis         │
│ 3. Mark as processed        │
│ ✅ Retries on failure        │
└─────────────────────────────┘
```

## Design

### Outbox Collection Schema

```go
// OutboxEntry represents an event waiting to be published
type OutboxEntry struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    EventID       string             `bson:"event_id"`
    EventType     string             `bson:"event_type"`
    AggregateID   string             `bson:"aggregate_id"`
    AggregateType string             `bson:"aggregate_type"`
    Payload       bson.Raw           `bson:"payload"`
    CreatedAt     time.Time          `bson:"created_at"`
    ProcessedAt   *time.Time         `bson:"processed_at,omitempty"`
    RetryCount    int                `bson:"retry_count"`
    LastError     string             `bson:"last_error,omitempty"`
}
```

### Outbox Interface

```go
// internal/application/appcore/outbox.go
type Outbox interface {
    // Add inserts event into outbox (called within transaction)
    Add(ctx context.Context, event event.DomainEvent) error
    
    // Poll retrieves unprocessed events
    Poll(ctx context.Context, batchSize int) ([]OutboxEntry, error)
    
    // MarkProcessed marks event as successfully published
    MarkProcessed(ctx context.Context, entryID string) error
    
    // MarkFailed records failure for retry
    MarkFailed(ctx context.Context, entryID string, err error) error
}
```

### Outbox Worker

```go
// internal/worker/outbox_worker.go
type OutboxWorker struct {
    outbox   appcore.Outbox
    eventBus event.Bus
    logger   *slog.Logger
    
    pollInterval time.Duration
    batchSize    int
    maxRetries   int
}

func (w *OutboxWorker) Run(ctx context.Context) error {
    ticker := time.NewTicker(w.pollInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            w.processBatch(ctx)
        }
    }
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
    entries, err := w.outbox.Poll(ctx, w.batchSize)
    if err != nil {
        w.logger.Error("failed to poll outbox", "error", err)
        return
    }
    
    for _, entry := range entries {
        if err := w.publishEvent(ctx, entry); err != nil {
            w.outbox.MarkFailed(ctx, entry.ID, err)
        } else {
            w.outbox.MarkProcessed(ctx, entry.ID)
        }
    }
}
```

## Implementation Steps

### Phase 1: Outbox Infrastructure

1. Create `internal/infrastructure/outbox/mongo_outbox.go`
2. Create `internal/application/appcore/outbox.go` interface
3. Add indexes for outbox collection
4. Write unit tests

### Phase 2: Repository Integration

1. Add Outbox to Repository constructors
2. Modify `Save()` to write to Outbox within transaction
3. Remove direct EventBus.Publish() from repositories

### Phase 3: Worker Implementation

1. Create `internal/worker/outbox_worker.go`
2. Add worker to `cmd/worker/main.go`
3. Configure poll interval, batch size, retries

### Phase 4: Cleanup & Monitoring

1. Add cleanup job for old processed entries
2. Add metrics for outbox lag
3. Add health check for outbox backlog

## Files to Create

| File | Purpose |
|------|---------|
| `internal/application/appcore/outbox.go` | Interface definition |
| `internal/infrastructure/outbox/mongo_outbox.go` | MongoDB implementation |
| `internal/infrastructure/outbox/mongo_outbox_test.go` | Tests |
| `internal/worker/outbox_worker.go` | Worker process |
| `internal/infrastructure/mongodb/indexes.go` | Add outbox indexes |

## Files to Modify

| File | Change |
|------|--------|
| `internal/infrastructure/repository/mongodb/chat_repository.go` | Add outbox write in transaction |
| `internal/infrastructure/repository/mongodb/task_repository.go` | Add outbox write in transaction |
| `cmd/api/container.go` | Create and inject outbox |
| `cmd/worker/main.go` | Add outbox worker |

## Configuration

```yaml
# configs/config.yaml
outbox:
  poll_interval: 100ms
  batch_size: 100
  max_retries: 5
  cleanup_after: 7d
```

## Acceptance Criteria

- [x] Outbox collection created with proper indexes
- [x] Events written to outbox within same transaction as EventStore
- [x] Worker processes outbox and publishes to EventBus
- [x] Failed publishes are retried with backoff
- [x] Old processed entries are cleaned up
- [ ] Metrics available for monitoring outbox lag (deferred to separate task)

## Trade-offs

**Pros:**
- Guaranteed event delivery
- Events survive Redis failures
- Replay capability

**Cons:**
- Increased latency (poll interval)
- Additional storage
- More complex infrastructure

## References

- [Outbox Pattern - Microservices.io](https://microservices.io/patterns/data/transactional-outbox.html)
- [Reliable Messaging with MongoDB](https://www.mongodb.com/blog/post/transactions-background-part-1)
