# Flowra

Комплексная система чата с интегрированным таск-трекером, help desk функциональностью и поддержкой команд.

## 📊 Текущее состояние проекта

**Версия:** 1.0.0-beta  
**Дата обновления:** 2026-01-31  
**Общий прогресс:** ~95% к MVP  
**Статус:** January 2026 Release Candidate

### Прогресс по слоям

| Слой | Статус | Прогресс | Файлов | Coverage |
|------|--------|----------|--------|----------|
| **Domain** | ✅ Complete | 100% | 48+ | 90%+ |
| **Application** | ✅ Complete | 100% | 139+ | 85%+ |
| **Infrastructure** | ✅ Complete | 95% | 45+ | 85%+ |
| **Interface** | ✅ Complete | 100% | 16+ | 80%+ |
| **Entry Points** | ✅ Complete | 100% | 6 | 75%+ |

### Что работает ✅

- ✅ **Domain Layer:** 6 Event-Sourced агрегатов, 30+ domain events
- ✅ **Application Layer:** 40+ use cases с 85% average coverage
- ✅ **MongoDB Repositories:** Все 6 репозиториев с интеграционными тестами
- ✅ **Event Store:** MongoDB Event Store с optimistic locking
- ✅ **HTTP Handlers:** Полный REST API с 40+ endpoints
- ✅ **WebSocket:** Real-time коммуникация с Hub pattern
- ✅ **Middleware:** Auth, CORS, Logging, Recovery, Workspace Access
- ✅ **Entry Points:** API server, Worker, Migrator
- ✅ **E2E Tests:** Полное покрытие критических flows
- ✅ **API Documentation:** OpenAPI 3.1, Postman collection

---

## 🚀 Quick Start (5 минут)

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Make

### 1. Clone & Setup

```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
make deps
```

### 2. Start Infrastructure

```bash
# Start MongoDB, Redis, Keycloak
docker-compose up -d

# Verify services
docker-compose ps
```

### 3. Run Migrations

```bash
make migrate-up
```

### 4. Start the Application

```bash
make dev
```

### 5. Verify

```bash
# Health check
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# API documentation
open http://localhost:8080/docs
```

### 6. Test Authentication

Access Keycloak at http://localhost:8090 (admin/admin123) to configure OAuth.

---

## 🔧 Development Commands

```bash
make help          # Show all available commands
make dev           # Run in development mode
make build         # Build binaries
make test          # Run all tests
make test-unit     # Run unit tests only
make test-e2e      # Run E2E tests
make lint          # Run linter and format code
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make test-coverage # Generate coverage report
```

---

## 🏗️ Основные возможности

- **Real-time чат** с поддержкой групп и direct messages
- **Система команд** для управления задачами прямо из чата
- **Task management** с state machine для статусов
- **Help Desk** функциональность с SLA tracking
- **Keycloak интеграция** для SSO и управления пользователями
- **HTMX + Alpine.js** для минимального использования JavaScript
- **WebSocket** для real-time обновлений
- **Event Sourcing** для полной истории изменений
- **Tag Processing** - система обработки команд через теги в сообщениях

---

## 🎯 Доменные модели

### Chat Aggregate
- **Типы**: Direct message, Group chat, Help Desk ticket
- **Операции**: Create, AddParticipant, RemoveParticipant, Rename, SetSeverity, SetPriority

### Message Aggregate
- **Возможности**: Content, attachments, reactions, threading
- **Операции**: Create, Edit, Delete, AddAttachment, AddReaction

### Task Aggregate
- **Типы**: Task, Bug, Feature, Support
- **States**: Todo, InProgress, Review, Done, Cancelled
- **Priority**: Low, Medium, High, Critical

### Notification Aggregate
- **Типы**: Task, Chat, Mention, System
- **Операции**: Create, MarkAsRead, MarkAllAsRead, Delete

### User & Workspace Entities
- **User**: Registration, Profile updates, Admin promotion
- **Workspace**: Create, Update, Member management

---

## 📋 Документация

### API Documentation
- [API Overview](./docs/api/README.md) - Полное описание API
- [OpenAPI Spec](./docs/api/openapi.yaml) - OpenAPI 3.1 спецификация
- [Postman Collection](./docs/api/postman_collection.json) - Готовая коллекция для тестирования

### Guides
- [Deployment Guide](./docs/DEPLOYMENT.md) - Инструкции по развёртыванию
- [Development Guide](./docs/DEVELOPMENT.md) - Настройка окружения разработчика
- [Architecture](./docs/ARCHITECTURE.md) - Обзор архитектуры системы

### Architecture & Design
- [Architecture Overview](./docs/01-architecture.md) - Детальная архитектура
- [Domain Model](./docs/02-domain-model.md) - Доменная модель
- [Security Model](./docs/04-security-model.md) - Модель безопасности
- [Event Flow](./docs/05-event-flow.md) - Потоки событий
- [API Contracts](./docs/06-api-contracts.md) - Контракты API

---

## 🛠 Технологический стек

### Backend
| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Primary language | 1.25+ |
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

## 📁 Структура проекта

```
.
├── cmd/                    # Application entry points
│   ├── api/               # HTTP API server
│   ├── worker/            # Background worker
│   └── migrator/          # Database migrations
│
├── internal/              # Private application code
│   ├── application/       # Application services (use cases)
│   ├── domain/           # Domain models and business logic
│   ├── handler/          # HTTP and WebSocket handlers
│   ├── infrastructure/   # External dependencies
│   ├── middleware/       # HTTP middleware
│   └── config/           # Configuration
│
├── tests/                # Test suites
│   ├── e2e/             # End-to-end tests
│   ├── integration/     # Integration tests
│   └── testutil/        # Test utilities
│
├── docs/                # Documentation
│   ├── api/            # API documentation
│   └── tasks/          # Task tracking
│
├── configs/             # Configuration files
└── docker-compose.yml   # Local development services
```

---

## 🔐 Безопасность

- **Authentication**: Keycloak SSO с JWT tokens
- **Authorization**: Role-based access control (RBAC)
- **Workspace Access**: Middleware проверки доступа
- **Input Validation**: Валидация на всех уровнях
- **Secure Defaults**: Безопасная конфигурация по умолчанию

---

## 🧪 Тестирование

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run E2E tests
make test-e2e

# Check coverage threshold (80%)
make test-coverage-check
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

### ✅ Completed (January 2026)

- Full domain layer with event sourcing
- Complete application layer with use cases
- MongoDB repositories with integration tests
- HTTP handlers for all endpoints
- WebSocket real-time communication
- Authentication & authorization middleware
- E2E test coverage
- API documentation (OpenAPI, Postman)
- Deployment documentation

### 🔜 Coming (February 2026)

- HTMX frontend templates
- Email notifications
- File attachments (S3)
- Search functionality
- Performance optimizations

### 📅 Future

- Mobile-friendly PWA
- Slack/Teams integration
- AI-powered features
- Analytics dashboard

---

## 📄 Лицензия

MIT License - см. [LICENSE](./LICENSE)

---

*Last updated: January 2026*
