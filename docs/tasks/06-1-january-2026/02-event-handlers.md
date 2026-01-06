# 02: Event Handlers

**Приоритет:** 🔴 Critical  
**Статус:** ✅ Завершено  
**Дни:** 1-3 января  
**Зависимости:** [01-event-bus.md](01-event-bus.md)

---

## Описание

Реализовать обработчики событий для Event Bus. Handlers подписываются на domain events и выполняют side-effects: создание уведомлений, логирование, обновление read models.

---

## Файлы

```
internal/infrastructure/eventbus/
├── handlers.go             (~200 LOC)
└── handlers_test.go        (~150 LOC)
```

---

## Детали реализации

### 1. NotificationHandler

Обработчик для создания уведомлений при событиях:

```go
type NotificationHandler struct {
    createNotifUC *notification.CreateNotificationUseCase
}

func (h *NotificationHandler) Handle(ctx context.Context, event domain.Event) error {
    switch e := event.(type) {
    case *chat.ChatCreated:
        // Уведомить участников о создании чата
    case *message.MessageSent:
        // Уведомить упомянутых пользователей
    case *task.TaskAssigned:
        // Уведомить assignee о назначении
    case *task.TaskStatusChanged:
        // Уведомить reporter об изменении статуса
    }
    return nil
}
```

**Обрабатываемые события:**
- `ChatCreated` → уведомление участникам
- `MessageSent` → уведомление упомянутым (@mentions)
- `TaskAssigned` → уведомление assignee
- `TaskStatusChanged` → уведомление reporter и watchers
- `TaskDueDateApproaching` → напоминание assignee

### 2. LoggingHandler

Обработчик для audit trail:

```go
type LoggingHandler struct {
    logger *slog.Logger
}

func (h *LoggingHandler) Handle(ctx context.Context, event domain.Event) error {
    h.logger.Info("domain event",
        "type", event.EventType(),
        "aggregate_id", event.AggregateID(),
        "timestamp", event.OccurredAt(),
        "data", event,
    )
    return nil
}
```

**Логируемая информация:**
- Тип события
- ID агрегата
- Timestamp
- Payload события
- User ID (если доступен в контексте)

### 3. Регистрация handlers

```go
func RegisterHandlers(bus EventBus, container *Container) error {
    notifHandler := NewNotificationHandler(container.CreateNotifUC)
    logHandler := NewLoggingHandler(container.Logger)
    
    // Notification events
    bus.Subscribe("chat.created", notifHandler)
    bus.Subscribe("message.sent", notifHandler)
    bus.Subscribe("task.assigned", notifHandler)
    bus.Subscribe("task.status_changed", notifHandler)
    
    // Logging - все события
    bus.Subscribe("*", logHandler)
    
    return nil
}
```

---

## Error Handling

### Retry Strategy

```go
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
}

func WithRetry(handler EventHandler, config RetryConfig) EventHandler {
    return func(ctx context.Context, event domain.Event) error {
        var lastErr error
        backoff := config.InitialBackoff
        
        for i := 0; i <= config.MaxRetries; i++ {
            if err := handler.Handle(ctx, event); err != nil {
                lastErr = err
                time.Sleep(backoff)
                backoff = min(backoff*time.Duration(config.Multiplier), config.MaxBackoff)
                continue
            }
            return nil
        }
        return fmt.Errorf("max retries exceeded: %w", lastErr)
    }
}
```

### Dead Letter Queue

```go
type DeadLetterHandler struct {
    redis  *redis.Client
    logger *slog.Logger
}

func (h *DeadLetterHandler) Handle(ctx context.Context, event domain.Event, err error) {
    // Сохранить в Redis для последующего анализа
    payload, _ := json.Marshal(event)
    h.redis.LPush(ctx, "events:dead_letter", string(payload))
    
    h.logger.Error("event processing failed",
        "event_type", event.EventType(),
        "error", err,
    )
}
```

---

## Тестирование

### Unit Tests

```go
func TestNotificationHandler_ChatCreated(t *testing.T) {
    // Given
    mockUC := &MockCreateNotificationUseCase{}
    handler := NewNotificationHandler(mockUC)
    
    event := chat.NewChatCreated(chatID, "Test Chat", []uuid.UUID{user1, user2})
    
    // When
    err := handler.Handle(context.Background(), event)
    
    // Then
    require.NoError(t, err)
    assert.Len(t, mockUC.CreatedNotifications, 2)
}

func TestLoggingHandler_LogsAllEvents(t *testing.T) {
    // Given
    var buf bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&buf, nil))
    handler := NewLoggingHandler(logger)
    
    event := message.NewMessageSent(msgID, chatID, userID, "Hello")
    
    // When
    err := handler.Handle(context.Background(), event)
    
    // Then
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "message.sent")
}
```

### Integration Tests

```go
func TestEventHandlers_Integration(t *testing.T) {
    // Given
    container := setupTestContainer(t)
    bus := container.EventBus
    RegisterHandlers(bus, container)
    bus.Start(context.Background())
    defer bus.Shutdown()
    
    // When - publish event
    event := task.NewTaskAssigned(taskID, assigneeID, assignerID)
    err := bus.Publish(context.Background(), event)
    require.NoError(t, err)
    
    // Then - notification created (eventually)
    assert.Eventually(t, func() bool {
        notifs, _ := container.NotifRepo.FindByUser(context.Background(), assigneeID)
        return len(notifs) == 1
    }, 5*time.Second, 100*time.Millisecond)
}
```

---

## Чеклист

### Реализация
- [x] `NotificationHandler` создан
- [x] `LoggingHandler` создан
- [x] Регистрация handlers при старте (`RegisterAllHandlers`, `HandlerRegistry`)
- [x] Retry logic реализован (в `RedisEventBus.executeHandler`)
- [x] Dead Letter Queue реализован (`DeadLetterHandler`)
- [x] Обработка всех основных событий

### События для обработки
- [x] `chat.created` → логирование (уведомления через `participant_added`)
- [x] `chat.participant_added` → уведомление добавленному участнику
- [x] `message.created` → уведомления для @mentions
- [x] `task.created` → уведомление assignee
- [x] `task.assignee_changed` → уведомление assignee
- [x] `task.status_changed` → логирование (TODO: уведомления watchers)
- [ ] `task.due_date_approaching` → напоминание (требует worker service)

### Тестирование
- [x] Unit tests для NotificationHandler
- [x] Unit tests для LoggingHandler
- [x] Unit tests для DeadLetterHandler
- [x] Unit tests для HandlerRegistry
- [x] Integration tests с реальным Event Bus
- [x] Coverage: 82%+

### Документация
- [x] Godoc комментарии
- [x] Список обрабатываемых событий
- [x] Примеры использования (`RegisterAllHandlers`)

---

## Зависимости

### Требуется до начала
- [01-event-bus.md](01-event-bus.md) — EventBus interface

### Использует
- `notification.CreateNotificationUseCase`
- `slog.Logger`
- Redis client (для Dead Letter Queue)

### Требуется для
- [08-websocket.md](08-websocket.md) — WebSocket broadcaster
- [10-e2e-tests.md](10-e2e-tests.md) — End-to-end tests

---

## Заметки

- Handlers должны быть идемпотентными — одно событие может быть обработано несколько раз
- Logging handler подписывается на wildcard `*` для логирования всех событий
- Notification handler должен проверять настройки пользователя (muted chats, notification preferences)
- Dead Letter Queue нужно периодически проверять и обрабатывать вручную

---

*Создано: 2026-01-01*  
*Завершено: 2026-01-01*

---

## Реализация

### Созданные файлы

- `internal/infrastructure/eventbus/handlers.go` (~730 LOC)
- `internal/infrastructure/eventbus/handlers_test.go` (~1200 LOC)

### Основные компоненты

1. **NotificationHandler** — обработчик уведомлений
   - Обрабатывает события: `chat.created`, `chat.participant_added`, `message.created`, `task.created`, `task.status_changed`, `task.assignee_changed`
   - Поддерживает @mentions в сообщениях через `UserResolver` интерфейс
   - Идемпотентная обработка

2. **LoggingHandler** — аудит-логгер
   - Логирует все события с метаданными
   - Truncation для больших payload (>500 символов)

3. **DeadLetterHandler** — хранение failed events
   - Сохраняет в Redis с лимитом очереди
   - Методы: `GetDeadLetters`, `ClearDeadLetters`, `QueueLength`

4. **HandlerRegistry** — регистрация handlers
   - `RegisterNotificationHandler` — регистрирует для notification events
   - `RegisterLoggingHandler` — регистрирует для указанных событий
   - `RegisterAllHandlers` — convenience функция

### Покрытие тестами

- Unit tests: 40+ тестов
- Integration tests: 2 теста с реальным Redis
- Coverage: 82.1%