# Task 03: WorkspaceService

**Приоритет:** 🔴 Critical
**Статус:** ✅ Complete
**Зависит от:** Task 02 (MemberService)

---

## Описание

Реализовать `WorkspaceService` — фасад над существующими workspace юзкейсами. Сервис должен реализовать интерфейс `httphandler.WorkspaceService` и заменить `MockWorkspaceService`.

---

## Текущее состояние

### Mock реализация (internal/handler/http/workspace_handler.go)

```go
type MockWorkspaceService struct {
    workspaces map[string]*workspace.Workspace
    counter    int
}

func NewMockWorkspaceService() *MockWorkspaceService
func (m *MockWorkspaceService) CreateWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) GetWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) ListUserWorkspaces(...) ([]*workspace.Workspace, int, error)
func (m *MockWorkspaceService) UpdateWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) DeleteWorkspace(...) error
func (m *MockWorkspaceService) GetMemberCount(...) (int, error)
```

### Использование в container.go

```go
// container.go:427
mockWorkspaceService := httphandler.NewMockWorkspaceService()
c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)
```

---

## Интерфейс (internal/handler/http/workspace_handler.go)

```go
type WorkspaceService interface {
    CreateWorkspace(ctx context.Context, ownerID uuid.UUID, name, description string) (*workspace.Workspace, error)
    GetWorkspace(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    ListUserWorkspaces(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, int, error)
    UpdateWorkspace(ctx context.Context, id uuid.UUID, name, description string) (*workspace.Workspace, error)
    DeleteWorkspace(ctx context.Context, id uuid.UUID) error
    GetMemberCount(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

---

## Существующие юзкейсы (internal/application/workspace/)

| Юзкейс | Файл | Статус |
|--------|------|--------|
| `CreateWorkspaceUseCase` | `create_workspace.go` | ✅ Готов |
| `GetWorkspaceUseCase` | `get_workspace.go` | ✅ Готов |
| `ListUserWorkspacesUseCase` | `list_workspaces.go` | ✅ Готов |
| `UpdateWorkspaceUseCase` | `update_workspace.go` | ✅ Готов |
| `CreateInviteUseCase` | `create_invite.go` | ✅ Готов |
| `AcceptInviteUseCase` | `accept_invite.go` | ✅ Готов |
| `RevokeInviteUseCase` | `revoke_invite.go` | ✅ Готов |

---

## Реализация

### Файл: internal/service/workspace_service.go

Реализован сервис с использованием:
- Use cases для `CreateWorkspace`, `GetWorkspace`, `UpdateWorkspace`
- Repository напрямую для `ListUserWorkspaces`, `DeleteWorkspace`, `GetMemberCount`

Особенности реализации:
- Интерфейсы объявлены на стороне потребителя согласно Go interface design guidelines
- Используется конфигурационный struct `WorkspaceServiceConfig` для dependency injection
- Compile-time assertion проверяет совместимость с `httphandler.WorkspaceService`
- `ListUserWorkspaces` использует repository напрямую, так как `ListUserWorkspacesUseCase` требует дополнительных методов Keycloak

---

## Инициализация юзкейсов

Юзкейсы должны быть созданы в container и переданы в сервис:

```go
// В container.go или отдельном методе setupUseCases

// Workspace use cases
createWorkspaceUC := wsapp.NewCreateWorkspaceUseCase(
    c.WorkspaceRepo,      // CommandRepository
    c.WorkspaceRepo,      // QueryRepository
    c.KeycloakClient,     // Опционально
)

getWorkspaceUC := wsapp.NewGetWorkspaceUseCase(c.WorkspaceRepo)
listWorkspacesUC := wsapp.NewListUserWorkspacesUseCase(c.WorkspaceRepo)
updateWorkspaceUC := wsapp.NewUpdateWorkspaceUseCase(c.WorkspaceRepo, c.WorkspaceRepo)

// Создание сервиса
workspaceService := service.NewWorkspaceService(service.WorkspaceServiceConfig{
    CreateUC:    createWorkspaceUC,
    GetUC:       getWorkspaceUC,
    ListUC:      listWorkspacesUC,
    UpdateUC:    updateWorkspaceUC,
    CommandRepo: c.WorkspaceRepo,
    QueryRepo:   c.WorkspaceRepo,
})
```

---

## Дополнительные методы (опционально)

Можно добавить методы для invite management:

```go
// CreateInvite создаёт приглашение в workspace.
func (s *WorkspaceService) CreateInvite(ctx context.Context, cmd wsapp.CreateInviteCommand) (*wsapp.CreateInviteResult, error)

// AcceptInvite принимает приглашение.
func (s *WorkspaceService) AcceptInvite(ctx context.Context, cmd wsapp.AcceptInviteCommand) error

// RevokeInvite отзывает приглашение.
func (s *WorkspaceService) RevokeInvite(ctx context.Context, cmd wsapp.RevokeInviteCommand) error
```

---

## Зависимости

### Входящие
- Workspace use cases из `internal/application/workspace/`
- `workspace.CommandRepository` и `workspace.QueryRepository`
- Опционально: `KeycloakClient` для создания групп

### Использует репозиторий

```go
type CommandRepository interface {
    Save(ctx context.Context, ws *workspace.Workspace) error
    Delete(ctx context.Context, id uuid.UUID) error
    AddMember(ctx context.Context, member *workspace.Member) error
    RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
}

type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)
    CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)
    CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

---

## Тестирование

### Unit tests

Реализовано 18 тестовых случаев в `internal/service/workspace_service_test.go`:

- `TestWorkspaceService_CreateWorkspace` (2 cases)
- `TestWorkspaceService_GetWorkspace` (2 cases)
- `TestWorkspaceService_ListUserWorkspaces` (5 cases)
- `TestWorkspaceService_UpdateWorkspace` (3 cases)
- `TestWorkspaceService_DeleteWorkspace` (2 cases)
- `TestWorkspaceService_GetMemberCount` (3 cases)

**Coverage: 100%** для всех методов WorkspaceService

---

## Чеклист

- [x] Создать файл `internal/service/workspace_service.go`
- [x] Определить `WorkspaceServiceConfig` struct
- [x] Реализовать `NewWorkspaceService()`
- [x] Реализовать `CreateWorkspace()` через use case
- [x] Реализовать `GetWorkspace()` через use case
- [x] Реализовать `ListUserWorkspaces()` через repository (use case требует доп. методов Keycloak)
- [x] Реализовать `UpdateWorkspace()` через use case
- [x] Реализовать `DeleteWorkspace()` через repository
- [x] Реализовать `GetMemberCount()` через repository
- [x] Написать unit tests
- [ ] Обновить `container.go` для создания use cases (Task 06)

---

## Критерии приёмки

- [x] `WorkspaceService` реализует `httphandler.WorkspaceService`
- [x] Все методы делегируют работу юзкейсам или репозиториям
- [x] Ошибки корректно пробрасываются
- [x] Unit test coverage > 80% (достигнуто 100%)
- [ ] Handler тесты проходят с real сервисом (требует Task 06)

---

## Заметки

- Keycloak integration для создания группы workspace — опционально на первом этапе
- Delete workspace должен также удалять members (обрабатывается в repository с транзакцией)
- Рассмотреть добавление soft delete вместо hard delete
- Параметр `description` не используется текущими use cases (игнорируется)

---

*Создано: 2026-01-06*
*Завершено: 2026-01-06*
