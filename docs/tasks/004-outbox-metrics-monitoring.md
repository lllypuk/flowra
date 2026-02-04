# Task 004: Outbox Metrics and Monitoring

**Status**: Complete
**Priority**: Low
**Depends on**: None
**Created**: 2026-02-04
**Completed**: 2026-02-04
**Source**: Deferred from Outbox Pattern implementation

---

## Overview

The outbox pattern is fully implemented with MongoDB storage, Redis-based event publishing, and basic health checks. This task adds comprehensive observability through Prometheus metrics and a visualization dashboard to monitor outbox backlog, processing latency, and overall system health.

---

## Current Implementation

### Outbox Architecture

```
Event Flow:
  Domain Event → Repository.Save() → Outbox.AddBatch()
                                          ↓
                              MongoDB "outbox" collection
                                          ↓
                        OutboxWorker.Poll() (every 100ms)
                                          ↓
                              Publish to Redis EventBus
                                          ↓
                         Outbox.MarkProcessed() / MarkFailed()
```

### Key Files

| File | Description |
|------|-------------|
| `internal/application/appcore/outbox.go:11-54` | Outbox interface and OutboxEntry struct |
| `internal/infrastructure/outbox/mongo_outbox.go` | MongoDB implementation |
| `internal/worker/outbox_worker.go` | Worker that polls and publishes events |
| `internal/infrastructure/healthcheck/outbox_backlog.go` | Health check implementation |
| `internal/config/config.go:188-199` | Configuration struct |
| `internal/infrastructure/mongodb/indexes.go:435-463` | MongoDB indexes |

### Existing Statistics API

**File**: `internal/worker/outbox_worker.go:206-221`

```go
type OutboxStats struct {
    PendingCount int64
}

func (w *OutboxWorker) GetStats(ctx context.Context) (OutboxStats, error)
```

### Existing Health Check

**File**: `internal/infrastructure/healthcheck/outbox_backlog.go:19-101`

```go
type OutboxBacklogChecker struct {
    outbox           appcore.Outbox
    warningThreshold int64  // default: 100
    criticalThreshold int64 // default: 1000
}

func (c *OutboxBacklogChecker) Check(ctx context.Context) HealthStatus {
    // Returns:
    // - backlog_count (pending events)
    // - oldest_event_age (lag from oldest event)
}
```

### Configuration

**File**: `internal/config/config.go:188-199`

```go
type OutboxConfig struct {
    Enabled         bool          // Default: true
    PollInterval    time.Duration // Default: 100ms
    BatchSize       int           // Default: 100
    MaxRetries      int           // Default: 5
    CleanupAge      time.Duration // Default: 7 days
    CleanupInterval time.Duration // Default: 1 hour
}
```

---

## Requirements

### Prometheus Metrics

Add the following metrics to track outbox performance:

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `flowra_outbox_events_pending` | Gauge | - | Current number of unprocessed events |
| `flowra_outbox_events_processed_total` | Counter | `event_type`, `status` | Total processed events (status: success/failed) |
| `flowra_outbox_processing_duration_seconds` | Histogram | `event_type` | Time from event creation to processing |
| `flowra_outbox_publish_duration_seconds` | Histogram | `event_type` | Time to publish event to Redis |
| `flowra_outbox_retry_total` | Counter | `event_type` | Number of retry attempts |
| `flowra_outbox_oldest_event_age_seconds` | Gauge | - | Age of oldest unprocessed event |
| `flowra_outbox_poll_batch_size` | Histogram | - | Size of each poll batch |
| `flowra_outbox_cleanup_deleted_total` | Counter | - | Number of events deleted by cleanup |

### Dashboard Requirements

Create Grafana dashboard with:

1. **Overview Panel**
   - Current backlog count (gauge)
   - Events processed per minute (graph)
   - Error rate percentage (graph)

2. **Latency Panel**
   - Processing latency percentiles (p50, p95, p99)
   - Publish latency percentiles
   - Oldest event age (graph)

3. **Event Types Panel**
   - Events by type (pie chart)
   - Processing rate by event type (stacked graph)
   - Failure rate by event type (table)

4. **Health Panel**
   - Health check status indicator
   - Retry count trends
   - Cleanup activity

---

## Implementation Plan

### Phase 1: Add Prometheus Metrics Package

- [x] Create `internal/infrastructure/metrics/outbox_metrics.go`
- [x] Define metric collectors using `prometheus/client_golang`
- [x] Register metrics in container initialization

**Example structure:**

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

type OutboxMetrics struct {
    EventsPending        prometheus.Gauge
    EventsProcessed      *prometheus.CounterVec
    ProcessingDuration   *prometheus.HistogramVec
    PublishDuration      *prometheus.HistogramVec
    RetryTotal           *prometheus.CounterVec
    OldestEventAge       prometheus.Gauge
    PollBatchSize        prometheus.Histogram
    CleanupDeletedTotal  prometheus.Counter
}

func NewOutboxMetrics(registerer prometheus.Registerer) *OutboxMetrics
```

### Phase 2: Instrument Outbox Worker

- [x] Inject `OutboxMetrics` into `OutboxWorker`
- [x] Add metric collection in `Poll()` method
- [x] Add metric collection in `processEntry()` method
- [x] Add metric collection in `cleanup()` method

**Key instrumentation points:**

| Location | Metric |
|----------|--------|
| `outbox_worker.go:125` after Poll() | `PollBatchSize` |
| `outbox_worker.go:164` processEntry start | Calculate `ProcessingDuration` |
| `outbox_worker.go:185` after Publish() | `PublishDuration`, `EventsProcessed` |
| `outbox_worker.go:193` on retry | `RetryTotal` |
| `outbox_worker.go:119` after cleanup | `CleanupDeletedTotal` |

### Phase 3: Add Periodic Stats Collection

- [x] Create background goroutine for gauge updates
- [x] Update `EventsPending` every poll interval
- [x] Update `OldestEventAge` every poll interval

### Phase 4: Add Metrics HTTP Endpoint

- [x] Register `/metrics` endpoint with Prometheus handler
- [x] Ensure endpoint is accessible from monitoring infrastructure
- [x] Add basic authentication if needed

**File**: `cmd/api/routes.go`

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

func (s *Server) setupRoutes() {
    // ... existing routes ...
    s.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
```

### Phase 5: Create Grafana Dashboard

- [x] Create `configs/grafana/dashboards/outbox.json`
- [x] Define dashboard panels with Prometheus queries
- [x] Add alert rules for critical thresholds
- [x] Test dashboard with sample data

**Alert rules to consider:**

| Alert | Condition | Severity |
|-------|-----------|----------|
| OutboxBacklogHigh | `flowra_outbox_events_pending > 100` | Warning |
| OutboxBacklogCritical | `flowra_outbox_events_pending > 1000` | Critical |
| OutboxProcessingStalled | `flowra_outbox_oldest_event_age_seconds > 60` | Critical |
| OutboxHighErrorRate | `rate(flowra_outbox_events_processed_total{status="failed"}[5m]) > 0.1` | Warning |

---

## Affected Files

| File | Change |
|------|--------|
| `internal/infrastructure/metrics/outbox_metrics.go` | New - metrics definitions |
| `internal/worker/outbox_worker.go` | Modify - add instrumentation |
| `cmd/api/container.go` | Modify - wire metrics |
| `cmd/api/routes.go` | Modify - add /metrics endpoint |
| `cmd/worker/main.go` | Modify - wire metrics |
| `configs/grafana/dashboards/outbox.json` | New - dashboard definition |
| `configs/prometheus/alerts/outbox.yml` | New - alert rules |
| `go.mod` | Modify - add prometheus dependency |

---

## Testing Plan

### Unit Tests

- [ ] Test metric increments in isolation
- [ ] Test metric labels are correctly applied
- [ ] Test histogram bucket boundaries

### Integration Tests

- [ ] Start worker, process events, verify metrics
- [ ] Test error scenarios update failure metrics
- [ ] Test cleanup updates deletion counter

### Manual Testing

1. Start server with metrics enabled
2. Send messages that generate events
3. Access `/metrics` endpoint
4. Verify all metrics appear with correct values
5. Import dashboard to Grafana
6. Verify graphs display data correctly

---

## Dependencies

- `github.com/prometheus/client_golang` - Prometheus client library
- Grafana instance for dashboard (optional for initial implementation)

---

## Success Criteria

1. [x] All defined metrics are exposed at `/metrics` endpoint
2. [x] Metrics update correctly during event processing
3. [x] Grafana dashboard displays real-time outbox statistics
4. [x] Alert rules trigger appropriately for threshold breaches
5. [x] No performance regression from metric collection
