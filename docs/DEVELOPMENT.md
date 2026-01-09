# Flowra Development Guide

This guide covers setting up the development environment and working with the Flowra codebase.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Project Setup](#project-setup)
3. [Development Workflow](#development-workflow)
4. [Code Organization](#code-organization)
5. [Testing](#testing)
6. [Code Quality](#code-quality)
7. [Database Operations](#database-operations)
8. [Debugging](#debugging)
9. [Contributing](#contributing)

---

## Prerequisites

### Required Tools

| Tool | Version | Installation |
|------|---------|--------------|
| Go | 1.25+ | [golang.org/dl](https://golang.org/dl/) |
| Docker | 24.0+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose | 2.20+ | Included with Docker Desktop |
| Make | 3.8+ | Usually pre-installed |
| golangci-lint | 1.55+ | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |

### Optional Tools

| Tool | Purpose |
|------|---------|
| [Air](https://github.com/cosmtrek/air) | Hot reload for Go |
| [mongosh](https://www.mongodb.com/try/download/shell) | MongoDB shell |
| [redis-cli](https://redis.io/docs/getting-started/) | Redis CLI |
| [VS Code](https://code.visualstudio.com/) | Recommended IDE |

### VS Code Extensions

Recommended extensions for VS Code:

- Go (golang.go)
- YAML
- OpenAPI (Swagger) Editor
- Docker
- EditorConfig

---

## Project Setup

### 1. Clone Repository

```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
```

### 2. Install Dependencies

```bash
# Download Go modules
make deps

# Or manually
go mod download
go mod tidy
```

### 3. Start Infrastructure Services

```bash
# Start MongoDB, Redis, and Keycloak
docker-compose up -d

# Verify services are running
docker-compose ps

# Expected output:
# flowra-mongodb   running   0.0.0.0:27017->27017/tcp
# flowra-redis     running   0.0.0.0:6379->6379/tcp
# flowra-keycloak  running   0.0.0.0:8090->8080/tcp
```

### 4. Configure Application

```bash
# Configuration is in configs/config.yaml
# Default values work for local development

# Optionally, override via environment variables:
export FLOWRA_LOG_LEVEL=debug
```

### 5. Start the Application

```bash
# Development mode
make dev

# Or directly
go run cmd/api/main.go
```

### 6. Verify Setup

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# Readiness check
curl http://localhost:8080/ready
# Expected: {"status":"ready","components":{...}}
```

---

## Development Workflow

### Make Commands

```bash
# Show all available commands
make help

# Common commands:
make dev           # Run in development mode
make build         # Build binaries
make test          # Run all tests
make test-unit     # Run unit tests only
make test-e2e      # Run E2E tests
make lint          # Run linter and format code
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make docker-logs   # View Docker logs
make clean         # Clean build artifacts
```

### Hot Reload (Optional)

For automatic reloading during development, use Air:

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Create .air.toml (if not exists)
air init

# Run with hot reload
air
```

Example `.air.toml`:

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/api"
bin = "./tmp/main"
include_ext = ["go", "yaml"]
exclude_dir = ["tmp", "vendor", "docs", "web"]
delay = 1000

[log]
time = true

[misc]
clean_on_exit = true
```

### Git Workflow

1. Create feature branch from `main`
2. Make changes with descriptive commits
3. Run tests and linter
4. Create pull request
5. Address review feedback
6. Squash and merge

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make commits
git commit -m "feat: add new feature"

# Run checks before pushing
make lint
make test

# Push and create PR
git push origin feature/my-feature
```

---

## Code Organization

### Directory Structure

```
.
├── cmd/                   # Application entry points
│   ├── api/               # HTTP API server
│   │   ├── main.go        # Entry point
│   │   ├── container.go   # Dependency injection
│   │   └── routes.go      # Route registration
│   └── worker/            # Background worker
│
├── internal/              # Private application code
│   ├── application/       # Application services (use cases)
│   ├── domain/           # Domain models and business logic
│   │   ├── chat/         # Chat aggregate
│   │   ├── message/      # Message aggregate
│   │   ├── task/         # Task aggregate
│   │   ├── notification/ # Notification aggregate
│   │   ├── user/         # User entity
│   │   ├── workspace/    # Workspace entity
│   │   ├── event/        # Domain events
│   │   ├── tag/          # Tag system
│   │   ├── uuid/         # UUID utilities
│   │   └── errs/         # Domain errors
│   ├── handler/          # HTTP and WebSocket handlers
│   │   ├── http/         # REST API handlers
│   │   └── websocket/    # WebSocket handler
│   ├── infrastructure/   # External dependencies
│   │   ├── repository/   # Data access
│   │   ├── eventstore/   # Event persistence
│   │   ├── httpserver/   # HTTP server utilities
│   │   └── websocket/    # WebSocket infrastructure
│   ├── middleware/       # HTTP middleware
│   └── config/           # Configuration
│
├── pkg/                   # Public packages (if any)
│
├── web/                   # Frontend resources
│   ├── templates/        # HTML templates
│   ├── static/           # Static assets
│   └── components/       # HTMX components
│
├── tests/                # Test suites
│   ├── e2e/             # End-to-end tests
│   ├── integration/     # Integration tests
│   └── testutil/        # Test utilities
│
├── configs/             # Configuration files
├── docs/                # Documentation
└── docker-compose.yml   # Local development services
```

### Architecture Layers

```
┌─────────────────────────────────────────────┐
│                  Handlers                    │
│         (HTTP, WebSocket, HTMX)             │
├─────────────────────────────────────────────┤
│              Application Layer               │
│        (Use Cases, Commands, Queries)        │
├─────────────────────────────────────────────┤
│                Domain Layer                  │
│   (Aggregates, Entities, Value Objects)     │
├─────────────────────────────────────────────┤
│            Infrastructure Layer              │
│   (Repositories, Event Store, External)     │
└─────────────────────────────────────────────┘
```

### Key Design Principles

1. **Event-Driven Architecture** - Business logic produces domain events
2. **Domain-Driven Design** - Rich domain models encapsulate business rules
3. **CQRS** - Separate command and query responsibilities
4. **Repository Pattern** - Abstract data access
5. **Dependency Injection** - Loose coupling via interfaces

---

## Testing

### Test Structure

```
tests/
├── e2e/              # End-to-end API tests
│   ├── auth_test.go
│   ├── workspace_test.go
│   └── ...
├── integration/      # Integration tests
└── testutil/         # Shared test utilities
    ├── db.go         # Database helpers
    ├── mongodb.go    # MongoDB test setup
    └── fixtures.go   # Test data fixtures
```

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run E2E tests (requires testcontainers)
make test-e2e

# Run E2E tests with docker-compose MongoDB
make test-e2e-docker

# Run specific test
go test -v ./internal/domain/chat/...

# Run with coverage
make test-coverage

# Check coverage threshold (80%)
make test-coverage-check
```

### Writing Tests

#### Unit Tests

```go
// internal/domain/chat/chat_test.go
package chat_test

import (
    "testing"

    "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestChat_Rename(t *testing.T) {
    // Arrange
    c, _ := chat.NewChat(uuid.NewUUID(), uuid.NewUUID(), "Original")

    // Act
    err := c.Rename("New Name", c.CreatedBy())

    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if c.Name() != "New Name" {
        t.Errorf("expected name 'New Name', got '%s'", c.Name())
    }
}
```

#### Integration Tests

```go
// tests/integration/repository_test.go
//go:build integration

package integration

import (
    "context"
    "testing"

    "github.com/lllypuk/flowra/tests/testutil"
)

func TestChatRepository_Save(t *testing.T) {
    // Setup test database
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)

    repo := repository.NewMongoChatRepository(db)

    // Test implementation...
}
```

#### E2E Tests

```go
// tests/e2e/workspace_test.go
//go:build e2e

package e2e

import (
    "net/http"
    "testing"
)

func TestWorkspace_Create(t *testing.T) {
    client := setupTestClient(t)

    resp, err := client.Post("/api/v1/workspaces", map[string]any{
        "name": "Test Workspace",
    })

    if err != nil {
        t.Fatalf("request failed: %v", err)
    }
    if resp.StatusCode != http.StatusCreated {
        t.Errorf("expected status 201, got %d", resp.StatusCode)
    }
}
```

### Test Utilities

```go
// Use testutil for common operations
import "github.com/lllypuk/flowra/tests/testutil"

// Create test database connection
db := testutil.SetupTestDB(t)

// Create test fixtures
user := testutil.CreateTestUser(t, db)
workspace := testutil.CreateTestWorkspace(t, db, user.ID())
```

---

## Code Quality

### Linting

```bash
# Run linter with auto-fix
make lint

# Or manually
golangci-lint run --fix
```

### Linter Configuration

See `.golangci.yml` for full configuration. Key rules:

- `gofmt` - Code formatting
- `goimports` - Import ordering
- `errcheck` - Error handling
- `staticcheck` - Static analysis
- `gosec` - Security issues
- `gocritic` - Code style

### Code Formatting

```bash
# Format all Go files
go fmt ./...

# Organize imports
goimports -w .
```

### Pre-commit Checks

Run before every commit:

```bash
# Quick check
make lint && make test-unit

# Full check
make lint && make test
```

---

## Database Operations

### MongoDB Shell

```bash
# Connect to MongoDB
mongosh "mongodb://admin:admin123@localhost:27017"

# Switch to flowra database
use flowra

# Common operations
db.chats.find().limit(10)
db.users.countDocuments()
db.messages.createIndex({ chat_id: 1, created_at: -1 })
```

### Redis CLI

```bash
# Connect to Redis
redis-cli -h localhost -p 6379

# Common operations
KEYS *
GET session:user:123
PUBLISH events.chat.message '{"type":"message_posted"}'
```

### Database Schema & Indexes

**MongoDB indexes are automatically managed in Go code.**

#### Index Management

Indexes are defined in `internal/infrastructure/mongodb/indexes.go` and created automatically:

- **On API startup**: Indexes are created when the API server starts (in `cmd/api/container.go`)
- **In tests**: Indexes are created during test database setup

#### Index Functions

```go
// Create all indexes for all collections
mongodb.CreateAllIndexes(ctx, db)

// Alias for semantic clarity
mongodb.EnsureIndexes(ctx, db)

// Get indexes for specific collections
mongodb.GetEventIndexes()
mongodb.GetUserIndexes()
mongodb.GetChatReadModelIndexes()
mongodb.GetTaskReadModelIndexes()
mongodb.GetMessageIndexes()
mongodb.GetNotificationIndexes()
mongodb.GetWorkspaceIndexes()
mongodb.GetMemberIndexes()
```

#### Adding New Indexes

To add a new index:

1. Add the index definition to the appropriate function in `indexes.go`
2. Restart the API server - indexes are created idempotently
3. No migration files needed

#### Collections

Main MongoDB collections:
- `events` - Event Store (event sourcing)
- `users` - User read model
- `workspaces` - Workspace read model
- `workspace_members` - Workspace membership
- `chat_read_model` - Chat read model
- `task_read_model` - Task read model
- `messages` - Message read model
- `notifications` - Notification read model

---

## Debugging

### Logging

```go
import "log/slog"

// Use structured logging
slog.Info("processing request",
    slog.String("user_id", userID.String()),
    slog.String("action", "create_chat"),
)

slog.Error("failed to save",
    slog.String("error", err.Error()),
    slog.String("chat_id", chatID.String()),
)
```

### Debug Mode

```bash
# Enable debug logging
export FLOWRA_LOG_LEVEL=debug
make dev
```

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Runtime profiling (if enabled)
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug application
dlv debug ./cmd/api

# Attach to running process
dlv attach <pid>
```

### VS Code Debug Configuration

`.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch API",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/api",
            "env": {
                "FLOWRA_LOG_LEVEL": "debug"
            }
        },
        {
            "name": "Test Current File",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${file}"
        }
    ]
}
```

---

## Contributing

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable and function names
- Keep functions small and focused
- Write comments for exported functions
- Handle all errors explicitly

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: resolve bug in chat creation
docs: update API documentation
refactor: simplify message handler
test: add tests for workspace service
chore: update dependencies
```

### Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add tests for new functionality
4. Request review from maintainers
5. Address feedback
6. Squash and merge

### Interface Guidelines

Follow the project's interface design principles (see CLAUDE.md):

- Declare interfaces on the consumer side
- Keep interfaces small and focused
- Return concrete types, accept interfaces

---

## Useful Resources

- [Go Documentation](https://golang.org/doc/)
- [Echo Framework](https://echo.labstack.com/guide/)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [gorilla/websocket](https://pkg.go.dev/github.com/gorilla/websocket)
- [Keycloak Documentation](https://www.keycloak.org/documentation)

---

*Last updated: January 2026*
