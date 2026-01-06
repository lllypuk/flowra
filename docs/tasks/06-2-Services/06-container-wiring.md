# Task 06: Container Wiring

**Приоритет:** 🔴 Critical
**Статус:** Pending
**Зависит от:** Tasks 01-05 (все сервисы)

---

## Описание

Обновить `cmd/api/container.go` для использования реальных сервисов вместо mock-реализаций. Это финальная задача, которая интегрирует все созданные сервисы.

---

## Текущее состояние (container.go:415-464)

```go
func (c *Container) setupHTTPHandlers() {
    c.Logger.Debug("setting up HTTP handlers with REAL implementations")

    // TODO: Wire real AuthService implementation when available
    c.Logger.Warn("AuthHandler: using mock implementation")
    mockAuthService := httphandler.NewMockAuthService()
    mockUserRepo := httphandler.NewMockUserRepository()
    c.AuthHandler = httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

    // TODO: Wire real WorkspaceService implementation when available
    c.Logger.Warn("WorkspaceHandler: using mock implementation")
    mockWorkspaceService := httphandler.NewMockWorkspaceService()
    mockMemberService := httphandler.NewMockMemberService()
    c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)

    // Inject services into template handler
    if c.TemplateHandler != nil {
        c.TemplateHandler.SetServices(mockWorkspaceService, mockMemberService)
    }

    // TODO: Wire real ChatService implementation when available
    c.Logger.Warn("ChatHandler: using mock implementation")
    mockChatService := httphandler.NewMockChatService()
    c.ChatHandler = httphandler.NewChatHandler(mockChatService)

    // WebSocket handler uses real Hub
    c.WSHandler = wshandler.NewHandler(...)

    // Setup token validator for auth middleware
    c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

    // TODO: Wire real WorkspaceAccessChecker implementation
    c.Logger.Warn("AccessChecker: using mock implementation")
    c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()
}
```

---

## Целевое состояние

```go
func (c *Container) setupHTTPHandlers() {
    c.Logger.Debug("setting up HTTP handlers with REAL implementations")

    // === 1. Access Checker (Task 01) ===
    c.AccessChecker = service.NewRealWorkspaceAccessChecker(c.WorkspaceRepo)

    // === 2. Member Service (Task 02) ===
    memberService := service.NewMemberService(
        c.WorkspaceRepo, // CommandRepository
        c.WorkspaceRepo, // QueryRepository
    )

    // === 3. Workspace Service (Task 03) ===
    workspaceService := c.createWorkspaceService(memberService)

    // === 4. Handlers with Real Services ===
    c.WorkspaceHandler = httphandler.NewWorkspaceHandler(workspaceService, memberService)

    // Inject services into template handler
    if c.TemplateHandler != nil {
        c.TemplateHandler.SetServices(workspaceService, memberService)
    }

    // === 5. Chat Service (Task 04) ===
    chatService := c.createChatService()
    c.ChatHandler = httphandler.NewChatHandler(chatService)

    // === 6. Auth Service (Task 05) ===
    authService := c.createAuthService()
    c.AuthHandler = httphandler.NewAuthHandler(authService, c.UserRepo)

    // === 7. WebSocket Handler (unchanged) ===
    c.WSHandler = wshandler.NewHandler(
        c.Hub,
        wshandler.WithHandlerLogger(c.Logger),
        wshandler.WithHandlerConfig(wshandler.HandlerConfig{
            ReadBufferSize:  c.Config.WebSocket.ReadBufferSize,
            WriteBufferSize: c.Config.WebSocket.WriteBufferSize,
            Logger:          c.Logger,
        }),
    )

    // === 8. Token Validator (unchanged) ===
    c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

    c.Logger.Info("HTTP handlers initialized with REAL implementations")
}
```

---

## Вспомогательные методы

### createWorkspaceService

```go
func (c *Container) createWorkspaceService(memberService *service.MemberService) *service.WorkspaceService {
    // Create use cases
    createUC := workspace.NewCreateWorkspaceUseCase(
        c.WorkspaceRepo,
        c.WorkspaceRepo,
        nil, // KeycloakClient - опционально
    )
    getUC := workspace.NewGetWorkspaceUseCase(c.WorkspaceRepo)
    listUC := workspace.NewListUserWorkspacesUseCase(c.WorkspaceRepo)
    updateUC := workspace.NewUpdateWorkspaceUseCase(c.WorkspaceRepo, c.WorkspaceRepo)

    return service.NewWorkspaceService(service.WorkspaceServiceConfig{
        CreateUC:    createUC,
        GetUC:       getUC,
        ListUC:      listUC,
        UpdateUC:    updateUC,
        CommandRepo: c.WorkspaceRepo,
        QueryRepo:   c.WorkspaceRepo,
    })
}
```

### createChatService

```go
func (c *Container) createChatService() *service.ChatService {
    // Create use cases
    createUC := chat.NewCreateChatUseCase(c.EventStore)
    getUC := chat.NewGetChatUseCase(c.ChatRepo)
    listUC := chat.NewListChatsUseCase(c.ChatRepo)
    renameUC := chat.NewRenameChatUseCase(c.ChatRepo)
    addPartUC := chat.NewAddParticipantUseCase(c.ChatRepo)
    removePartUC := chat.NewRemoveParticipantUseCase(c.ChatRepo)

    return service.NewChatService(service.ChatServiceConfig{
        CreateUC:     createUC,
        GetUC:        getUC,
        ListUC:       listUC,
        RenameUC:     renameUC,
        AddPartUC:    addPartUC,
        RemovePartUC: removePartUC,
        CommandRepo:  c.ChatRepo,
    })
}
```

### createAuthService

```go
func (c *Container) createAuthService() httphandler.AuthService {
    // Если AuthService ещё не готов, можно вернуть mock
    if c.Config.Auth.UseMockAuth {
        c.Logger.Warn("using mock auth service (AUTH_USE_MOCK=true)")
        return httphandler.NewMockAuthService()
    }

    // Real implementation
    oauthClient := keycloak.NewOAuthClient(c.Config.Keycloak)
    tokenStore := auth.NewTokenStore(c.Redis)

    return service.NewAuthService(service.AuthServiceConfig{
        OAuthClient: oauthClient,
        TokenStore:  tokenStore,
        UserRepo:    c.UserRepo,
        Logger:      c.Logger,
    })
}
```

---

## Обновление Container struct

Добавить поля для use cases (опционально, если нужен доступ извне):

```go
type Container struct {
    // ... existing fields ...

    // Services
    WorkspaceService *service.WorkspaceService
    MemberService    *service.MemberService
    ChatService      *service.ChatService
    AuthService      httphandler.AuthService
}
```

---

## Обновление imports

```go
import (
    // ... existing imports ...

    "github.com/lllypuk/flowra/internal/service"
    wsapp "github.com/lllypuk/flowra/internal/application/workspace"
    chatapp "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    "github.com/lllypuk/flowra/internal/infrastructure/auth"
)
```

---

## Feature Flags

Добавить возможность переключения между mock и real:

```go
// config/config.go
type AppConfig struct {
    // ... existing fields ...
    UseMockAuth      bool `env:"AUTH_USE_MOCK" default:"false"`
    UseMockWorkspace bool `env:"WORKSPACE_USE_MOCK" default:"false"`
}
```

```go
// container.go
func (c *Container) setupHTTPHandlers() {
    if c.Config.App.UseMockWorkspace {
        c.Logger.Warn("using MOCK workspace services")
        c.setupMockWorkspaceHandlers()
    } else {
        c.setupRealWorkspaceHandlers()
    }
    // ...
}
```

---

## Валидация wiring

Обновить `validateWiring()` для проверки real сервисов:

```go
func (c *Container) validateWiring() error {
    var errs []error

    // ... existing validation ...

    // Validate services are properly initialized
    if c.AccessChecker == nil {
        errs = append(errs, errors.New("access checker not initialized"))
    }

    // Check that we're not accidentally using mocks in production
    if c.Config.IsProduction() {
        if _, isMock := c.AccessChecker.(*middleware.MockWorkspaceAccessChecker); isMock {
            errs = append(errs, errors.New("mock access checker used in production"))
        }
    }

    // ... rest of validation ...
}
```

---

## Порядок инициализации

```
1. setupInfrastructure()
   ├── MongoDB
   ├── Redis
   ├── EventStore
   ├── EventBus
   └── WebSocket Hub

2. setupRepositories()
   ├── UserRepo
   ├── WorkspaceRepo
   ├── ChatRepo
   ├── MessageRepo
   ├── TaskRepo
   └── NotificationRepo

3. setupUseCases() [NEW - расширить]
   ├── Workspace use cases
   ├── Chat use cases
   └── Notification use case

4. setupServices() [NEW]
   ├── WorkspaceAccessChecker
   ├── MemberService
   ├── WorkspaceService
   ├── ChatService
   └── AuthService

5. setupTemplateRenderer()

6. setupHTTPHandlers()
   ├── AuthHandler
   ├── WorkspaceHandler
   ├── ChatHandler
   ├── WSHandler
   └── TokenValidator

7. validateWiring()
```

---

## Тестирование

### Integration test

```go
// cmd/api/container_test.go

func TestContainer_RealWiring(t *testing.T) {
    // Setup test MongoDB and Redis
    cfg := testutil.LoadTestConfig()

    container, err := NewContainer(cfg)
    require.NoError(t, err)
    defer container.Close()

    // Verify all services are real implementations
    assert.NotNil(t, container.WorkspaceHandler)
    assert.NotNil(t, container.ChatHandler)
    assert.NotNil(t, container.AuthHandler)
    assert.NotNil(t, container.AccessChecker)

    // Verify not using mocks (in real mode)
    _, isMock := container.AccessChecker.(*middleware.MockWorkspaceAccessChecker)
    assert.False(t, isMock, "should not use mock in real mode")
}
```

---

## Чеклист

### Подготовка
- [ ] Создать `internal/service/` директорию
- [ ] Убедиться, что все сервисы из Tasks 01-05 реализованы

### Container updates
- [ ] Добавить import для `internal/service`
- [ ] Создать `createWorkspaceService()` method
- [ ] Создать `createChatService()` method
- [ ] Создать `createAuthService()` method
- [ ] Обновить `setupHTTPHandlers()` для real сервисов
- [ ] Обновить `validateWiring()` для проверки real сервисов

### Configuration
- [ ] Добавить feature flags для mock/real switching
- [ ] Обновить `.env.example` с новыми переменными

### Testing
- [ ] Написать integration test для real wiring
- [ ] Проверить, что все существующие тесты проходят
- [ ] E2E тест с HTMX frontend

### Cleanup
- [ ] Удалить TODO комментарии про mock
- [ ] Обновить логирование (убрать Warn про mocks)

---

## Критерии приёмки

- [ ] Все mock-сервисы заменены на real в `setupHTTPHandlers()`
- [ ] Feature flags позволяют вернуться к mock для debugging
- [ ] `validateWiring()` предупреждает о mocks в production
- [ ] Все существующие тесты проходят
- [ ] HTMX frontend работает с реальными данными

---

## Rollback план

Если что-то пойдёт не так:

1. Установить `WORKSPACE_USE_MOCK=true`, `AUTH_USE_MOCK=true`
2. Перезапустить приложение
3. Mock-сервисы будут использоваться

---

## Заметки

- Рекомендуется делать изменения инкрементально (сначала AccessChecker, потом остальные)
- Каждое изменение должно быть отдельным коммитом для лёгкого rollback
- Мониторить логи на ошибки после каждого изменения

---

*Создано: 2026-01-06*
