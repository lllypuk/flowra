# Domain Model - Chat-Based Task Tracker

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

Система таск-трекера построена на концепции **chat-first**: любая сущность (задача, баг, эпик) является чатом, управление происходит через теги в сообщениях. Используется подход Event Sourcing с eventual consistency.

## Core Use Cases (MVP)

### Must-Have для MVP

1. **Создание задачи через чат**
   - Вариант А: В общем чате проекта написать `#task Название` → создаётся отдельный чат-задача
   - Вариант Б: Кнопка "New Task" → открывается чат, первое сообщение с `#task Название` инициализирует задачу

2. **Изменение статуса**
   - Через чат: `#status In Progress`
   - Через канбан: drag-n-drop → автоматически добавляется сообщение от пользователя в чат

3. **Назначение исполнителя**
   - `#assignee @username` в сообщении
   - Исполнитель получает уведомление

4. **Просмотр задач на канбане**
   - Актуальное состояние всех задач
   - Группировка по статусам
   - Фильтрация, поиск

5. **Чтение истории задачи**
   - Весь чат с обсуждениями
   - История изменений через сообщения с тегами

6. **Обсуждение в контексте задачи**
   - Обычные сообщения без тегов = обсуждение
   - Упоминания @user

7. **Уведомления**
   - Назначение, упоминание, изменение статуса

8. **Поиск по задачам**
   - По содержимому чата, тегам, участникам

### Отложено на V2

- Кастомизация статусных моделей (в MVP: фиксированные статусы)
- Метрики и аналитика
- Связи между задачами (#parent, #blocks)
- Дедлайны и автоматические напоминания
- Шаблоны задач

## Ключевые архитектурные решения

### 1. Синтаксис тегов

```
#task Название задачи       → создаёт Task
#bug Описание бага          → создаёт Bug
#epic Название эпика        → создаёт Epic

#status In Progress         → меняет статус
#assignee @username         → назначает исполнителя
#priority High              → устанавливает приоритет
#due 2025-10-20            → устанавливает дедлайн
```

**Правила парсинга:**
- Теги срабатывают только в **начале сообщения или отдельной строкой**
- В обычном тексте `#status` не срабатывает (защита от случайного упоминания)
- Множественные теги в одном сообщении: `#status Done #assignee @bob` → оба применяются

### 2. Типизация сущностей

- **Переопределение типа:** можно написать `#bug` в чате-задаче → задача становится багом
- **Чат без типа:** обычный discussion-чат не отображается на канбане
- **Превращение чата в задачу:** в любой момент можно добавить `#task` → чат появляется на доске

### 3. Управление через drag-n-drop

- Перетащили карточку в другую колонку → автоматически добавляется сообщение **от имени пользователя**: `@alex: #status In Progress`
- Не system-сообщение, а обычное от юзера

### 4. Удаление сообщений с тегами

- **Теги необратимы:** удаление сообщения с тегом НЕ откатывает изменение
- Статус остаётся, история сохраняется

### 5. Идентификация

- `Chat.id == TaskEntity.id` (один UUID для обоих)
- Один чат может иметь максимум одну типизированную сущность

## Bounded Contexts

### 1. Chat Context (Core Domain)
**Ответственность:**
- Управление чатами (создание, удаление)
- Поток сообщений
- Участники чата
- Реал-тайм коммуникация (WebSocket)

**Ключевые концепции:**
- Chat (aggregate root)
- Message (entity)
- Participant (value object)

### 2. Task Management Context
**Ответственность:**
- Задачи, баги, эпики как специализация чатов
- Статусы и их валидация
- Канбан-доска (материализованное представление)
- Типизация сущностей

**Ключевые концепции:**
- TaskEntity (aggregate)
- EntityState (value object)
- StatusTransition (value object)

### 3. Collaboration Context
**Ответственность:**
- Упоминания пользователей (@mentions)
- Уведомления
- Присутствие (presence tracking)
- Права доступа

### 4. Tag Processing Context
**Ответственность:**
- Парсинг тегов из сообщений
- Валидация синтаксиса тегов
- Генерация команд из тегов

## Domain Model

### Aggregates

#### Chat (Aggregate Root)

```go
type ChatType string

const (
    Discussion ChatType = "discussion"
    Task       ChatType = "task"
    Bug        ChatType = "bug"
    Epic       ChatType = "epic"
)

type Chat struct {
    ID           UUID
    Type         ChatType
    CreatedBy    UserID
    CreatedAt    time.Time
    Participants []Participant
    TaskEntity   *TaskEntity // nil для discussion

    // Event sourcing
    version           int
    uncommittedEvents []DomainEvent
}

// Commands
func (c *Chat) PostMessage(author UserID, content string) (*Message, error)
func (c *Chat) ConvertToTask(title string, initiatedBy UserID) error
func (c *Chat) ConvertToBug(title string, initiatedBy UserID) error
func (c *Chat) AddParticipant(userID UserID) error
func (c *Chat) RemoveParticipant(userID UserID) error

// Event application (для восстановления из event stream)
func (c *Chat) Apply(event DomainEvent) error

// Helpers
func (c *Chat) IsTyped() bool { return c.Type != Discussion }
```

**Инвариант:** Chat с типом Task/Bug/Epic ДОЛЖЕН иметь TaskEntity.

#### TaskEntity

```go
type TaskEntity struct {
    ID      UUID // == Chat.ID
    Title   string
    State   EntityState

    version int
}

type EntityState struct {
    Status       string
    Assignee     *UserID
    Priority     *string
    DueDate      *time.Time
    CustomFields map[string]string // для кастомных тегов (#sprint, #component, etc.)
}

// Commands
func (t *TaskEntity) ChangeStatus(newStatus string, by UserID) error
func (t *TaskEntity) Assign(userID UserID, by UserID) error
func (t *TaskEntity) SetPriority(priority string, by UserID) error
func (t *TaskEntity) SetDueDate(date time.Time, by UserID) error
func (t *TaskEntity) SetCustomField(key, value string, by UserID) error
```

**Инвариант:** Title не может быть пустым.

#### Message (Entity)

```go
type Message struct {
    ID              UUID
    ChatID          UUID
    Author          UserID
    Content         string
    CreatedAt       time.Time
    Tags            []ParsedTag
    IsSystemMessage bool
}

type ParsedTag struct {
    Key   string  // "status", "assignee", "priority"
    Value string  // "Done", "@bob", "High"
}
```

**Примечание:** Messages хранятся отдельно от Chat aggregate (event stream).

### Value Objects

```go
type Participant struct {
    UserID   UUID
    JoinedAt time.Time
    Role     ParticipantRole // admin, member, viewer
}

type StatusTransition struct {
    From      string
    To        string
    By        UserID
    MessageID UUID
    Timestamp time.Time
}
```

### Domain Events

#### Chat Context Events

```go
type ChatCreated struct {
    ChatID    UUID
    Type      ChatType
    CreatedBy UserID
    Timestamp time.Time
}

type MessagePosted struct {
    ChatID    UUID
    MessageID UUID
    Author    UserID
    Content   string
    Timestamp time.Time
}

type ChatTypeChanged struct {
    ChatID      UUID
    OldType     ChatType
    NewType     ChatType
    InitiatedBy UserID
    Timestamp   time.Time
}

type ParticipantJoined struct {
    ChatID    UUID
    UserID    UUID
    Timestamp time.Time
}
```

#### Task Management Events

```go
type TaskCreated struct {
    TaskID    UUID
    ChatID    UUID
    Title     string
    Type      ChatType
    CreatedBy UserID
    Timestamp time.Time
}

type StatusChanged struct {
    TaskID      UUID
    OldStatus   string
    NewStatus   string
    ChangedBy   UserID
    MessageID   UUID
    Timestamp   time.Time
}

type AssigneeChanged struct {
    TaskID      UUID
    OldAssignee *UserID
    NewAssignee *UserID
    ChangedBy   UserID
    MessageID   UUID
    Timestamp   time.Time
}

type PriorityChanged struct {
    TaskID      UUID
    OldPriority *string
    NewPriority string
    ChangedBy   UserID
    MessageID   UUID
    Timestamp   time.Time
}
```

## Domain Services

### TagParserService

```go
type TagParserService struct{}

func (s *TagParserService) Parse(content string) []ParsedTag
func (s *TagParserService) ExtractCommands(tags []ParsedTag) []Command
func (s *TagParserService) IsValidTagSyntax(content string) bool
```

**Ответственность:**
- Парсинг текста сообщения
- Извлечение тегов согласно грамматике
- Генерация команд из тегов

### CommandExecutor

```go
type CommandExecutor struct {
    taskRepo TaskRepository
    eventBus EventBus
}

func (e *CommandExecutor) Execute(taskID UUID, cmd Command, executedBy UserID) error
```

**Ответственность:**
- Применение команд к TaskEntity
- Валидация бизнес-правил
- Публикация domain events

### ChatToTaskConverter

```go
type ChatToTaskConverter struct {
    chatRepo ChatRepository
}

func (c *ChatToTaskConverter) Convert(chatID UUID, toType ChatType, title string, by UserID) error
```

**Ответственность:**
- Превращение Discussion в Task/Bug/Epic
- Инициализация TaskEntity
- Валидация возможности конвертации

### NotificationService

```go
type NotificationService struct{}

func (s *NotificationService) DetermineRecipients(event DomainEvent) []UserID
func (s *NotificationService) CreateNotification(event DomainEvent, recipient UserID) Notification
```

**Ответственность:**
- Определение получателей уведомлений
- Создание уведомлений на основе domain events

## Context Integration

### Event Flow

```
User action
    ↓
[Chat Context] PostMessage
    ↓
MessagePosted event → Event Bus (Redis Pub/Sub)
    ↓
[Tag Processing Context] TagParserService
    ↓
TagsParsed event → Commands
    ↓
[Task Management Context] CommandExecutor
    ↓
StatusChanged / AssigneeChanged events
    ↓
[Collaboration Context] NotificationService
    ↓
Notification sent to user
```

### Integration Patterns

1. **Chat → Tag Processing**
   - Pattern: Publisher/Subscriber
   - Event: `MessagePosted`
   - Асинхронная обработка

2. **Tag Processing → Task Management**
   - Pattern: Command via Events
   - Event: `TagsParsed` с массивом команд
   - Eventual consistency

3. **Task Management → Collaboration**
   - Pattern: Publisher/Subscriber
   - Events: `StatusChanged`, `AssigneeChanged`
   - Асинхронная обработка

### Eventual Consistency

**Сценарий:**
```
1. User: "#status Done"
2. MessagePosted сохранено ✅
3. [асинхронно] TagsParsed
4. [асинхронно] StatusChanged
5. [асинхронно] Канбан обновлён
```

**Время консистентности:** < 100ms (Redis Pub/Sub)

**Обработка ошибок:**
- Если валидация команды упала (например, неверный статус):
  - Сообщение остаётся в чате
  - Публикуется `CommandValidationFailed` event
  - Бот отвечает в чат: "❌ Invalid status 'Dne'. Available: To Do, In Progress, Done"

## Бизнес-правила (MVP)

### Статусы (hardcoded в MVP)

```go
var TaskStatuses = []string{"To Do", "In Progress", "Done"}
var BugStatuses = []string{"New", "Investigating", "Fixed", "Verified"}
var EpicStatuses = []string{"Planned", "In Progress", "Completed"}
```

**V2:** Кастомизация статусных моделей, валидация переходов (state machine).

### Валидация

**MVP (минимальная валидация):**
- Статус должен быть из списка допустимых для типа
- Title не может быть пустым
- Assignee должен существовать в системе
- Due date (если указан) должен быть парсируемым

**V2 (дополнительные правила):**
- Нельзя установить Done без assignee
- Нельзя установить due_date в прошлом
- Права на изменение статуса (assignee vs admin)

### CustomFields

```go
CustomFields map[string]string
```

Любой тег, не являющийся системным (status, assignee, priority, due, type), сохраняется как custom field:

```
#sprint Sprint-42  → CustomFields["sprint"] = "Sprint-42"
#component Auth    → CustomFields["component"] = "Auth"
```

## Persistence Strategy

### Event Sourcing Architecture

**Документная БД:** MongoDB или Cassandra (TBD)

**Подход:** Chat-first Event Sourcing с материализованными представлениями

### Collections/Tables

#### 1. Events (Event Store)

```json
{
  "_id": "uuid",
  "aggregateId": "chat-uuid",
  "aggregateType": "Chat",
  "eventType": "MessagePosted",
  "eventData": {
    "messageId": "msg-uuid",
    "author": "user-uuid",
    "content": "#status Done",
    "timestamp": "2025-09-30T10:00:00Z"
  },
  "version": 42,
  "timestamp": "2025-09-30T10:00:00Z",
  "metadata": {
    "correlationId": "correlation-uuid",
    "causationId": "causation-uuid"
  }
}
```

**Индексы:**
- `aggregateId + version` (unique) — для загрузки событий aggregate
- `timestamp` — для хронологического порядка
- `eventType` — для проекций

#### 2. Chat Snapshots (Read Model)

```json
{
  "_id": "chat-uuid",
  "type": "task",
  "createdBy": "user-uuid",
  "createdAt": "2025-09-30T09:00:00Z",
  "participants": [
    {"userId": "user-1", "joinedAt": "...", "role": "admin"},
    {"userId": "user-2", "joinedAt": "...", "role": "member"}
  ],
  "version": 42,
  "snapshotAt": "2025-09-30T10:00:00Z"
}
```

**Обновление:** Каждые N событий (например, каждые 50) или по времени.

#### 3. Task Projections (Read Model для канбана)

```json
{
  "_id": "task-uuid",
  "chatId": "chat-uuid",
  "type": "task",
  "title": "Implement authentication",
  "status": "In Progress",
  "assignee": "user-uuid",
  "priority": "High",
  "dueDate": "2025-10-20",
  "customFields": {
    "sprint": "Sprint-42",
    "component": "Auth"
  },
  "createdAt": "2025-09-30T09:00:00Z",
  "updatedAt": "2025-09-30T10:00:00Z",
  "lastMessageId": "msg-uuid"
}
```

**Индексы:**
- `status` — для группировки на канбане
- `assignee` — для фильтрации "My Tasks"
- `type` — для фильтрации Task/Bug/Epic
- `customFields.sprint` — для фильтрации по спринтам

#### 4. Messages (отдельная коллекция)

```json
{
  "_id": "msg-uuid",
  "chatId": "chat-uuid",
  "author": "user-uuid",
  "content": "#status Done",
  "tags": [
    {"key": "status", "value": "Done"}
  ],
  "isSystemMessage": false,
  "createdAt": "2025-09-30T10:00:00Z"
}
```

**Индексы:**
- `chatId + createdAt` — для загрузки истории чата
- `tags.key` — для поиска по тегам

### Восстановление Aggregate

```go
func LoadChat(aggregateId UUID) (*Chat, error) {
    // 1. Попытаться загрузить snapshot
    snapshot, err := snapshotRepo.Load(aggregateId)
    if err == nil {
        // 2. Загрузить события после snapshot
        events := eventStore.LoadAfter(aggregateId, snapshot.Version)
        chat := snapshot.ToAggregate()
        for _, event := range events {
            chat.Apply(event)
        }
        return chat, nil
    }

    // 3. Если snapshot нет, загрузить все события
    events := eventStore.Load(aggregateId)
    chat := &Chat{}
    for _, event := range events {
        chat.Apply(event)
    }
    return chat, nil
}
```

### Материализация Read Models

**Механизм:** Event handlers подписаны на события, обновляют проекции.

```go
// Пример: обновление Task Projection при StatusChanged
func (h *TaskProjectionHandler) Handle(event StatusChanged) {
    taskProjection := h.repo.FindByID(event.TaskID)
    taskProjection.Status = event.NewStatus
    taskProjection.UpdatedAt = event.Timestamp
    h.repo.Save(taskProjection)
}
```

## Technology Stack (MVP)

- **Event Bus:** Redis Pub/Sub (eventual consistency)
- **Database:** MongoDB (документная модель, гибкость схемы)
- **Backend:** Go 1.25+ с Echo v4
- **WebSocket:** Горутины + channels для реал-тайм
- **Frontend:** HTMX 2 + Pico CSS v2

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. **TODO:** Детальная грамматика тегов
4. **TODO:** Права доступа и security model
5. **TODO:** API контракты (HTTP + WebSocket)
6. **TODO:** Структура кода (внутри internal/)
7. **TODO:** План реализации MVP (roadmap)
