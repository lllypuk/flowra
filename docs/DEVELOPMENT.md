# Flowra Development Guide

This guide covers setting up the development environment and working with the Flowra codebase.

Local runs must not use the live install's ports (`127.0.0.1:8080`, `27017`, `6379`) or its Compose containers;
see [Working Rules](../AGENTS.md#working-rules).

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
| Go | 1.26+ | [golang.org/dl](https://golang.org/dl/) |
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

### 3. Start Infrastructure Services (Optional)

```bash
# Start MongoDB and Redis manually (optional)
docker-compose up -d

# Verify services are running
docker-compose ps

# Expected output:
# flowra-mongodb   running   127.0.0.1:27017->27017/tcp
# flowra-redis     running   127.0.0.1:6379->6379/tcp
```

`make dev` already starts these services automatically. Manual startup is useful when you need to run `make reset-data`
before launching the full stack.

### 4. Configure Application

```bash
# Configuration is in configs/config.yaml; the API needs an owner token digest
token=$(openssl rand -hex 32); echo "$token"
export AUTH_OWNER_TOKEN_SHA256=$(printf %s "$token" | sha256sum | cut -d' ' -f1)

# Optionally, override via environment variables:
export LOG_LEVEL=debug
```

### 5. Start the Application

```bash
# Infra + one API process with the worker inside (make dev-local is the same)
make dev

# API only, infra already running
make dev-lite
```

### 6. Verify Setup

```bash
# Health check
curl http://127.0.0.1:8080/health
# Expected: {"status":"healthy"}

# Readiness check
curl http://127.0.0.1:8080/ready
# Expected: {"status":"ready","components":{...}}
```

### 7. Sign-in

The server is for a single owner on one machine. It refuses to start on a non-loopback `server.host`, and
nothing falls back to mock or development auth.

- Web: http://127.0.0.1:8080/login takes the token and sets an HttpOnly session cookie.
- API: `Authorization: Bearer <token>`.
- More tokens (one per device) go under `auth.tokens` in the config as `name` + `sha256`; every token
  resolves to the same `owner` user, so rotating a token does not change the user.

### 8. Lead Feed

The feed is a projection of the lead's task journal, kept apart from chats and tasks. The owner's answers to
questions are stored in Flowra and delivered through `lead-task answer`; the journal is never written directly.

```bash
make feed-sync                         # lead-task export -> temp file -> bin/leadfeed import
make feed-status
bin/leadfeed import snapshot.jsonl     # or from stdin: ... | bin/leadfeed import
```

`scripts/leadfeed-sync.sh` imports only after the exporter has exited with success. Exit codes of
`leadfeed import`: `0` published or confirmed, `1` not published (the previous feed is intact),
`3` commit result unknown (the feed holds the old or the new snapshot in full).

A snapshot is refused when it is truncated, marked incomplete, of an unknown schema, larger than
`-max-bytes`, edited under an unchanged revision, or older than the published one. The reason is shown
by `make feed-status`, by `GET /api/v1/feed/status` and on the feed pages.

To run the parser against real exports, which must stay out of the repository:

```bash
LEADFEED_SAMPLE_DIR=/path/to/exports go test ./internal/application/leadfeed/
```

---

## Development Workflow

### Make Commands

```bash
# Show all available commands
make help

# Common commands:
make dev           # Run infra + API with the worker inside (needs AUTH_OWNER_TOKEN_SHA256)
make dev-lite      # Run lightweight API-only mode (no worker)
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
├── cmd/
│   ├── api/               # HTTP API server (main.go, container.go DI, routes.go)
│   ├── worker/            # Standalone background worker
│   ├── leadfeed/          # Feed CLI: import, deliver, status
│   └── tools/reset_data/  # Local data reset (make reset-data)
├── internal/
│   ├── application/       # Use cases
│   │   ├── leadfeed/      # Lead feed projection, owner questions, answer delivery
│   │   └── ...            # appcore, auth, chat, eventhandler, message, notification, task, user, workspace
│   ├── domain/            # Aggregates, entities, events, tags, errors
│   ├── service/           # Task actions and workspace/chat/member services
│   ├── handler/           # http/ (REST, HTMX pages), websocket/
│   ├── infrastructure/    # mongodb (indexes), eventstore, eventbus, outbox, projector, repair,
│   │                      # repository, filestorage, httpserver, healthcheck, metrics, websocket
│   ├── middleware/        # HTTP middleware
│   ├── worker/            # Outbox and repair workers
│   └── config/            # Configuration
├── web/                   # templates/, static/, components/ (HTMX), embed.go
├── scripts/
│   ├── units/             # systemd user unit templates
│   └── *.sh               # dev stack, feed sync, unit install, data reset
├── tests/
│   ├── e2e/               # API tests; frontend/ holds Playwright tests
│   ├── integration/       # Testcontainers-based tests
│   ├── scripts/           # Tests of scripts/
│   ├── fixtures/leadfeed/ # Synthetic feed snapshots
│   ├── load/              # Manual load tests (k6, agentbench)
│   ├── mocks/             # Reusable mocks
│   └── testutil/          # Test helpers
├── configs/               # config.yaml, dev/prod overlays, Prometheus/Grafana
├── docs/                  # Documentation; plans/ is design history
└── docker-compose.yml     # Local MongoDB and Redis on loopback
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

See the `tests/` tree above and [tests/README.md](../tests/README.md).

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run E2E tests (requires testcontainers)
make test-e2e

# Run frontend E2E tests (Playwright)
make test-e2e-frontend

# Run focused board+sidebar smoke regression
make test-e2e-frontend-smoke

# Run integration tests (requires Docker/testcontainers)
make test-integration

# Run specific test
go test -v ./internal/domain/chat/...

# Run with coverage
make test-coverage
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

#### Integration and E2E Tests

Both are build-tagged (`integration`, `e2e`) and start containers; see `tests/integration/` and `tests/e2e/setup_test.go`.

### Test Utilities

`tests/testutil` provides MongoDB and Redis setup (`SetupTestMongoDB`, `SetupSharedTestMongoDB`, ...) on testcontainers.

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
mongosh "mongodb://127.0.0.1:27017/?replicaSet=rs0"

# Switch to flowra database
use flowra

# Common operations
db.users.countDocuments()
db.leadfeed_tasks.find().limit(5)
```

### Redis CLI

```bash
# Connect to Redis
redis-cli -h 127.0.0.1 -p 6379

PING
```

### Database Schema & Indexes

**MongoDB indexes are automatically managed in Go code.**

#### Index Management

Indexes are defined in `internal/infrastructure/mongodb/indexes.go` and created automatically:

- **On API startup**: `CreateAllIndexes` in `cmd/api/container.go`
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
// also Outbox, RepairQueue, FileMetadata, LeadFeed, OwnerAnswer
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
- `chats_read_model` - Chat read model
- `tasks_read_model` - Task read model
- `messages` - Message read model
- `notifications` - Notification read model
- `outbox`, `repair_queue`, `file_metadata` - infrastructure
- `leadfeed_tasks`, `leadfeed_records`, `leadfeed_state`, `owner_answers` - lead feed and owner answers

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
export LOG_LEVEL=debug
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
                "LOG_LEVEL": "debug"
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

Follow the project's interface design principles (see AGENTS.md):

- Declare interfaces on the consumer side
- Keep interfaces small and focused
- Return concrete types, accept interfaces

---

## Useful Resources

- [Go Documentation](https://golang.org/doc/)
- [Echo Framework](https://echo.labstack.com/guide/)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [gorilla/websocket](https://pkg.go.dev/github.com/gorilla/websocket)

---

*Last updated: November 2026*
