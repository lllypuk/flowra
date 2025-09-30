# Event Flow and Event Sourcing Architecture

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

Система использует **Event Sourcing** для хранения изменений состояния и **Event-Driven Architecture** для коммуникации между bounded contexts. События являются единственным source of truth, все read models (проекции) строятся из event stream.

## Архитектурные принципы

- **Event Sourcing** — состояние восстанавливается из событий
- **CQRS** — разделение команд (write) и запросов (read)
- **Eventual Consistency** — между bounded contexts
- **Idempotency** — события обрабатываются ровно один раз
- **Ordering** — гарантия порядка для одного aggregate
- **At-most-once delivery (MVP)** — упрощённая модель доставки

---

## Event Store

### Структура хранения

**MongoDB коллекция:** `events`

```javascript
{
  "_id": "event-uuid",
  "aggregateId": "chat-uuid",
  "aggregateType": "Chat",
  "eventType": "MessagePosted",
  "eventData": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "authorId": "user-uuid",
    "content": "Закончил работу\n#status Done",
    "timestamp": "2025-09-30T10:00:00Z"
  },
  "version": 142,
  "timestamp": "2025-09-30T10:00:00Z",
  "metadata": {
    "correlationId": "req-uuid",      // ID HTTP запроса
    "causationId": "parent-event-id", // ID события, вызвавшего это
    "userId": "user-uuid"
  }
}
```

**Индексы:**

```javascript
// Для загрузки событий aggregate
db.events.createIndex({ "aggregateId": 1, "version": 1 }, { unique: true })

// Для хронологической последовательности
db.events.createIndex({ "timestamp": 1 })

// Для фильтрации по типу события
db.events.createIndex({ "eventType": 1, "timestamp": 1 })

// Для поиска по типу aggregate
db.events.createIndex({ "aggregateType": 1, "timestamp": 1 })
```

### Event Schema

**Базовая структура:**

```go
type DomainEvent interface {
    GetEventID() UUID
    GetAggregateID() UUID
    GetAggregateType() string
    GetEventType() string
    GetVersion() int
    GetTimestamp() time.Time
    GetMetadata() EventMetadata
}

type EventMetadata struct {
    CorrelationID UUID // ID запроса/операции
    CausationID   UUID // ID события-причины
    UserID        UUID // Кто инициировал
}

type BaseEvent struct {
    EventID       UUID          `bson:"_id"`
    AggregateID   UUID          `bson:"aggregateId"`
    AggregateType string        `bson:"aggregateType"`
    EventType     string        `bson:"eventType"`
    EventData     interface{}   `bson:"eventData"`
    Version       int           `bson:"version"`
    Timestamp     time.Time     `bson:"timestamp"`
    Metadata      EventMetadata `bson:"metadata"`
}
```

### Event Versioning

**Стратегия:** Flexible Schema (MongoDB)

```go
// V1: Initial version
type MessagePosted struct {
    MessageID UUID
    ChatID    UUID
    Content   string
}

// V2: Added AuthorID
type MessagePosted struct {
    MessageID UUID
    ChatID    UUID
    AuthorID  *UUID  // nullable для обратной совместимости
    Content   string
}

// Обработчик должен проверять наличие полей
func (h *Handler) Handle(event MessagePosted) {
    if event.AuthorID == nil {
        // Старая версия события
        event.AuthorID = &systemUserID
    }

    // Обработка с AuthorID
}
```

**Правила эволюции:**
- Новые поля всегда опциональны (nullable/pointer)
- Нельзя удалять существующие поля
- Нельзя менять тип существующих полей
- Переименование = новое поле + deprecated старое

### Append-only гарантии

```go
func (s *EventStore) Append(event DomainEvent) error {
    // 1. Проверяем optimistic concurrency
    currentVersion := s.GetCurrentVersion(event.GetAggregateID())
    if event.GetVersion() != currentVersion+1 {
        return ErrConcurrencyConflict
    }

    // 2. Сохраняем событие (append-only)
    _, err := s.collection.InsertOne(context.Background(), event)
    if err != nil {
        // Unique constraint violation на aggregateId + version
        return ErrConcurrencyConflict
    }

    return nil
}
```

---

## Event Bus (Redis Pub/Sub)

### Топология каналов

**Стратегия:** По типу события

```
Channel: events.MessagePosted
Channel: events.ChatTypeChanged
Channel: events.TagsParsed
Channel: events.StatusChanged
Channel: events.AssigneeChanged
Channel: events.TaskCreated
Channel: events.UserNotified
```

**Формат сообщения:**

```json
{
  "eventId": "event-uuid",
  "eventType": "MessagePosted",
  "aggregateId": "chat-uuid",
  "payload": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "authorId": "user-uuid",
    "content": "...",
    "timestamp": "2025-09-30T10:00:00Z"
  },
  "metadata": {
    "correlationId": "req-uuid",
    "causationId": null,
    "userId": "user-uuid"
  }
}
```

### Публикация событий

```go
type EventBus interface {
    Publish(event DomainEvent) error
    Subscribe(eventType string, handler EventHandler) error
}

type RedisEventBus struct {
    client *redis.Client
}

func (b *RedisEventBus) Publish(event DomainEvent) error {
    channel := fmt.Sprintf("events.%s", event.GetEventType())

    message := EventMessage{
        EventID:     event.GetEventID(),
        EventType:   event.GetEventType(),
        AggregateID: event.GetAggregateID(),
        Payload:     event.GetEventData(),
        Metadata:    event.GetMetadata(),
    }

    data, err := json.Marshal(message)
    if err != nil {
        return err
    }

    return b.client.Publish(context.Background(), channel, data).Err()
}
```

### Подписка на события

```go
func (b *RedisEventBus) Subscribe(eventType string, handler EventHandler) error {
    channel := fmt.Sprintf("events.%s", eventType)

    pubsub := b.client.Subscribe(context.Background(), channel)

    go func() {
        for msg := range pubsub.Channel() {
            var eventMsg EventMessage
            if err := json.Unmarshal([]byte(msg.Payload), &eventMsg); err != nil {
                log.Error("Failed to unmarshal event", "error", err)
                continue
            }

            // Обрабатываем событие с retry и idempotency
            b.processEvent(eventMsg, handler)
        }
    }()

    return nil
}
```

### Delivery Guarantees

**MVP: At-most-once**

- Redis Pub/Sub не гарантирует доставку
- Если подписчик offline → событие потеряно
- **Mitigation:** События хранятся в Event Store, можно восстановить состояние

**Преимущества:**
- Простая реализация
- Низкая latency
- Достаточно для MVP

**V2: At-least-once (Transactional Outbox)**

```go
// Outbox table для гарантированной доставки
type OutboxEvent struct {
    ID          UUID
    EventID     UUID
    EventType   string
    Payload     []byte
    PublishedAt *time.Time
    CreatedAt   time.Time
}

// При сохранении события также сохраняем в outbox
func (s *EventStore) AppendWithOutbox(event DomainEvent) error {
    // 1. Сохраняем событие
    s.Append(event)

    // 2. Сохраняем в outbox
    outboxEvent := OutboxEvent{
        EventID:   event.GetEventID(),
        EventType: event.GetEventType(),
        Payload:   serialize(event),
        CreatedAt: time.Now(),
    }
    s.outboxRepo.Save(outboxEvent)

    return nil
}

// Worker периодически читает outbox и публикует
func (w *OutboxWorker) Run() {
    ticker := time.NewTicker(1 * time.Second)

    for range ticker.C {
        events := w.repo.GetUnpublished(limit: 100)

        for _, event := range events {
            err := w.eventBus.Publish(deserialize(event.Payload))
            if err == nil {
                now := time.Now()
                event.PublishedAt = &now
                w.repo.Update(event)
            }
        }
    }
}
```

---

## Idempotency

### Проблема

```
Сценарий:
1. MessagePosted event → TagParserService
2. TagParserService обрабатывает → теги распарсены
3. Redis переотправляет событие (reconnect)
4. TagParserService обрабатывает повторно → дублирование?
```

### Решение: Processed Events Tracking

**Коллекция:** `processed_events`

```javascript
{
  "_id": ObjectId(),
  "eventId": "event-uuid",
  "handlerName": "TagParserService",
  "processedAt": ISODate("2025-09-30T10:00:00Z"),
  "expiresAt": ISODate("2025-10-07T10:00:00Z") // TTL = 7 дней
}
```

**Индексы:**

```javascript
// Compound unique для быстрой проверки
db.processed_events.createIndex(
  { "eventId": 1, "handlerName": 1 },
  { unique: true }
)

// TTL index для автоматической очистки
db.processed_events.createIndex(
  { "expiresAt": 1 },
  { expireAfterSeconds: 0 }
)
```

### Реализация

```go
type IdempotencyChecker interface {
    IsProcessed(eventID UUID, handlerName string) bool
    MarkProcessed(eventID UUID, handlerName string) error
}

type MongoIdempotencyChecker struct {
    collection *mongo.Collection
}

func (c *MongoIdempotencyChecker) IsProcessed(eventID UUID, handlerName string) bool {
    filter := bson.M{
        "eventId":     eventID,
        "handlerName": handlerName,
    }

    count, err := c.collection.CountDocuments(context.Background(), filter)
    if err != nil {
        log.Error("Failed to check idempotency", "error", err)
        return false
    }

    return count > 0
}

func (c *MongoIdempotencyChecker) MarkProcessed(eventID UUID, handlerName string) error {
    doc := bson.M{
        "eventId":     eventID,
        "handlerName": handlerName,
        "processedAt": time.Now(),
        "expiresAt":   time.Now().Add(7 * 24 * time.Hour), // TTL 7 дней
    }

    _, err := c.collection.InsertOne(context.Background(), doc)
    if err != nil {
        // Игнорируем duplicate key error (уже обработано)
        if mongo.IsDuplicateKeyError(err) {
            return nil
        }
        return err
    }

    return nil
}
```

### Использование в обработчиках

```go
type EventHandler interface {
    Handle(event DomainEvent) error
    GetName() string
}

type TagParserHandler struct {
    parser             *TagParser
    eventBus           EventBus
    idempotencyChecker IdempotencyChecker
}

func (h *TagParserHandler) Handle(event DomainEvent) error {
    // 1. Idempotency check
    if h.idempotencyChecker.IsProcessed(event.GetEventID(), h.GetName()) {
        log.Info("Event already processed, skipping",
            "eventId", event.GetEventID(),
            "handler", h.GetName())
        return nil
    }

    // 2. Обрабатываем событие
    err := h.processEvent(event)
    if err != nil {
        return err
    }

    // 3. Отмечаем как обработанное
    return h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())
}

func (h *TagParserHandler) GetName() string {
    return "TagParserHandler"
}
```

---

## Retry и Error Handling

### Стратегия

**Exponential Backoff + Dead Letter Queue**

```
1. Обработка события упала с ошибкой
2. Retry с exponential backoff (1s, 2s, 4s, 8s, 16s)
3. После MaxRetries → Dead Letter Queue
4. Ручной replay администратором
```

### Реализация

```go
type RetryConfig struct {
    MaxRetries  int           // 5
    InitialDelay time.Duration // 1 second
}

type EventProcessor struct {
    handler            EventHandler
    idempotencyChecker IdempotencyChecker
    dlqRepo            DeadLetterQueueRepository
    retryConfig        RetryConfig
}

func (p *EventProcessor) Process(event DomainEvent) {
    var lastError error

    for attempt := 0; attempt <= p.retryConfig.MaxRetries; attempt++ {
        err := p.handler.Handle(event)

        if err == nil {
            // Успешно обработано
            return
        }

        lastError = err

        log.Warn("Event handler failed, will retry",
            "handler", p.handler.GetName(),
            "eventId", event.GetEventID(),
            "attempt", attempt+1,
            "maxRetries", p.retryConfig.MaxRetries,
            "error", err)

        if attempt < p.retryConfig.MaxRetries {
            // Exponential backoff: 1s, 2s, 4s, 8s, 16s
            delay := p.retryConfig.InitialDelay * time.Duration(math.Pow(2, float64(attempt)))
            time.Sleep(delay)
        }
    }

    // Все попытки исчерпаны → Dead Letter Queue
    p.sendToDLQ(event, lastError)
}

func (p *EventProcessor) sendToDLQ(event DomainEvent, lastError error) {
    dlqEntry := DeadLetterEntry{
        ID:          uuid.New(),
        EventID:     event.GetEventID(),
        EventType:   event.GetEventType(),
        AggregateID: event.GetAggregateID(),
        Payload:     serialize(event),
        Error:       lastError.Error(),
        Attempts:    p.retryConfig.MaxRetries + 1,
        CreatedAt:   time.Now(),
        Status:      "pending", // pending, replayed, discarded
    }

    err := p.dlqRepo.Save(dlqEntry)
    if err != nil {
        log.Error("Failed to save to DLQ", "error", err, "eventId", event.GetEventID())
    }

    // Алерт администраторам
    p.alertService.SendAlert(AlertCritical, fmt.Sprintf(
        "Event processing failed: %s (EventID: %s)",
        event.GetEventType(),
        event.GetEventID(),
    ))
}
```

### Dead Letter Queue

**Коллекция:** `dead_letter_queue`

```javascript
{
  "_id": "dlq-entry-uuid",
  "eventId": "event-uuid",
  "eventType": "MessagePosted",
  "aggregateId": "chat-uuid",
  "payload": { /* full event data */ },
  "error": "Failed to parse tags: invalid syntax",
  "attempts": 6,
  "createdAt": ISODate("2025-09-30T10:00:00Z"),
  "status": "pending",        // pending, replayed, discarded
  "replayedAt": null,
  "replayedBy": null,
  "notes": ""
}
```

**Индексы:**

```javascript
db.dead_letter_queue.createIndex({ "status": 1, "createdAt": 1 })
db.dead_letter_queue.createIndex({ "eventType": 1, "createdAt": 1 })
db.dead_letter_queue.createIndex({ "eventId": 1 }, { unique: true })
```

### Ручной Replay

**Admin UI:**

```
GET /admin/dlq
→ Список событий в DLQ

POST /admin/dlq/{entryId}/replay
→ Повторная обработка события

POST /admin/dlq/{entryId}/discard
→ Отметить как "discarded" (игнорировать)
```

**Реализация replay:**

```go
func (s *DLQService) Replay(entryID UUID, adminUserID UUID) error {
    // 1. Загружаем entry из DLQ
    entry := s.repo.FindByID(entryID)
    if entry.Status != "pending" {
        return errors.New("entry already processed")
    }

    // 2. Десериализуем событие
    event := deserialize(entry.Payload)

    // 3. Публикуем событие заново в Event Bus
    err := s.eventBus.Publish(event)
    if err != nil {
        return err
    }

    // 4. Обновляем статус
    now := time.Now()
    entry.Status = "replayed"
    entry.ReplayedAt = &now
    entry.ReplayedBy = &adminUserID

    return s.repo.Update(entry)
}
```

---

## Event Ordering

### Проблема

```
User отправляет два сообщения подряд:
1. "#status In Progress"
2. "#status Done"

События могут обработаться в неправильном порядке:
1. StatusChanged (Done)
2. StatusChanged (In Progress)

→ Финальный статус = "In Progress" (неправильно!)
```

### Решение: Партиционирование по AggregateID

**Концепция:**
- События для одного aggregate обрабатываются последовательно
- События для разных aggregates обрабатываются параллельно

**Реализация:**

```go
type PartitionedEventBus struct {
    redis      *redis.Client
    partitions map[UUID]*Partition
    mu         sync.RWMutex
}

type Partition struct {
    aggregateID UUID
    eventQueue  chan DomainEvent
    handler     EventHandler
    processor   *EventProcessor
}

func (b *PartitionedEventBus) Subscribe(eventType string, handler EventHandler) {
    channel := fmt.Sprintf("events.%s", eventType)
    pubsub := b.redis.Subscribe(context.Background(), channel)

    go func() {
        for msg := range pubsub.Channel() {
            var eventMsg EventMessage
            json.Unmarshal([]byte(msg.Payload), &eventMsg)

            event := deserializeEvent(eventMsg)

            // Получаем или создаём партицию для aggregateID
            partition := b.getOrCreatePartition(event.GetAggregateID(), handler)

            // Отправляем событие в очередь партиции
            partition.eventQueue <- event
        }
    }()
}

func (b *PartitionedEventBus) getOrCreatePartition(aggregateID UUID, handler EventHandler) *Partition {
    b.mu.Lock()
    defer b.mu.Unlock()

    partition, exists := b.partitions[aggregateID]
    if !exists {
        partition = &Partition{
            aggregateID: aggregateID,
            eventQueue:  make(chan DomainEvent, 100),
            handler:     handler,
            processor:   NewEventProcessor(handler, b.idempotencyChecker, b.dlqRepo),
        }

        // Запускаем worker для партиции
        go partition.run()

        b.partitions[aggregateID] = partition
    }

    return partition
}

func (p *Partition) run() {
    for event := range p.eventQueue {
        // События обрабатываются последовательно для одного aggregate
        p.processor.Process(event)
    }
}
```

**Гарантии:**
- ✅ События для `chat-uuid-1` обрабатываются в порядке поступления
- ✅ События для `chat-uuid-2` обрабатываются параллельно с `chat-uuid-1`
- ✅ Нет race conditions на одном aggregate

**Ограничения:**
- Партиции живут в памяти (при перезапуске создаются заново)
- Если много aggregates → много горутин (но они idle, когда нет событий)

---

## Complete Event Flow — Детальный сценарий

### Сценарий: User меняет статус задачи

```
User в чате задачи:
"Закончил работу
#status Done"
```

#### Шаг 1: HTTP Request

```http
POST /api/chats/chat-uuid/messages HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbG...
Content-Type: application/json

{
  "content": "Закончил работу\n#status Done"
}
```

#### Шаг 2: Handler → Service

```go
func (h *ChatHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
    chatID := chi.URLParam(r, "chatId")
    userID := GetUserIDFromContext(r.Context())
    correlationID := uuid.New() // для трейсинга

    var req PostMessageRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Проверка прав
    chat := h.chatRepo.FindByID(chatID)
    if !hasWriteAccess(userID, chat) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // Создаём команду
    cmd := PostMessageCommand{
        ChatID:        chatID,
        UserID:        userID,
        Content:       req.Content,
        CorrelationID: correlationID,
    }

    // Отправляем в service
    message, err := h.chatService.PostMessage(cmd)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(message)
}
```

#### Шаг 3: ChatService — Command to Event

```go
func (s *ChatService) PostMessage(cmd PostMessageCommand) (*Message, error) {
    // 1. Загружаем Chat aggregate из Event Store
    chat, err := s.loadChatFromEvents(cmd.ChatID)
    if err != nil {
        return nil, err
    }

    // 2. Применяем команду к aggregate (бизнес-логика)
    message, err := chat.PostMessage(cmd.UserID, cmd.Content)
    if err != nil {
        return nil, err
    }

    // 3. Chat aggregate сгенерировал событие
    events := chat.GetUncommittedEvents()
    // events = [MessagePosted{...}]

    // 4. Сохраняем события в Event Store
    for _, event := range events {
        event.SetMetadata(EventMetadata{
            CorrelationID: cmd.CorrelationID,
            CausationID:   uuid.Nil, // первое событие в цепочке
            UserID:        cmd.UserID,
        })

        err := s.eventStore.Append(event)
        if err != nil {
            return nil, err
        }
    }

    // 5. Публикуем события в Event Bus
    for _, event := range events {
        s.eventBus.Publish(event)
    }

    // 6. Сохраняем Message в read model (messages collection)
    s.messageRepo.Save(message)

    // 7. Broadcast через WebSocket
    s.wsHub.BroadcastToChat(cmd.ChatID, WebSocketMessage{
        Type: "chat.message.posted",
        Data: message,
    })

    // 8. Очищаем uncommitted events в aggregate
    chat.MarkEventsAsCommitted()

    return message, nil
}
```

#### Шаг 4: Event Store — Append

```javascript
// MongoDB: events collection
{
  "_id": "event-uuid-1",
  "aggregateId": "chat-uuid",
  "aggregateType": "Chat",
  "eventType": "MessagePosted",
  "eventData": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "authorId": "user-uuid",
    "content": "Закончил работу\n#status Done",
    "timestamp": "2025-09-30T10:00:00.000Z"
  },
  "version": 142,
  "timestamp": "2025-09-30T10:00:00.123Z",
  "metadata": {
    "correlationId": "correlation-uuid",
    "causationId": "00000000-0000-0000-0000-000000000000",
    "userId": "user-uuid"
  }
}
```

#### Шаг 5: Event Bus — Publish

```
Redis PUBLISH events.MessagePosted
{
  "eventId": "event-uuid-1",
  "eventType": "MessagePosted",
  "aggregateId": "chat-uuid",
  "payload": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "authorId": "user-uuid",
    "content": "Закончил работу\n#status Done",
    "timestamp": "2025-09-30T10:00:00.000Z"
  },
  "metadata": {
    "correlationId": "correlation-uuid",
    "causationId": "00000000-0000-0000-0000-000000000000",
    "userId": "user-uuid"
  }
}
```

#### Шаг 6: TagParserHandler — Subscriber

```go
func (h *TagParserHandler) Handle(event DomainEvent) error {
    messagePosted := event.(*MessagePosted)

    // 1. Idempotency check
    if h.idempotencyChecker.IsProcessed(event.GetEventID(), h.GetName()) {
        return nil
    }

    // 2. Парсим теги
    tags := h.parser.Parse(messagePosted.Content)

    if len(tags) == 0 {
        // Нет тегов → просто отмечаем как обработанное
        h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())
        return nil
    }

    // 3. Генерируем команды из тегов
    commands := []Command{}
    for _, tag := range tags {
        cmd, err := h.tagToCommand(tag, messagePosted.ChatID, messagePosted.AuthorID)
        if err != nil {
            log.Warn("Invalid tag", "tag", tag, "error", err)
            // Невалидный тег → игнорируем, продолжаем
            continue
        }
        commands = append(commands, cmd)
    }

    // 4. Создаём и сохраняем событие TagsParsed
    tagsParsedEvent := TagsParsed{
        EventID:   uuid.New(),
        MessageID: messagePosted.MessageID,
        ChatID:    messagePosted.ChatID,
        Commands:  commands,
        Timestamp: time.Now(),
    }

    tagsParsedEvent.SetMetadata(EventMetadata{
        CorrelationID: event.GetMetadata().CorrelationID, // тот же correlation
        CausationID:   event.GetEventID(),                // MessagePosted вызвало это
        UserID:        event.GetMetadata().UserID,
    })

    // Сохраняем в Event Store
    h.eventStore.Append(tagsParsedEvent)

    // Публикуем в Event Bus
    h.eventBus.Publish(tagsParsedEvent)

    // 5. Отмечаем MessagePosted как обработанное
    h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())

    return nil
}
```

#### Шаг 7: Event Store + Event Bus

```javascript
// events collection
{
  "_id": "event-uuid-2",
  "aggregateId": "chat-uuid",
  "aggregateType": "Chat",
  "eventType": "TagsParsed",
  "eventData": {
    "messageId": "msg-uuid",
    "chatId": "chat-uuid",
    "commands": [
      {
        "type": "ChangeStatusCommand",
        "status": "Done"
      }
    ],
    "timestamp": "2025-09-30T10:00:00.234Z"
  },
  "version": 143,
  "timestamp": "2025-09-30T10:00:00.234Z",
  "metadata": {
    "correlationId": "correlation-uuid",    // тот же
    "causationId": "event-uuid-1",         // MessagePosted
    "userId": "user-uuid"
  }
}
```

```
Redis PUBLISH events.TagsParsed { ... }
```

#### Шаг 8: CommandExecutorHandler — Subscriber

```go
func (h *CommandExecutorHandler) Handle(event DomainEvent) error {
    tagsParsed := event.(*TagsParsed)

    // 1. Idempotency check
    if h.idempotencyChecker.IsProcessed(event.GetEventID(), h.GetName()) {
        return nil
    }

    // 2. Загружаем TaskEntity (read model)
    task := h.taskRepo.FindByChatID(tagsParsed.ChatID)
    if task == nil {
        // Не typed чат → игнорируем
        h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())
        return nil
    }

    // 3. Применяем команды
    results := []CommandResult{}

    for _, cmd := range tagsParsed.Commands {
        result := h.executeCommand(task, cmd, tagsParsed.Metadata.UserID)
        results = append(results, result)

        if result.Success {
            // Сохраняем событие в Event Store
            result.Event.SetMetadata(EventMetadata{
                CorrelationID: event.GetMetadata().CorrelationID,
                CausationID:   event.GetEventID(), // TagsParsed вызвало это
                UserID:        event.GetMetadata().UserID,
            })

            h.eventStore.Append(result.Event)
            h.eventBus.Publish(result.Event)
        }
    }

    // 4. Обновляем TaskEntity (read model)
    h.taskRepo.Save(task)

    // 5. Отправляем feedback в чат (бот-сообщение)
    h.sendFeedbackToChat(tagsParsed.ChatID, results)

    // 6. Broadcast обновление канбана через WebSocket
    h.wsHub.BroadcastToWorkspace(task.WorkspaceID, WebSocketMessage{
        Type: "task.updated",
        Data: task,
    })

    // 7. Отмечаем TagsParsed как обработанное
    h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())

    return nil
}

func (h *CommandExecutorHandler) executeCommand(
    task *TaskEntity,
    cmd Command,
    executedBy UUID,
) CommandResult {
    switch c := cmd.(type) {
    case ChangeStatusCommand:
        oldStatus := task.State.Status

        err := task.ChangeStatus(c.Status, executedBy)
        if err != nil {
            return CommandResult{
                Success: false,
                Error:   err,
                Message: fmt.Sprintf("❌ Invalid status '%s'. Available: %s",
                    c.Status, getValidStatuses(task.Type)),
            }
        }

        event := StatusChanged{
            EventID:   uuid.New(),
            TaskID:    task.ID,
            OldStatus: oldStatus,
            NewStatus: c.Status,
            ChangedBy: executedBy,
            Timestamp: time.Now(),
        }

        return CommandResult{
            Success: true,
            Event:   event,
            Message: fmt.Sprintf("✅ Status changed to %s", c.Status),
        }

    // ... другие команды
    }
}
```

#### Шаг 9: Event Store + Event Bus

```javascript
// events collection
{
  "_id": "event-uuid-3",
  "aggregateId": "task-uuid",
  "aggregateType": "TaskEntity",
  "eventType": "StatusChanged",
  "eventData": {
    "taskId": "task-uuid",
    "oldStatus": "In Progress",
    "newStatus": "Done",
    "changedBy": "user-uuid",
    "timestamp": "2025-09-30T10:00:00.345Z"
  },
  "version": 23,
  "timestamp": "2025-09-30T10:00:00.345Z",
  "metadata": {
    "correlationId": "correlation-uuid",
    "causationId": "event-uuid-2",  // TagsParsed
    "userId": "user-uuid"
  }
}
```

```
Redis PUBLISH events.StatusChanged { ... }
```

#### Шаг 10: NotificationHandler — Subscriber

```go
func (h *NotificationHandler) Handle(event DomainEvent) error {
    statusChanged := event.(*StatusChanged)

    // 1. Idempotency check
    if h.idempotencyChecker.IsProcessed(event.GetEventID(), h.GetName()) {
        return nil
    }

    // 2. Определяем получателей уведомления
    recipients := h.determineRecipients(statusChanged)
    // Recipients: [assignee, chat participants, watchers]

    // 3. Создаём уведомления
    for _, recipientID := range recipients {
        notification := Notification{
            ID:         uuid.New(),
            UserID:     recipientID,
            Type:       "task.status_changed",
            Title:      "Task status changed",
            Message:    fmt.Sprintf("Task status changed to %s", statusChanged.NewStatus),
            ResourceID: statusChanged.TaskID,
            CreatedAt:  time.Now(),
            ReadAt:     nil,
        }

        h.notificationRepo.Save(notification)

        // Отправляем через WebSocket (если онлайн)
        h.wsHub.SendToUser(recipientID, WebSocketMessage{
            Type: "notification.new",
            Data: notification,
        })

        // Опционально: email (асинхронно)
        if h.shouldSendEmail(recipientID, notification) {
            h.emailQueue.Enqueue(EmailTask{
                To:      h.userRepo.GetEmail(recipientID),
                Subject: notification.Title,
                Body:    notification.Message,
            })
        }
    }

    // 4. Отмечаем StatusChanged как обработанное
    h.idempotencyChecker.MarkProcessed(event.GetEventID(), h.GetName())

    return nil
}
```

#### Итоговый Event Chain

```
[event-uuid-1] MessagePosted
    ↓ (causationId)
[event-uuid-2] TagsParsed
    ↓ (causationId)
[event-uuid-3] StatusChanged
    ↓ (causationId)
[event-uuid-4] UserNotified

Все события имеют одинаковый correlationId = "correlation-uuid"
→ Можно проследить всю цепочку обработки запроса
```

---

## Aggregate Recovery

### Восстановление из Event Stream

```go
func (r *ChatRepository) LoadChat(chatID UUID) (*Chat, error) {
    // 1. Пытаемся загрузить snapshot (если есть)
    snapshot, err := r.snapshotRepo.Load(chatID)
    if err == nil && snapshot != nil {
        // 2. Загружаем события после snapshot
        events := r.eventStore.LoadAfter(chatID, snapshot.Version)

        // 3. Восстанавливаем aggregate из snapshot
        chat := snapshot.ToAggregate()

        // 4. Применяем события после snapshot
        for _, event := range events {
            chat.Apply(event)
        }

        return chat, nil
    }

    // 5. Snapshot нет → загружаем все события
    events := r.eventStore.Load(chatID)

    // 6. Создаём пустой aggregate
    chat := &Chat{ID: chatID}

    // 7. Применяем все события
    for _, event := range events {
        chat.Apply(event)
    }

    return chat, nil
}
```

### Snapshots (оптимизация)

**Проблема:** Aggregate с 10000+ событий долго восстанавливать.

**Решение:** Периодические snapshots.

```javascript
// Коллекция: chat_snapshots
{
  "_id": "chat-uuid",
  "type": "task",
  "createdBy": "user-uuid",
  "createdAt": ISODate("2025-09-30T09:00:00Z"),
  "participants": [
    { "userId": "user-1", "role": "admin", "joinedAt": "..." }
  ],
  "version": 9900,  // версия события, на котором сделан snapshot
  "snapshotAt": ISODate("2025-09-30T12:00:00Z")
}
```

**Стратегия создания:**
- Каждые N событий (например, каждые 100)
- Или периодически (каждые 1 час)
- Асинхронно (не блокирует обработку)

```go
// Snapshot Worker
func (w *SnapshotWorker) Run() {
    ticker := time.NewTicker(10 * time.Minute)

    for range ticker.C {
        // Находим aggregates, которым нужен snapshot
        aggregates := w.findAggregatesNeedingSnapshot()

        for _, aggID := range aggregates {
            // Загружаем aggregate
            chat := w.chatRepo.LoadChat(aggID)

            // Создаём snapshot
            snapshot := ChatSnapshot{
                ID:           chat.ID,
                Type:         chat.Type,
                CreatedBy:    chat.CreatedBy,
                CreatedAt:    chat.CreatedAt,
                Participants: chat.Participants,
                Version:      chat.Version,
                SnapshotAt:   time.Now(),
            }

            w.snapshotRepo.Save(snapshot)
        }
    }
}
```

---

## Monitoring и Observability

### Метрики

```go
// Prometheus metrics
var (
    eventsPublished = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "events_published_total",
            Help: "Total number of events published",
        },
        []string{"event_type"},
    )

    eventsProcessed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "events_processed_total",
            Help: "Total number of events processed",
        },
        []string{"handler", "event_type", "status"}, // status: success, error
    )

    eventProcessingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "event_processing_duration_seconds",
            Help:    "Event processing duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"handler", "event_type"},
    )

    dlqSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "dead_letter_queue_size",
            Help: "Number of events in Dead Letter Queue",
        },
    )
)

// В обработчике событий
func (p *EventProcessor) Process(event DomainEvent) {
    start := time.Now()
    err := p.handler.Handle(event)
    duration := time.Since(start)

    status := "success"
    if err != nil {
        status = "error"
    }

    eventsProcessed.WithLabelValues(
        p.handler.GetName(),
        event.GetEventType(),
        status,
    ).Inc()

    eventProcessingDuration.WithLabelValues(
        p.handler.GetName(),
        event.GetEventType(),
    ).Observe(duration.Seconds())
}
```

### Distributed Tracing

```go
// Используем OpenTelemetry для трейсинга

func (s *ChatService) PostMessage(cmd PostMessageCommand) (*Message, error) {
    ctx := context.Background()

    // Создаём span для операции
    ctx, span := tracer.Start(ctx, "ChatService.PostMessage",
        trace.WithAttributes(
            attribute.String("chat.id", cmd.ChatID.String()),
            attribute.String("user.id", cmd.UserID.String()),
        ))
    defer span.End()

    // ... бизнес-логика

    // При публикации события передаём trace context
    event.SetMetadata(EventMetadata{
        CorrelationID: cmd.CorrelationID,
        TraceID:       span.SpanContext().TraceID().String(),
        SpanID:        span.SpanContext().SpanID().String(),
    })

    return message, nil
}

// В обработчике события извлекаем trace context
func (h *TagParserHandler) Handle(event DomainEvent) error {
    // Извлекаем trace context из metadata
    traceID := event.GetMetadata().TraceID
    spanID := event.GetMetadata().SpanID

    // Продолжаем trace
    ctx := createContextFromTrace(traceID, spanID)
    ctx, span := tracer.Start(ctx, "TagParserHandler.Handle")
    defer span.End()

    // ... обработка
}
```

### Logging

```go
// Structured logging с correlation ID

log.Info("Event published",
    "eventId", event.GetEventID(),
    "eventType", event.GetEventType(),
    "aggregateId", event.GetAggregateID(),
    "correlationId", event.GetMetadata().CorrelationID,
    "causationId", event.GetMetadata().CausationID,
)

log.Warn("Event handler failed",
    "handler", handler.GetName(),
    "eventId", event.GetEventID(),
    "correlationId", event.GetMetadata().CorrelationID,
    "error", err,
)

// Можно искать все логи по одному запросу:
// grep correlationId=correlation-uuid logs.txt
```

---

## Резюме архитектурных решений

| Аспект | Решение | Обоснование |
|--------|---------|-------------|
| **Event Store** | Одна MongoDB коллекция | Простота, достаточно индексов |
| **Event Bus** | Redis Pub/Sub по типу события | Проще роутинг, меньше шума |
| **Delivery** | At-most-once (MVP) | Простота, достаточно для MVP |
| **Idempotency** | Processed events с TTL 7 дней | Защита от дубликатов |
| **Retry** | Exponential backoff + DLQ | Устойчивость к временным сбоям |
| **DLQ Replay** | Ручной через Admin UI | Контроль администратора |
| **Ordering** | Партиционирование по aggregateId | Гарантия порядка для aggregate |
| **Versioning** | Flexible schema (MongoDB) | Простота эволюции |
| **Snapshots** | Периодические (каждые 100 событий) | Оптимизация восстановления |
| **Tracing** | OpenTelemetry + correlationId | Debugging, observability |

## V2 Enhancements

**Transactional Outbox:**
- At-least-once delivery гарантия
- Worker публикует события из outbox table
- Retry при неудаче публикации

**Saga Pattern:**
- Координация multi-step процессов
- Откат при ошибках (compensating transactions)
- State machine для отслеживания прогресса

**Event Sourcing Projections:**
- Множественные read models из одного event stream
- Rebuild проекций из Event Store
- Versioned projections

**CQRS Read Replicas:**
- Оптимизированные read models для разных use cases
- Eventual consistency между write и read
- Кеширование в Redis

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. ✅ Детальная грамматика тегов
4. ✅ Права доступа и security model
5. ✅ Event flow детально
6. **TODO:** API контракты (HTTP + WebSocket)
7. **TODO:** Структура кода (внутри internal/)
8. **TODO:** План реализации MVP (roadmap)
