# 03: HTTP Server Setup

**Приоритет:** 🟡 High  
**Дни:** 4-5 января  
**Статус:** ✅ Выполнено

---

## Описание

Настройка базовой HTTP инфраструктуры с Echo v4 framework. Это подготовительная задача, использующая бонусное время от досрочно выполненных задач (Task Repository и MongoDB Indexes).

---

## Цели

1. Настроить Echo server с базовой конфигурацией
2. Реализовать базовые middleware (logging, recovery, CORS)
3. Создать response helpers для унифицированных ответов

---

## Файлы

```
internal/infrastructure/httpserver/
├── server.go               (~155 LOC)
├── server_test.go          (~310 LOC)
├── response.go             (~124 LOC)
└── response_test.go        (~229 LOC)

internal/middleware/
├── cors.go                 (~82 LOC)
├── cors_test.go            (~282 LOC)
├── logging.go              (~147 LOC)
├── logging_test.go         (~496 LOC)
├── recovery.go             (~112 LOC)
└── recovery_test.go        (~408 LOC)
```

---

## Детали реализации

### 1. Echo Server Setup

Создать базовый Echo server:
- Настройка Echo instance
- Подключение middleware
- Базовая конфигурация (timeouts, body limits)

### 2. Базовые Middleware

**CORS Middleware:**
- Разрешить origins из конфигурации
- Настроить allowed methods и headers

**Logging Middleware:**
- Request/response logging
- Request ID tracking
- Latency измерение

**Recovery Middleware:**
- Catch panics
- Stack trace logging
- Graceful error response

### 3. Response Helpers

Унифицированные функции для ответов:
- `RespondJSON(c, code, data)` — успешный ответ
- `RespondError(c, err)` — ошибка

---

## Критерии приёмки

- [x] Echo server запускается на указанном порту
- [x] CORS middleware настроен и работает
- [x] Logging middleware логирует requests/responses
- [x] Recovery middleware перехватывает panic
- [x] Response helpers создают консистентные ответы
- [x] Unit tests для middleware

---

## Зависимости

**Внешние:**
- `github.com/labstack/echo/v4`

**Внутренние:**
- Нет зависимостей от других задач

---

## Блокирует

- [04-middleware.md](04-middleware.md) — расширенные middleware
- [05-handlers-auth-workspace.md](05-handlers-auth-workspace.md) — HTTP handlers

---

## Заметки

- Эта задача использует бонусное время (Task Repository и MongoDB Indexes выполнены досрочно)
- Полная реализация router и расширенных middleware — в задаче 04
- Цель — минимальный работающий HTTP server