package metrics_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/infrastructure/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestOutboxMetrics_Registration(t *testing.T) {
	// Create a new registry for testing
	registry := prometheus.NewRegistry()

	// Create metrics
	outboxMetrics := metrics.NewOutboxMetrics(registry)

	// Verify all metrics are registered
	if outboxMetrics.EventsPending == nil {
		t.Error("EventsPending metric not initialized")
	}
	if outboxMetrics.EventsProcessed == nil {
		t.Error("EventsProcessed metric not initialized")
	}
	if outboxMetrics.ProcessingDuration == nil {
		t.Error("ProcessingDuration metric not initialized")
	}
	if outboxMetrics.PublishDuration == nil {
		t.Error("PublishDuration metric not initialized")
	}
	if outboxMetrics.RetryTotal == nil {
		t.Error("RetryTotal metric not initialized")
	}
	if outboxMetrics.OldestEventAge == nil {
		t.Error("OldestEventAge metric not initialized")
	}
	if outboxMetrics.PollBatchSize == nil {
		t.Error("PollBatchSize metric not initialized")
	}
	if outboxMetrics.CleanupDeletedTotal == nil {
		t.Error("CleanupDeletedTotal metric not initialized")
	}

	// Test setting a simple gauge value
	outboxMetrics.EventsPending.Set(42)

	// Verify the value
	got := testutil.ToFloat64(outboxMetrics.EventsPending)
	if got != 42 {
		t.Errorf("EventsPending.Set(42) = %v, want 42", got)
	}
}

func TestOutboxMetrics_CounterIncrement(t *testing.T) {
	registry := prometheus.NewRegistry()
	outboxMetrics := metrics.NewOutboxMetrics(registry)

	// Increment success counter
	outboxMetrics.EventsProcessed.WithLabelValues("TestEvent", "success").Inc()
	outboxMetrics.EventsProcessed.WithLabelValues("TestEvent", "success").Inc()

	// Verify count
	got := testutil.ToFloat64(outboxMetrics.EventsProcessed.WithLabelValues("TestEvent", "success"))
	if got != 2 {
		t.Errorf("EventsProcessed count = %v, want 2", got)
	}
}

func TestOutboxMetrics_HistogramObserve(_ *testing.T) {
	registry := prometheus.NewRegistry()
	outboxMetrics := metrics.NewOutboxMetrics(registry)

	// Observe some durations
	outboxMetrics.ProcessingDuration.WithLabelValues("TestEvent").Observe(0.5)
	outboxMetrics.ProcessingDuration.WithLabelValues("TestEvent").Observe(1.5)

	// For histograms, we just verify they accept observations without errors
	// The actual histogram bucket counts would require more complex validation
	// which is beyond the scope of this basic test
	outboxMetrics.PublishDuration.WithLabelValues("TestEvent").Observe(0.1)
	outboxMetrics.PollBatchSize.Observe(50)
}
