# Task 05: Workspace Domain Use Cases

**Дата:** 2025-10-19
**Статус:** 📝 Pending
**Зависимости:** Task 01 (Architecture), Task 04 (User UseCases)
**Оценка:** 4-5 часов

## Цель

Реализовать Use Cases для Workspace domain. Workspace - это организация/команда, которая группирует пользователей и чаты.

## Контекст

**Workspace entity:**
- ID, Name, KeycloakGroupID
- Invite system (токены с expiration и max uses)
- Интеграция с Keycloak groups
- CRUD модель

## Use Cases для реализации

### Command Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| CreateWorkspaceUseCase | Создание workspace + Keycloak group | Критичный | 1.5 ч |
| UpdateWorkspaceUseCase | Обновление названия | Средний | 0.5 ч |
| CreateInviteUseCase | Создание инвайта с токеном | Критичный | 1 ч |
| AcceptInviteUseCase | Принятие инвайта + добавление в Keycloak | Критичный | 1.5 ч |
| RevokeInviteUseCase | Отзыв инвайта | Средний | 0.5 ч |

### Query Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| GetWorkspaceUseCase | Получение по ID | Критичный | 0.5 ч |
| ListUserWorkspacesUseCase | Список workspace пользователя | Высокий | 1 ч |

## Структура файлов

```
internal/application/workspace/
├── commands.go
├── queries.go
├── results.go
├── errors.go
│
├── create_workspace.go
├── update_workspace.go
├── create_invite.go
├── accept_invite.go
├── revoke_invite.go
│
├── get_workspace.go
├── list_user_workspaces.go
│
└── *_test.go
```

## Commands

```go
package workspace

import (
    "time"

    "github.com/google/uuid"
)

// CreateWorkspaceCommand - создание workspace
type CreateWorkspaceCommand struct {
    Name      string
    CreatedBy uuid.UUID
}

func (c CreateWorkspaceCommand) CommandName() string { return "CreateWorkspace" }

// UpdateWorkspaceCommand - обновление workspace
type UpdateWorkspaceCommand struct {
    WorkspaceID uuid.UUID
    Name        string
    UpdatedBy   uuid.UUID
}

func (c UpdateWorkspaceCommand) CommandName() string { return "UpdateWorkspace" }

// CreateInviteCommand - создание инвайта
type CreateInviteCommand struct {
    WorkspaceID uuid.UUID
    ExpiresAt   *time.Time     // опционально, default: 7 дней
    MaxUses     *int           // опционально, default: unlimited
    CreatedBy   uuid.UUID
}

func (c CreateInviteCommand) CommandName() string { return "CreateInvite" }

// AcceptInviteCommand - принятие инвайта
type AcceptInviteCommand struct {
    Token  string
    UserID uuid.UUID
}

func (c AcceptInviteCommand) CommandName() string { return "AcceptInvite" }

// RevokeInviteCommand - отзыв инвайта
type RevokeInviteCommand struct {
    InviteID  uuid.UUID
    RevokedBy uuid.UUID
}

func (c RevokeInviteCommand) CommandName() string { return "RevokeInvite" }
```

## CreateWorkspaceUseCase (пример)

```go
package workspace

import (
    "context"
    "fmt"

    "github.com/lllypuk/teams-up/internal/application/shared"
    "github.com/lllypuk/teams-up/internal/domain/workspace"
    "github.com/lllypuk/teams-up/internal/infrastructure/keycloak"
)

type CreateWorkspaceUseCase struct {
    workspaceRepo workspace.Repository
    keycloakClient *keycloak.Client
}

func NewCreateWorkspaceUseCase(
    workspaceRepo workspace.Repository,
    keycloakClient *keycloak.Client,
) *CreateWorkspaceUseCase {
    return &CreateWorkspaceUseCase{
        workspaceRepo:  workspaceRepo,
        keycloakClient: keycloakClient,
    }
}

func (uc *CreateWorkspaceUseCase) Execute(
    ctx context.Context,
    cmd CreateWorkspaceCommand,
) (WorkspaceResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return WorkspaceResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Создание группы в Keycloak
    keycloakGroupID, err := uc.keycloakClient.CreateGroup(ctx, cmd.Name)
    if err != nil {
        return WorkspaceResult{}, fmt.Errorf("failed to create Keycloak group: %w", err)
    }

    // Создание workspace
    ws := workspace.NewWorkspace(cmd.Name, keycloakGroupID)

    // Сохранение
    if err := uc.workspaceRepo.Save(ctx, ws); err != nil {
        // Rollback Keycloak group
        _ = uc.keycloakClient.DeleteGroup(ctx, keycloakGroupID)
        return WorkspaceResult{}, fmt.Errorf("failed to save workspace: %w", err)
    }

    // Добавление создателя в группу Keycloak
    if err := uc.keycloakClient.AddUserToGroup(ctx, cmd.CreatedBy.String(), keycloakGroupID); err != nil {
        // Не критично, можно залогировать
    }

    return WorkspaceResult{
        Result: shared.Result[*workspace.Workspace]{
            Value: ws,
        },
    }, nil
}

func (uc *CreateWorkspaceUseCase) validate(cmd CreateWorkspaceCommand) error {
    if err := shared.ValidateRequired("name", cmd.Name); err != nil {
        return err
    }
    if err := shared.ValidateMaxLength("name", cmd.Name, 100); err != nil {
        return err
    }
    if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
        return err
    }
    return nil
}
```

## AcceptInviteUseCase (сложный пример)

```go
package workspace

import (
    "context"
    "fmt"

    "github.com/lllypuk/teams-up/internal/application/shared"
    "github.com/lllypuk/teams-up/internal/domain/workspace"
    "github.com/lllypuk/teams-up/internal/infrastructure/keycloak"
)

type AcceptInviteUseCase struct {
    workspaceRepo workspace.Repository
    keycloakClient *keycloak.Client
}

func NewAcceptInviteUseCase(
    workspaceRepo workspace.Repository,
    keycloakClient *keycloak.Client,
) *AcceptInviteUseCase {
    return &AcceptInviteUseCase{
        workspaceRepo:  workspaceRepo,
        keycloakClient: keycloakClient,
    }
}

func (uc *AcceptInviteUseCase) Execute(
    ctx context.Context,
    cmd AcceptInviteCommand,
) (WorkspaceResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return WorkspaceResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Поиск workspace по инвайту
    ws, err := uc.workspaceRepo.FindByInviteToken(ctx, cmd.Token)
    if err != nil {
        return WorkspaceResult{}, ErrInviteNotFound
    }

    // Валидация инвайта
    invite := ws.GetInviteByToken(cmd.Token)
    if invite == nil {
        return WorkspaceResult{}, ErrInviteNotFound
    }

    if err := invite.Validate(); err != nil {
        return WorkspaceResult{}, err
    }

    // Инкремент использований
    invite.IncrementUses()

    // Сохранение workspace
    if err := uc.workspaceRepo.Save(ctx, ws); err != nil {
        return WorkspaceResult{}, fmt.Errorf("failed to save workspace: %w", err)
    }

    // Добавление пользователя в Keycloak группу
    if err := uc.keycloakClient.AddUserToGroup(
        ctx,
        cmd.UserID.String(),
        ws.KeycloakGroupID(),
    ); err != nil {
        return WorkspaceResult{}, fmt.Errorf("failed to add user to Keycloak group: %w", err)
    }

    return WorkspaceResult{
        Result: shared.Result[*workspace.Workspace]{
            Value: ws,
        },
    }, nil
}

func (uc *AcceptInviteUseCase) validate(cmd AcceptInviteCommand) error {
    if err := shared.ValidateRequired("token", cmd.Token); err != nil {
        return err
    }
    if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
        return err
    }
    return nil
}
```

## Keycloak Integration

```go
// internal/infrastructure/keycloak/client.go
package keycloak

import "context"

type Client struct {
    // Keycloak admin client configuration
}

func (c *Client) CreateGroup(ctx context.Context, name string) (string, error) {
    // Создание группы в Keycloak
    // Возвращает groupID
}

func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
    // Удаление группы
}

func (c *Client) AddUserToGroup(ctx context.Context, userID, groupID string) error {
    // Добавление пользователя в группу
}

func (c *Client) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
    // Удаление пользователя из группы
}
```

## Tests

```go
func TestCreateWorkspaceUseCase_Success(t *testing.T) {
    workspaceRepo := mocks.NewWorkspaceRepository()
    keycloakClient := mocks.NewKeycloakClient()

    keycloakClient.On("CreateGroup", mock.Anything, "Test Workspace").
        Return("keycloak-group-id", nil)

    useCase := NewCreateWorkspaceUseCase(workspaceRepo, keycloakClient)

    cmd := CreateWorkspaceCommand{
        Name:      "Test Workspace",
        CreatedBy: uuid.New(),
    }

    result, err := useCase.Execute(context.Background(), cmd)

    assert.NoError(t, err)
    assert.NotNil(t, result.Value)
    assert.Equal(t, cmd.Name, result.Value.Name())

    // Verify Keycloak group was created
    keycloakClient.AssertCalled(t, "CreateGroup", mock.Anything, "Test Workspace")

    // Verify workspace was saved
    assert.Equal(t, 1, workspaceRepo.SaveCallCount())
}

func TestAcceptInviteUseCase_InviteExpired(t *testing.T) {
    workspaceRepo := mocks.NewWorkspaceRepository()
    keycloakClient := mocks.NewKeycloakClient()

    // Setup expired invite
    ws := workspace.NewWorkspace("Test", "keycloak-id")
    expiredTime := time.Now().Add(-24 * time.Hour)
    ws.CreateInvite(&expiredTime, nil)

    workspaceRepo.AddWorkspace(ws)

    useCase := NewAcceptInviteUseCase(workspaceRepo, keycloakClient)

    cmd := AcceptInviteCommand{
        Token:  ws.Invites()[0].Token(),
        UserID: uuid.New(),
    }

    result, err := useCase.Execute(context.Background(), cmd)

    assert.Error(t, err)
    assert.ErrorIs(t, err, workspace.ErrInviteExpired)
}
```

## Checklist

- [ ] Создать `commands.go`, `queries.go`, `results.go`, `errors.go`
- [ ] CreateWorkspaceUseCase + tests
- [ ] UpdateWorkspaceUseCase + tests
- [ ] CreateInviteUseCase + tests
- [ ] AcceptInviteUseCase + tests
- [ ] RevokeInviteUseCase + tests
- [ ] GetWorkspaceUseCase + tests
- [ ] ListUserWorkspacesUseCase + tests
- [ ] Keycloak client implementation
- [ ] Integration tests (workspace lifecycle)

## Следующие шаги

- **Task 06**: Notification UseCases
- Keycloak integration testing
