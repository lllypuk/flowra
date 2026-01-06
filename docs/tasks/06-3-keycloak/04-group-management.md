# 04: Group Management

**Приоритет:** 🟡 High
**Статус:** ✅ Выполнено
**Зависит от:** [01-realm-setup.md](01-realm-setup.md)

---

## Описание

Реализовать Keycloak Admin API клиент для управления группами. Группы используются для представления workspaces — каждый workspace соответствует группе в Keycloak.

---

## Текущее состояние

Реализован полноценный Keycloak Admin API клиент для управления группами:

```go
// internal/infrastructure/keycloak/group_client.go
type GroupClient struct {
    config       GroupClientConfig
    tokenManager *AdminTokenManager
    httpClient   *http.Client
}

// Implements workspace.KeycloakClient interface
func (c *GroupClient) CreateGroup(ctx context.Context, name string) (string, error)
func (c *GroupClient) DeleteGroup(ctx context.Context, groupID string) error
func (c *GroupClient) AddUserToGroup(ctx context.Context, userID, groupID string) error
func (c *GroupClient) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
```

**Дополнительно реализовано:**
- `GetGroup` — получение информации о группе
- `GetUserGroups` — получение списка групп пользователя

---

## Keycloak Admin API

### Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create group | POST | `/admin/realms/{realm}/groups` |
| Delete group | DELETE | `/admin/realms/{realm}/groups/{id}` |
| Get group | GET | `/admin/realms/{realm}/groups/{id}` |
| Add user to group | PUT | `/admin/realms/{realm}/users/{userId}/groups/{groupId}` |
| Remove user from group | DELETE | `/admin/realms/{realm}/users/{userId}/groups/{groupId}` |
| List user groups | GET | `/admin/realms/{realm}/users/{userId}/groups` |

### Authentication

```
POST /realms/master/protocol/openid-connect/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
client_id=admin-cli
client_secret=<secret>

# Or with password grant
grant_type=password
client_id=admin-cli
username=admin
password=admin123
```

---

## Файлы

```
internal/infrastructure/keycloak/
├── admin_token.go           # Admin token management ✅
├── admin_token_test.go      # Tests ✅
├── group_client.go          # Group management client ✅
├── group_client_test.go     # Tests ✅
├── jwt_validator.go         # JWT validation (existing)
├── jwt_validator_test.go    # Tests (existing)
├── oauth_client.go          # OAuth client (existing)
└── oauth_client_test.go     # Tests (existing)
```

---

## Реализация

### Interface (уже существует)

```go
// internal/application/workspace/keycloak_client.go

type KeycloakClient interface {
    CreateGroup(ctx context.Context, name string) (groupID string, err error)
    DeleteGroup(ctx context.Context, groupID string) error
    AddUserToGroup(ctx context.Context, userID, groupID string) error
    RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
}
```

### Admin Token Manager

```go
// internal/infrastructure/keycloak/admin_token.go

package keycloak

type AdminTokenConfig struct {
    KeycloakURL   string
    Realm         string        // Usually "master" for admin operations
    ClientID      string        // Usually "admin-cli"
    ClientSecret  string        // Or use username/password
    Username      string
    Password      string
    TokenBuffer   time.Duration // Refresh before expiry (default 30s)
    HTTPClient    *http.Client
}

type AdminTokenManager struct {
    config     AdminTokenConfig
    httpClient *http.Client

    mu        sync.RWMutex
    token     string
    expiresAt time.Time
}

// NewAdminTokenManager creates a new AdminTokenManager
func NewAdminTokenManager(config AdminTokenConfig) *AdminTokenManager

// GetToken returns valid admin token, refreshing if needed
func (m *AdminTokenManager) GetToken(ctx context.Context) (string, error)

// InvalidateToken clears the cached token
func (m *AdminTokenManager) InvalidateToken()
```

### Group Client

```go
// internal/infrastructure/keycloak/group_client.go

package keycloak

type GroupClientConfig struct {
    KeycloakURL string
    Realm       string
    HTTPClient  *http.Client
}

type GroupClient struct {
    config       GroupClientConfig
    tokenManager *AdminTokenManager
    httpClient   *http.Client
}

func NewGroupClient(config GroupClientConfig, tokenManager *AdminTokenManager) *GroupClient

// Implements workspace.KeycloakClient
func (c *GroupClient) CreateGroup(ctx context.Context, name string) (string, error)
func (c *GroupClient) DeleteGroup(ctx context.Context, groupID string) error
func (c *GroupClient) AddUserToGroup(ctx context.Context, userID, groupID string) error
func (c *GroupClient) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error

// Additional methods
func (c *GroupClient) GetGroup(ctx context.Context, groupID string) (*Group, error)
func (c *GroupClient) GetUserGroups(ctx context.Context, userID string) ([]Group, error)
```

---

## Container Integration

```go
// cmd/api/container.go

func (c *Container) createWorkspaceService() *service.WorkspaceService {
    var keycloakClient wsapp.KeycloakClient
    if c.Config.Keycloak.URL != "" && c.Config.Keycloak.AdminUsername != "" {
        c.Logger.Debug("using real Keycloak GroupClient for workspace service",
            slog.String("url", c.Config.Keycloak.URL),
            slog.String("realm", c.Config.Keycloak.Realm),
        )

        tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
            KeycloakURL: c.Config.Keycloak.URL,
            Realm:       "master",
            ClientID:    "admin-cli",
            Username:    c.Config.Keycloak.AdminUsername,
            Password:    c.Config.Keycloak.AdminPassword,
            TokenBuffer: 30 * time.Second,
        })

        keycloakClient = keycloak.NewGroupClient(keycloak.GroupClientConfig{
            KeycloakURL: c.Config.Keycloak.URL,
            Realm:       c.Config.Keycloak.Realm,
        }, tokenManager)
    } else {
        c.Logger.Debug("using NoOp Keycloak client for workspace service (admin not configured)")
        keycloakClient = service.NewNoOpKeycloakClient()
    }

    // ... rest of workspace service creation
}
```

---

## Чеклист

### Admin Token
- [x] `AdminTokenManager` реализован
- [x] Token caching работает
- [x] Auto-refresh before expiry
- [x] Password grant поддержан
- [x] Client credentials grant поддержан

### Group Client
- [x] `CreateGroup` реализован
- [x] `DeleteGroup` реализован
- [x] `AddUserToGroup` реализован
- [x] `RemoveUserFromGroup` реализован
- [x] `GetGroup` реализован (дополнительно)
- [x] `GetUserGroups` реализован (дополнительно)
- [x] Error handling

### Testing
- [x] Unit tests с mock HTTP (21 тест для AdminTokenManager)
- [x] Unit tests с mock HTTP (18 тестов для GroupClient)
- [ ] Integration test с реальным Keycloak

### Integration
- [x] Container создаёт real client когда настроен
- [x] Fallback на NoOp когда не настроен
- [x] WorkspaceService использует client

---

## Критерии приёмки

- [x] При создании workspace создаётся группа в Keycloak
- [x] При добавлении member пользователь добавляется в группу
- [x] При удалении member пользователь удаляется из группы
- [x] При удалении workspace группа удаляется
- [x] Admin token автоматически обновляется
- [x] Graceful degradation при недоступности Keycloak

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — Admin API настроен

### Исходящие
- [05-user-sync.md](05-user-sync.md) — может использовать group client
- [06-integration-tests.md](06-integration-tests.md) — тестирует группы

---

*Обновлено: 2026-01-06*
