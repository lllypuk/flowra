package eventbus_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent is a concrete event type for testing.
type testEvent struct {
	event.BaseEvent

	Message string `json:"message"`
}

func newTestEvent(eventType, aggregateID, message string) *testEvent {
	return &testEvent{
		BaseEvent: event.NewBaseEvent(
			eventType,
			aggregateID,
			"test",
			1,
			event.NewMetadata("user-1", "correlation-1", "causation-1"),
		),
		Message: message,
	}
}

func TestNewRedisEventBus(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("creates with defaults", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)

		assert.NotNil(t, bus)
		assert.False(t, bus.IsRunning())
		assert.Equal(t, 0, bus.HandlerCount("any.event"))
	})

	t.Run("applies options", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		retryConfig := eventbus.RetryConfig{
			MaxRetries:     5,
			InitialBackoff: 200 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
			BackoffFactor:  3.0,
		}

		bus := eventbus.NewRedisEventBus(client,
			eventbus.WithLogger(logger),
			eventbus.WithRetryConfig(retryConfig),
			eventbus.WithChannelPrefix("test-events:"),
		)

		assert.NotNil(t, bus)
	})
}

func TestRedisEventBus_Subscribe(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client)

	t.Run("registers handler successfully", func(t *testing.T) {
		handler := func(_ context.Context, _ event.DomainEvent) error {
			return nil
		}

		err := bus.Subscribe("user.created", handler)
		require.NoError(t, err)

		assert.Equal(t, 1, bus.HandlerCount("user.created"))
	})

	t.Run("allows multiple handlers for same event type", func(t *testing.T) {
		newBus := eventbus.NewRedisEventBus(client)

		handler1 := func(_ context.Context, _ event.DomainEvent) error { return nil }
		handler2 := func(_ context.Context, _ event.DomainEvent) error { return nil }

		err := newBus.Subscribe("order.created", handler1)
		require.NoError(t, err)

		err = newBus.Subscribe("order.created", handler2)
		require.NoError(t, err)

		assert.Equal(t, 2, newBus.HandlerCount("order.created"))
	})

	t.Run("returns error for empty event type", func(t *testing.T) {
		handler := func(_ context.Context, _ event.DomainEvent) error { return nil }

		err := bus.Subscribe("", handler)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event type cannot be empty")
	})

	t.Run("returns error for nil handler", func(t *testing.T) {
		err := bus.Subscribe("user.created", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler cannot be nil")
	})
}

func TestRedisEventBus_Publish(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client)
	ctx := context.Background()

	t.Run("publishes event successfully", func(t *testing.T) {
		evt := newTestEvent("user.created", "user-123", "Hello World")

		err := bus.Publish(ctx, evt)
		require.NoError(t, err)
	})

	t.Run("returns error for nil event", func(t *testing.T) {
		err := bus.Publish(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event cannot be nil")
	})
}

func TestRedisEventBus_PublishAndReceive(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("handler receives published event", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)

		received := make(chan event.DomainEvent, 1)
		handler := func(_ context.Context, e event.DomainEvent) error {
			received <- e
			return nil
		}

		err := bus.Subscribe("user.created", handler)
		require.NoError(t, err)

		// Start bus in background
		go func() {
			_ = bus.Start(ctx)
		}()

		// Give the bus time to start
		time.Sleep(100 * time.Millisecond)

		// Publish event
		evt := newTestEvent("user.created", "user-123", "Hello World")
		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for event to be received
		select {
		case receivedEvt := <-received:
			assert.Equal(t, "user.created", receivedEvt.EventType())
			assert.Equal(t, "user-123", receivedEvt.AggregateID())
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		err = bus.Shutdown()
		require.NoError(t, err)
	})

	t.Run("multiple handlers receive same event", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)

		var count atomic.Int32
		var wg sync.WaitGroup
		wg.Add(3)

		for range 3 {
			handler := func(_ context.Context, _ event.DomainEvent) error {
				count.Add(1)
				wg.Done()
				return nil
			}
			err := bus.Subscribe("order.created", handler)
			require.NoError(t, err)
		}

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		evt := newTestEvent("order.created", "order-456", "New order")
		err := bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for all handlers to complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			assert.Equal(t, int32(3), count.Load())
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for handlers")
		}

		err = bus.Shutdown()
		require.NoError(t, err)
	})
}

func TestRedisEventBus_EventSerialization(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("preserves event metadata", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)

		received := make(chan event.DomainEvent, 1)
		handler := func(_ context.Context, e event.DomainEvent) error {
			received <- e
			return nil
		}

		err := bus.Subscribe("metadata.test", handler)
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		originalEvt := newTestEvent("metadata.test", "agg-123", "test message")
		err = bus.Publish(ctx, originalEvt)
		require.NoError(t, err)

		select {
		case receivedEvt := <-received:
			assert.Equal(t, originalEvt.EventType(), receivedEvt.EventType())
			assert.Equal(t, originalEvt.AggregateID(), receivedEvt.AggregateID())
			assert.Equal(t, originalEvt.AggregateType(), receivedEvt.AggregateType())
			assert.Equal(t, originalEvt.Version(), receivedEvt.Version())
			assert.Equal(t, originalEvt.Metadata().UserID, receivedEvt.Metadata().UserID)
			assert.Equal(t, originalEvt.Metadata().CorrelationID, receivedEvt.Metadata().CorrelationID)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		err = bus.Shutdown()
		require.NoError(t, err)
	})

	t.Run("includes payload in deserialized event", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)

		received := make(chan event.DomainEvent, 1)
		handler := func(_ context.Context, e event.DomainEvent) error {
			received <- e
			return nil
		}

		err := bus.Subscribe("payload.test", handler)
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		evt := newTestEvent("payload.test", "agg-789", "payload content")
		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		select {
		case de := <-received:
			// Check if we can access payload through interface
			if payloader, ok := de.(interface{ Payload() json.RawMessage }); ok {
				var parsed map[string]interface{}
				unmarshalErr := json.Unmarshal(payloader.Payload(), &parsed)
				require.NoError(t, unmarshalErr)
				assert.Equal(t, "payload content", parsed["message"])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		err = bus.Shutdown()
		require.NoError(t, err)
	})
}

func TestRedisEventBus_RetryLogic(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("retries failed handler", func(t *testing.T) {
		retryConfig := eventbus.RetryConfig{
			MaxRetries:     2,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     100 * time.Millisecond,
			BackoffFactor:  2.0,
		}
		bus := eventbus.NewRedisEventBus(client, eventbus.WithRetryConfig(retryConfig))

		var attempts atomic.Int32
		done := make(chan struct{})

		handler := func(_ context.Context, _ event.DomainEvent) error {
			count := attempts.Add(1)
			if count < 3 {
				return errors.New("temporary error")
			}
			close(done)
			return nil
		}

		err := bus.Subscribe("retry.test", handler)
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		evt := newTestEvent("retry.test", "agg-retry", "retry me")
		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		select {
		case <-done:
			assert.Equal(t, int32(3), attempts.Load())
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for retries")
		}

		err = bus.Shutdown()
		require.NoError(t, err)
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		retryConfig := eventbus.RetryConfig{
			MaxRetries:     2,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     50 * time.Millisecond,
			BackoffFactor:  2.0,
		}
		bus := eventbus.NewRedisEventBus(client, eventbus.WithRetryConfig(retryConfig))

		var attempts atomic.Int32

		handler := func(_ context.Context, _ event.DomainEvent) error {
			attempts.Add(1)
			return errors.New("persistent error")
		}

		err := bus.Subscribe("retry.fail", handler)
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		evt := newTestEvent("retry.fail", "agg-fail", "fail me")
		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for all retries to complete
		time.Sleep(500 * time.Millisecond)

		// 1 initial attempt + 2 retries = 3 total attempts
		assert.Equal(t, int32(3), attempts.Load())

		err = bus.Shutdown()
		require.NoError(t, err)
	})
}

func TestRedisEventBus_GracefulShutdown(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("waits for handlers to complete", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		ctx := context.Background()

		handlerStarted := make(chan struct{})
		handlerCompleted := atomic.Bool{}

		handler := func(_ context.Context, _ event.DomainEvent) error {
			close(handlerStarted)
			time.Sleep(200 * time.Millisecond)
			handlerCompleted.Store(true)
			return nil
		}

		err := bus.Subscribe("shutdown.test", handler)
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		evt := newTestEvent("shutdown.test", "agg-shutdown", "shutdown test")
		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for handler to start
		select {
		case <-handlerStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for handler to start")
		}

		// Shutdown should wait for handler
		err = bus.Shutdown()
		require.NoError(t, err)

		assert.True(t, handlerCompleted.Load(), "handler should have completed before shutdown returned")
	})

	t.Run("shutdown is idempotent", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		ctx := context.Background()

		err := bus.Subscribe("shutdown.idem", func(_ context.Context, _ event.DomainEvent) error {
			return nil
		})
		require.NoError(t, err)

		go func() {
			_ = bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// First shutdown
		err = bus.Shutdown()
		require.NoError(t, err)

		// Second shutdown should not error
		err = bus.Shutdown()
		require.NoError(t, err)
	})

	t.Run("cannot start twice", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := bus.Subscribe("start.twice", func(_ context.Context, _ event.DomainEvent) error {
			return nil
		})
		require.NoError(t, err)

		started := make(chan error, 2)

		// First start
		go func() {
			started <- bus.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// Second start should fail
		go func() {
			started <- bus.Start(ctx)
		}()

		var errCount int
		for range 2 {
			select {
			case startErr := <-started:
				if startErr != nil && startErr.Error() == "event bus is already running" {
					errCount++
				}
			case <-time.After(3 * time.Second):
			}
		}

		assert.Equal(t, 1, errCount, "one Start should have failed")

		_ = bus.Shutdown()
	})
}

func TestRedisEventBus_IsRunning(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	bus := eventbus.NewRedisEventBus(client)
	ctx := context.Background()

	assert.False(t, bus.IsRunning())

	err := bus.Subscribe("running.test", func(_ context.Context, _ event.DomainEvent) error {
		return nil
	})
	require.NoError(t, err)

	go func() {
		_ = bus.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	assert.True(t, bus.IsRunning())

	err = bus.Shutdown()
	require.NoError(t, err)

	assert.False(t, bus.IsRunning())
}

func TestRedisEventBus_ChannelPrefix(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("different prefixes isolate events", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bus1 := eventbus.NewRedisEventBus(client, eventbus.WithChannelPrefix("bus1:"))
		bus2 := eventbus.NewRedisEventBus(client, eventbus.WithChannelPrefix("bus2:"))

		received1 := atomic.Int32{}
		received2 := atomic.Int32{}

		_ = bus1.Subscribe("test.event", func(_ context.Context, _ event.DomainEvent) error {
			received1.Add(1)
			return nil
		})

		_ = bus2.Subscribe("test.event", func(_ context.Context, _ event.DomainEvent) error {
			received2.Add(1)
			return nil
		})

		go func() { _ = bus1.Start(ctx) }()
		go func() { _ = bus2.Start(ctx) }()

		time.Sleep(100 * time.Millisecond)

		// Publish only to bus1's channel
		evt := newTestEvent("test.event", "agg-1", "message")
		_ = bus1.Publish(ctx, evt)

		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, int32(1), received1.Load())
		assert.Equal(t, int32(0), received2.Load())

		_ = bus1.Shutdown()
		_ = bus2.Shutdown()
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := eventbus.DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.InitialBackoff)
	assert.Equal(t, 5*time.Second, config.MaxBackoff)
	assert.InDelta(t, 2.0, config.BackoffFactor, 0.001)
}
