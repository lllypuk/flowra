# 11: Documentation & Demo

**Приоритет:** 🟢 Medium  
**Статус:** ⏳ Не начато  
**Дни:** 28-31 января  
**Зависит от:** [09-entry-points.md](09-entry-points.md), [10-e2e-tests.md](10-e2e-tests.md)

---

## Описание

Финальная задача января: документирование API, создание гайдов для разработчиков, подготовка демо для stakeholders. Включает bug fixing найденных проблем из E2E тестов.

---

## Цели

1. Создать полную API документацию (OpenAPI/Swagger)
2. Написать deployment guide
3. Обновить developer documentation
4. Подготовить демонстрацию для stakeholders
5. Исправить баги из E2E тестов

---

## Deliverables

### 1. API Documentation

**Файлы:**
```
docs/api/
├── openapi.yaml            (~2000 lines)
└── README.md               (~200 lines)
```

**Содержание OpenAPI spec:**
- Все 40+ endpoints
- Request/response schemas
- Authentication описание
- Error responses
- Examples для каждого endpoint

### 2. Deployment Guide

**Файл:** `docs/DEPLOYMENT.md`

**Разделы:**
- Prerequisites (Go, Docker, etc.)
- Configuration overview
- Environment variables
- Docker Compose setup
- Manual deployment steps
- Health checks
- Troubleshooting

### 3. Developer Guide

**Обновить файлы:**
- `README.md` — Quick start
- `docs/DEVELOPMENT.md` — Local development setup
- `docs/ARCHITECTURE.md` — System overview
- `CLAUDE.md` — AI assistant context

### 4. Postman Collection

**Файл:** `docs/api/postman_collection.json`

**Содержание:**
- Все endpoints сгруппированы по ресурсам
- Environment variables
- Pre-request scripts для auth
- Tests для каждого request

### 5. Demo Materials

**Файлы:**
```
docs/demo/
├── DEMO_SCRIPT.md          (сценарий демо)
├── screenshots/            (скриншоты key flows)
└── demo-video.md           (инструкции для записи)
```

---

## Bug Fixing

### Процесс

1. Собрать список проблем из E2E тестов
2. Приоритизировать по severity
3. Исправить Critical и High
4. Документировать Known Issues

### Категории багов

| Severity | Описание | Действие |
|----------|----------|----------|
| Critical | Блокирует основной flow | Исправить немедленно |
| High | Значительное влияние | Исправить в рамках задачи |
| Medium | Неудобство для пользователя | Задокументировать, создать issue |
| Low | Косметические проблемы | Создать issue на будущее |

---

## Чеклист

### API Documentation
- [ ] OpenAPI spec создан
- [ ] Все endpoints документированы
- [ ] Request/response schemas описаны
- [ ] Authentication документирована
- [ ] Error codes описаны
- [ ] Examples для всех endpoints

### Deployment Guide
- [ ] Prerequisites описаны
- [ ] Docker Compose инструкции
- [ ] Environment variables table
- [ ] Health check endpoints
- [ ] Troubleshooting section

### Developer Guide
- [ ] README.md обновлён
- [ ] Quick start работает
- [ ] Architecture diagram добавлен
- [ ] Code structure описана
- [ ] Testing guide добавлен

### Postman Collection
- [ ] Collection создан
- [ ] Все endpoints добавлены
- [ ] Environment настроен
- [ ] Auth flow работает
- [ ] Examples проверены

### Demo
- [ ] Demo script написан
- [ ] Screenshots сделаны
- [ ] Video инструкции готовы
- [ ] Demo проведено для stakeholders

### Bug Fixing
- [ ] Critical bugs исправлены
- [ ] High priority bugs исправлены
- [ ] Known issues документированы
- [ ] Regression tests добавлены

---

## Критерии приёмки

- [ ] OpenAPI spec валиден (проходит lint)
- [ ] Swagger UI работает с нашим spec
- [ ] Postman collection импортируется и работает
- [ ] `docker-compose up` запускает приложение
- [ ] README quick start выполним за 5 минут
- [ ] Demo script покрывает основные сценарии
- [ ] Все Critical и High баги исправлены
- [ ] Known issues документированы

---

## OpenAPI Spec Structure

```yaml
openapi: 3.1.0
info:
  title: Teams Up API
  version: 1.0.0
  description: Chat System with Task Management

servers:
  - url: http://localhost:8080/api/v1
    description: Local development

security:
  - bearerAuth: []

tags:
  - name: Auth
  - name: Workspaces
  - name: Chats
  - name: Messages
  - name: Tasks
  - name: Notifications
  - name: Users

paths:
  /auth/login:
    post: ...
  /workspaces:
    get: ...
    post: ...
  # ... все endpoints

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  
  schemas:
    Workspace: ...
    Chat: ...
    Message: ...
    Task: ...
    # ... все модели

  responses:
    UnauthorizedError: ...
    NotFoundError: ...
    ValidationError: ...
```

---

## Demo Script Outline

### 1. Introduction (2 min)
- Обзор системы
- Ключевые возможности

### 2. User Flow (5 min)
- Login через Keycloak
- Создание workspace
- Приглашение участников

### 3. Chat Flow (5 min)
- Создание группового чата
- Отправка сообщений
- Real-time delivery через WebSocket

### 4. Task Management (5 min)
- Создание задачи из чата
- Назначение исполнителя
- Изменение статуса
- Уведомления

### 5. Q&A (3 min)

---

## Зависимости

### Входящие
- [09-entry-points.md](09-entry-points.md) — работающее приложение
- [10-e2e-tests.md](10-e2e-tests.md) — список багов для исправления

### Исходящие
- Финальная задача января
- Подготовка к февральскому этапу (Frontend)

---

## Заметки

- OpenAPI spec можно генерировать автоматически из annotations (swaggo/swag)
- Postman collection можно экспортировать из OpenAPI spec
- Demo лучше записать на видео для асинхронного просмотра
- Known issues должны быть в GitHub Issues с label `known-issue`
- После демо собрать feedback и создать issues на февраль

---

## Ресурсы

- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0)
- [Swagger Editor](https://editor.swagger.io/)
- [swaggo/swag](https://github.com/swaggo/swag) — Go annotations to OpenAPI
- [Postman Learning Center](https://learning.postman.com/)

---

*Создано: 2026-01-01*