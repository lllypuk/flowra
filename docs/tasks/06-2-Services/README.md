# Service Layer Implementation Plan

Этот каталог содержит задачи по реализации сервисного слоя — фасадов, которые соединяют HTTP handlers с существующими use cases.

## Обзор

**Проблема:** В `container.go:setupHTTPHandlers()` используются mock-сервисы вместо реальных реализаций. Юзкейсы полностью готовы в `internal/application/`, но хендлеры не подключены к ним.

**Решение:** Создать сервисы-фасады, которые:
1. Реализуют интерфейсы, ожидаемые хендлерами
2. Делегируют работу существующим юзкейсам
3. Обеспечивают единую точку входа для бизнес-логики

## Текущее состояние

### Mock-сервисы в использовании (container.go:415-464)

| Компонент | Mock | Нужен Real | Блокирует |
|-----------|------|------------|-----------|
| `AuthService` | `NewMockAuthService()` | Да | Auth flow |
| `UserRepository` | `NewMockUserRepository()` | Да | User lookup |
| `WorkspaceService` | `NewMockWorkspaceService()` | Да | HTMX frontend |
| `MemberService` | `NewMockMemberService()` | Да | HTMX frontend |
| `ChatService` | `NewMockChatService()` | Да | Chat UI |
| `WorkspaceAccessChecker` | `NewMockWorkspaceAccessChecker()` | Да | Authorization |

### Готовые юзкейсы (internal/application/)

| Домен | Юзкейсы | Статус |
|-------|---------|--------|
| `workspace/` | Create, Get, List, Update, Invite, Accept, Revoke | ✅ Готовы |
| `chat/` | Create, Get, List, Rename, AddParticipant, Remove, Convert* | ✅ Готовы |
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
│                    Service Layer (NEW)                      │
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

## Структура задач

### Phase 1: Инфраструктура доступа

| Задача | Файл | Приоритет | Описание |
|--------|------|-----------|----------|
| **Task 01** | [01-workspace-access-checker.md](01-workspace-access-checker.md) | 🔴 Critical | Real WorkspaceAccessChecker для middleware |

### Phase 2: Core сервисы

| Задача | Файл | Приоритет | Описание |
|--------|------|-----------|----------|
| **Task 02** | [02-member-service.md](02-member-service.md) | 🔴 Critical | MemberService для управления участниками |
| **Task 03** | [03-workspace-service.md](03-workspace-service.md) | 🔴 Critical | WorkspaceService — фасад над workspace юзкейсами |
| **Task 04** | [04-chat-service.md](04-chat-service.md) | 🟡 High | ChatService — фасад над chat юзкейсами |

### Phase 3: Аутентификация

| Задача | Файл | Приоритет | Описание |
|--------|------|-----------|----------|
| **Task 05** | [05-auth-service.md](05-auth-service.md) | 🟡 High | AuthService с Keycloak интеграцией |

### Phase 4: Интеграция

| Задача | Файл | Приоритет | Описание |
|--------|------|-----------|----------|
| **Task 06** | [06-container-wiring.md](06-container-wiring.md) | 🔴 Critical | Обновление container.go для real сервисов |

## Порядок выполнения

```
Phase 1: 01 WorkspaceAccessChecker
           ↓
Phase 2: 02 MemberService → 03 WorkspaceService → 04 ChatService
           ↓
Phase 3: 05 AuthService (может выполняться параллельно с Phase 2)
           ↓
Phase 4: 06 Container Wiring
```

**Зависимости:**
- Task 03 зависит от Task 02 (WorkspaceService использует MemberService)
- Task 06 зависит от Tasks 01-05

## Файловая структура (результат)

```
internal/service/                    # НОВАЯ папка
├── workspace_access_checker.go      # Task 01
├── member_service.go                # Task 02
├── workspace_service.go             # Task 03
├── chat_service.go                  # Task 04
├── auth_service.go                  # Task 05
└── service_test.go                  # Unit tests

cmd/api/
└── container.go                     # Task 06 - обновление setupHTTPHandlers()
```

## Принципы реализации

### 1. Consumer-Side Interfaces

Интерфейсы уже объявлены в handler layer:
- `httphandler.AuthService`
- `httphandler.WorkspaceService`
- `httphandler.MemberService`
- `httphandler.ChatService`
- `middleware.WorkspaceAccessChecker`

Сервисы должны имплементировать эти интерфейсы.

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

Сервисы могут содержать:
- Преобразование между форматами (handler DTO → use case command)
- Композицию нескольких юзкейсов
- Обработку ошибок

Сервисы НЕ должны содержать:
- Бизнес-правила (это в domain)
- Валидацию (это в use cases)
- Прямую работу с БД (это в repositories)

## Критерии приёмки (общие)

- [ ] Все mock-сервисы заменены на real в `setupHTTPHandlers()`
- [ ] HTMX frontend работает с реальными данными из MongoDB
- [ ] Все существующие тесты проходят
- [ ] Unit tests для каждого сервиса
- [ ] Integration tests с MongoDB

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

## Ресурсы

- Handler interfaces: `internal/handler/http/auth_handler.go`, `workspace_handler.go`, `chat_handler.go`
- Use cases: `internal/application/workspace/`, `internal/application/chat/`
- Mock implementations: в handler файлах (`NewMock*` функции)
- Container: `cmd/api/container.go`

---

*Создано: 2026-01-06*
*Статус: 0% Complete*
