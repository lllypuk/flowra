# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Chat System with Task Management** built in Go. It's a comprehensive chat platform with integrated task tracking, help desk functionality, and command support. The project uses a microservices architecture with event-driven design.

**Key Technologies:**
- **Backend**: Go 1.25+ with Echo v4 framework
- **Database**: MongoDB 8+ (main), Redis (cache/pub-sub)
- **Frontend**: HTMX 2+ for dynamic updates, Pico CSS v2 for styling
- **Auth**: Keycloak for SSO and user management
- **Infrastructure**: Docker Compose for development

## Development Commands

### Environment Setup
```bash
# Start infrastructure services
docker-compose up -d mongodb redis keycloak

# Start the main application (when implemented)
go run cmd/api/main.go
```

### Code Quality
```bash
# Run linting with comprehensive Go linting rules
golangci-lint run

# Run tests (when implemented)
go test ./...
go test ./tests/integration -tags=integration
go test ./tests/e2e -tags=e2e
```

### Build and Development
```bash
# Build application (when build targets are added to Makefile)
make build

# Development mode with hot reload (when implemented)
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

- **Primary**: MongoDB 8+ (document store)
- **Cache**: Redis for sessions, pub/sub, caching
- Main collections: Users, Chats, Messages, Tasks, Chat_members, Audit_log
- Schema versioning handled through application code

## Development Notes

- This is currently in **planning/architecture phase** - no Go source code exists yet
- The project follows the planned structure described in extensive documentation
- All business logic will be event-driven with proper domain boundaries
- HTMX will be used for minimal JavaScript frontend with server-side rendering
- Comprehensive linting rules are configured in `.golangci.yml`
- Security-first approach with Keycloak SSO, RBAC, and secure defaults

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
