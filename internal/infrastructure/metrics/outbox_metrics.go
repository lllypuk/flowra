package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// OutboxMetrics contains Prometheus metrics for monitoring outbox performance.
type OutboxMetrics struct {
	EventsPending       prometheus.Gauge
	EventsProcessed     *prometheus.CounterVec
	ProcessingDuration  *prometheus.HistogramVec
	PublishDuration     *prometheus.HistogramVec
	RetryTotal          *prometheus.CounterVec
	OldestEventAge      prometheus.Gauge
	PollBatchSize       prometheus.Histogram
	CleanupDeletedTotal prometheus.Counter
}

// NewOutboxMetrics creates and registers outbox metrics with the given registerer.
func NewOutboxMetrics(registerer prometheus.Registerer) *OutboxMetrics {
	metrics := &OutboxMetrics{
		EventsPending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "flowra_outbox_events_pending",
			Help: "Current number of unprocessed events in the outbox",
		}),
		EventsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flowra_outbox_events_processed_total",
				Help: "Total number of processed events",
			},
			[]string{"event_type", "status"}, // status: success/failed
		),
		ProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flowra_outbox_processing_duration_seconds",
				Help:    "Time from event creation to processing completion",
				Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"event_type"},
		),
		PublishDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flowra_outbox_publish_duration_seconds",
				Help:    "Time to publish event to Redis event bus",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}, // Shorter buckets for publish
			},
			[]string{"event_type"},
		),
		RetryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flowra_outbox_retry_total",
				Help: "Total number of retry attempts for failed events",
			},
			[]string{"event_type"},
		),
		OldestEventAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "flowra_outbox_oldest_event_age_seconds",
			Help: "Age in seconds of the oldest unprocessed event",
		}),
		PollBatchSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "flowra_outbox_poll_batch_size",
			Help:    "Number of events retrieved in each poll batch",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		CleanupDeletedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "flowra_outbox_cleanup_deleted_total",
			Help: "Total number of processed events deleted by cleanup",
		}),
	}

	// Register all metrics
	registerer.MustRegister(
		metrics.EventsPending,
		metrics.EventsProcessed,
		metrics.ProcessingDuration,
		metrics.PublishDuration,
		metrics.RetryTotal,
		metrics.OldestEventAge,
		metrics.PollBatchSize,
		metrics.CleanupDeletedTotal,
	)

	return metrics
}
