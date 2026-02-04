# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Chat System with Task Management** built in Go. It's a comprehensive chat platform with integrated task tracking, help desk functionality, and command support. The project uses a microservices architecture with event-driven design.

**Key Technologies:**
- **Backend**: Go 1.24+ with Echo v4 framework
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

# Development mode (starts infrastructure + API server)
make dev
```

### Local Testing and Development

**Starting the Development Server:**
```bash
# Quick start (recommended)
make dev

# Manual start
docker-compose up -d  # Start infrastructure
go run cmd/api/main.go  # Start API server

# Alternative: build and run
go build -o /tmp/flowra-api cmd/api/*.go
/tmp/flowra-api
```

**Test User Credentials:**
- Username: `testuser`
- Password: `test123`

**Testing with Chrome DevTools MCP:**
When testing UI changes or debugging frontend issues, use the Chrome DevTools MCP server:

1. Navigate to the application: `http://localhost:8080`
2. Login with test credentials
3. Use DevTools commands:
   - `take_snapshot` - Get accessibility tree of current page
   - `take_screenshot` - Capture visual state
   - `click`, `fill`, `navigate_page` - Interact with UI
   - `evaluate_script` - Run JavaScript for debugging

Example workflow:
```bash
# 1. Start server
make dev

# 2. In Claude Code, use DevTools to test
# Navigate to: http://localhost:8080/workspaces/{workspace_id}/chats/{chat_id}
# Login: testuser / test123
# Test UI elements, check layout, verify functionality
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
- **Worker Service** - Background tasks (SLA monitoring, notifications, user sync)
- **Command Processor** - Chat command parsing and execution

### Directory Layout
```
cmd/                           # Application entry points
├── api/                      # HTTP API server (main.go, container.go, routes.go)
└── worker/                   # Background workers (user sync)

internal/                     # Internal application code (296 files)
├── application/             # Use cases (139 files, CQRS)
│   ├── appcore/            # Shared interfaces and utilities
│   ├── auth/               # Authentication use cases
│   ├── chat/               # Chat use cases
│   ├── eventhandler/       # Event handlers
│   ├── message/            # Message use cases
│   ├── notification/       # Notification use cases
│   ├── task/               # Task use cases
│   ├── user/               # User use cases
│   └── workspace/          # Workspace use cases
├── domain/                  # Business logic and models (48 files)
│   ├── chat/               # Chat aggregate
│   ├── message/            # Message aggregate
│   ├── task/               # Task aggregate
│   ├── user/               # User aggregate
│   ├── workspace/          # Workspace aggregate
│   ├── notification/       # Notification aggregate
│   ├── tag/                # Tag/command processing system
│   ├── event/              # Event base types
│   ├── uuid/               # UUID utilities
│   └── errs/               # Domain errors
├── infrastructure/          # External dependencies (50 files)
│   ├── mongodb/            # MongoDB index setup
│   ├── repository/         # MongoDB repositories (6 repos)
│   ├── eventstore/         # Event store implementation
│   ├── eventbus/           # Redis pub/sub event bus
│   ├── httpserver/         # HTTP server utilities
│   ├── websocket/          # WebSocket hub and client
│   ├── keycloak/           # Keycloak SSO integration
│   └── auth/               # Token store
├── handler/                 # HTTP/WS handlers (28 files)
│   ├── http/               # REST API handlers
│   └── websocket/          # WebSocket handler
├── middleware/              # HTTP middleware (14 files)
├── service/                 # Business services (13 files)
├── worker/                  # Background workers
└── config/                  # Configuration loading

web/                          # Frontend resources (53 files)
├── templates/               # HTML templates
│   ├── layout/             # Base layout (base, navbar, footer)
│   ├── components/         # Reusable HTMX components
│   ├── auth/               # Auth pages
│   ├── workspace/          # Workspace pages
│   ├── chat/               # Chat pages
│   ├── task/               # Task pages
│   ├── board/              # Board pages
│   └── notification/       # Notification pages
├── components/              # Reusable components
├── static/                  # CSS, JS assets
└── embed.go                 # Go embed for static files

tests/                        # Test suites (33 files)
├── e2e/                     # End-to-end tests
├── integration/             # Integration tests
├── testutil/                # Test utilities
├── fixtures/                # Test data fixtures
└── mocks/                   # Mock implementations

configs/                      # Configuration files (YAML)
docs/                         # Documentation (9+ files)
```

## Configuration

- Main config: `configs/config.yaml`
- Development config: `configs/config.dev.yaml`
- Production config: `configs/config.prod.yaml`
- Environment-specific values override via environment variables
- Docker services configured in `docker-compose.yml`
- Comprehensive settings for database, Redis, JWT, OAuth, email, etc.

## Index Management

**MongoDB indexes are managed in Go code, not external migration files.**

- **Index definitions**: `internal/infrastructure/mongodb/indexes.go`
- **Automatic creation**: Indexes are created automatically when the API server starts
- **Test setup**: Indexes are also created in test utilities for integration tests

### Index Creation Functions:
- `CreateAllIndexes(ctx, db)` - Creates all indexes for all collections
- `EnsureIndexes(ctx, db)` - Alias for CreateAllIndexes
- Individual getters: `GetEventIndexes()`, `GetUserIndexes()`, `GetChatReadModelIndexes()`, etc.

### When Indexes Are Created:
1. **API Startup** - In `cmd/api/container.go` during MongoDB initialization
2. **Test Setup** - In `tests/testutil/db.go` and `tests/testutil/mongodb_shared.go`

**No separate migration runner is needed** - the application manages its own schema.

## Database

- **Primary**: MongoDB 6+ (document store with replica set)
- **Cache**: Redis for sessions, pub/sub, caching
- Main collections: Users, Chats, Messages, Tasks, Chat_members, Notifications, Events
- Schema and indexes managed through application code (see Index Management section above)

## Development Notes

- **Project Status**: February 2026 Release Candidate (~95% complete)
- **Backend**: Fully production-ready with all layers implemented
- **Frontend**: ~25% complete (framework ready, auth + workspace + notifications UI done)

### Layer Implementation Status
| Layer | Status | Files | Coverage |
|-------|--------|-------|----------|
| Domain | Complete | 48 | 90%+ |
| Application | Complete | 139 | 85%+ |
| Infrastructure | Complete | 50 | 85%+ |
| Handlers | Complete | 28 | 80%+ |
| Middleware | Complete | 14 | 80%+ |
| Services | Complete | 13 | 80%+ |
| Frontend | In Progress | ~54 | - |

### Key Implementation Details
- 6 Event-Sourced Aggregates (Chat, Message, Task, User, Workspace, Notification)
- 40+ Use Cases with CQRS pattern
- 40+ REST API endpoints
- Real-time WebSocket with Hub pattern
- Full Keycloak SSO integration
- E2E tests cover all critical workflows
- Comprehensive linting rules in `.golangci.yml`
- Security-first approach with RBAC and secure defaults

## Documentation Structure

- **API Documentation**: `docs/api/` - OpenAPI spec, Postman collection
- **Deployment Guide**: `docs/DEPLOYMENT.md` - Docker, environment setup
- **Development Guide**: `docs/DEVELOPMENT.md` - Local setup, testing
- **Architecture Overview**: `docs/ARCHITECTURE.md` - System design, decisions
- **Frontend Guide**: `docs/FRONTEND_DEV_GUIDE.md` - Frontend development
- **User Guide**: `docs/USER_GUIDE.md` - End user documentation

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

| Service | URL | Credentials |
|---------|-----|-------------|
| API Server | http://localhost:8080 | testuser / test123 |
| API Health | http://localhost:8080/health | - |
| API Docs | http://localhost:8080/docs | - |
| Keycloak Admin | http://localhost:8090 | admin / admin123 |
| MongoDB | localhost:27017 | admin / admin123 |
| Redis | localhost:6379 | - |

**Test User:** Use `testuser` / `test123` to login to the application via SSO.

## Testing Strategy

- Unit tests for all business logic
- Integration tests with MongoDB (testcontainers)
- E2E tests for user workflows
- Load testing for performance validation
- Test database uses in-memory MongoDB (testcontainers)

### Test Commands
```bash
make test               # Run all tests
make test-unit          # Run unit tests only
make test-integration   # Run integration tests
make test-e2e           # Run E2E tests
make test-coverage      # Generate coverage report
make test-coverage-check # Check 80% threshold
```

**Note**: All tests automatically create MongoDB indexes during test database setup.

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
- ❌ "Estimate: 3-4 hours"
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

## Code Style Guidelines

### Avoid Reflection

**Prefer type assertions and generics over reflection (`reflect` package).**

This includes:
- ❌ Using `reflect.ValueOf()`, `reflect.TypeOf()`, etc.
- ❌ Runtime type inspection when compile-time alternatives exist
- ❌ Generic operations that can be done with type assertions

**Why:**
- Reflection is slower and less performant
- Type assertions are compile-time safe
- Code is more explicit and easier to understand
- Better IDE support and tooling

**When to use reflection:**
- ✅ Only when absolutely necessary (JSON/YAML marshaling, ORM libraries, etc.)
- ✅ When dealing with truly dynamic types from external sources
- ✅ In library code that must handle arbitrary types

**Prefer instead:**
- ✅ Type assertions with type switches
- ✅ Generics (Go 1.18+) for type-safe generic code
- ✅ Interface-based polymorphism
- ✅ Code generation tools for boilerplate

**Example:**
```go
// ❌ BAD: Using reflection
func length(v any) int {
    rv := reflect.ValueOf(v)
    if rv.Kind() == reflect.Slice {
        return rv.Len()
    }
    return 0
}

// ✅ GOOD: Using type assertions
func length(v any) int {
    switch val := v.(type) {
    case string:
        return len(val)
    case []any:
        return len(val)
    case []string:
        return len(val)
    default:
        return 0
    }
}
```

## Language Requirements

**All code, comments, and documentation must be written in English.**

This includes:
- ✅ Code comments
- ✅ Function and variable names
- ✅ Error messages and string literals
- ✅ Documentation (markdown files, README, etc.)
- ✅ Commit messages
- ✅ Task descriptions and notes

**Why:**
- English is the standard language for software development
- Ensures accessibility for international contributors
- Maintains consistency across the codebase
- Improves searchability and tooling compatibility

**When writing comments:**
- Use clear, concise English
- Follow standard Go documentation conventions
- Prefer "is returned when..." over "returns when..." for error comments
- Prefer "represents..." or "is a..." for type comments

## Output Guidelines

**DO NOT create summary or documentation markdown files after completing work unless explicitly requested.**

This includes:
- ❌ Creating `docs/fixes/*.md` files after bug fixes
- ❌ Creating `docs/changes/*.md` files after feature implementation
- ❌ Creating summary documents in any location
- ❌ Any unsolicited documentation files

**Why:**
- Clutters the repository with redundant documentation
- Documentation should be intentional and requested
- Code changes and commit messages are sufficient record
- User may have specific documentation standards

**Exceptions:**
- ✅ User explicitly requests documentation: "document this fix in a markdown file"
- ✅ Updating existing documentation files (README, ARCHITECTURE.md, etc.)
- ✅ Adding inline code comments and docstrings
- ✅ Updating CHANGELOG if one exists and is actively maintained

**Instead:**
- Provide summary in chat response
- Update existing documentation if relevant
- Add/improve code comments
- Let the user decide if permanent documentation is needed
