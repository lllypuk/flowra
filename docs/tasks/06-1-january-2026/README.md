# Задачи на январь 2026

**Период:** 1-31 января 2026  
**Цель:** Завершить Infrastructure Layer и запустить первое работающее API  
**Статус:** 🟢 В работе

---

## Обзор

Этот каталог содержит детализированные задачи на январь 2026 года, сформированные на основе [JANUARY_2026_PLAN.md](../../JANUARY_2026_PLAN.md).

### Предварительно выполнено (до января)
- ✅ Task Repository с Event Sourcing
- ✅ MongoDB Indexes для всех коллекций

### Приоритеты на январь
1. 🔴 **КРИТИЧЕСКИЙ** — Event Bus, Entry Points
2. 🟡 **ВЫСОКИЙ** — HTTP Handlers, WebSocket
3. 🟢 **СРЕДНИЙ** — Documentation, E2E Tests

---

## Структура задач

### Неделя 1: Infrastructure Completion (1-7 января)

| № | Задача | Файл | Приоритет | Дни | Статус |
|---|--------|------|-----------|-----|--------|
| 01 | Event Bus (Redis Pub/Sub) | [01-event-bus.md](01-event-bus.md) | 🔴 Critical | 1-3 | ⏳ |
| 02 | Event Handlers | [02-event-handlers.md](02-event-handlers.md) | 🔴 Critical | 1-3 | ⏳ |
| 03 | HTTP Server Setup | [03-http-server.md](03-http-server.md) | 🟡 High | 4-5 | ⏳ |

### Неделя 2: HTTP Infrastructure (8-14 января)

| № | Задача | Файл | Приоритет | Дни | Статус |
|---|--------|------|-----------|-----|--------|
| 04 | Echo Router & Middleware | [04-middleware.md](04-middleware.md) | 🔴 Critical | 8-10 | ⏳ |
| 05 | Auth & Workspace Handlers | [05-handlers-auth-workspace.md](05-handlers-auth-workspace.md) | 🔴 Critical | 11-12 | ⏳ |
| 06 | Chat & Message Handlers | [06-handlers-chat-message.md](06-handlers-chat-message.md) | 🔴 Critical | 13-14 | ⏳ |

### Неделя 3: More Handlers & WebSocket (15-21 января)

| № | Задача | Файл | Приоритет | Дни | Статус |
|---|--------|------|-----------|-----|--------|
| 07 | Task & Notification Handlers | [07-handlers-task-notification.md](07-handlers-task-notification.md) | 🟡 High | 15-17 | ⏳ |
| 08 | WebSocket Server | [08-websocket.md](08-websocket.md) | 🟡 High | 18-21 | ⏳ |

### Неделя 4: Entry Points & Integration (22-31 января)

| № | Задача | Файл | Приоритет | Дни | Статус |
|---|--------|------|-----------|-----|--------|
| 09 | Entry Points (cmd/api) | [09-entry-points.md](09-entry-points.md) | 🔴 Critical | 22-24 | ⏳ |
| 10 | E2E Tests | [10-e2e-tests.md](10-e2e-tests.md) | 🟡 High | 25-27 | ⏳ |
| 11 | Documentation & Demo | [11-documentation.md](11-documentation.md) | 🟢 Medium | 28-31 | ⏳ |

---

## Метрики успеха

### К концу января
- [ ] Event Bus публикует события асинхронно
- [ ] 40+ HTTP endpoints функциональны
- [ ] WebSocket real-time updates работают
- [ ] `./api` запускает приложение
- [ ] E2E tests проходят
- [ ] API documentation готова

### Coverage targets
- Infrastructure: 85%+
- Interface (handlers): 70%+

---

## Зависимости между задачами

```
[01 Event Bus] ──┬──> [02 Event Handlers]
                 │
                 └──> [08 WebSocket] ──> [10 E2E Tests]
                           ↑
[03 HTTP Server] ──> [04 Middleware] ──> [05-07 Handlers] ──┘
                                                │
                                                v
                                    [09 Entry Points] ──> [11 Documentation]
```

---

## Легенда статусов

- ⏳ — Не начато
- 🔄 — В процессе
- ✅ — Завершено
- ❌ — Заблокировано
- ⏸️ — Приостановлено

---

*Создано: 2026-01-01*  
*Обновлено: 2026-01-01*
