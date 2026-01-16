# Task 006: Add Consistency Health Checks

## Status: Pending

## Priority: Medium

## Parent Task: [001-event-architecture-overview](./001-event-architecture-overview.md)

## Summary

Add health checks and monitoring to detect desynchronization between EventStore and ReadModels, and between EventStore and EventBus delivery.

## Problem Statement

Currently there's no way to know if:
- ReadModel is out of sync with EventStore
- Events failed to publish to EventBus
- Outbox has growing backlog
- Repair queue has unprocessed items

Issues can go undetected for days until users report incorrect data.

## Design

### Health Check Interface

```go
// internal/application/appcore/healthcheck.go
type HealthChecker interface {
    // Check performs health check and returns status
    Check(ctx context.Context) HealthStatus
    
    // Name returns the name of this health checker
    Name() string
}

type HealthStatus struct {
    Healthy     bool              `json:"healthy"`
    Message     string            `json:"message,omitempty"`
    Details     map[string]any    `json:"details,omitempty"`
    CheckedAt   time.Time         `json:"checked_at"`
}
```

### Consistency Health Checks

#### 1. EventStore-ReadModel Sync Check

```go
// internal/infrastructure/healthcheck/readmodel_sync.go
type ReadModelSyncChecker struct {
    eventStore    appcore.EventStore
    chatReadModel *mongo.Collection
    taskReadModel *mongo.Collection
    sampleSize    int // How many aggregates to verify
}

func (c *ReadModelSyncChecker) Check(ctx context.Context) HealthStatus {
    // 1. Sample random aggregates from EventStore
    // 2. Compare EventStore version with ReadModel version
    // 3. Report any mismatches
    
    mismatches := c.findMismatches(ctx, c.sampleSize)
    
    return HealthStatus{
        Healthy: len(mismatches) == 0,
        Message: fmt.Sprintf("checked %d aggregates, %d mismatches", c.sampleSize, len(mismatches)),
        Details: map[string]any{
            "mismatches": mismatches,
            "sample_size": c.sampleSize,
        },
    }
}
```

#### 2. Outbox Backlog Check

```go
// internal/infrastructure/healthcheck/outbox_backlog.go
type OutboxBacklogChecker struct {
    outbox           appcore.Outbox
    warningThreshold int
    criticalThreshold int
}

func (c *OutboxBacklogChecker) Check(ctx context.Context) HealthStatus {
    count, oldest, err := c.outbox.Stats(ctx)
    if err != nil {
        return HealthStatus{Healthy: false, Message: err.Error()}
    }
    
    lag := time.Since(oldest)
    healthy := count < c.warningThreshold
    
    return HealthStatus{
        Healthy: healthy,
        Message: fmt.Sprintf("outbox backlog: %d events, oldest: %v ago", count, lag),
        Details: map[string]any{
            "backlog_count": count,
            "oldest_event_age": lag.String(),
            "warning_threshold": c.warningThreshold,
        },
    }
}
```

#### 3. Repair Queue Check

```go
// internal/infrastructure/healthcheck/repair_queue.go
type RepairQueueChecker struct {
    repairQueue appcore.RepairQueue
    threshold   int
}

func (c *RepairQueueChecker) Check(ctx context.Context) HealthStatus {
    pending, failed, err := c.repairQueue.Stats(ctx)
    if err != nil {
        return HealthStatus{Healthy: false, Message: err.Error()}
    }
    
    healthy := pending < c.threshold && failed == 0
    
    return HealthStatus{
        Healthy: healthy,
        Details: map[string]any{
            "pending_repairs": pending,
            "failed_repairs": failed,
        },
    }
}
```

#### 4. Dead Letter Queue Check

```go
// internal/infrastructure/healthcheck/dead_letter.go
type DeadLetterChecker struct {
    dlqHandler *eventbus.DeadLetterHandler
}

func (c *DeadLetterChecker) Check(ctx context.Context) HealthStatus {
    count, err := c.dlqHandler.QueueLength(ctx)
    if err != nil {
        return HealthStatus{Healthy: false, Message: err.Error()}
    }
    
    return HealthStatus{
        Healthy: count == 0,
        Message: fmt.Sprintf("dead letter queue: %d events", count),
        Details: map[string]any{
            "dead_letters": count,
        },
    }
}
```

### Health Endpoint

```go
// internal/handler/http/health_handler.go
func (h *HealthHandler) GetHealth(c echo.Context) error {
    checks := map[string]HealthStatus{
        "database":        h.dbChecker.Check(c.Request().Context()),
        "redis":          h.redisChecker.Check(c.Request().Context()),
        "readmodel_sync": h.syncChecker.Check(c.Request().Context()),
        "outbox":         h.outboxChecker.Check(c.Request().Context()),
        "repair_queue":   h.repairChecker.Check(c.Request().Context()),
        "dead_letters":   h.dlqChecker.Check(c.Request().Context()),
    }
    
    allHealthy := true
    for _, status := range checks {
        if !status.Healthy {
            allHealthy = false
            break
        }
    }
    
    statusCode := http.StatusOK
    if !allHealthy {
        statusCode = http.StatusServiceUnavailable
    }
    
    return c.JSON(statusCode, map[string]any{
        "healthy": allHealthy,
        "checks":  checks,
    })
}
```

### Metrics

```go
// internal/infrastructure/metrics/event_metrics.go
var (
    ReadModelSyncMismatches = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "flowra_readmodel_sync_mismatches",
            Help: "Number of ReadModel sync mismatches detected",
        },
        []string{"aggregate_type"},
    )
    
    OutboxBacklog = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "flowra_outbox_backlog_count",
            Help: "Number of events waiting in outbox",
        },
    )
    
    OutboxLagSeconds = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "flowra_outbox_lag_seconds",
            Help: "Age of oldest event in outbox",
        },
    )
    
    RepairQueuePending = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "flowra_repair_queue_pending",
            Help: "Number of pending repair tasks",
        },
    )
    
    DeadLetterCount = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "flowra_dead_letter_count",
            Help: "Number of events in dead letter queue",
        },
    )
)
```

## Files to Create

| File | Purpose |
|------|---------|
| `internal/application/appcore/healthcheck.go` | Interface definition |
| `internal/infrastructure/healthcheck/readmodel_sync.go` | Sync checker |
| `internal/infrastructure/healthcheck/outbox_backlog.go` | Outbox checker |
| `internal/infrastructure/healthcheck/repair_queue.go` | Repair queue checker |
| `internal/infrastructure/healthcheck/dead_letter.go` | DLQ checker |
| `internal/infrastructure/metrics/event_metrics.go` | Prometheus metrics |

## Files to Modify

| File | Change |
|------|--------|
| `internal/handler/http/health_handler.go` | Add consistency checks |
| `cmd/api/container.go` | Wire up health checkers |
| `cmd/api/routes.go` | Add detailed health endpoint |

## API Endpoints

```
GET /health              # Basic health (existing)
GET /health/detailed     # Detailed with all checks
GET /health/readmodel    # ReadModel sync status only
GET /health/outbox       # Outbox status only
```

## Example Response

```json
{
  "healthy": false,
  "checks": {
    "database": {
      "healthy": true,
      "message": "connected",
      "checked_at": "2026-01-16T10:00:00Z"
    },
    "readmodel_sync": {
      "healthy": false,
      "message": "checked 100 aggregates, 2 mismatches",
      "details": {
        "mismatches": [
          {"aggregate_id": "abc-123", "expected_version": 5, "actual_version": 3},
          {"aggregate_id": "def-456", "expected_version": 12, "actual_version": 10}
        ]
      }
    },
    "outbox": {
      "healthy": true,
      "message": "outbox backlog: 15 events, oldest: 2s ago",
      "details": {
        "backlog_count": 15,
        "oldest_event_age": "2s"
      }
    },
    "dead_letters": {
      "healthy": true,
      "message": "dead letter queue: 0 events"
    }
  }
}
```

## Alerting Recommendations

| Metric | Warning | Critical |
|--------|---------|----------|
| `readmodel_sync_mismatches` | > 0 | > 10 |
| `outbox_backlog_count` | > 100 | > 1000 |
| `outbox_lag_seconds` | > 10 | > 60 |
| `repair_queue_pending` | > 10 | > 100 |
| `dead_letter_count` | > 0 | > 10 |

## Acceptance Criteria

- [ ] Health endpoint shows consistency check results
- [ ] Prometheus metrics exported for all queues
- [ ] Sample-based ReadModel sync verification works
- [ ] Alerts can be configured based on metrics
- [ ] Detailed health endpoint returns JSON with all check details

## Testing

```bash
# Test health checks
go test ./internal/infrastructure/healthcheck/... -v

# Manual verification
curl http://localhost:8080/health/detailed | jq
```
