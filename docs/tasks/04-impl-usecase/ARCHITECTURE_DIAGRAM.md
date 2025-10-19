# Architecture Diagrams - UseCase Layer

Визуальное представление архитектуры UseCase слоя и его взаимодействия с другими слоями.

## 1. Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                     Presentation Layer                       │
│  ┌───────────────────────┐  ┌──────────────────────────┐   │
│  │   HTTP Handlers       │  │  WebSocket Handlers      │   │
│  │   (Echo)              │  │  (gorilla/websocket)     │   │
│  └───────────┬───────────┘  └──────────┬───────────────┘   │
└──────────────┼─────────────────────────┼───────────────────┘
               │                         │
               ▼                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                 UseCases (Commands)                  │   │
│  │  • CreateChatUseCase                                 │   │
│  │  • SendMessageUseCase                                │   │
│  │  • AssignUserUseCase                                 │   │
│  │  • ...                                               │   │
│  └──────────────────┬───────────────────────────────────┘   │
│                     │                                        │
│  ┌──────────────────▼───────────────────────────────────┐   │
│  │               UseCases (Queries)                     │   │
│  │  • GetChatUseCase                                    │   │
│  │  • ListMessagesUseCase                               │   │
│  │  • GetUserUseCase                                    │   │
│  │  • ...                                               │   │
│  └──────────────────┬───────────────────────────────────┘   │
└───────────────────┼─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────────┐
│                      Domain Layer                            │
│  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐  │
│  │  Chat          │  │  Message       │  │  User        │  │
│  │  (Aggregate)   │  │  (Entity)      │  │  (Entity)    │  │
│  └────────────────┘  └────────────────┘  └──────────────┘  │
│                                                              │
│  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐  │
│  │  Task          │  │  Workspace     │  │  Notification│  │
│  │  (Aggregate)   │  │  (Entity)      │  │  (Entity)    │  │
│  └────────────────┘  └────────────────┘  └──────────────┘  │
└───────────────────┬─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────────┐
│                  Infrastructure Layer                        │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────┐  │
│  │  MongoDB         │  │  Redis           │  │ Keycloak │  │
│  │  Repositories    │  │  Event Bus       │  │ Client   │  │
│  └──────────────────┘  └──────────────────┘  └──────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 2. UseCase Execution Flow

```
┌──────────────┐
│   HTTP       │
│   Request    │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────────────────┐
│            HTTP Handler                               │
│  1. Parse request → Command                          │
│  2. Extract user from context                        │
│  3. Call UseCase.Execute(ctx, command)              │
└──────┬───────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────┐
│              UseCase                                  │
│                                                       │
│  Step 1: Валидация                                   │
│  ┌─────────────────────────────────────────┐        │
│  │ • ValidateUUID                          │        │
│  │ • ValidateRequired                      │        │
│  │ • ValidateEnum                          │        │
│  └─────────────────────────────────────────┘        │
│                     │                                 │
│                     ▼                                 │
│  Step 2: Авторизация                                 │
│  ┌─────────────────────────────────────────┐        │
│  │ • Check permissions                     │        │
│  │ • Verify workspace membership           │        │
│  └─────────────────────────────────────────┘        │
│                     │                                 │
│                     ▼                                 │
│  Step 3: Бизнес-логика                               │
│  ┌─────────────────────────────────────────┐        │
│  │ • Load aggregate (if needed)            │        │
│  │ • Execute domain method                 │        │
│  │ • Collect uncommitted events            │        │
│  └─────────────────────────────────────────┘        │
│                     │                                 │
│                     ▼                                 │
│  Step 4: Сохранение                                  │
│  ┌─────────────────────────────────────────┐        │
│  │ • Repository.Save()                     │        │
│  │ • Handle optimistic locking             │        │
│  └─────────────────────────────────────────┘        │
│                     │                                 │
│                     ▼                                 │
│  Step 5: Публикация событий                          │
│  ┌─────────────────────────────────────────┐        │
│  │ • EventBus.Publish()                    │        │
│  │ • Mark events as committed              │        │
│  └─────────────────────────────────────────┘        │
│                     │                                 │
│                     ▼                                 │
│  Step 6: Возврат результата                          │
│  ┌─────────────────────────────────────────┐        │
│  │ • Result with Value, Version, Events    │        │
│  └─────────────────────────────────────────┘        │
└──────┬───────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────┐
│            HTTP Handler                               │
│  1. Map Result → Response DTO                        │
│  2. Return JSON                                      │
└──────┬───────────────────────────────────────────────┘
       │
       ▼
┌──────────────┐
│   HTTP       │
│   Response   │
└──────────────┘
```

## 3. CQRS Pattern

```
                    ┌─────────────────┐
                    │   Application   │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
    ┌─────────────────┐           ┌─────────────────┐
    │   Commands      │           │    Queries      │
    │  (Write Model)  │           │   (Read Model)  │
    └────────┬────────┘           └────────┬────────┘
             │                              │
             ▼                              ▼
    ┌─────────────────┐           ┌─────────────────┐
    │  Event Store    │           │  Read Database  │
    │  (MongoDB)      │           │  (MongoDB)      │
    └────────┬────────┘           └────────┬────────┘
             │                              │
             │   ┌──────────────┐          │
             └──►│  Event Bus   │──────────┘
                 │  (Redis)     │
                 └──────┬───────┘
                        │
                        ▼
                 ┌──────────────┐
                 │  Projections │
                 │  (Update     │
                 │   Read Model)│
                 └──────────────┘

Commands:
• CreateChatCommand → Event Store → Events
• ChangeStatusCommand → Event Store → Events

Queries:
• GetChatQuery → Read Model → Chat DTO
• ListChatsQuery → Read Model → Chat[] DTO

Event Handlers:
• ChatCreatedEvent → Update Read Model
• StatusChangedEvent → Update Read Model
• StatusChangedEvent → Create Notification
```

## 4. Event Sourcing Flow (Chat Aggregate)

```
┌────────────────────────────────────────────────────────────┐
│                 CreateChatUseCase                           │
└──────┬─────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────┐
│  Create new Chat        │
│  aggregate              │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐      ┌─────────────────────────┐
│  Call domain method:    │      │  Aggregate emits:       │
│  chat.AddParticipant()  │─────►│  • ParticipantAdded     │
└─────────────────────────┘      │    Event                │
                                  └──────┬──────────────────┘
                                         │
       ┌─────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│            Aggregate.GetUncommittedEvents()              │
│  Returns: [ChatCreatedEvent, ParticipantAddedEvent]      │
└──────┬───────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│           EventStore.SaveEvents()                        │
│                                                           │
│  MongoDB collection: chat_events                         │
│  Document:                                               │
│  {                                                       │
│    aggregateID: "chat-uuid",                            │
│    version: 1,                                          │
│    events: [                                            │
│      { type: "ChatCreated", data: {...} },             │
│      { type: "ParticipantAdded", data: {...} }         │
│    ]                                                    │
│  }                                                       │
└──────┬───────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│              EventBus.Publish()                          │
│                                                           │
│  Redis pub/sub:                                          │
│  • PUBLISH chat.created {...}                           │
│  • PUBLISH chat.participant.added {...}                 │
└──────┬───────────────────────────────────────────────────┘
       │
       ├────────────────────────────────────────────────────┐
       │                                                     │
       ▼                                                     ▼
┌─────────────────────┐                    ┌─────────────────────┐
│  Event Handler:     │                    │  Event Handler:     │
│  Update Read Model  │                    │  Create Notification│
│                     │                    │                     │
│  MongoDB:           │                    │  MongoDB:           │
│  chat_read_models   │                    │  notifications      │
│  {                  │                    │  {                  │
│    id: "...",       │                    │    userID: "...",   │
│    title: "...",    │                    │    type: "...",     │
│    participants: [] │                    │    message: "..."   │
│  }                  │                    │  }                  │
└─────────────────────┘                    └─────────────────────┘
```

## 5. Tag Processing Integration

```
┌──────────────────────────────────────────────────────────┐
│         User sends message with tags                     │
│         "Let's create a task !task #priority:high"       │
└──────┬───────────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│            SendMessageUseCase                            │
│                                                           │
│  1. Validate & authorize                                 │
│  2. Create Message entity                                │
│  3. Save to database                                     │
│  4. Publish MessageSentEvent                             │
│  5. Trigger tag processing (async) ─────────┐           │
└──────────────────────────────────────────────┼───────────┘
                                               │
                                               ▼
                                    ┌──────────────────────┐
                                    │   Tag Parser         │
                                    │                      │
                                    │  Tokenize message:   │
                                    │  • !task             │
                                    │  • #priority:high    │
                                    └──────┬───────────────┘
                                           │
                                           ▼
                                    ┌──────────────────────┐
                                    │   Tag Processor      │
                                    │                      │
                                    │  Generate commands:  │
                                    │  • CreateTaskCmd     │
                                    │  • SetPriorityCmd    │
                                    └──────┬───────────────┘
                                           │
                                           ▼
                                    ┌──────────────────────┐
                                    │  Tag Executor        │
                                    │                      │
                                    │  Execute commands:   │
                                    └──────┬───────────────┘
                                           │
                    ┌──────────────────────┴────────────────────────┐
                    │                                               │
                    ▼                                               ▼
        ┌───────────────────────┐                    ┌───────────────────────┐
        │ ConvertToTaskUseCase  │                    │ SetPriorityUseCase    │
        │                       │                    │                       │
        │  • Load Chat          │                    │  • Load Chat          │
        │  • Convert to Task    │                    │  • Set Priority       │
        │  • Save & Publish     │                    │  • Save & Publish     │
        └───────────────────────┘                    └───────────────────────┘
                    │                                               │
                    └───────────────────┬───────────────────────────┘
                                        │
                                        ▼
                            ┌───────────────────────┐
                            │  Chat updated with:   │
                            │  • Type = Task        │
                            │  • Priority = High    │
                            └───────────────────────┘
```

## 6. Cross-Domain Integration (Notifications)

```
┌─────────────────────────────────────────────────────────────┐
│                CreateChatUseCase                             │
│                                                              │
│  1. Create Chat aggregate                                   │
│  2. Add participant                                         │
│  3. Save to Event Store                                     │
│  4. Publish events ────────────────────────┐                │
└─────────────────────────────────────────────┼───────────────┘
                                              │
                   ┌──────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │     Event Bus        │
        │     (Redis)          │
        └──────┬───────────────┘
               │
               ├─────────────────────────────────────────────┐
               │                                             │
               ▼                                             ▼
┌──────────────────────────┐              ┌──────────────────────────┐
│  Event Handler:          │              │  Event Handler:          │
│  Update Chat Read Model  │              │  Create Notification     │
│                          │              │                          │
│  Subscribe:              │              │  Subscribe:              │
│  • ChatCreatedEvent      │              │  • ChatCreatedEvent      │
│                          │              │  • ParticipantAdded      │
│  Action:                 │              │  • StatusChanged         │
│  • Create/Update         │              │  • UserAssigned          │
│    chat_read_models      │              │                          │
│    collection            │              │  Action:                 │
│                          │              │  • Call                  │
│  Result:                 │              │    CreateNotification    │
│  • Fast queries          │              │    UseCase               │
│  • No event replay       │              │                          │
└──────────────────────────┘              └──────┬───────────────────┘
                                                 │
                                                 ▼
                                    ┌────────────────────────┐
                                    │ CreateNotification     │
                                    │ UseCase                │
                                    │                        │
                                    │ • Create Notification  │
                                    │ • Save to DB           │
                                    │ • Trigger WebSocket    │
                                    │   update               │
                                    └────────────────────────┘
```

## 7. Dependency Injection Setup

```
┌─────────────────────────────────────────────────────────────┐
│                       main.go                                │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│              Infrastructure Setup                            │
│                                                              │
│  chatRepo := mongodb.NewChatRepository(db)                  │
│  messageRepo := mongodb.NewMessageRepository(db)            │
│  userRepo := mongodb.NewUserRepository(db)                  │
│  eventBus := redis.NewEventBus(redisClient)                 │
│  keycloakClient := keycloak.NewClient(config)               │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│              Application Setup (UseCases)                    │
│                                                              │
│  // Chat UseCases                                           │
│  createChat := chat.NewCreateChatUseCase(chatRepo, eventBus)│
│  addParticipant := chat.NewAddParticipantUseCase(...)       │
│  convertToTask := chat.NewConvertToTaskUseCase(...)         │
│                                                              │
│  // Message UseCases                                        │
│  sendMessage := message.NewSendMessageUseCase(...)          │
│                                                              │
│  // User UseCases                                           │
│  registerUser := user.NewRegisterUserUseCase(...)           │
│                                                              │
│  // Workspace UseCases                                      │
│  createWorkspace := workspace.NewCreateWorkspaceUseCase(...)│
│                                                              │
│  // Notification UseCases                                   │
│  createNotification := notification.NewCreateNotification...│
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│              Event Handlers Setup                            │
│                                                              │
│  notificationHandler := eventhandlers.New...(...,           │
│      createNotification)                                    │
│                                                              │
│  eventBus.Subscribe(chat.EventTypeChatCreated,              │
│      notificationHandler.HandleChatCreated)                 │
│  eventBus.Subscribe(chat.EventTypeUserAssigned,             │
│      notificationHandler.HandleUserAssigned)                │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│              HTTP Handlers Setup                             │
│                                                              │
│  chatHandler := handlers.NewChatHandler(                    │
│      createChat,                                            │
│      addParticipant,                                        │
│      convertToTask,                                         │
│      ...                                                    │
│  )                                                          │
│                                                              │
│  messageHandler := handlers.NewMessageHandler(...)          │
│                                                              │
│  // Echo routing                                            │
│  e.POST("/api/chats", chatHandler.CreateChat)              │
│  e.POST("/api/messages", messageHandler.SendMessage)       │
└─────────────────────────────────────────────────────────────┘
```

## 8. Test Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Test Pyramid                              │
│                                                              │
│                       ▲                                      │
│                      ╱ ╲         E2E Tests                  │
│                     ╱   ╲        (5%)                       │
│                    ╱─────╲                                  │
│                   ╱       ╲      Integration Tests          │
│                  ╱         ╲     (15%)                      │
│                 ╱───────────╲                               │
│                ╱             ╲   Unit Tests                 │
│               ╱               ╲  (80%)                      │
│              ╱─────────────────╲                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘

Unit Tests:
• Each UseCase tested in isolation
• Mock repositories, event bus
• Fast execution (<1s)

Integration Tests:
• Multiple UseCases interact
• In-memory repositories
• Event Bus integration
• Moderate speed (~5-10s)

E2E Tests:
• Complete user workflows
• All layers involved
• Real dependencies (Docker containers)
• Slower execution (~30-60s)
```

---

Эти диаграммы помогают визуализировать:
- **Слои архитектуры** и их зависимости
- **Поток выполнения** UseCase
- **CQRS разделение** команд и запросов
- **Event Sourcing** механизм
- **Tag processing** integration
- **Cross-domain** взаимодействие
- **Dependency Injection** структуру
- **Test pyramid** стратегию
