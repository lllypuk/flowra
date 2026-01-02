# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Chat System with Task Management** built in Go. It's a comprehensive chat platform with integrated task tracking, help desk functionality, and command support. The project uses a microservices architecture with event-driven design.

**Key Technologies:**
- **Backend**: Go 1.25+ with Echo v4 framework
- **Database**: MongoDB 6+ (main with Go Driver v2), Redis (cache/pub-sub)
- **Frontend**: HTMX 2+ for dynamic updates, Pico CSS v2 for styling
- **Auth**: Keycloak for SSO and user management
- **Infrastructure**: Docker Compose for development

## Development Commands

### Environment Setup
```bash
# Start infrastructure services
docker-compose up -d mongodb redis keycloak

# Start the main application
go run cmd/api/main.go
```

### Code Quality
```bash
# Run linting with comprehensive Go linting rules
golangci-lint run

# Run tests
go test ./...
go test ./tests/integration -tags=integration
go test ./tests/e2e -tags=e2e
```

### Build and Development
```bash
# Build application
make build

# Development mode
make dev
```

## Architecture

### Core Design Principles
- **Event-driven architecture** for loose coupling
- **Domain-Driven Design** for business logic organization
- **CQRS** pattern for command/query separation
- **Repository pattern** for data access abstraction

### Service Structure
The system is designed around multiple services:
- **API Gateway** (Echo) - HTTP/HTMX requests, static files, WebSocket upgrade
- **WebSocket Server** - Real-time communication, presence tracking
- **Worker Service** - Background tasks (SLA monitoring, notifications)
- **Command Processor** - Chat command parsing and execution

### Directory Layout
```
cmd/                    # Application entry points
├── api/               # HTTP API server
├── websocket/         # WebSocket server
├── worker/            # Background workers
└── migrator/          # Database migrations

internal/              # Internal application code
├── domain/           # Business logic and models
├── repository/       # Data access layer
├── service/          # Service layer
├── handler/          # HTTP/WS handlers
├── auth/             # Authentication (Keycloak integration)
├── command/          # Command processors
└── event/            # Event bus

pkg/                   # Reusable packages
web/                   # Frontend resources
├── templates/        # HTML templates
├── static/           # CSS, JS assets
└── components/       # HTMX components

migrations/           # SQL migrations
configs/              # Configuration files
```

## Configuration

- Main config: `configs/config.yaml`
- Environment-specific values override via environment variables
- Docker services configured in `docker-compose.yml`
- Comprehensive settings for database, Redis, JWT, OAuth, email, etc.

## Database

- **Primary**: MongoDB 6+ (document store)
- **Cache**: Redis for sessions, pub/sub, caching
- Main collections: Users, Chats, Messages, Tasks, Chat_members, Audit_log
- Schema versioning handled through application code

## Development Notes

- **Project Status**: January 2026 Release Candidate (~95% complete)
- All domain, application, and infrastructure layers are fully implemented
- HTTP handlers and WebSocket support are production-ready
- Entry points (API server, Worker, Migrator) are implemented and tested
- E2E tests provide full coverage of critical workflows
- HTMX frontend is planned for February 2026
- Comprehensive linting rules are configured in `.golangci.yml`
- Security-first approach with Keycloak SSO, RBAC, and secure defaults

## Documentation Structure

- **API Documentation**: `docs/api/` - OpenAPI spec, Postman collection
- **Deployment Guide**: `docs/DEPLOYMENT.md` - Docker, environment setup
- **Development Guide**: `docs/DEVELOPMENT.md` - Local setup, testing
- **Architecture Overview**: `docs/ARCHITECTURE.md` - System design, decisions

## MongoDB Driver

**Current Version**: Go Driver v2 (go.mongodb.org/mongo-driver/v2)

**Migration Notes**:
- Project was migrated from v1 to v2 on 2025-10-21
- Key v2 API changes to remember:
  - `mongo.Connect()` no longer takes `context` as first argument
  - `StartSession()` returns `*mongo.Session` (pointer) instead of value type
  - `UseSession` callback now receives `context.Context`, use `mongo.SessionFromContext()` to get session
  - `Distinct` results use `DistinctResult` with `.Decode()` method instead of returning `[]any`

**Implementation Reference**:
- Test utilities: `tests/testutil/db.go` (v2 integration tests)
- Test utilities: `tests/testutil/mongodb.go` (v2 connection setup)

## Application Access

- **Main App**: http://localhost:8080 (when implemented)
- **Keycloak**: http://localhost:8090 (admin/admin123)
- **Traefik Dashboard**: http://localhost:8080 (reverse proxy)
- **MongoDB**: localhost:27017 (admin/admin123)
- **Redis**: localhost:6379

## Testing Strategy

- Unit tests for all business logic
- Integration tests with MongoDB
- E2E tests for user workflows
- Load testing for performance validation
- Test database uses in-memory MongoDB (testcontainers)

## Interface Design Guidelines

This project follows **idiomatic Go interface patterns**. Always follow these rules when working with interfaces:

### Core Principle: Accept Interfaces, Return Structs

> **"Interfaces should be declared on the consumer side, not the producer side"**

### Rules for Interface Declaration

1. **Declare interfaces where they are used (consumer side)**
   - ✅ CORRECT: Application layer declares `EventStore` interface it depends on
   - ❌ WRONG: Infrastructure layer declares `EventStore` interface it implements

2. **Keep interfaces small and focused**
   - Prefer small, single-purpose interfaces
   - Large interfaces are acceptable if the consumer needs all methods

3. **Cross-domain dependencies require local interfaces**
   - If domain A uses domain B's repository, declare the interface in domain A
   - Example: `tag` domain declares `ChatRepository`, `UserRepository`, `MessageRepository` locally

### Directory-Specific Patterns

#### Application Layer (`internal/application/`)
```go
// ✅ CORRECT: Application layer owns interfaces it depends on
package shared

type EventStore interface {
    SaveEvents(...) error
    LoadEvents(...) error
}

type UserRepository interface {
    Exists(ctx context.Context, userID uuid.UUID) (bool, error)
}
```

#### Domain Layer (`internal/domain/`)
```go
// ✅ CORRECT: Domain declares interfaces for dependencies it needs
package tag

type ChatRepository interface {
    Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)
    Save(ctx context.Context, chat *chat.Chat) error
}

// Note: Still import domain types (chat.Chat), but NOT their repository interfaces
```

#### Infrastructure Layer (`internal/infrastructure/`)
```go
// ✅ CORRECT: Infrastructure implements interfaces, doesn't declare them
package eventstore

import "github.com/lllypuk/flowra/internal/application/shared"

type InMemoryEventStore struct { ... }

// Implements shared.EventStore interface
func (s *InMemoryEventStore) SaveEvents(...) error { ... }
```

### Anti-Patterns to Avoid

❌ **DON'T declare interfaces in infrastructure**
```go
// BAD: infrastructure/eventstore/eventstore.go
type EventStore interface { ... }  // ❌ Wrong location!
```

❌ **DON'T import repository interfaces across domains**
```go
// BAD: domain/tag/executor.go
import "github.com/lllypuk/flowra/internal/domain/chat"

type Executor struct {
    chatRepo chat.Repository  // ❌ Cross-domain interface dependency!
}
```

❌ **DON'T create generic repository interfaces in shared packages** (unless truly needed everywhere)
```go
// BAD (usually): domain/shared/repository.go
type Repository[T any] interface { ... }  // ❌ Premature abstraction!
```

### When to Share Interfaces

Only share interfaces when multiple consumers need the **exact same interface**:
- `application/shared/` - for interfaces used by multiple use cases
- NOT for "might be reused someday" - wait until actual reuse happens

### Migration Checklist

When adding a new dependency:
1. ✅ Declare interface in the **consumer** package
2. ✅ Import only domain **types**, not interfaces
3. ✅ Implementation imports interface from consumer
4. ✅ Use dependency injection to wire everything together

### Example: Correct Interface Usage

```go
// Package: internal/application/task
type TaskUseCase struct {
    eventStore shared.EventStore      // ✅ Interface from shared (consumer side)
    userRepo   shared.UserRepository   // ✅ Interface from shared (consumer side)
}

// Package: internal/domain/tag
type CommandExecutor struct {
    chatRepo ChatRepository    // ✅ Interface declared in THIS package
    userRepo UserRepository    // ✅ Interface declared in THIS package
}

// Package: internal/infrastructure/persistence
type MongoUserRepository struct { ... }

// ✅ Implements shared.UserRepository (from consumer)
func (r *MongoUserRepository) Exists(...) (bool, error) { ... }
```

### Benefits of This Approach

- **Loose coupling**: Consumers don't depend on infrastructure packages
- **Testability**: Easy to mock dependencies
- **Flexibility**: Change implementations without affecting consumers
- **Idiomatic Go**: Follows community best practices
- **Clear ownership**: Interface changes driven by consumer needs

## Task Documentation Guidelines

When creating or updating task documentation in markdown files:

### No Time Estimates

**DO NOT include time estimates in task files.** This includes:
- ❌ "Оценка: 3-4 часа"
- ❌ "Time spent: ~4 hours"
- ❌ "Estimated time: 2h"
- ❌ Any form of time prediction or tracking

**Why:**
- Time estimates are often inaccurate and become outdated
- They create unnecessary pressure and false expectations
- Focus should be on task completion, not time tracking
- Actual time spent varies significantly based on context

**Instead, focus on:**
- ✅ Clear task description
- ✅ Checklist of deliverables
- ✅ Dependencies between tasks
- ✅ Status (Pending, In Progress, Complete)
- ✅ Priority when relevant
