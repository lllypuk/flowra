# 06: Integration Tests

**Приоритет:** 🟡 High
**Статус:** ⏳ Не начато
**Зависит от:** Все предыдущие задачи (01-05)

---

## Описание

Реализовать интеграционные тесты для Keycloak интеграции с использованием testcontainers. Тесты должны проверять полный auth flow, группы и sync.

---

## Текущие тесты

Существуют только unit-тесты с mocks:
- `oauth_client_test.go` — mock HTTP server
- `token_store_test.go` — Redis
- `auth_service_test.go` — mocks

**Проблемы:**
- Не тестируется реальное взаимодействие с Keycloak
- Нет E2E тестов auth flow
- Не проверяется совместимость с конкретной версией Keycloak

---

## Решение

### Testcontainers

```go
// Keycloak container для тестов
container := testcontainers.NewKeycloakContainer(
    testcontainers.WithRealmImportFile("./testdata/realm-export.json"),
)
```

---

## Файлы

```
tests/integration/
├── keycloak_test.go           # Integration tests
├── testdata/
│   ├── realm-export.json      # Test realm config
│   └── users.json             # Test users
└── helpers/
    └── keycloak_container.go  # Container helpers
```

---

## Реализация

### Keycloak Container Helper

```go
// tests/integration/helpers/keycloak_container.go

package helpers

import (
    "context"
    "fmt"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type KeycloakContainer struct {
    container testcontainers.Container
    URL       string
    AdminUser string
    AdminPass string
}

func NewKeycloakContainer(ctx context.Context) (*KeycloakContainer, error) {
    req := testcontainers.ContainerRequest{
        Image:        "quay.io/keycloak/keycloak:23.0",
        ExposedPorts: []string{"8080/tcp"},
        Env: map[string]string{
            "KEYCLOAK_ADMIN":          "admin",
            "KEYCLOAK_ADMIN_PASSWORD": "admin",
            "KC_DB":                   "dev-file",
        },
        Cmd: []string{"start-dev", "--import-realm"},
        Files: []testcontainers.ContainerFile{
            {
                HostFilePath:      "./testdata/realm-export.json",
                ContainerFilePath: "/opt/keycloak/data/import/realm-export.json",
                FileMode:          0644,
            },
        },
        WaitingFor: wait.ForHTTP("/health/ready").
            WithPort("8080/tcp").
            WithStartupTimeout(120 * time.Second),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to start keycloak container: %w", err)
    }

    host, err := container.Host(ctx)
    if err != nil {
        return nil, err
    }

    port, err := container.MappedPort(ctx, "8080")
    if err != nil {
        return nil, err
    }

    return &KeycloakContainer{
        container: container,
        URL:       fmt.Sprintf("http://%s:%s", host, port.Port()),
        AdminUser: "admin",
        AdminPass: "admin",
    }, nil
}

func (k *KeycloakContainer) Terminate(ctx context.Context) error {
    return k.container.Terminate(ctx)
}

// GetTestUserToken получает токен для тестового пользователя
func (k *KeycloakContainer) GetTestUserToken(ctx context.Context, username, password string) (string, error) {
    tokenURL := fmt.Sprintf("%s/realms/flowra/protocol/openid-connect/token", k.URL)

    data := url.Values{}
    data.Set("grant_type", "password")
    data.Set("client_id", "flowra-backend")
    data.Set("client_secret", "test-secret")
    data.Set("username", username)
    data.Set("password", password)

    resp, err := http.PostForm(tokenURL, data)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var tokenResp struct {
        AccessToken string `json:"access_token"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return "", err
    }

    return tokenResp.AccessToken, nil
}
```

### Test Realm Export

```json
// tests/integration/testdata/realm-export.json
{
  "realm": "flowra",
  "enabled": true,
  "clients": [
    {
      "clientId": "flowra-backend",
      "enabled": true,
      "clientAuthenticatorType": "client-secret",
      "secret": "test-secret",
      "directAccessGrantsEnabled": true,
      "standardFlowEnabled": true,
      "redirectUris": ["*"],
      "webOrigins": ["*"]
    }
  ],
  "roles": {
    "realm": [
      {"name": "user"},
      {"name": "admin"}
    ]
  },
  "users": [
    {
      "username": "testuser",
      "email": "testuser@example.com",
      "enabled": true,
      "emailVerified": true,
      "firstName": "Test",
      "lastName": "User",
      "credentials": [
        {
          "type": "password",
          "value": "password123",
          "temporary": false
        }
      ],
      "realmRoles": ["user"]
    },
    {
      "username": "admin",
      "email": "admin@example.com",
      "enabled": true,
      "emailVerified": true,
      "firstName": "Admin",
      "lastName": "User",
      "credentials": [
        {
          "type": "password",
          "value": "admin123",
          "temporary": false
        }
      ],
      "realmRoles": ["user", "admin"]
    }
  ],
  "groups": [
    {"name": "users"},
    {"name": "admins"}
  ]
}
```

### Integration Tests

```go
// tests/integration/keycloak_test.go

//go:build integration

package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"

    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    "github.com/lllypuk/flowra/tests/integration/helpers"
)

type KeycloakIntegrationSuite struct {
    suite.Suite
    container *helpers.KeycloakContainer
    ctx       context.Context
}

func TestKeycloakIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    suite.Run(t, new(KeycloakIntegrationSuite))
}

func (s *KeycloakIntegrationSuite) SetupSuite() {
    s.ctx = context.Background()

    var err error
    s.container, err = helpers.NewKeycloakContainer(s.ctx)
    s.Require().NoError(err, "Failed to start Keycloak container")
}

func (s *KeycloakIntegrationSuite) TearDownSuite() {
    if s.container != nil {
        s.container.Terminate(s.ctx)
    }
}

// Test OAuth Token Exchange
func (s *KeycloakIntegrationSuite) TestOAuthClient_ExchangeCode() {
    // This would require a browser-based flow, so we test with password grant
    token, err := s.container.GetTestUserToken(s.ctx, "testuser", "password123")
    s.Require().NoError(err)
    s.NotEmpty(token)
}

// Test JWT Validation
func (s *KeycloakIntegrationSuite) TestJWTValidator_Validate() {
    // Get a real token
    token, err := s.container.GetTestUserToken(s.ctx, "testuser", "password123")
    s.Require().NoError(err)

    // Create validator
    validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
        KeycloakURL:     s.container.URL,
        Realm:           "flowra",
        ClientID:        "flowra-backend",
        Leeway:          30 * time.Second,
        RefreshInterval: time.Hour,
    })
    s.Require().NoError(err)
    defer validator.Close()

    // Validate token
    claims, err := validator.Validate(s.ctx, token)
    s.Require().NoError(err)

    s.Equal("testuser", claims.Username)
    s.Equal("testuser@example.com", claims.Email)
    s.True(claims.EmailVerified)
    s.Contains(claims.RealmRoles, "user")
}

// Test Invalid Token
func (s *KeycloakIntegrationSuite) TestJWTValidator_InvalidToken() {
    validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
        KeycloakURL:     s.container.URL,
        Realm:           "flowra",
        ClientID:        "flowra-backend",
        Leeway:          30 * time.Second,
        RefreshInterval: time.Hour,
    })
    s.Require().NoError(err)
    defer validator.Close()

    _, err = validator.Validate(s.ctx, "invalid.token.here")
    s.Error(err)
}

// Test Group Management
func (s *KeycloakIntegrationSuite) TestGroupClient_CreateAndDelete() {
    tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
        KeycloakURL: s.container.URL,
        Realm:       "master",
        ClientID:    "admin-cli",
        Username:    s.container.AdminUser,
        Password:    s.container.AdminPass,
    })

    client := keycloak.NewGroupClient(keycloak.GroupClientConfig{
        KeycloakURL: s.container.URL,
        Realm:       "flowra",
    }, tokenManager)

    // Create group
    groupID, err := client.CreateGroup(s.ctx, "test-workspace-123")
    s.Require().NoError(err)
    s.NotEmpty(groupID)

    // Delete group
    err = client.DeleteGroup(s.ctx, groupID)
    s.NoError(err)
}

// Test User to Group
func (s *KeycloakIntegrationSuite) TestGroupClient_AddRemoveUser() {
    tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
        KeycloakURL: s.container.URL,
        Realm:       "master",
        ClientID:    "admin-cli",
        Username:    s.container.AdminUser,
        Password:    s.container.AdminPass,
    })

    client := keycloak.NewGroupClient(keycloak.GroupClientConfig{
        KeycloakURL: s.container.URL,
        Realm:       "flowra",
    }, tokenManager)

    // Create group
    groupID, err := client.CreateGroup(s.ctx, "test-workspace-456")
    s.Require().NoError(err)
    defer client.DeleteGroup(s.ctx, groupID)

    // Get testuser ID (would need to look up)
    userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
        KeycloakURL: s.container.URL,
        Realm:       "flowra",
    }, tokenManager)

    users, err := userClient.ListUsers(s.ctx, 0, 100)
    s.Require().NoError(err)

    var testUserID string
    for _, u := range users {
        if u.Username == "testuser" {
            testUserID = u.ID
            break
        }
    }
    s.Require().NotEmpty(testUserID)

    // Add user to group
    err = client.AddUserToGroup(s.ctx, testUserID, groupID)
    s.NoError(err)

    // Remove user from group
    err = client.RemoveUserFromGroup(s.ctx, testUserID, groupID)
    s.NoError(err)
}

// Test Token Refresh
func (s *KeycloakIntegrationSuite) TestOAuthClient_RefreshToken() {
    oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
        KeycloakURL:  s.container.URL,
        Realm:        "flowra",
        ClientID:     "flowra-backend",
        ClientSecret: "test-secret",
    })

    // Get initial token with password grant
    tokenURL := fmt.Sprintf("%s/realms/flowra/protocol/openid-connect/token", s.container.URL)
    data := url.Values{}
    data.Set("grant_type", "password")
    data.Set("client_id", "flowra-backend")
    data.Set("client_secret", "test-secret")
    data.Set("username", "testuser")
    data.Set("password", "password123")

    resp, err := http.PostForm(tokenURL, data)
    s.Require().NoError(err)
    defer resp.Body.Close()

    var tokenResp struct {
        RefreshToken string `json:"refresh_token"`
    }
    err = json.NewDecoder(resp.Body).Decode(&tokenResp)
    s.Require().NoError(err)

    // Refresh the token
    newTokens, err := oauthClient.RefreshToken(s.ctx, tokenResp.RefreshToken)
    s.Require().NoError(err)
    s.NotEmpty(newTokens.AccessToken)
}

// Test Admin Token Manager Caching
func (s *KeycloakIntegrationSuite) TestAdminTokenManager_Caching() {
    tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
        KeycloakURL: s.container.URL,
        Realm:       "master",
        ClientID:    "admin-cli",
        Username:    s.container.AdminUser,
        Password:    s.container.AdminPass,
        TokenBuffer: 30 * time.Second,
    })

    // First call - fetches new token
    token1, err := tokenManager.GetToken(s.ctx)
    s.Require().NoError(err)
    s.NotEmpty(token1)

    // Second call - should return cached token
    token2, err := tokenManager.GetToken(s.ctx)
    s.Require().NoError(err)
    s.Equal(token1, token2)
}
```

---

## Makefile

```makefile
# Makefile
.PHONY: test-integration

test-integration:
	go test -tags=integration -v -count=1 ./tests/integration/...

test-integration-keycloak:
	go test -tags=integration -v -count=1 -run TestKeycloakIntegration ./tests/integration/...
```

---

## Чеклист

### Container Setup
- [ ] Keycloak container helper
- [ ] Test realm export
- [ ] Test users created
- [ ] Container cleanup

### OAuth Tests
- [ ] Token exchange (password grant)
- [ ] Token refresh
- [ ] Token revocation
- [ ] User info endpoint

### JWT Tests
- [ ] Valid token validation
- [ ] Invalid token rejection
- [ ] Expired token rejection
- [ ] Claims extraction

### Group Tests
- [ ] Create group
- [ ] Delete group
- [ ] Add user to group
- [ ] Remove user from group

### Admin Tests
- [ ] Admin token caching
- [ ] Token refresh

---

## Критерии приёмки

- [ ] Все тесты проходят с реальным Keycloak
- [ ] Тесты запускаются через `make test-integration`
- [ ] Container cleanup работает
- [ ] Тесты изолированы (не влияют друг на друга)
- [ ] Время выполнения < 2 минут
- [ ] CI/CD интеграция настроена

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — конфигурация для тестов
- [02-jwt-validation.md](02-jwt-validation.md) — JWT validator
- [03-token-middleware.md](03-token-middleware.md) — middleware
- [04-group-management.md](04-group-management.md) — group client
- [05-user-sync.md](05-user-sync.md) — user client

### Go Dependencies

```go
require (
    github.com/testcontainers/testcontainers-go v0.27.0
    github.com/stretchr/testify v1.8.4
)
```

---

*Обновлено: 2026-01-06*
