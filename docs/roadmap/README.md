# Development Roadmap - Detailed Implementation Guide

Этот каталог содержит детальные руководства по реализации для каждой фазы разработки проекта **new-teams-up**.

**Основной роадмап:** [DEVELOPMENT_ROADMAP_2025.md](../DEVELOPMENT_ROADMAP_2025.md)

---

## 📁 Структура

Задачи организованы по фазам разработки:

```
docs/roadmap/
├── README.md                    (этот файл)
├── phase-0/                     (Критические исправления, 0-2 недели)
├── phase-1/                     (Infrastructure Layer, 3-6 недель)
├── phase-2/                     (Interface Layer, 7-10 недель)
├── phase-3/                     (Entry Points & DI, 11-12 недель)
└── phase-4/                     (Minimal Frontend, 13-16 недель)
```

---

## 🎯 Phase 0: Критические исправления (0-2 недели)

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Цель:** Устранить блокеры, завершить Application Layer

### Задачи:

1. **[task-0.1-chat-usecases-testing.md](phase-0/task-0.1-chat-usecases-testing.md)**
   - 🔴 БЛОКЕР
   - Создать comprehensive test suite для Chat UseCases
   - Покрытие: 0% → 85%+
   - Время: 3-4 часа

2. **[task-0.2-chat-query-usecases.md](phase-0/task-0.2-chat-query-usecases.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - Реализовать GetChat, ListChats, ListParticipants
   - 3 query use cases + 15 unit tests
   - Время: 1.5-2 часа

3. **[task-0.3-documentation-sync.md](phase-0/task-0.3-documentation-sync.md)**
   - 🟡 MEDIUM
   - Синхронизировать README, архитектурную документацию
   - Создать API_USAGE.md с примерами
   - Время: 1 час

**После Phase 0:**
- ✅ Application Layer: 100% реализован
- ✅ Test Coverage: 75%+
- ✅ Нет критических блокеров

---

## 🏗️ Phase 1: Infrastructure Layer (Недели 3-6)

**Приоритет:** 🟡 HIGH
**Цель:** Реализовать persistence, event bus, Keycloak integration

### Задачи:

1. **[task-1.1.1-mongodb-event-store.md](phase-1/task-1.1.1-mongodb-event-store.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - Production-ready MongoDB Event Store
   - Optimistic concurrency control, event sourcing
   - Время: 3-4 дня

2. **[task-1.1.2-mongodb-repositories.md](phase-1/task-1.1.2-mongodb-repositories.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - 5 repositories: Chat, Message, User, Workspace, Notification
   - Event sourcing для Chat, read models
   - Время: 5-6 дней

3. **[task-1.1.3-redis-repositories.md](phase-1/task-1.1.3-redis-repositories.md)**
   - 🟡 MEDIUM
   - Session, Idempotency, Cache repositories
   - Время: 2 дня

4. **[task-1.2.1-redis-event-bus.md](phase-1/task-1.2.1-redis-event-bus.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - Redis Pub/Sub Event Bus
   - Notification handler, Dead Letter Queue
   - Время: 4-5 дней

5. **[task-1.3.1-keycloak-integration.md](phase-1/task-1.3.1-keycloak-integration.md)**
   - 🟡 HIGH
   - OAuth2/OIDC flow, JWT validation
   - Group management для Workspaces
   - Время: 4-5 дней

**После Phase 1:**
- ✅ Persistence работает (MongoDB + Redis)
- ✅ Event Bus доставляет события
- ✅ Keycloak OAuth2 интегрирован

---

## 🌐 Phase 2: Interface Layer (Недели 7-10)

**Приоритет:** 🟡 HIGH
**Цель:** HTTP API, WebSocket, Middleware

### Задачи:

1. **[task-2.1-http-infrastructure.md](phase-2/task-2.1-http-infrastructure.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - Echo v4 router setup
   - 6 middleware: Auth, Workspace, Rate Limiting, Logging, CORS
   - Время: 4-5 дней

2. **[task-2.2-http-handlers.md](phase-2/task-2.2-http-handlers.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - 7 handlers, 40+ endpoints
   - Auth, Workspace, Chat, Message, Notification
   - Время: 8-10 дней

3. **[task-2.3-websocket-server.md](phase-2/task-2.3-websocket-server.md)**
   - 🟡 MEDIUM
   - WebSocket для real-time updates
   - Hub, Client, Event broadcaster
   - Время: 5-6 дней

**После Phase 2:**
- ✅ REST API полностью функционален
- ✅ WebSocket real-time обновления
- ✅ Middleware защищают endpoints

---

## 🚀 Phase 3: Entry Points & Dependency Injection (Недели 11-12)

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Цель:** Собрать приложение воедино, запустить

### Задачи:

1. **[task-3.1-entry-points.md](phase-3/task-3.1-entry-points.md)**
   - 🔴 КРИТИЧЕСКИЙ
   - API Server (cmd/api/main.go)
   - Worker Service (cmd/worker/main.go)
   - Database Migrator (cmd/migrator/main.go)
   - Configuration management
   - Время: 4-5 дней

**После Phase 3:**
- ✅ Приложение запускается: `./api`
- ✅ Worker обрабатывает events
- ✅ Graceful shutdown работает

---

## 🎨 Phase 4: Minimal Frontend (Недели 13-16)

**Приоритет:** 🟡 MEDIUM
**Цель:** HTMX + Pico CSS для базового UI

### Задачи:

1. **[task-4-minimal-frontend.md](phase-4/task-4-minimal-frontend.md)**
   - 🟡 MEDIUM
   - HTMX 2.0 + Pico CSS v2
   - Base templates, components
   - Core pages: Auth, Workspace, Chat, Kanban, Notifications
   - CSS customization, JS utilities
   - Время: 2-3 недели

**После Phase 4:**
- ✅ Работающий UI для всех основных функций
- ✅ Real-time WebSocket updates
- ✅ MVP готов к production! 🎉

---

## 🗺️ Навигация

### Быстрый старт

1. **Я новый разработчик** → Начни с [Phase 0, Task 0.3](phase-0/task-0.3-documentation-sync.md) (Documentation)
2. **Хочу начать кодить** → [Phase 0, Task 0.1](phase-0/task-0.1-chat-usecases-testing.md) (Chat Testing)
3. **Интересует архитектура** → [Phase 1, Task 1.1.1](phase-1/task-1.1.1-mongodb-event-store.md) (Event Store)
4. **Хочу запустить приложение** → [Phase 3, Task 3.1](phase-3/task-3.1-entry-points.md) (Entry Points)

### По технологиям

- **MongoDB** → Phase 1: [Event Store](phase-1/task-1.1.1-mongodb-event-store.md), [Repositories](phase-1/task-1.1.2-mongodb-repositories.md)
- **Redis** → Phase 1: [Redis Repos](phase-1/task-1.1.3-redis-repositories.md), [Event Bus](phase-1/task-1.2.1-redis-event-bus.md)
- **Keycloak** → Phase 1: [Keycloak Integration](phase-1/task-1.3.1-keycloak-integration.md)
- **Echo (HTTP)** → Phase 2: [HTTP Infrastructure](phase-2/task-2.1-http-infrastructure.md), [Handlers](phase-2/task-2.2-http-handlers.md)
- **WebSocket** → Phase 2: [WebSocket Server](phase-2/task-2.3-websocket-server.md)
- **HTMX Frontend** → Phase 4: [Minimal Frontend](phase-4/task-4-minimal-frontend.md)

---

## 📊 Прогресс трекинг

Каждый файл задачи содержит:

- **Критерии успеха** (✅ checklist)
- **Оценка времени**
- **Зависимости**
- **Детальный план реализации**
- **Примеры кода**
- **Тестирование**

Используй чеклисты в каждом файле для отслеживания прогресса!

---

## 🔗 Полезные ссылки

- **Основной роадмап:** [DEVELOPMENT_ROADMAP_2025.md](../DEVELOPMENT_ROADMAP_2025.md)
- **Архитектура:** [docs/01-architecture.md](../01-architecture.md)
- **README:** [README.md](../../README.md)

---

## 📝 Формат файлов

Каждый файл задачи следует единой структуре:

1. **Заголовок** (приоритет, статус, время, зависимости)
2. **Проблема** (что не работает)
3. **Цель** (что нужно достичь)
4. **Файлы для создания**
5. **Детальный план реализации** (пошаговый)
6. **Примеры кода** (с комментариями)
7. **Тестирование** (примеры тестов)
8. **Критерии успеха** (чеклист)
9. **Следующий шаг**

---

**Удачи в разработке! 🚀**

*Last updated: 2025-11-11*
