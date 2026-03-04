# Flowra Architecture

This document provides a comprehensive overview of the Flowra system architecture, design decisions, and key components.

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Principles](#core-principles)
4. [Layer Architecture](#layer-architecture)
5. [Key Components](#key-components)
6. [Tag System](#tag-system)
7. [Data Flow](#data-flow)
8. [Event Sourcing & Event Flow](#event-sourcing--event-flow)
9. [Technology Stack](#technology-stack)
10. [Architectural Decisions](#architectural-decisions)
11. [Security Model](#security-model)
12. [Scalability Considerations](#scalability-considerations)

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
│  ┌──────────────────┐                                                       │
│  │   File Storage   │                                                       │
│  │    (Local FS)    │                                                       │
│  │                  │                                                       │
│  │ • Message files  │                                                       │
│  │ • Task files     │                                                       │
│  │ • Metadata in DB │                                                       │
│  └──────────────────┘                                                       │
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
| **Attachment** | File attached to message or task | Message, Task |

### Domain Events

| Event | Trigger | Handlers |
|-------|---------|----------|
| `ChatCreated` | New chat | Notification, Analytics |
| `MessagePosted` | New message | WebSocket, Notification |
| `TaskStatusChanged` | Status update | WebSocket, Notification, SLA |
| `UserMentioned` | @mention in message | Notification |

---

## Tag System

Tags are the primary mechanism for task management through chat. All task operations (status changes, assignments, priority updates) happen via tags in messages.

### Design Principles

| Principle | Description |
|-----------|-------------|
| **Simplicity** | Easy to remember basic syntax |
| **Tag names are case-sensitive** | `#status` ≠ `#Status` (prevents accidental triggers) |
| **Enum values are forgiving** | Values like status/priority/severity are matched case-insensitively and normalized |
| **Partial Application** | Valid tags apply even when others fail |
| **Known Tags Only** | Only registered tags are parsed (`#` in regular text is ignored) |

### Tag Positioning

Tags can appear:
1. **At the start of a message** (first line)
2. **On a separate line** after regular text

```
Valid:
#status Done #assignee @alex
Finished the work, ready for review

Also valid:
Finished working on the task
#status Done
#assignee @alex

Invalid (not parsed):
Finished work #status Done — sending for review
```

### System Tags

User-facing reference with examples and troubleshooting: `docs/TAGS_USER_GUIDE.md`

#### Entity Creation Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `#task <title>` | Create a Task | `#task Implement OAuth` |
| `#bug <title>` | Create a Bug | `#bug Login fails on Chrome` |
| `#epic <title>` | Create an Epic | `#epic User Management` |

#### Entity Management Tags

| Tag | Format | Description |
|-----|--------|-------------|
| `#status <value>` | Enum (case-insensitive input) | Change status |
| `#assignee @user` | Username | Assign to user |
| `#priority <value>` | High/Medium/Low | Set priority |
| `#due <date>` | ISO 8601 (YYYY-MM-DD) | Set deadline |
| `#title <text>` | Free text | Change task title |
| `#severity <value>` | Critical/Major/Minor/Trivial | Bug severity only |

#### Participant Management Tags

| Tag | Format | Description |
|-----|--------|-------------|
| `#invite @user` | Username | Add participant to chat |
| `#remove @user` | Username | Remove participant from chat |

#### Chat Lifecycle Tags

| Tag | Format | Description |
|-----|--------|-------------|
| `#close` | No value | Close/archive current chat |
| `#reopen` | No value | Reopen chat |
| `#delete` | No value | Delete chat (parses today; execution currently not implemented) |

**Status Values by Type:**
- **Task:** To Do, In Progress, Done
- **Bug:** New, Investigating, Fixed, Verified
- **Epic:** Planned, In Progress, Completed

**Priority via tags:** `High`, `Medium`, `Low` (tag commands do not currently support `Critical`).

### Validation Strategy

**Partial Application:** Each tag is validated independently. Valid tags are applied, invalid tags are reported but don't block others.

```
Input: "#status Done #assignee @unknown #priority High"

Result:
✅ status → "Done" (applied)
❌ assignee → error "user @unknown not found: ..." (not applied)
✅ priority → "High" (applied)

Bot response:
"✅ Status changed to Done
 ✅ Priority changed to High
 ❌ user @unknown not found: ..."
```

### Error Types

| Type | Example | Message Format |
|------|---------|----------------|
| **Syntax** | `#assignee alex` (missing @) | `❌ invalid assignee format. Use @username` |
| **Semantic** | `#status Completed` (invalid value) | `❌ invalid status 'Completed' for Task. Available: To Do, In Progress, Done` |
| **Business** | `#assignee @nonexistent` | `❌ user @nonexistent not found: ...` |

Business-layer errors may include wrapped context from executors/use cases (for example `failed to assign user: ...`).

**Important:** Messages are always saved, even if all tags are invalid. Tag errors don't prevent message posting.

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

## Event Sourcing & Event Flow

The system uses **Event Sourcing** for storing state changes and **Event-Driven Architecture** for communication between bounded contexts. Events are the single source of truth; all read models (projections) are built from the event stream.

### Event Store

**MongoDB Collection:** `events`

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
    "content": "Finished work\n#status Done",
    "timestamp": "2025-09-30T10:00:00Z"
  },
  "version": 142,
  "timestamp": "2025-09-30T10:00:00Z",
  "metadata": {
    "correlationId": "req-uuid",
    "causationId": "parent-event-id",
    "userId": "user-uuid"
  }
}
```

**Key Indexes:**
- `{ aggregateId: 1, version: 1 }` — unique, for loading aggregate events
- `{ eventType: 1, timestamp: 1 }` — for filtering by event type
- `{ timestamp: 1 }` — for chronological queries

### Event Metadata

| Field | Purpose |
|-------|---------|
| `correlationId` | Traces all events from a single user request |
| `causationId` | Links to the event that caused this one |
| `userId` | Who initiated the action |

This enables full request tracing through the event chain.

### Event Bus (Redis Pub/Sub)

**Channel Strategy:** By event type

```
Channel: events.MessagePosted
Channel: events.ChatTypeChanged
Channel: events.TagsParsed
Channel: events.StatusChanged
Channel: events.TaskCreated
```

### Delivery Guarantees

**MVP: At-most-once**
- Redis Pub/Sub doesn't guarantee delivery
- If subscriber is offline, event is lost
- **Mitigation:** Events stored in Event Store; state can be rebuilt

**V2: At-least-once** (Transactional Outbox pattern)

### Idempotency

**Problem:** Events may be redelivered (reconnections, retries).

**Solution:** Track processed events.

```javascript
// Collection: processed_events
{
  "eventId": "event-uuid",
  "handlerName": "TagParserService",
  "processedAt": ISODate("..."),
  "expiresAt": ISODate("...") // TTL = 7 days
}
```

Each handler checks if event was already processed before handling.

### Retry & Error Handling

**Strategy:** Exponential Backoff + Dead Letter Queue

```
1. Event processing fails
2. Retry: 1s → 2s → 4s → 8s → 16s
3. After MaxRetries → Dead Letter Queue
4. Manual replay by administrator
```

**Dead Letter Queue Collection:** `dead_letter_queue`
- Stores failed events with error details
- Admin can replay or discard entries

### Event Ordering

**Problem:** Events for same aggregate may process out of order.

**Solution:** Partition by aggregateId.

- Events for `chat-uuid-1` process sequentially
- Events for `chat-uuid-2` process in parallel with `chat-uuid-1`
- No race conditions on same aggregate

### Example Event Chain

```
User sends: "Finished work\n#status Done"

[1] MessagePosted
    ↓ (causationId)
[2] TagsParsed
    ↓ (causationId)
[3] StatusChanged
    ↓ (causationId)
[4] UserNotified

All events share same correlationId
```

### Aggregate Recovery

Aggregates are rebuilt from event stream:

1. Load snapshot (if exists)
2. Load events after snapshot version
3. Apply events to rebuild current state

**Snapshots:** Created every ~100 events to optimize recovery time.

---

## Technology Stack

### Backend

| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Primary language | 1.26+ |
| **Echo** | HTTP framework | v4 |
| **gorilla/websocket** | WebSocket | Latest |
| **MongoDB Go Driver** | Database driver | v2 |
| **go-redis** | Redis client | v9 |

### Frontend

| Technology | Purpose | Version |
|------------|---------|---------|
| **HTMX** | Dynamic updates | 2+ |
| **Pico CSS** | Styling | v2 |
| **Vanilla JS (ES Modules)** | Client interactions | Built-in |

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

## Security Model

The system uses **Keycloak** for user management, authentication, and authorization. User and role logic is delegated to Keycloak; the application works with JWT tokens in a stateless manner.

### Core Principles

- **Keycloak as Source of Truth** for users, roles, workspace membership
- **Stateless Authorization** via JWT tokens
- **RBAC** (Role-Based Access Control) at Keycloak level
- **Workspace Isolation** — users work within workspace context
- **Self-Service** — users can create workspaces and invite others

### Keycloak Configuration

```
Realm: flowra

Realm Roles:
├─ user              — base role (all registered users)
└─ system-admin      — superadmin (full access)

Client: flowra-app
├─ Client ID: flowra-app
├─ Protocol: openid-connect
├─ Access Type: confidential
└─ Client Roles:
   ├─ workspace-admin   — workspace administrator
   └─ workspace-member  — workspace member

Groups (created dynamically):
├─ "Engineering Team"
│  ├─ Attributes: { workspace_id: "uuid" }
│  └─ Members with roles
└─ "Marketing Team"
   └─ ...
```

### JWT Token Structure

```json
{
  "sub": "user-uuid",
  "email": "alice@example.com",
  "preferred_username": "alice",
  "realm_access": {
    "roles": ["user"]
  },
  "resource_access": {
    "flowra-app": {
      "roles": ["workspace-admin", "workspace-member"]
    }
  },
  "groups": ["/Engineering Team", "/Marketing Team"],
  "aud": "flowra-app"
}
```

### Access Hierarchy

```
System Level (Keycloak Realm)
    ↓
Workspace Level (Keycloak Groups)
    ↓
Chat Level (Application)
    ↓
Message Level (Application)
```

### Permission Tables

#### System Level

| Role | Capabilities |
|------|-------------|
| **system-admin** | Access all workspaces, manage any chat/task, view logs |
| **user** | Create workspaces, join via invite |

#### Workspace Level

| Action | workspace-admin | workspace-member | non-member |
|--------|----------------|------------------|------------|
| View public chats | ✅ | ✅ | ❌ |
| Create chat | ✅ | ✅ | ❌ |
| Generate invite links | ✅ | ❌ | ❌ |
| Manage settings | ✅ | ❌ | ❌ |
| Remove members | ✅ | ❌ | ❌ |

#### Chat Level

| Action | Chat Admin | Chat Member | Workspace Member (not in chat) |
|--------|------------|-------------|-------------------------------|
| View private chat | ✅ | ✅ | ❌ |
| View public chat | ✅ | ✅ | ✅ (read-only) |
| Send messages | ✅ | ✅ | ❌ (needs Join) |
| Apply tags | ✅ | ✅ | ❌ |
| Add/remove participants | ✅ | ❌ | ❌ |
| Delete chat | ✅ | ❌ | ❌ |

#### Message Level

| Action | Author (< 5 min) | Author (> 5 min) | Chat Admin |
|--------|-----------------|------------------|------------|
| Edit own message | ✅ | ❌ | ✅ |
| Delete own message | ✅ | ❌ | ✅ |
| Delete others' messages | ❌ | ❌ | ✅ |

### Workspace Management

**Self-Service Creation:**
1. User clicks "Create Workspace"
2. Backend creates Keycloak Group with workspace attributes
3. User is added as workspace-admin
4. Workspace record created in database

**Invite Links:**
```
InviteLink:
├─ Token: "secure-random-token"
├─ WorkspaceID: UUID
├─ ExpiresAt: timestamp
├─ MaxUses: int (null = unlimited)
└─ UsedCount: int
```

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

### WebSocket Authentication

WebSocket doesn't support custom headers after handshake. Solution: pass token at connection time.

```javascript
const wsURL = `ws://localhost:8080/ws?token=${accessToken}`;
const ws = new WebSocket(wsURL);
```

Backend validates JWT from query parameter, extracts user ID, and registers the client.

### Security Best Practices

| Practice | Implementation |
|----------|----------------|
| **JWT Validation** | Verify signature via Keycloak JWKS endpoint, check audience/issuer/expiry |
| **CORS** | Whitelist allowed origins, enable credentials for cookies |
| **Rate Limiting** | Per IP + UserID, stricter for auth endpoints |
| **Input Validation** | Validate and sanitize all user input |
| **Audit Logging** | Log security-relevant actions with user/IP/timestamp |

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

*Last updated: February 2026*
