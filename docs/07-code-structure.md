# Code Structure — Project Layout

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

Проект использует **Clean Architecture** с разделением на слои по ответственности. Структура организована по принципу **layered architecture** с явными границами между слоями.

## Принципы организации

- **Domain-первый подход** — бизнес-логика независима от фреймворков
- **Dependency Rule** — зависимости направлены внутрь (к domain)
- **Явные зависимости** — constructor injection, без глобальных переменных
- **Bounded Contexts** — логическое разделение доменов
- **Testability** — каждый слой тестируется изолированно

---

## Project Layout

```
new-flowra/
├── cmd/                           # Application entry points
│   ├── api/                      # HTTP API + WebSocket server
│   │   └── main.go
│   ├── worker/                   # Background workers (event handlers)
│   │   └── main.go
│   └── migrator/                 # Database migrations runner
│       └── main.go
│
├── internal/                      # Private application code
│   ├── domain/                   # Domain layer (business logic)
│   │   ├── chat/                # Chat aggregate
│   │   ├── task/                # Task aggregate
│   │   ├── user/                # User aggregate
│   │   ├── workspace/           # Workspace aggregate
│   │   ├── notification/        # Notification aggregate
│   │   └── event/               # Domain events
│   │
│   ├── application/              # Application layer (use cases)
│   │   ├── chat/                # Chat use cases
│   │   ├── task/                # Task use cases
│   │   ├── workspace/           # Workspace use cases
│   │   ├── auth/                # Auth use cases
│   │   └── command/             # CQRS commands
│   │
│   ├── infrastructure/           # Infrastructure layer
│   │   ├── repository/          # Data access
│   │   │   ├── mongodb/
│   │   │   └── redis/
│   │   ├── eventbus/            # Event bus (Redis Pub/Sub)
│   │   ├── eventstore/          # Event Store (MongoDB)
│   │   ├── keycloak/            # Keycloak client
│   │   ├── email/               # Email service
│   │   └── websocket/           # WebSocket hub
│   │
│   ├── handler/                  # Interface adapters
│   │   ├── http/                # HTTP handlers (REST API)
│   │   └── websocket/           # WebSocket handlers
│   │
│   ├── middleware/               # HTTP middleware
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── logging.go
│   │
│   └── config/                   # Configuration
│       └── config.go
│
├── pkg/                          # Public libraries (reusable)
│   ├── logger/                  # Structured logging wrapper
│   ├── validator/               # Validation helpers
│   └── errors/                  # Error handling utilities
│
├── web/                          # Frontend resources
│   ├── templates/               # HTML templates
│   ├── static/                  # CSS, JS, images
│   │   ├── css/
│   │   ├── js/
│   │   └── assets/
│   └── components/              # HTMX components
│
├── migrations/                   # Database migrations
│   └── mongodb/
│       ├── 001_initial_schema.js
│       └── 002_add_indexes.js
│
├── configs/                      # Configuration files
│   ├── config.yaml              # Default config
│   ├── config.dev.yaml          # Development overrides
│   └── config.prod.yaml         # Production overrides
│
├── scripts/                      # Utility scripts
│   ├── setup.sh                 # Development setup
│   ├── seed.sh                  # Seed test data
│   └── deploy.sh                # Deployment script
│
├── docs/                         # Documentation
│   ├── 01-architecture.md
│   ├── 02-domain-model.md
│   ├── ...
│   └── api/                     # API documentation
│
├── tests/                        # Integration and E2E tests
│   ├── integration/
│   │   ├── chat_test.go
│   │   ├── task_test.go
│   │   └── helpers.go
│   └── e2e/
│       ├── create_task_test.go
│       └── chat_flow_test.go
│
├── .github/                      # GitHub workflows
│   └── workflows/
│       ├── ci.yml
│       └── deploy.yml
│
├── docker-compose.yml            # Local development services
├── Dockerfile                    # Application container
├── Makefile                      # Build and dev commands
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── .golangci.yml                # Linter configuration
├── .gitignore
├── LICENSE
├── README.md
├── ROADMAP.md
└── CLAUDE.md                     # Project instructions for Claude
```

---

## Domain Layer (`internal/domain/`)

### Структура domain пакета

```
internal/domain/
├── chat/
│   ├── chat.go              # Chat aggregate root
│   ├── chat_test.go         # Unit tests
│   ├── message.go           # Message entity
│   ├── participant.go       # Participant value object
│   ├── events.go            # Domain events (ChatCreated, MessagePosted, etc.)
│   └── repository.go        # Repository interface
│
├── task/
│   ├── task.go              # TaskEntity aggregate
│   ├── task_test.go
│   ├── entity_state.go      # EntityState value object
│   ├── status.go            # Status enum/validation
│   ├── events.go            # Domain events (StatusChanged, AssigneeChanged, etc.)
│   └── repository.go
│
├── user/
│   ├── user.go              # User aggregate
│   ├── user_test.go
│   └── repository.go
│
├── workspace/
│   ├── workspace.go         # Workspace aggregate
│   ├── workspace_test.go
│   ├── invite.go            # Invite entity
│   └── repository.go
│
├── notification/
│   ├── notification.go
│   ├── notification_test.go
│   └── repository.go
│
└── event/
    ├── event.go             # Base event interface
    ├── metadata.go          # Event metadata
    └── types.go             # Event type constants
```

### Пример: Chat Aggregate

**internal/domain/chat/chat.go:**

```go
package chat

import (
    "time"
    "github.com/google/uuid"
    "github.com/yourorg/flowra/internal/domain/event"
)

type ChatType string

const (
    Discussion ChatType = "discussion"
    Task       ChatType = "task"
    Bug        ChatType = "bug"
    Epic       ChatType = "epic"
)

// Chat is an aggregate root
type Chat struct {
    id                UUID
    workspaceID       UUID
    chatType          ChatType
    isPublic          bool
    createdBy         UUID
    createdAt         time.Time
    participants      []Participant

    // Event sourcing
    version           int
    uncommittedEvents []event.DomainEvent
}

// Constructor
func NewChat(workspaceID, createdBy UUID, chatType ChatType) *Chat {
    chat := &Chat{
        id:          uuid.New(),
        workspaceID: workspaceID,
        chatType:    chatType,
        isPublic:    false,
        createdBy:   createdBy,
        createdAt:   time.Now(),
        version:     0,
    }

    // Create with creator as admin
    chat.AddParticipant(createdBy, ParticipantAdmin)

    // Raise domain event
    chat.raise(ChatCreated{
        ChatID:      chat.id,
        WorkspaceID: workspaceID,
        Type:        chatType,
        CreatedBy:   createdBy,
        CreatedAt:   chat.createdAt,
    })

    return chat
}

// Commands (business logic)
func (c *Chat) PostMessage(authorID UUID, content string) (*Message, error) {
    // Validate
    if !c.HasParticipant(authorID) {
        return nil, ErrNotParticipant
    }

    if content == "" {
        return nil, ErrEmptyMessage
    }

    // Create message
    message := &Message{
        ID:        uuid.New(),
        ChatID:    c.id,
        AuthorID:  authorID,
        Content:   content,
        CreatedAt: time.Now(),
    }

    // Raise event
    c.raise(MessagePosted{
        MessageID: message.ID,
        ChatID:    c.id,
        AuthorID:  authorID,
        Content:   content,
        Timestamp: message.CreatedAt,
    })

    return message, nil
}

func (c *Chat) AddParticipant(userID UUID, role ParticipantRole) error {
    if c.HasParticipant(userID) {
        return ErrAlreadyParticipant
    }

    participant := Participant{
        UserID:   userID,
        Role:     role,
        JoinedAt: time.Now(),
    }

    c.participants = append(c.participants, participant)

    c.raise(ParticipantJoined{
        ChatID:   c.id,
        UserID:   userID,
        Role:     role,
        JoinedAt: participant.JoinedAt,
    })

    return nil
}

func (c *Chat) ConvertToTask(title string) error {
    if c.chatType != Discussion {
        return ErrAlreadyTyped
    }

    if title == "" {
        return ErrEmptyTitle
    }

    c.chatType = Task

    c.raise(ChatTypeChanged{
        ChatID:  c.id,
        OldType: Discussion,
        NewType: Task,
        Title:   title,
    })

    return nil
}

// Queries
func (c *Chat) ID() UUID                      { return c.id }
func (c *Chat) WorkspaceID() UUID             { return c.workspaceID }
func (c *Chat) Type() ChatType                { return c.chatType }
func (c *Chat) IsPublic() bool                { return c.isPublic }
func (c *Chat) Participants() []Participant   { return c.participants }
func (c *Chat) Version() int                  { return c.version }

func (c *Chat) HasParticipant(userID UUID) bool {
    for _, p := range c.participants {
        if p.UserID == userID {
            return true
        }
    }
    return false
}

func (c *Chat) IsParticipantAdmin(userID UUID) bool {
    for _, p := range c.participants {
        if p.UserID == userID && p.Role == ParticipantAdmin {
            return true
        }
    }
    return false
}

// Event sourcing methods
func (c *Chat) raise(event event.DomainEvent) {
    c.uncommittedEvents = append(c.uncommittedEvents, event)
    c.version++
}

func (c *Chat) GetUncommittedEvents() []event.DomainEvent {
    return c.uncommittedEvents
}

func (c *Chat) MarkEventsAsCommitted() {
    c.uncommittedEvents = nil
}

func (c *Chat) Apply(event event.DomainEvent) error {
    switch e := event.(type) {
    case ChatCreated:
        c.id = e.ChatID
        c.workspaceID = e.WorkspaceID
        c.chatType = e.Type
        c.createdBy = e.CreatedBy
        c.createdAt = e.CreatedAt

    case ParticipantJoined:
        c.participants = append(c.participants, Participant{
            UserID:   e.UserID,
            Role:     e.Role,
            JoinedAt: e.JoinedAt,
        })

    case ChatTypeChanged:
        c.chatType = e.NewType

    // ... other events
    }

    c.version++
    return nil
}
```

**internal/domain/chat/repository.go:**

```go
package chat

import "github.com/google/uuid"

// Repository defines the interface for chat persistence
type Repository interface {
    // Load chat from event store
    Load(chatID UUID) (*Chat, error)

    // Save events to event store
    Save(chat *Chat) error

    // Query methods (read model)
    FindByID(chatID UUID) (*ChatReadModel, error)
    FindByWorkspace(workspaceID UUID, filters Filters) ([]ChatReadModel, error)
}

// ChatReadModel is a projection for queries
type ChatReadModel struct {
    ID              UUID
    WorkspaceID     UUID
    Type            ChatType
    Title           string
    IsPublic        bool
    ParticipantCount int
    UnreadCount     int
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Filters struct {
    Type     *ChatType
    Status   *string
    IsPublic *bool
    Limit    int
    Cursor   string
}
```

---

## Application Layer (`internal/application/`)

### Структура application пакета

```
internal/application/
├── chat/
│   ├── service.go           # ChatService (use cases)
│   ├── service_test.go
│   ├── commands.go          # Command structs
│   └── queries.go           # Query structs
│
├── task/
│   ├── service.go
│   ├── service_test.go
│   ├── commands.go
│   └── command_executor.go  # Executes commands from tags
│
├── workspace/
│   ├── service.go
│   ├── service_test.go
│   └── invite_service.go
│
├── auth/
│   ├── service.go
│   └── token_validator.go
│
└── eventhandler/            # Event handlers (subscribers)
    ├── tag_parser_handler.go
    ├── notification_handler.go
    └── projection_handler.go
```

### Пример: Chat Service

**internal/application/chat/service.go:**

```go
package chat

import (
    "context"
    "github.com/google/uuid"
    "github.com/yourorg/flowra/internal/domain/chat"
    "github.com/yourorg/flowra/internal/domain/event"
    "github.com/yourorg/flowra/internal/infrastructure/eventbus"
    "github.com/yourorg/flowra/internal/infrastructure/eventstore"
)

// Service handles chat use cases
type Service struct {
    chatRepo   chat.Repository
    eventStore eventstore.EventStore
    eventBus   eventbus.EventBus
}

func NewService(
    chatRepo chat.Repository,
    eventStore eventstore.EventStore,
    eventBus eventbus.EventBus,
) *Service {
    return &Service{
        chatRepo:   chatRepo,
        eventStore: eventStore,
        eventBus:   eventBus,
    }
}

// PostMessage handles posting a message to a chat
func (s *Service) PostMessage(ctx context.Context, cmd PostMessageCommand) (*MessageDTO, error) {
    // 1. Load chat aggregate from event store
    chatAggregate, err := s.chatRepo.Load(cmd.ChatID)
    if err != nil {
        return nil, err
    }

    // 2. Execute business logic (domain method)
    message, err := chatAggregate.PostMessage(cmd.UserID, cmd.Content)
    if err != nil {
        return nil, err
    }

    // 3. Save events to event store
    events := chatAggregate.GetUncommittedEvents()
    for _, event := range events {
        event.SetMetadata(event.EventMetadata{
            CorrelationID: cmd.CorrelationID,
            CausationID:   uuid.Nil,
            UserID:        cmd.UserID,
        })

        if err := s.eventStore.Append(ctx, event); err != nil {
            return nil, err
        }
    }

    // 4. Publish events to event bus
    for _, event := range events {
        if err := s.eventBus.Publish(ctx, event); err != nil {
            // Log error, but don't fail (event is in store)
            log.Error("Failed to publish event", "error", err)
        }
    }

    // 5. Mark events as committed
    chatAggregate.MarkEventsAsCommitted()

    // 6. Return DTO
    return &MessageDTO{
        ID:        message.ID,
        ChatID:    message.ChatID,
        AuthorID:  message.AuthorID,
        Content:   message.Content,
        CreatedAt: message.CreatedAt,
    }, nil
}

// CreateChat handles creating a new chat
func (s *Service) CreateChat(ctx context.Context, cmd CreateChatCommand) (*ChatDTO, error) {
    // 1. Create new chat aggregate
    chatAggregate := chat.NewChat(cmd.WorkspaceID, cmd.CreatedBy, cmd.Type)

    // 2. If typed, convert immediately
    if cmd.Type != chat.Discussion && cmd.Title != "" {
        if err := chatAggregate.ConvertToTask(cmd.Title); err != nil {
            return nil, err
        }
    }

    // 3. If initial message, post it
    if cmd.InitialMessage != "" {
        if _, err := chatAggregate.PostMessage(cmd.CreatedBy, cmd.InitialMessage); err != nil {
            return nil, err
        }
    }

    // 4. Save aggregate (events to event store)
    if err := s.chatRepo.Save(chatAggregate); err != nil {
        return nil, err
    }

    // 5. Publish events
    for _, event := range chatAggregate.GetUncommittedEvents() {
        s.eventBus.Publish(ctx, event)
    }

    // 6. Return DTO
    return s.toChatDTO(chatAggregate), nil
}

// GetChat handles retrieving chat details
func (s *Service) GetChat(ctx context.Context, chatID uuid.UUID) (*ChatDTO, error) {
    // Query from read model (not event store)
    chatReadModel, err := s.chatRepo.FindByID(chatID)
    if err != nil {
        return nil, err
    }

    return s.toDTO(chatReadModel), nil
}

// Helper methods
func (s *Service) toChatDTO(c *chat.Chat) *ChatDTO {
    return &ChatDTO{
        ID:          c.ID(),
        WorkspaceID: c.WorkspaceID(),
        Type:        string(c.Type()),
        IsPublic:    c.IsPublic(),
        CreatedAt:   c.CreatedAt,
    }
}

func (s *Service) toDTO(rm *chat.ChatReadModel) *ChatDTO {
    return &ChatDTO{
        ID:               rm.ID,
        WorkspaceID:      rm.WorkspaceID,
        Type:             string(rm.Type),
        Title:            rm.Title,
        IsPublic:         rm.IsPublic,
        ParticipantCount: rm.ParticipantCount,
        UnreadCount:      rm.UnreadCount,
        CreatedAt:        rm.CreatedAt,
        UpdatedAt:        rm.UpdatedAt,
    }
}
```

**internal/application/chat/commands.go:**

```go
package chat

import (
    "github.com/google/uuid"
    "github.com/yourorg/flowra/internal/domain/chat"
)

// Commands (write operations)

type PostMessageCommand struct {
    ChatID        uuid.UUID
    UserID        uuid.UUID
    Content       string
    CorrelationID uuid.UUID
}

type CreateChatCommand struct {
    WorkspaceID    uuid.UUID
    CreatedBy      uuid.UUID
    Type           chat.ChatType
    Title          string
    IsPublic       bool
    InitialMessage string
    CorrelationID  uuid.UUID
}

type AddParticipantCommand struct {
    ChatID        uuid.UUID
    UserID        uuid.UUID
    AddedBy       uuid.UUID
    Role          chat.ParticipantRole
    CorrelationID uuid.UUID
}

// DTOs (data transfer objects)

type MessageDTO struct {
    ID        uuid.UUID
    ChatID    uuid.UUID
    AuthorID  uuid.UUID
    Content   string
    CreatedAt time.Time
}

type ChatDTO struct {
    ID               uuid.UUID
    WorkspaceID      uuid.UUID
    Type             string
    Title            string
    IsPublic         bool
    ParticipantCount int
    UnreadCount      int
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

---

## Infrastructure Layer (`internal/infrastructure/`)

### Структура infrastructure пакета

```
internal/infrastructure/
├── repository/
│   ├── mongodb/
│   │   ├── chat_repository.go
│   │   ├── task_repository.go
│   │   ├── user_repository.go
│   │   └── connection.go
│   └── redis/
│       ├── session_repository.go
│       └── idempotency_repository.go
│
├── eventstore/
│   ├── eventstore.go        # Interface
│   ├── mongodb_store.go     # MongoDB implementation
│   └── snapshot_store.go
│
├── eventbus/
│   ├── eventbus.go          # Interface
│   ├── redis_bus.go         # Redis Pub/Sub implementation
│   └── partitioned_bus.go   # Partitioned event bus
│
├── keycloak/
│   ├── client.go            # Keycloak client
│   ├── admin_api.go         # Admin API methods
│   └── token_validator.go
│
├── email/
│   ├── service.go
│   └── smtp_provider.go
│
└── websocket/
    ├── hub.go               # WebSocket hub (connection manager)
    ├── client.go            # WebSocket client
    └── message.go           # WebSocket message types
```

### Пример: MongoDB Chat Repository

**internal/infrastructure/repository/mongodb/chat_repository.go:**

```go
package mongodb

import (
    "context"
    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson"

    "github.com/yourorg/flowra/internal/domain/chat"
    "github.com/yourorg/flowra/internal/infrastructure/eventstore"
)

type ChatRepository struct {
    eventStore eventstore.EventStore
    collection *mongo.Collection // для read model
}

func NewChatRepository(
    eventStore eventstore.EventStore,
    db *mongo.Database,
) *ChatRepository {
    return &ChatRepository{
        eventStore: eventStore,
        collection: db.Collection("chats"),
    }
}

// Load восстанавливает aggregate из событий
func (r *ChatRepository) Load(chatID uuid.UUID) (*chat.Chat, error) {
    events, err := r.eventStore.Load(context.Background(), chatID)
    if err != nil {
        return nil, err
    }

    if len(events) == 0 {
        return nil, chat.ErrChatNotFound
    }

    // Создаём пустой aggregate
    chatAggregate := &chat.Chat{}

    // Применяем все события
    for _, event := range events {
        if err := chatAggregate.Apply(event); err != nil {
            return nil, err
        }
    }

    return chatAggregate, nil
}

// Save сохраняет события в event store
func (r *ChatRepository) Save(c *chat.Chat) error {
    ctx := context.Background()

    events := c.GetUncommittedEvents()
    for _, event := range events {
        if err := r.eventStore.Append(ctx, event); err != nil {
            return err
        }
    }

    return nil
}

// FindByID возвращает read model (проекцию)
func (r *ChatRepository) FindByID(chatID uuid.UUID) (*chat.ChatReadModel, error) {
    var result chat.ChatReadModel

    err := r.collection.FindOne(
        context.Background(),
        bson.M{"_id": chatID},
    ).Decode(&result)

    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, chat.ErrChatNotFound
        }
        return nil, err
    }

    return &result, nil
}

// FindByWorkspace возвращает список чатов workspace (read model)
func (r *ChatRepository) FindByWorkspace(
    workspaceID uuid.UUID,
    filters chat.Filters,
) ([]chat.ChatReadModel, error) {
    ctx := context.Background()

    // Build filter
    filter := bson.M{"workspaceId": workspaceID}

    if filters.Type != nil {
        filter["type"] = *filters.Type
    }

    if filters.IsPublic != nil {
        filter["isPublic"] = *filters.IsPublic
    }

    // Query
    cursor, err := r.collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var results []chat.ChatReadModel
    if err := cursor.All(ctx, &results); err != nil {
        return nil, err
    }

    return results, nil
}
```

### Пример: Event Store

**internal/infrastructure/eventstore/mongodb_store.go:**

```go
package eventstore

import (
    "context"
    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson"

    "github.com/yourorg/flowra/internal/domain/event"
)

type MongoDBEventStore struct {
    collection *mongo.Collection
}

func NewMongoDBEventStore(db *mongo.Database) *MongoDBEventStore {
    return &MongoDBEventStore{
        collection: db.Collection("events"),
    }
}

func (s *MongoDBEventStore) Append(ctx context.Context, event event.DomainEvent) error {
    doc := bson.M{
        "_id":           event.GetEventID(),
        "aggregateId":   event.GetAggregateID(),
        "aggregateType": event.GetAggregateType(),
        "eventType":     event.GetEventType(),
        "eventData":     event.GetEventData(),
        "version":       event.GetVersion(),
        "timestamp":     event.GetTimestamp(),
        "metadata":      event.GetMetadata(),
    }

    _, err := s.collection.InsertOne(ctx, doc)
    if err != nil {
        // Check for duplicate key (optimistic concurrency)
        if mongo.IsDuplicateKeyError(err) {
            return ErrConcurrencyConflict
        }
        return err
    }

    return nil
}

func (s *MongoDBEventStore) Load(ctx context.Context, aggregateID uuid.UUID) ([]event.DomainEvent, error) {
    filter := bson.M{"aggregateId": aggregateID}
    opts := options.Find().SetSort(bson.D{{"version", 1}})

    cursor, err := s.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var events []event.DomainEvent
    for cursor.Next(ctx) {
        var doc bson.M
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }

        event, err := deserializeEvent(doc)
        if err != nil {
            return nil, err
        }

        events = append(events, event)
    }

    return events, nil
}

func (s *MongoDBEventStore) LoadAfter(
    ctx context.Context,
    aggregateID uuid.UUID,
    afterVersion int,
) ([]event.DomainEvent, error) {
    filter := bson.M{
        "aggregateId": aggregateID,
        "version":     bson.M{"$gt": afterVersion},
    }
    opts := options.Find().SetSort(bson.D{{"version", 1}})

    cursor, err := s.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var events []event.DomainEvent
    for cursor.Next(ctx) {
        var doc bson.M
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }

        event, err := deserializeEvent(doc)
        if err != nil {
            return nil, err
        }

        events = append(events, event)
    }

    return events, nil
}
```

---

## Handler Layer (`internal/handler/`)

### Структура handler пакета

```
internal/handler/
├── http/
│   ├── router.go            # HTTP router setup
│   ├── auth_handler.go
│   ├── workspace_handler.go
│   ├── chat_handler.go
│   ├── message_handler.go
│   ├── task_handler.go
│   ├── notification_handler.go
│   ├── admin_handler.go
│   └── response.go          # Response helpers
│
└── websocket/
    ├── handler.go           # WebSocket connection handler
    ├── hub.go               # Connection hub
    ├── client.go            # Client connection
    └── message_handler.go   # Message routing
```

### Пример: HTTP Chat Handler

**internal/handler/http/chat_handler.go:**

```go
package http

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "github.com/yourorg/flowra/internal/application/chat"
    "github.com/yourorg/flowra/pkg/logger"
)

type ChatHandler struct {
    chatService *chat.Service
    log         *logger.Logger
}

func NewChatHandler(chatService *chat.Service, log *logger.Logger) *ChatHandler {
    return &ChatHandler{
        chatService: chatService,
        log:         log,
    }
}

// RegisterRoutes registers chat routes
func (h *ChatHandler) RegisterRoutes(r chi.Router) {
    r.Route("/chats", func(r chi.Router) {
        r.Post("/", h.CreateChat)
        r.Get("/{chatId}", h.GetChat)
        r.Put("/{chatId}", h.UpdateChat)
        r.Delete("/{chatId}", h.DeleteChat)

        r.Post("/{chatId}/messages", h.PostMessage)
        r.Get("/{chatId}/messages", h.GetMessages)

        r.Post("/{chatId}/join", h.JoinChat)
        r.Post("/{chatId}/leave", h.LeaveChat)
    })
}

// PostMessage handles POST /chats/{chatId}/messages
func (h *ChatHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    chatID, err := uuid.Parse(chi.URLParam(r, "chatId"))
    if err != nil {
        respondError(w, http.StatusBadRequest, "invalid chat ID")
        return
    }

    userID := getUserIDFromContext(ctx)
    correlationID := uuid.New()

    var req struct {
        Content string `json:"content"`
    }

    if err := decodeJSON(r.Body, &req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Validate
    if req.Content == "" {
        respondError(w, http.StatusBadRequest, "content is required")
        return
    }

    // Create command
    cmd := chat.PostMessageCommand{
        ChatID:        chatID,
        UserID:        userID,
        Content:       req.Content,
        CorrelationID: correlationID,
    }

    // Execute
    message, err := h.chatService.PostMessage(ctx, cmd)
    if err != nil {
        h.log.Error("Failed to post message", "error", err, "chatId", chatID)
        respondError(w, http.StatusInternalServerError, "failed to post message")
        return
    }

    // Respond
    respondJSON(w, http.StatusCreated, message)
}

// CreateChat handles POST /w/{workspaceId}/chats
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    workspaceID, err := uuid.Parse(chi.URLParam(r, "workspaceId"))
    if err != nil {
        respondError(w, http.StatusBadRequest, "invalid workspace ID")
        return
    }

    userID := getUserIDFromContext(ctx)

    var req struct {
        Type           string `json:"type"`
        Title          string `json:"title"`
        IsPublic       bool   `json:"isPublic"`
        InitialMessage string `json:"initialMessage"`
    }

    if err := decodeJSON(r.Body, &req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Create command
    cmd := chat.CreateChatCommand{
        WorkspaceID:    workspaceID,
        CreatedBy:      userID,
        Type:           chat.ChatType(req.Type),
        Title:          req.Title,
        IsPublic:       req.IsPublic,
        InitialMessage: req.InitialMessage,
        CorrelationID:  uuid.New(),
    }

    // Execute
    result, err := h.chatService.CreateChat(ctx, cmd)
    if err != nil {
        h.log.Error("Failed to create chat", "error", err)
        respondError(w, http.StatusInternalServerError, "failed to create chat")
        return
    }

    // Respond
    w.Header().Set("Location", "/api/v1/chats/"+result.ID.String())
    respondJSON(w, http.StatusCreated, result)
}
```

**internal/handler/http/response.go:**

```go
package http

import (
    "encoding/json"
    "net/http"
)

type ErrorResponse struct {
    Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondErrorWithCode(w, status, mapStatusToCode(status), message, nil)
}

func respondErrorWithCode(
    w http.ResponseWriter,
    status int,
    code string,
    message string,
    details map[string]string,
) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ErrorResponse{
        Error: ErrorDetail{
            Code:    code,
            Message: message,
            Details: details,
        },
    })
}

func mapStatusToCode(status int) string {
    switch status {
    case http.StatusBadRequest:
        return "VALIDATION_ERROR"
    case http.StatusUnauthorized:
        return "UNAUTHORIZED"
    case http.StatusForbidden:
        return "FORBIDDEN"
    case http.StatusNotFound:
        return "NOT_FOUND"
    case http.StatusConflict:
        return "CONFLICT"
    case http.StatusTooManyRequests:
        return "RATE_LIMIT_EXCEEDED"
    default:
        return "INTERNAL_ERROR"
    }
}

func decodeJSON(r io.Reader, v interface{}) error {
    return json.NewDecoder(r).Decode(v)
}
```

---

## Entry Points (`cmd/`)

### API Server

**cmd/api/main.go:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "github.com/yourorg/flowra/internal/application/chat"
    "github.com/yourorg/flowra/internal/config"
    httphandler "github.com/yourorg/flowra/internal/handler/http"
    "github.com/yourorg/flowra/internal/infrastructure/eventbus"
    "github.com/yourorg/flowra/internal/infrastructure/eventstore"
    "github.com/yourorg/flowra/internal/infrastructure/repository/mongodb"
    "github.com/yourorg/flowra/internal/middleware"
    "github.com/yourorg/flowra/pkg/logger"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // 2. Initialize logger
    logger := logger.New(cfg.LogLevel)

    // 3. Connect to MongoDB
    mongoClient, err := mongo.Connect(
        context.Background(),
        options.Client().ApplyURI(cfg.MongoDB.URI),
    )
    if err != nil {
        log.Fatal("Failed to connect to MongoDB:", err)
    }
    defer mongoClient.Disconnect(context.Background())

    db := mongoClient.Database(cfg.MongoDB.Database)

    // 4. Initialize infrastructure
    eventStore := eventstore.NewMongoDBEventStore(db)
    eventBus := eventbus.NewRedisEventBus(cfg.Redis)

    // 5. Initialize repositories
    chatRepo := mongodb.NewChatRepository(eventStore, db)
    // ... other repositories

    // 6. Initialize application services
    chatService := chat.NewService(chatRepo, eventStore, eventBus)
    // ... other services

    // 7. Initialize HTTP handlers
    chatHandler := httphandler.NewChatHandler(chatService, logger)
    // ... other handlers

    // 8. Setup router
    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(appmiddleware.CORS(cfg.CORS))

    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        // Auth required for all routes except /auth
        r.Group(func(r chi.Router) {
            r.Use(appmiddleware.Auth(cfg.Keycloak))

            // Register handlers
            chatHandler.RegisterRoutes(r)
            // ... other handlers
        })

        // Public routes
        r.Group(func(r chi.Router) {
            // auth routes (no auth required)
        })
    })

    // 9. Start server
    server := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        logger.Info("Starting server", "addr", server.Addr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Failed to start server:", err)
        }
    }()

    <-done
    logger.Info("Server stopping...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    logger.Info("Server stopped")
}
```

### Worker (Event Handlers)

**cmd/worker/main.go:**

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/yourorg/flowra/internal/application/eventhandler"
    "github.com/yourorg/flowra/internal/config"
    "github.com/yourorg/flowra/internal/infrastructure/eventbus"
    "github.com/yourorg/flowra/pkg/logger"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // 2. Initialize logger
    logger := logger.New(cfg.LogLevel)

    // 3. Initialize event bus
    eventBus := eventbus.NewRedisEventBus(cfg.Redis)

    // 4. Initialize event handlers
    tagParserHandler := eventhandler.NewTagParserHandler(...)
    commandExecutorHandler := eventhandler.NewCommandExecutorHandler(...)
    notificationHandler := eventhandler.NewNotificationHandler(...)

    // 5. Subscribe handlers to events
    eventBus.Subscribe("MessagePosted", tagParserHandler)
    eventBus.Subscribe("TagsParsed", commandExecutorHandler)
    eventBus.Subscribe("StatusChanged", notificationHandler)
    eventBus.Subscribe("AssigneeChanged", notificationHandler)

    logger.Info("Worker started, listening for events...")

    // 6. Wait for interrupt signal
    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

    <-done
    logger.Info("Worker stopping...")

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    eventBus.Shutdown(ctx)

    logger.Info("Worker stopped")
}
```

---

## Configuration (`internal/config/`)

**internal/config/config.go:**

```go
package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server    ServerConfig
    MongoDB   MongoDBConfig
    Redis     RedisConfig
    Keycloak  KeycloakConfig
    CORS      CORSConfig
    LogLevel  string
}

type ServerConfig struct {
    Host string
    Port int
}

type MongoDBConfig struct {
    URI      string
    Database string
}

type RedisConfig struct {
    Host     string
    Port     int
    Password string
    DB       int
}

type KeycloakConfig struct {
    URL          string
    Realm        string
    ClientID     string
    ClientSecret string
}

type CORSConfig struct {
    AllowedOrigins []string
    AllowedMethods []string
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./configs")
    viper.AddConfigPath(".")

    // Environment variables override
    viper.AutomaticEnv()
    viper.SetEnvPrefix("FLOWRA")

    // Defaults
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("loglevel", "info")

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

---

## Testing

### Unit Tests

**internal/domain/chat/chat_test.go:**

```go
package chat_test

import (
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/yourorg/flowra/internal/domain/chat"
)

func TestNewChat(t *testing.T) {
    workspaceID := uuid.New()
    createdBy := uuid.New()

    c := chat.NewChat(workspaceID, createdBy, chat.Task)

    assert.NotNil(t, c)
    assert.Equal(t, workspaceID, c.WorkspaceID())
    assert.Equal(t, chat.Task, c.Type())
    assert.True(t, c.HasParticipant(createdBy))
    assert.True(t, c.IsParticipantAdmin(createdBy))
}

func TestPostMessage(t *testing.T) {
    c := chat.NewChat(uuid.New(), uuid.New(), chat.Discussion)
    authorID := c.Participants()[0].UserID

    message, err := c.PostMessage(authorID, "Hello, world!")

    assert.NoError(t, err)
    assert.NotNil(t, message)
    assert.Equal(t, "Hello, world!", message.Content)
    assert.Equal(t, authorID, message.AuthorID)
}

func TestPostMessage_NotParticipant(t *testing.T) {
    c := chat.NewChat(uuid.New(), uuid.New(), chat.Discussion)
    nonParticipant := uuid.New()

    _, err := c.PostMessage(nonParticipant, "Hello!")

    assert.Error(t, err)
    assert.Equal(t, chat.ErrNotParticipant, err)
}

func TestConvertToTask(t *testing.T) {
    c := chat.NewChat(uuid.New(), uuid.New(), chat.Discussion)

    err := c.ConvertToTask("My Task")

    assert.NoError(t, err)
    assert.Equal(t, chat.Task, c.Type())
}
```

### Integration Tests

**tests/integration/chat_test.go:**

```go
//go:build integration

package integration

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/yourorg/flowra/internal/application/chat"
    "github.com/yourorg/flowra/internal/domain/chat"
)

func TestChatService_CreateAndPostMessage(t *testing.T) {
    // Setup
    testDB := setupTestDB(t)
    defer cleanupTestDB(t, testDB)

    chatService := setupChatService(testDB)

    // Create chat
    createCmd := chat.CreateChatCommand{
        WorkspaceID: uuid.New(),
        CreatedBy:   uuid.New(),
        Type:        chat.Task,
        Title:       "Integration Test Task",
    }

    result, err := chatService.CreateChat(context.Background(), createCmd)
    require.NoError(t, err)
    require.NotNil(t, result)

    // Post message
    postCmd := chat.PostMessageCommand{
        ChatID:  result.ID,
        UserID:  createCmd.CreatedBy,
        Content: "Test message",
    }

    message, err := chatService.PostMessage(context.Background(), postCmd)
    require.NoError(t, err)
    assert.Equal(t, "Test message", message.Content)

    // Verify in DB
    savedChat, err := chatService.GetChat(context.Background(), result.ID)
    require.NoError(t, err)
    assert.Equal(t, result.ID, savedChat.ID)
}
```

---

## Makefile

**Makefile:**

```makefile
.PHONY: help build test lint run migrate docker-up docker-down clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	go build -o bin/api cmd/api/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/migrator cmd/migrator/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	go test -v -race -tags=integration ./tests/integration/...

test-e2e: ## Run E2E tests
	go test -v -tags=e2e ./tests/e2e/...

lint: ## Run linter
	golangci-lint run

run: ## Run the application
	go run cmd/api/main.go

run-worker: ## Run worker
	go run cmd/worker/main.go

migrate: ## Run database migrations
	go run cmd/migrator/main.go up

docker-up: ## Start Docker services
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

dev: docker-up ## Start development environment
	@echo "Waiting for services to start..."
	@sleep 5
	make migrate
	make run

.DEFAULT_GOAL := help
```

---

## Резюме архитектурных решений

| Аспект | Решение | Обоснование |
|--------|---------|-------------|
| **Layout** | Clean Architecture (layers) | Чёткое разделение ответственности |
| **Domain** | DDD aggregates + Event Sourcing | Бизнес-логика в центре |
| **Application** | Use cases + Commands/Queries | CQRS pattern |
| **Infrastructure** | Repository + Event Bus | Абстракция внешних систем |
| **DI** | Constructor injection | Явные зависимости |
| **Config** | Viper (yaml + env) | Гибкость настройки |
| **Logging** | slog (standard library) | Structured logging |
| **Testing** | Unit + Integration + E2E | Comprehensive coverage |

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. ✅ Детальная грамматика тегов
4. ✅ Права доступа и security model
5. ✅ Event flow детально
6. ✅ API контракты (HTTP + WebSocket)
7. ✅ Структура кода (внутри internal/)
8. **TODO:** План реализации MVP (roadmap)
