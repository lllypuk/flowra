# 04: Group Management

**Приоритет:** 🟡 High
**Статус:** ⏳ Не начато
**Зависит от:** [01-realm-setup.md](01-realm-setup.md)

---

## Описание

Реализовать Keycloak Admin API клиент для управления группами. Группы используются для представления workspaces — каждый workspace соответствует группе в Keycloak.

---

## Текущее состояние

Сейчас используется NoOp реализация:

```go
// internal/application/workspace/noop_keycloak_client.go
type NoOpKeycloakClient struct{}

func (c *NoOpKeycloakClient) CreateGroup(ctx context.Context, name string) (string, error) {
    return uuid.New().String(), nil  // Fake group ID
}
```

**Проблемы:**
- Группы не создаются в Keycloak
- Пользователи не добавляются в группы
- Нет синхронизации доступа workspace ↔ Keycloak groups

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
├── group_client.go           # Group management client
├── group_client_test.go      # Tests
├── admin_token.go            # Admin token management
└── admin_token_test.go       # Tests
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

import (
    "context"
    "sync"
    "time"
)

// AdminTokenManager manages admin API tokens
type AdminTokenManager struct {
    config      AdminTokenConfig
    httpClient  *http.Client

    mu          sync.RWMutex
    token       string
    expiresAt   time.Time
}

type AdminTokenConfig struct {
    KeycloakURL   string
    Realm         string        // Usually "master" for admin operations
    ClientID      string        // Usually "admin-cli"
    ClientSecret  string        // Or use username/password
    Username      string
    Password      string
    TokenBuffer   time.Duration // Refresh before expiry (default 30s)
}

func NewAdminTokenManager(config AdminTokenConfig) *AdminTokenManager {
    return &AdminTokenManager{
        config:     config,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

// GetToken returns valid admin token, refreshing if needed
func (m *AdminTokenManager) GetToken(ctx context.Context) (string, error) {
    m.mu.RLock()
    if m.token != "" && time.Now().Add(m.config.TokenBuffer).Before(m.expiresAt) {
        token := m.token
        m.mu.RUnlock()
        return token, nil
    }
    m.mu.RUnlock()

    return m.refreshToken(ctx)
}

func (m *AdminTokenManager) refreshToken(ctx context.Context) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Double-check after acquiring write lock
    if m.token != "" && time.Now().Add(m.config.TokenBuffer).Before(m.expiresAt) {
        return m.token, nil
    }

    tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
        m.config.KeycloakURL, m.config.Realm)

    data := url.Values{}
    if m.config.ClientSecret != "" {
        data.Set("grant_type", "client_credentials")
        data.Set("client_id", m.config.ClientID)
        data.Set("client_secret", m.config.ClientSecret)
    } else {
        data.Set("grant_type", "password")
        data.Set("client_id", m.config.ClientID)
        data.Set("username", m.config.Username)
        data.Set("password", m.config.Password)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := m.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("token request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("token request failed: %s", body)
    }

    var tokenResp struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return "", err
    }

    m.token = tokenResp.AccessToken
    m.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

    return m.token, nil
}
```

### Group Client

```go
// internal/infrastructure/keycloak/group_client.go

package keycloak

import (
    "context"
    "fmt"
    "net/http"
)

type GroupClient struct {
    config       GroupClientConfig
    tokenManager *AdminTokenManager
    httpClient   *http.Client
}

type GroupClientConfig struct {
    KeycloakURL string
    Realm       string
}

func NewGroupClient(config GroupClientConfig, tokenManager *AdminTokenManager) *GroupClient {
    return &GroupClient{
        config:       config,
        tokenManager: tokenManager,
        httpClient:   &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *GroupClient) CreateGroup(ctx context.Context, name string) (string, error) {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get admin token: %w", err)
    }

    url := fmt.Sprintf("%s/admin/realms/%s/groups", c.config.KeycloakURL, c.config.Realm)

    body := map[string]string{"name": name}
    jsonBody, _ := json.Marshal(body)

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
    if err != nil {
        return "", err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("create group request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("create group failed: %s", body)
    }

    // Extract group ID from Location header
    location := resp.Header.Get("Location")
    parts := strings.Split(location, "/")
    groupID := parts[len(parts)-1]

    return groupID, nil
}

func (c *GroupClient) DeleteGroup(ctx context.Context, groupID string) error {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return fmt.Errorf("failed to get admin token: %w", err)
    }

    url := fmt.Sprintf("%s/admin/realms/%s/groups/%s",
        c.config.KeycloakURL, c.config.Realm, groupID)

    req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("delete group request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("delete group failed: %s", body)
    }

    return nil
}

func (c *GroupClient) AddUserToGroup(ctx context.Context, userID, groupID string) error {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return fmt.Errorf("failed to get admin token: %w", err)
    }

    url := fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s",
        c.config.KeycloakURL, c.config.Realm, userID, groupID)

    req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("add user to group request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("add user to group failed: %s", body)
    }

    return nil
}

func (c *GroupClient) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
    token, err := c.tokenManager.GetToken(ctx)
    if err != nil {
        return fmt.Errorf("failed to get admin token: %w", err)
    }

    url := fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s",
        c.config.KeycloakURL, c.config.Realm, userID, groupID)

    req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("remove user from group request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("remove user from group failed: %s", body)
    }

    return nil
}
```

---

## Container Integration

```go
// cmd/api/container.go

func (c *Container) createKeycloakClient() workspace.KeycloakClient {
    // Use NoOp if Keycloak not configured
    if c.Config.Keycloak.URL == "" || c.Config.Keycloak.AdminUsername == "" {
        c.Logger.Warn("Keycloak admin not configured, using NoOp client")
        return workspace.NewNoOpKeycloakClient()
    }

    tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
        KeycloakURL: c.Config.Keycloak.URL,
        Realm:       "master",
        ClientID:    "admin-cli",
        Username:    c.Config.Keycloak.AdminUsername,
        Password:    c.Config.Keycloak.AdminPassword,
        TokenBuffer: 30 * time.Second,
    })

    return keycloak.NewGroupClient(keycloak.GroupClientConfig{
        KeycloakURL: c.Config.Keycloak.URL,
        Realm:       c.Config.Keycloak.Realm,
    }, tokenManager)
}
```

---

## Чеклист

### Admin Token
- [ ] `AdminTokenManager` реализован
- [ ] Token caching работает
- [ ] Auto-refresh before expiry
- [ ] Password grant поддержан
- [ ] Client credentials grant поддержан

### Group Client
- [ ] `CreateGroup` реализован
- [ ] `DeleteGroup` реализован
- [ ] `AddUserToGroup` реализован
- [ ] `RemoveUserFromGroup` реализован
- [ ] Error handling

### Testing
- [ ] Unit tests с mock HTTP
- [ ] Integration test с реальным Keycloak

### Integration
- [ ] Container создаёт real client когда настроен
- [ ] Fallback на NoOp когда не настроен
- [ ] WorkspaceService использует client

---

## Критерии приёмки

- [ ] При создании workspace создаётся группа в Keycloak
- [ ] При добавлении member пользователь добавляется в группу
- [ ] При удалении member пользователь удаляется из группы
- [ ] При удалении workspace группа удаляется
- [ ] Admin token автоматически обновляется
- [ ] Graceful degradation при недоступности Keycloak

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — Admin API настроен

### Исходящие
- [05-user-sync.md](05-user-sync.md) — может использовать group client
- [06-integration-tests.md](06-integration-tests.md) — тестирует группы

---

*Обновлено: 2026-01-06*
