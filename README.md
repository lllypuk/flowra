# Flowra

A comprehensive chat system with integrated task tracker, help desk functionality, and team support.

## 📊 Current Project Status

**Version:** 1.0.0-beta
**Last Updated:** 2026-03-02
**Overall Progress:** ~98% to MVP
**Status:** March 2026 Release Candidate

### Progress by Layer

| Layer | Status | Progress | Files | Coverage |
|-------|--------|----------|-------|----------|
| **Domain** | ✅ Complete | 100% | 48 | 90%+ |
| **Application** | ✅ Complete | 100% | 139 | 85%+ |
| **Infrastructure** | ✅ Complete | 100% | 50 | 85%+ |
| **Handlers** | ✅ Complete | 100% | 28 | 80%+ |
| **Middleware** | ✅ Complete | 100% | 14 | 80%+ |
| **Services** | ✅ Complete | 100% | 13 | 80%+ |
| **Frontend** | 🔄 Mostly Complete | 85% | ~70 | - |
| **Entry Points** | ✅ Complete | 100% | 6 | 75%+ |

### What Works ✅

- ✅ **Domain Layer:** 6 Event-Sourced aggregates, 30+ domain events
- ✅ **Application Layer:** 40+ use cases with 85% average coverage
- ✅ **MongoDB Repositories:** All 6 repositories with integration tests
- ✅ **Event Store:** MongoDB Event Store with optimistic locking
- ✅ **Event Bus:** Redis pub/sub for cross-service events
- ✅ **HTTP Handlers:** Full REST API with 40+ endpoints
- ✅ **WebSocket:** Real-time communication with Hub pattern
- ✅ **Middleware:** Auth, CORS, Logging, Recovery, Rate Limiting, Workspace Access
- ✅ **Services:** Workspace Access Checker, Chat, Member, Auth services
- ✅ **Keycloak Integration:** Full SSO integration (JWT, OAuth, User Sync)
- ✅ **Entry Points:** API server, Worker (with User Sync)
- ✅ **E2E Tests:** Full coverage of critical flows
- ✅ **API Documentation:** OpenAPI 3.1, Postman collection
- ✅ **Tag System Frontend:** Human-readable messages, visual grouping, sidebar integration
- ✅ **Tag Processing Test Coverage:** Concurrent integration tests + bot-response E2E scenarios
- ✅ **Tag System Load Testing:** k6 script/profiles for tag-heavy message flows (`tests/load/tag-system`)
- ✅ **Landing Page:** Redesigned with distinctive typography, scroll animations, responsive feature grid
- ✅ **Chat=SoT Stabilization:** projection-safe typed mutations, strict assignee validation, frontend smoke regression coverage

### In Development 🔄

- 🔄 **Frontend:** HTMX + Pico CSS (~85% complete — all core features, settings, enhancements, dark mode, search, file uploads done)

---

## 🚀 Quick Start (5 minutes)

### Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Make

### 1. Clone & Setup

```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
make deps
```

### 2. Start Full-Stack Runtime

```bash
# Starts infra (MongoDB, Redis, Keycloak) + worker + API
make dev
```

### 3. Verify

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# API documentation
open http://localhost:8080/docs
```

### 4. Test Authentication

Access Keycloak at http://localhost:8090 (admin/admin123) to configure OAuth.

---

## 🔧 Development Commands

```bash
make help          # Show all available commands
make dev                # Run full-stack development mode (infra + worker + API)
make dev-lite           # Run API-only development mode (limited: no worker)
make build              # Build binaries
make test               # Run all tests
make test-unit          # Run unit tests only
make test-integration   # Run integration tests (Docker/testcontainers)
make test-e2e           # Run E2E API tests
make test-e2e-frontend  # Run frontend browser E2E tests (requires running server)
make test-e2e-frontend-smoke # Run focused board+sidebar smoke regression
make test-load-tags     # Run k6 tag-system load test (requires k6 + AUTH_TOKEN)
make lint               # Run linter and format code
make docker-up          # Start Docker services
make docker-down        # Stop Docker services
make test-coverage      # Generate coverage report
make playwright-install # Install Playwright browsers for frontend tests
make reset-data         # Reset local/dev Chat=SoT data collections and recreate indexes
```

When switching between branches around the Chat=SoT refactor, run:
`make docker-up && make reset-data` before `make dev` to avoid stale
event/read-model shape mismatches.

---

## 🏗️ Key Features

- **Real-time chat** with group and direct message support
- **Command system** for managing tasks directly from chat
- **Task management** with state machine for statuses
- **Help Desk** functionality with SLA tracking
- **Keycloak integration** for SSO and user management
- **HTMX + Alpine.js** for minimal JavaScript usage
- **WebSocket** for real-time updates
- **Event Sourcing** for complete change history
- **Tag Processing** - command processing system via message tags
- **Human-readable system messages** - "John changed status to In Progress" format
- **Real-time UI updates** - sidebar actions create visible system messages in chat

---

## 🎯 Domain Models

### Chat Aggregate
- **Types**: Direct message, Group chat, Help Desk ticket
- **Operations**: Create, AddParticipant, RemoveParticipant, Rename, SetSeverity, SetPriority

### Message Aggregate
- **Capabilities**: Content, attachments, reactions, threading
- **Operations**: Create, Edit, Delete, AddAttachment, AddReaction

### Typed Entities (Chat = SoT)
- **Types**: Task, Bug, Epic (as typed chat variants)
- **State Ownership**: status/priority/assignee/due date are written through Chat commands
- **Events**: business writes emit `chat.*` events only

### Notification Aggregate
- **Types**: Task, Chat, Mention, System
- **Operations**: Create, MarkAsRead, MarkAllAsRead, Delete

### User & Workspace Entities
- **User**: Registration, Profile updates, Admin promotion
- **Workspace**: Create, Update, Member management

---

## 📋 Documentation

### API Documentation
- [API Overview](./docs/api/README.md) - Complete API description
- [OpenAPI Spec](./docs/api/openapi.yaml) - OpenAPI 3.1 specification
- [Postman Collection](./docs/api/postman_collection.json) - Ready-to-use collection for testing

### Guides
- [Deployment Guide](./docs/DEPLOYMENT.md) - Deployment instructions
- [Development Guide](./docs/DEVELOPMENT.md) - Developer environment setup
- [Architecture](./docs/ARCHITECTURE.md) - System architecture overview
- [Tag Load Test Guide](./tests/load/tag-system/README.md) - k6 profiles, auth setup, and metrics for tag-system load runs

### Architecture & Design
- [Architecture Overview](./docs/01-architecture.md) - Detailed architecture
- [Domain Model](./docs/02-domain-model.md) - Domain model
- [Security Model](./docs/04-security-model.md) - Security model
- [Event Flow](./docs/05-event-flow.md) - Event flows
- [API Contracts](./docs/06-api-contracts.md) - API contracts

---

## 🛠 Technology Stack

### Backend
| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Primary language | 1.26+ |
| **Echo** | HTTP framework | v4 |
| **gorilla/websocket** | WebSocket | Latest |
| **MongoDB** | Primary database | 6+ |
| **Redis** | Cache/Pub-Sub | 7+ |
| **Keycloak** | Authentication | 23+ |

### Frontend (Planned)
| Technology | Purpose | Version |
|------------|---------|---------|
| **HTMX** | Dynamic updates | 2+ |
| **Pico CSS** | Styling | v2 |
| **Alpine.js** | Interactions | 3+ |

---

## 📁 Project Structure

```
.
├── cmd/                        # Application entry points
│   ├── api/                   # HTTP API server (main, container, routes)
│   └── worker/                # Background worker (user sync)
│
├── internal/                  # Private application code (296 files)
│   ├── application/           # Use cases (139 files, 40+ use cases)
│   │   ├── appcore/          # Shared interfaces
│   │   ├── chat/             # Chat use cases
│   │   ├── message/          # Message use cases
│   │   ├── task/             # Task use cases
│   │   ├── notification/     # Notification use cases
│   │   ├── workspace/        # Workspace use cases
│   │   └── user/             # User use cases
│   ├── domain/               # Domain models (48 files, 6 aggregates)
│   │   ├── chat/             # Chat aggregate
│   │   ├── message/          # Message aggregate
│   │   ├── task/             # Shared task entity state/event contracts (query-side support)
│   │   ├── user/             # User aggregate
│   │   ├── workspace/        # Workspace aggregate
│   │   ├── notification/     # Notification aggregate
│   │   └── tag/              # Tag/command system
│   ├── infrastructure/        # External dependencies (50 files)
│   │   ├── repository/       # MongoDB repositories
│   │   ├── eventstore/       # Event store
│   │   ├── eventbus/         # Redis event bus
│   │   ├── websocket/        # WebSocket hub
│   │   └── keycloak/         # Keycloak integration
│   ├── handler/              # HTTP/WS handlers (28 files)
│   ├── middleware/           # HTTP middleware (14 files)
│   └── service/              # Business services (13 files)
│   └── config/               # Configuration
│
├── web/                       # Frontend (~70 files, HTMX + Pico CSS)
│   ├── templates/            # HTML templates
│   ├── components/           # Reusable components
│   └── static/               # CSS, JS assets
│
├── tests/                     # Test suites
│   ├── e2e/                  # End-to-end tests (incl. bot response/tag flow scenarios)
│   ├── integration/          # Integration tests (incl. concurrent tag processing)
│   ├── load/                 # Manual load tests (k6 scripts and docs)
│   ├── testutil/             # Test utilities (shared Mongo/Keycloak containers)
│   └── mocks/                # Mock implementations
│
├── docs/                      # Documentation
│   ├── api/                  # API documentation
│   ├── deployment/           # Deployment guides
│   └── development/          # Development guides
│
├── configs/                   # Configuration files
└── docker-compose.yml         # Local development services
```

---

## 🔐 Security

- **Authentication**: Keycloak SSO with JWT tokens
- **Authorization**: Role-based access control (RBAC)
- **Workspace Access**: Access verification middleware
- **Input Validation**: Validation at all levels
- **Secure Defaults**: Secure default configuration

---

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run E2E tests
make test-e2e

# Run frontend browser E2E tests
make test-e2e-frontend

# Run focused board+sidebar smoke regression
make test-e2e-frontend-smoke
```

### Test Coverage Targets

| Layer | Target | Current |
|-------|--------|---------|
| Domain | 90% | ✅ 90%+ |
| Application | 80% | ✅ 85%+ |
| Infrastructure | 80% | ✅ 85%+ |
| Handlers | 75% | ✅ 80%+ |

---

## 📊 Application Access

| Service | URL | Credentials |
|---------|-----|-------------|
| **API Server** | http://localhost:8080 | JWT Token |
| **API Docs** | http://localhost:8080/docs | - |
| **Keycloak** | http://localhost:8090 | admin/admin123 |
| **MongoDB** | localhost:27017 | admin/admin123 |
| **Redis** | localhost:6379 | - |

---

## 📈 Roadmap

### ✅ Completed (March 2026)

- Full domain layer with event sourcing (6 aggregates, 30+ events)
- Complete application layer with use cases (40+ use cases)
- MongoDB repositories with integration tests (6 repositories)
- HTTP handlers for all endpoints (40+ REST endpoints)
- WebSocket real-time communication (Hub pattern)
- Authentication & authorization middleware (7 middleware components)
- Keycloak SSO integration (JWT, OAuth, User Sync)
- Business services (Workspace Access, Chat, Member, Auth)
- E2E test coverage (all critical flows)
- API documentation (OpenAPI 3.1, Postman collection)
- Deployment and development documentation
- Frontend framework setup (HTMX + Pico CSS)
- Landing page with distinctive design (Playfair Display + DM Sans, scroll animations)
- Authentication UI (login, logout, callback)
- Workspace management UI (CRUD + full admin settings)
- Chat UI with real-time messaging, typing indicators, tag autocomplete, mention autocomplete, message editing, reactions
- Board UI with drag-and-drop Kanban, filters, bulk operations, real-time updates
- Notifications UI with real-time updates, improved UX
- User profile & settings page
- Task detail improvements (inline editing, activity timeline)
- Dark mode toggle UI
- Workspace-wide global search
- File uploads (attachments for messages and tasks)
- Chat=SoT hardening for board/sidebar flows:
  - projection-safe task chat creation and post-action sync
  - unified task action execution path with consistent system messages
  - mandatory assignee existence validation in chat write path
  - startup warning for legacy read-model collections (`chat_read_model`, `task_read_model`)
  - HTMX WebSocket close workaround and chat empty-state first-message fix

### 🔄 In Progress (March 2026)

- Mobile-responsive polish
- Remaining frontend edge cases

### 🔜 Coming (March 2026)

- Email notifications
- Performance optimizations

### 📅 Future

- Mobile-friendly PWA
- Slack/Teams integration
- AI-powered features
- Analytics dashboard

---

## 📄 License

MIT License - see [LICENSE](./LICENSE)

---

*Last updated: March 2, 2026*
