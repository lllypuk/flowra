# Service Layer Implementation Plan

Этот каталог содержит задачи по реализации сервисного слоя — фасадов, которые соединяют HTTP handlers с существующими use cases.

## Обзор

**Проблема:** В `container.go:setupHTTPHandlers()` использовались mock-сервисы вместо реальных реализаций. Юзкейсы были полностью готовы в `internal/application/`, но хендлеры не были подключены к ним.

**Решение:** Созданы сервисы-фасады, которые:
1. Реализуют интерфейсы, ожидаемые хендлерами
2. Делегируют работу существующим юзкейсам
3. Обеспечивают единую точку входа для бизнес-логики

## Текущее состояние

### ✅ Все mock-сервисы заменены на реальные реализации

| Компонент | Было | Стало | Статус |
|-----------|------|-------|--------|
| `AccessChecker` | `MockWorkspaceAccessChecker` | `RealWorkspaceAccessChecker` | ✅ |
| `MemberService` | `MockMemberService` | `service.MemberService` | ✅ |
| `WorkspaceService` | `MockWorkspaceService` | `service.WorkspaceService` | ✅ |
| `ChatService` | `MockChatService` | `service.ChatService` | ✅ |
| `AuthService` | `MockAuthService` | `service.AuthService` / NoOp | ✅ |

### Готовые юзкейсы (internal/application/)

| Домен | Юзкейсы | Статус |
|-------|---------|--------|
| `workspace/` | Create, Get, List, Update, Invite, Accept, Revoke | ✅ Используются |
| `chat/` | Create, Get, List, Rename, AddParticipant, Remove, Convert* | ✅ Используются |
| `notification/` | Create | ✅ Готов |

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Handlers                          │
│  (internal/handler/http/)                                   │
├─────────────────────────────────────────────────────────────┤
│  AuthHandler    WorkspaceHandler    ChatHandler             │
│       │               │    │              │                 │
│       ▼               ▼    ▼              ▼                 │
├─────────────────────────────────────────────────────────────┤
│                    Service Layer ✅                         │
│  (internal/service/)                                        │
├─────────────────────────────────────────────────────────────┤
│  AuthService    WorkspaceService  MemberService  ChatService│
│       │               │              │              │       │
│       ▼               ▼              ▼              ▼       │
├─────────────────────────────────────────────────────────────┤
│                   Application Layer                         │
│  (internal/application/)                                    │
├─────────────────────────────────────────────────────────────┤
│  Use Cases: CreateWorkspaceUC, GetChatUC, etc.             │
│       │                                                     │
│       ▼                                                     │
├─────────────────────────────────────────────────────────────┤
│                  Infrastructure Layer                       │
│  (internal/infrastructure/)                                 │
├─────────────────────────────────────────────────────────────┤
│  MongoWorkspaceRepo    MongoChatRepo    MongoEventStore    │
└─────────────────────────────────────────────────────────────┘
```

## Выполненные задачи

### Phase 1: Инфраструктура доступа

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 01** | [01-workspace-access-checker.md](01-workspace-access-checker.md) | ✅ Complete | Real WorkspaceAccessChecker для middleware |

### Phase 2: Core сервисы

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 02** | [02-member-service.md](02-member-service.md) | ✅ Complete | MemberService для управления участниками |
| **Task 03** | [03-workspace-service.md](03-workspace-service.md) | ✅ Complete | WorkspaceService — фасад над workspace юзкейсами |
| **Task 04** | [04-chat-service.md](04-chat-service.md) | ✅ Complete | ChatService — фасад над chat юзкейсами |

### Phase 3: Аутентификация

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 05** | [05-auth-service.md](05-auth-service.md) | ✅ Complete | AuthService с Keycloak интеграцией |

### Phase 4: Интеграция

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 06** | [06-container-wiring.md](06-container-wiring.md) | ✅ Complete | Обновление container.go для real сервисов |

## Файловая структура (результат)

```
internal/service/                    # ✅ Создано
├── workspace_access_checker.go      # Task 01 ✅
├── workspace_access_checker_test.go # Task 01 ✅
├── member_service.go                # Task 02 ✅
├── member_service_test.go           # Task 02 ✅
├── workspace_service.go             # Task 03 ✅
├── workspace_service_test.go        # Task 03 ✅
├── chat_service.go                  # Task 04 ✅
├── chat_service_test.go             # Task 04 ✅
├── auth_service.go                  # Task 05 ✅
├── auth_service_test.go             # Task 05 ✅
├── noop_keycloak_client.go          # Task 06 ✅
└── noop_keycloak_client_test.go     # Task 06 ✅

internal/infrastructure/keycloak/    # ✅ Создано
├── oauth_client.go                  # Task 05 ✅
└── oauth_client_test.go             # Task 05 ✅

internal/infrastructure/auth/        # ✅ Создано
├── token_store.go                   # Task 05 ✅
└── token_store_test.go              # Task 05 ✅

cmd/api/
├── container.go                     # Task 06 ✅ - обновлён для real сервисов
└── container_test.go                # Task 06 ✅

tests/integration/
├── service/
│   └── workspace_access_checker_test.go  # Task 01 ✅
└── container_wiring_test.go         # Task 06 ✅
```

## Принципы реализации

### 1. Consumer-Side Interfaces

Интерфейсы объявлены в handler layer:
- `httphandler.AuthService`
- `httphandler.WorkspaceService`
- `httphandler.MemberService`
- `httphandler.ChatService`
- `middleware.WorkspaceAccessChecker`

Сервисы имплементируют эти интерфейсы с compile-time assertions.

### 2. Делегирование юзкейсам

Сервисы не содержат бизнес-логику — они делегируют работу юзкейсам:

```go
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, ownerID uuid.UUID, name, description string) (*workspace.Workspace, error) {
    result, err := s.createUC.Execute(ctx, workspace.CreateWorkspaceCommand{
        Name:      name,
        CreatedBy: ownerID,
    })
    if err != nil {
        return nil, err
    }
    return s.queryRepo.FindByID(ctx, result.WorkspaceID)
}
```

### 3. Минимальная логика в сервисах

Сервисы содержат:
- Преобразование между форматами (handler DTO → use case command)
- Композицию нескольких юзкейсов
- Обработку ошибок

Сервисы НЕ содержат:
- Бизнес-правила (это в domain)
- Валидацию (это в use cases)
- Прямую работу с БД (это в repositories)

## Критерии приёмки ✅

- [x] Все mock-сервисы заменены на real в `setupHTTPHandlers()`
- [x] NoOp fallback для Keycloak когда не настроен
- [x] Все существующие тесты проходят
- [x] Unit tests для каждого сервиса (100% coverage)
- [x] Integration tests с MongoDB
- [ ] HTMX frontend работает с реальными данными (February 2026)

## Зависимости

### Входящие
- [06-january-2026/05-handlers-auth-workspace.md](../06-january-2026/05-handlers-auth-workspace.md) — определяет интерфейсы хендлеров
- [05-impl-mongodb-repositories/](../05-impl-mongodb-repositories/) — MongoDB репозитории

### Использует
- `internal/application/workspace/` — workspace юзкейсы
- `internal/application/chat/` — chat юзкейсы
- `internal/infrastructure/repository/mongodb/` — MongoDB репозитории

### Исходящие
- [07-frontend/](../07-frontend/) — HTMX frontend зависит от работающих сервисов

## Конфигурация

### APP_MODE

Переменная окружения для контроля режима:
- `APP_MODE=real` (default) — используются реальные сервисы
- `APP_MODE=mock` — используются mock сервисы (для отладки)

### Keycloak

Если Keycloak не настроен (`KEYCLOAK_URL` пустой), используется `NoOpKeycloakClient`.

## Ресурсы

- Handler interfaces: `internal/handler/http/auth_handler.go`, `workspace_handler.go`, `chat_handler.go`
- Use cases: `internal/application/workspace/`, `internal/application/chat/`
- Container: `cmd/api/container.go`
- Service implementations: `internal/service/`

---

*Создано: 2026-01-06*
*Завершено: 2026-01-06*
*Статус: 100% Complete* ✅
