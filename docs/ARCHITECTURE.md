# Flowra Architecture

This document provides a comprehensive overview of the Flowra system architecture, design decisions, and key components.

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Principles](#core-principles)
4. [Layer Architecture](#layer-architecture)
5. [Key Components](#key-components)
6. [Data Flow](#data-flow)
7. [Technology Stack](#technology-stack)
8. [Architectural Decisions](#architectural-decisions)

---

## System Overview

Flowra is a **Chat System with Task Management** designed for team collaboration. It combines real-time messaging with integrated task tracking, supporting both traditional chat workflows and help desk scenarios.

### Key Capabilities

- **Real-time Communication** - WebSocket-based chat with presence tracking
- **Task Management** - Kanban-style workflows with status transitions
- **Workspace Organization** - Multi-tenant workspaces with role-based access
- **Event-Driven Architecture** - Loosely coupled components via domain events
- **SSO Integration** - Keycloak-based authentication

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                 CLIENTS                                      │
│                                                                             │
│    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                │
│    │  Web Browser │    │ Mobile App   │    │  API Client  │                │
│    │   (HTMX)     │    │  (Future)    │    │  (REST)      │                │
│    └──────┬───────┘    └──────┬───────┘    └──────┬───────┘                │
│           │                    │                    │                       │
└───────────┼────────────────────┼────────────────────┼───────────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           REVERSE PROXY (Traefik)                           │
│                     TLS Termination • Load Balancing                        │
└─────────────────────────────────────────────────────────────────────────────┘
            │                    │
            ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           APPLICATION LAYER                                  │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        API SERVER (Echo)                              │  │
│  │                                                                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │  │
│  │  │    Auth     │  │  Workspace  │  │    Chat     │  │   Message   │ │  │
│  │  │  Handlers   │  │  Handlers   │  │  Handlers   │  │  Handlers   │ │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │  │
│  │                                                                       │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │  │
│  │  │    Task     │  │Notification │  │    User     │  │  WebSocket  │ │  │
│  │  │  Handlers   │  │  Handlers   │  │  Handlers   │  │   Handler   │ │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                     WEBSOCKET SERVER (gorilla/websocket)              │  │
│  │                                                                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │  │
│  │  │   Connection    │  │    Message      │  │    Presence     │      │  │
│  │  │     Hub         │  │   Broadcasting  │  │    Tracking     │      │  │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘      │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        WORKER SERVICE                                 │  │
│  │                                                                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │  │
│  │  │  Event Handler  │  │  Notification   │  │   SLA Monitor   │      │  │
│  │  │   (Projections) │  │    Sender       │  │    (Future)     │      │  │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘      │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           INFRASTRUCTURE LAYER                               │
│                                                                             │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │     MongoDB      │  │      Redis       │  │    Keycloak      │          │
│  │   (Primary DB)   │  │  (Cache/PubSub)  │  │   (Auth/SSO)     │          │
│  │                  │  │                  │  │                  │          │
│  │ • Documents      │  │ • Session cache  │  │ • User mgmt      │          │
│  │ • Event store    │  │ • Event pub/sub  │  │ • OAuth 2.0      │          │
│  │ • Read models    │  │ • Rate limiting  │  │ • JWT tokens     │          │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Principles

### 1. Domain-Driven Design (DDD)

Business logic is organized around domain concepts:

- **Aggregates** - Consistency boundaries (Chat, Message, Task, Notification)
- **Entities** - Objects with identity (User, Workspace)
- **Value Objects** - Immutable domain concepts (UUID, Priority, Status)
- **Domain Events** - Business facts that happened

### 2. Event-Driven Architecture

Components communicate through domain events:

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Command    │───▶│   Domain     │───▶│    Event     │
│   Handler    │    │   Logic      │    │   Published  │
└──────────────┘    └──────────────┘    └──────────────┘
                                               │
        ┌──────────────────────────────────────┼──────────────────┐
        │                                      │                  │
        ▼                                      ▼                  ▼
┌──────────────┐               ┌──────────────┐     ┌──────────────┐
│  Projection  │               │  Notification │     │   WebSocket  │
│   Handler    │               │    Service    │     │   Broadcast  │
└──────────────┘               └──────────────┘     └──────────────┘
```

### 3. CQRS (Command Query Responsibility Segregation)

Separate models for reading and writing:

- **Commands** - Change state (CreateChat, SendMessage, AssignTask)
- **Queries** - Read state (ListChats, GetMessages, GetTasks)
- **Read Models** - Optimized for queries (task board view, chat list)

### 4. Clean Architecture

Dependencies point inward:

```
┌─────────────────────────────────────────────────┐
│                  Handlers                        │
│              (HTTP, WebSocket)                   │
│  ┌─────────────────────────────────────────┐    │
│  │           Application Layer              │    │
│  │         (Use Cases, Services)            │    │
│  │  ┌─────────────────────────────────┐    │    │
│  │  │         Domain Layer            │    │    │
│  │  │  (Aggregates, Entities, Events) │    │    │
│  │  └─────────────────────────────────┘    │    │
│  └─────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
        ▲                            │
        │                            ▼
┌─────────────────────────────────────────────────┐
│              Infrastructure Layer                │
│    (Repositories, Event Store, External APIs)   │
└─────────────────────────────────────────────────┘
```

---

## Layer Architecture

### 1. Handler Layer (`internal/handler/`)

Handles HTTP requests and WebSocket connections.

```go
// HTTP handler example
type ChatHandler struct {
    chatService ChatService  // Interface defined by handler
}

func (h *ChatHandler) Create(c echo.Context) error {
    var req CreateChatRequest
    if err := c.Bind(&req); err != nil {
        return httpserver.RespondError(c, err)
    }
    
    result, err := h.chatService.CreateChat(ctx, cmd)
    if err != nil {
        return httpserver.RespondError(c, err)
    }
    
    return httpserver.RespondCreated(c, ToChatResponse(result))
}
```

### 2. Application Layer (`internal/application/`)

Orchestrates use cases and coordinates domain logic.

```go
// Application service example
type ChatUseCase struct {
    chatRepo   ChatRepository    // Interfaces
    eventStore EventStore
    eventBus   EventBus
}

func (uc *ChatUseCase) CreateChat(ctx context.Context, cmd CreateChatCommand) (Result, error) {
    // 1. Load or create aggregate
    chat, err := chat.NewChat(cmd.WorkspaceID, cmd.CreatorID, cmd.Name)
    if err != nil {
        return Result{}, err
    }
    
    // 2. Save aggregate and events
    if err := uc.chatRepo.Save(ctx, chat); err != nil {
        return Result{}, err
    }
    
    // 3. Publish domain events
    for _, event := range chat.Events() {
        uc.eventBus.Publish(ctx, event)
    }
    
    return Result{Value: chat}, nil
}
```

### 3. Domain Layer (`internal/domain/`)

Contains business logic and domain models.

```go
// Aggregate example
type Chat struct {
    id           uuid.UUID
    workspaceID  uuid.UUID
    name         string
    participants []Participant
    events       []event.Event  // Uncommitted domain events
}

func (c *Chat) Rename(newName string, byUser uuid.UUID) error {
    // Business rule validation
    if !c.hasPermission(byUser, PermissionRename) {
        return ErrNotChatAdmin
    }
    
    if newName == c.name {
        return nil
    }
    
    // Apply change
    c.name = newName
    
    // Record domain event
    c.recordEvent(ChatRenamed{
        ChatID:   c.id,
        NewName:  newName,
        RenamedBy: byUser,
    })
    
    return nil
}
```

### 4. Infrastructure Layer (`internal/infrastructure/`)

Implements interfaces defined by upper layers.

```go
// Repository implementation
type MongoChatRepository struct {
    collection *mongo.Collection
}

func (r *MongoChatRepository) Save(ctx context.Context, chat *chat.Chat) error {
    doc := toDocument(chat)
    _, err := r.collection.ReplaceOne(ctx, 
        bson.M{"_id": chat.ID()},
        doc,
        options.Replace().SetUpsert(true),
    )
    return err
}
```

---

## Key Components

### Domain Aggregates

| Aggregate | Description | Key Operations |
|-----------|-------------|----------------|
| **Chat** | Conversation room | Create, Rename, AddParticipant, RemoveParticipant |
| **Message** | Chat message | Send, Edit, Delete, AddReaction |
| **Task** | Trackable work item | Create, ChangeStatus, Assign, SetPriority |
| **Notification** | User notification | Create, MarkAsRead, Delete |

### Domain Entities

| Entity | Description | Belongs To |
|--------|-------------|------------|
| **User** | System user | Global |
| **Workspace** | Organizational unit | Global |
| **Participant** | Chat member with role | Chat |
| **Attachment** | File attached to message | Message |

### Domain Events

| Event | Trigger | Handlers |
|-------|---------|----------|
| `ChatCreated` | New chat | Notification, Analytics |
| `MessagePosted` | New message | WebSocket, Notification |
| `TaskStatusChanged` | Status update | WebSocket, Notification, SLA |
| `UserMentioned` | @mention in message | Notification |

---

## Data Flow

### REST API Request Flow

```
Client Request
     │
     ▼
┌─────────────────┐
│    Middleware   │  ─── Auth, Logging, CORS, Rate Limiting
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Handler     │  ─── Parse request, validate input
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Application   │  ─── Execute use case
│     Service     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Domain      │  ─── Business logic, create events
│    Aggregate    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Repository    │  ─── Persist changes
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Event Bus    │  ─── Publish domain events
└────────┬────────┘
         │
         ▼
Response to Client
```

### WebSocket Message Flow

```
Client Message
     │
     ▼
┌─────────────────┐
│    WebSocket    │  ─── Validate connection, parse message
│     Handler     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│      Hub        │  ─── Route message based on type
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
Subscribe   Typing
to Chat    Indicator
    │         │
    ▼         ▼
Update    Broadcast
Subscriptions  to Chat
```

### Event Processing Flow

```
Domain Event Published
        │
        ▼
┌─────────────────┐
│   Event Bus     │  ─── Redis Pub/Sub
│    (Redis)      │
└────────┬────────┘
         │
    ┌────┴────┬────────────┐
    │         │            │
    ▼         ▼            ▼
Projection  Notification  WebSocket
Handler     Service       Broadcast
    │         │            │
    ▼         ▼            ▼
Update     Create      Push to
Read Model Notification  Clients
```

---

## Technology Stack

### Backend

| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Primary language | 1.25+ |
| **Echo** | HTTP framework | v4 |
| **gorilla/websocket** | WebSocket | Latest |
| **MongoDB Go Driver** | Database driver | v2 |
| **go-redis** | Redis client | v9 |

### Frontend

| Technology | Purpose | Version |
|------------|---------|---------|
| **HTMX** | Dynamic updates | 2+ |
| **Pico CSS** | Styling | v2 |
| **Alpine.js** | Minimal JS interactions | 3+ |

### Infrastructure

| Technology | Purpose | Version |
|------------|---------|---------|
| **MongoDB** | Primary database | 6+ |
| **Redis** | Cache, Pub/Sub | 7+ |
| **Keycloak** | Authentication | 23+ |
| **Docker** | Containerization | 24+ |

---

## Architectural Decisions

### ADR-001: MongoDB for Primary Storage

**Context:** Need a database for storing documents with flexible schema.

**Decision:** Use MongoDB as the primary database.

**Rationale:**
- Document model fits well with aggregates
- Flexible schema for evolving domain
- Good Go driver support (v2)
- Built-in sharding for scale

**Consequences:**
- Need to manage consistency at application level
- No ACID transactions across documents (acceptable for our use case)

---

### ADR-002: Event-Driven Architecture

**Context:** Need loose coupling between components and real-time updates.

**Decision:** Implement event-driven architecture with domain events.

**Rationale:**
- Decouples components (producers don't know consumers)
- Enables real-time updates via WebSocket
- Supports future event sourcing
- Facilitates audit logging

**Consequences:**
- Eventual consistency between components
- Need dead letter queue for failed events
- More complex debugging

---

### ADR-003: Redis for Pub/Sub and Caching

**Context:** Need fast cache and inter-service communication.

**Decision:** Use Redis for caching, session storage, and event pub/sub.

**Rationale:**
- Fast in-memory operations
- Built-in pub/sub for events
- Simple deployment
- Excellent Go client

**Consequences:**
- Additional infrastructure component
- Need persistence configuration for reliability

---

### ADR-004: Keycloak for Authentication

**Context:** Need secure authentication with SSO capabilities.

**Decision:** Use Keycloak for authentication and user management.

**Rationale:**
- Industry-standard OAuth 2.0 / OpenID Connect
- Built-in user management
- Supports multiple identity providers
- Reduces security burden

**Consequences:**
- Additional infrastructure dependency
- Learning curve for configuration
- JWT token handling complexity

---

### ADR-005: CQRS Pattern

**Context:** Different read and write requirements.

**Decision:** Implement CQRS with separate read models.

**Rationale:**
- Read models optimized for queries
- Write models focused on consistency
- Easier scaling of read-heavy workloads
- Clear separation of concerns

**Consequences:**
- Eventual consistency between read/write models
- Need to maintain projections
- More complex data management

---

### ADR-006: Interface Declaration on Consumer Side

**Context:** Need clean architecture with proper dependency management.

**Decision:** Declare interfaces where they are used (consumer side).

**Rationale:**
- Follows Go idioms ("Accept interfaces, return structs")
- Consumers define their dependencies
- Loose coupling between packages
- Easier testing with mocks

**Consequences:**
- May have similar interfaces in multiple places
- Need clear naming conventions

---

## Security Architecture

### Authentication Flow

```
┌────────┐     ┌────────┐     ┌──────────┐
│ Client │────▶│  API   │────▶│ Keycloak │
└────────┘     └────────┘     └──────────┘
     │              │               │
     │  1. Login    │               │
     │─────────────▶│   2. Validate │
     │              │──────────────▶│
     │              │   3. Token    │
     │              │◀──────────────│
     │  4. JWT      │               │
     │◀─────────────│               │
     │              │               │
     │  5. Request  │               │
     │─────────────▶│               │
     │   + JWT      │ 6. Validate   │
     │              │   locally     │
     │  7. Response │               │
     │◀─────────────│               │
```

### Authorization Model

| Level | Mechanism | Checked By |
|-------|-----------|------------|
| **API Access** | JWT Token | Auth Middleware |
| **Workspace Access** | Membership | Workspace Middleware |
| **Chat Access** | Participant | Handler |
| **Action Permission** | Role-based | Domain Logic |

---

## Scalability Considerations

### Horizontal Scaling

```
                    ┌──────────────┐
                    │ Load Balancer│
                    └──────┬───────┘
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │ API Server │  │ API Server │  │ API Server │
    │     #1     │  │     #2     │  │     #3     │
    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
    ┌─────────────────────┼─────────────────────┐
    │                     │                     │
    ▼                     ▼                     ▼
┌────────┐           ┌────────┐           ┌────────┐
│MongoDB │           │ Redis  │           │Keycloak│
│Replica │           │Cluster │           │HA Mode │
│  Set   │           │        │           │        │
└────────┘           └────────┘           └────────┘
```

### Performance Optimizations

- **Connection pooling** for MongoDB and Redis
- **Read replicas** for query-heavy workloads
- **Caching** of frequently accessed data
- **Event batching** for high-volume scenarios
- **Pagination** for all list endpoints

---

*Last updated: January 2026*
