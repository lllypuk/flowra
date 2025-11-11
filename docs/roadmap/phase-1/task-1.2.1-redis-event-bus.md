# Task 1.2.1: Redis Event Bus

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** Blocked
**Время:** 4-5 дней
**Зависимости:** Task 1.1.3 (Redis setup)

---

## Проблема

Domain events публикуются, но нет механизма доставки. Event handlers (notifications, projections, tag processing) не могут подписаться на события.

---

## Цель

Реализовать Redis Pub/Sub Event Bus для:
- Publishing domain events to subscribers
- Multiple handlers per event type
- Dead Letter Queue (DLQ) для failed events
- Graceful shutdown

---

## Архитектура

```
┌─────────────┐                    ┌──────────────┐
│  Use Case   │─(publish)─────────▶│  Event Bus   │
│             │                    │  (Redis)     │
└─────────────┘                    └───────┬──────┘
                                          │
                    ┌─────────────────────┼─────────────────────┐
                    │                     │                     │
                    ▼                     ▼                     ▼
            ┌───────────────┐   ┌──────────────────┐   ┌─────────────┐
            │ Notification  │   │   Projection     │   │   Tag       │
            │   Handler     │   │    Handler       │   │  Processor  │
            └───────────────┘   └──────────────────┘   └─────────────┘
```

---

## Файлы для создания

```
internal/infrastructure/eventbus/
├── eventbus.go              (interface)
├── redis_bus.go             (implementation)
├── redis_bus_test.go        (unit tests)
├── handler.go               (base handler interface)
└── dlq.go                   (dead letter queue)

internal/application/eventhandler/
├── notification_handler.go
├── notification_handler_test.go
└── projection_handler.go    (optional для MVP)
```

---

## 1. Event Bus Interface

```go
package eventbus

import (
    "context"
    "github.com/lllypuk/flowra/internal/application/shared"
)

type EventBus interface {
    // Publish event to all subscribers
    Publish(ctx context.Context, event shared.DomainEvent) error

    // Subscribe handler to event type
    Subscribe(eventType string, handler EventHandler) error

    // Unsubscribe handler
    Unsubscribe(eventType string) error

    // Shutdown gracefully
    Shutdown() error
}

type EventHandler interface {
    Handle(ctx context.Context, event shared.DomainEvent) error
}
```

---

## 2. Redis Event Bus Implementation

```go
package eventbus

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "github.com/redis/go-redis/v9"

    "github.com/lllypuk/flowra/internal/application/shared"
)

type RedisEventBus struct {
    client   *redis.Client
    handlers map[string][]EventHandler  // eventType → handlers
    pubsub   *redis.PubSub
    mu       sync.RWMutex
    shutdown chan struct{}
    wg       sync.WaitGroup
}

func NewRedisEventBus(client *redis.Client) *RedisEventBus {
    return &RedisEventBus{
        client:   client,
        handlers: make(map[string][]EventHandler),
        shutdown: make(chan struct{}),
    }
}

// Publish event to Redis channel
func (b *RedisEventBus) Publish(ctx context.Context, event shared.DomainEvent) error {
    channel := fmt.Sprintf("events.%s", event.EventType())

    // Serialize event
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    // Publish to Redis
    return b.client.Publish(ctx, channel, data).Err()
}

// Subscribe handler to event type
func (b *RedisEventBus) Subscribe(eventType string, handler EventHandler) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    // Add handler to map
    b.handlers[eventType] = append(b.handlers[eventType], handler)

    // Subscribe to Redis channel
    channel := fmt.Sprintf("events.%s", eventType)

    if b.pubsub == nil {
        b.pubsub = b.client.Subscribe(context.Background(), channel)
    } else {
        b.pubsub.Subscribe(context.Background(), channel)
    }

    // Start listening (if not already started)
    b.wg.Add(1)
    go b.listen()

    return nil
}

// listen - background goroutine receiving messages
func (b *RedisEventBus) listen() {
    defer b.wg.Done()

    ch := b.pubsub.Channel()

    for {
        select {
        case msg := <-ch:
            b.handleMessage(msg)
        case <-b.shutdown:
            return
        }
    }
}

// handleMessage - dispatch message to handlers
func (b *RedisEventBus) handleMessage(msg *redis.Message) {
    // Extract event type from channel name
    eventType := msg.Channel[7:]  // remove "events." prefix

    b.mu.RLock()
    handlers := b.handlers[eventType]
    b.mu.RUnlock()

    // Deserialize event
    var event shared.DomainEvent
    if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
        // Log error
        return
    }

    // Dispatch to all handlers
    for _, handler := range handlers {
        if err := b.handleWithRetry(context.Background(), handler, event); err != nil {
            // Send to DLQ
            b.sendToDLQ(event, err)
        }
    }
}

// handleWithRetry - retry handler with exponential backoff
func (b *RedisEventBus) handleWithRetry(ctx context.Context, handler EventHandler, event shared.DomainEvent) error {
    maxRetries := 3
    baseDelay := 100 * time.Millisecond

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := handler.Handle(ctx, event)
        if err == nil {
            return nil  // success
        }

        // Exponential backoff
        delay := baseDelay * time.Duration(1<<uint(attempt))
        time.Sleep(delay)
    }

    return fmt.Errorf("handler failed after %d retries", maxRetries)
}

// Shutdown - graceful shutdown
func (b *RedisEventBus) Shutdown() error {
    close(b.shutdown)
    b.wg.Wait()

    if b.pubsub != nil {
        return b.pubsub.Close()
    }

    return nil
}
```

---

## 3. Dead Letter Queue (dlq.go)

```go
package eventbus

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
)

type DeadLetterQueue struct {
    client *redis.Client
}

func NewDeadLetterQueue(client *redis.Client) *DeadLetterQueue {
    return &DeadLetterQueue{client: client}
}

func (dlq *DeadLetterQueue) Send(event shared.DomainEvent, err error) error {
    data, _ := json.Marshal(map[string]interface{}{
        "event": event,
        "error": err.Error(),
        "timestamp": time.Now(),
    })

    key := "dlq:events"
    return dlq.client.LPush(context.Background(), key, data).Err()
}

func (dlq *DeadLetterQueue) GetFailedEvents(limit int) ([]map[string]interface{}, error) {
    // Retrieve failed events from DLQ
    // ...
}
```

---

## 4. Notification Handler

```go
package eventhandler

import (
    "context"
    "fmt"
    "github.com/google/uuid"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/application/notification"
    chatevents "github.com/lllypuk/flowra/internal/domain/chat/events"
    messageevents "github.com/lllypuk/flowra/internal/domain/message/events"
)

type NotificationHandler struct {
    createNotifUseCase *notification.CreateNotificationUseCase
}

func NewNotificationHandler(createNotifUseCase *notification.CreateNotificationUseCase) *NotificationHandler {
    return &NotificationHandler{
        createNotifUseCase: createNotifUseCase,
    }
}

func (h *NotificationHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
    switch e := event.(type) {
    case *chatevents.ChatCreated:
        return h.handleChatCreated(ctx, e)

    case *chatevents.UserAssigned:
        return h.handleUserAssigned(ctx, e)

    case *chatevents.StatusChanged:
        return h.handleStatusChanged(ctx, e)

    case *messageevents.MessagePosted:
        return h.handleMessagePosted(ctx, e)

    default:
        // Unknown event type, skip
        return nil
    }
}

func (h *NotificationHandler) handleUserAssigned(ctx context.Context, event *chatevents.UserAssigned) error {
    cmd := notification.CreateNotificationCommand{
        UserID:  event.AssignedTo,
        Type:    "TaskAssigned",
        Title:   "New task assigned",
        Content: fmt.Sprintf("You have been assigned to a task in chat %s", event.ChatID),
        Link:    fmt.Sprintf("/chats/%s", event.ChatID),
    }

    _, err := h.createNotifUseCase.Execute(ctx, cmd)
    return err
}

func (h *NotificationHandler) handleMessagePosted(ctx context.Context, event *messageevents.MessagePosted) error {
    // Notify all chat participants except sender
    // ...
}
```

---

## 5. Event Bus Setup in main.go

```go
func main() {
    // ... setup

    // Initialize Event Bus
    eventBus := eventbus.NewRedisEventBus(redisClient)

    // Initialize handlers
    notificationHandler := eventhandler.NewNotificationHandler(createNotificationUC)

    // Subscribe handlers
    eventBus.Subscribe("ChatCreated", notificationHandler)
    eventBus.Subscribe("UserAssigned", notificationHandler)
    eventBus.Subscribe("StatusChanged", notificationHandler)
    eventBus.Subscribe("MessagePosted", notificationHandler)

    // Integration with Use Cases
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    createChatUC.SetEventBus(eventBus)  // inject event bus

    // ... start server

    // Graceful shutdown
    defer eventBus.Shutdown()
}
```

---

## Тестирование

```go
func TestEventBus_PublishAndSubscribe(t *testing.T) {
    client := testutil.SetupRedis(t)
    eventBus := eventbus.NewRedisEventBus(client)

    // Mock handler
    handler := &MockHandler{}
    eventBus.Subscribe("ChatCreated", handler)

    // Publish event
    event := &chatevents.ChatCreated{
        ChatID: uuid.New(),
        // ...
    }

    err := eventBus.Publish(context.Background(), event)
    require.NoError(t, err)

    // Wait for handler
    time.Sleep(100 * time.Millisecond)

    // Verify handler called
    handler.AssertCalled(t, "Handle", mock.Anything, event)
}
```

---

## Критерии успеха

- ✅ **Events published and delivered**
- ✅ **Multiple handlers work**
- ✅ **DLQ captures failed events**
- ✅ **Retry mechanism works**
- ✅ **Graceful shutdown**
- ✅ **Test coverage >80%**

---

## Следующий шаг

→ **Task 1.3.1: Keycloak Integration**
