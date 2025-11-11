# Task 1.3.1: Keycloak Integration

**Приоритет:** 🟡 HIGH
**Статус:** Ready
**Время:** 4-5 дней
**Зависимости:** Нет (может выполняться параллельно)

---

## Проблема

Приложению нужна аутентификация и авторизация пользователей. Keycloak обеспечивает:
- OAuth2/OIDC authentication
- Group management (для Workspaces)
- JWT token validation
- User management

---

## Цель

Интегрировать Keycloak для OAuth2 flow и group sync.

---

## Файлы для создания

```
internal/infrastructure/keycloak/
├── client.go                (interface)
├── http_client.go           (implementation)
├── http_client_test.go      (unit tests)
├── token_validator.go       (JWT validation)
├── token_validator_test.go
└── integration_test.go      (with Keycloak testcontainer)
```

---

## OAuth2 Flow

```
┌──────┐                                      ┌─────────┐
│ User │                                      │ Keycloak│
└───┬──┘                                      └────┬────┘
    │                                              │
    │  1. GET /auth/login                         │
    ├────────────────────────────────────────────▶│
    │                                              │
    │  2. Redirect to Keycloak login              │
    │◀─────────────────────────────────────────────┤
    │                                              │
    │  3. User enters credentials                 │
    ├─────────────────────────────────────────────▶│
    │                                              │
    │  4. Redirect to /auth/callback?code=xxx     │
    │◀─────────────────────────────────────────────┤
    │                                              │
┌───▼──────┐                                      │
│   App    │  5. Exchange code for tokens         │
│          ├─────────────────────────────────────▶│
│          │                                       │
│          │  6. Access + Refresh tokens           │
│          │◀──────────────────────────────────────┤
│          │                                       │
│          │  7. Set session cookie                │
└──────────┘
```

---

## 1. Keycloak Client Interface

```go
package keycloak

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type KeycloakClient interface {
    // OAuth2/OIDC
    GetAuthURL(redirectURI string, state string) string
    ExchangeCode(ctx context.Context, code string, redirectURI string) (*TokenResponse, error)
    RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
    ValidateToken(ctx context.Context, token string) (*Claims, error)
    RevokeToken(ctx context.Context, token string) error

    // Group Management
    CreateGroup(ctx context.Context, name string) (string, error)
    AddUserToGroup(ctx context.Context, userID, groupID string) error
    RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
    ListGroupMembers(ctx context.Context, groupID string) ([]string, error)

    // User Management
    GetUser(ctx context.Context, userID string) (*User, error)
    CreateUser(ctx context.Context, user *CreateUserRequest) (string, error)
}

type TokenResponse struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
    TokenType    string
}

type Claims struct {
    UserID    uuid.UUID
    Username  string
    Email     string
    Roles     []string
    Groups    []string
    ExpiresAt time.Time
}

type User struct {
    ID       string
    Username string
    Email    string
    Enabled  bool
}
```

---

## 2. HTTP Client Implementation

```go
package keycloak

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
)

type HTTPKeycloakClient struct {
    baseURL      string
    realm        string
    clientID     string
    clientSecret string
    httpClient   *http.Client
}

func NewHTTPKeycloakClient(baseURL, realm, clientID, clientSecret string) *HTTPKeycloakClient {
    return &HTTPKeycloakClient{
        baseURL:      baseURL,
        realm:        realm,
        clientID:     clientID,
        clientSecret: clientSecret,
        httpClient:   &http.Client{Timeout: 10 * time.Second},
    }
}

// GetAuthURL - construct authorization URL
func (c *HTTPKeycloakClient) GetAuthURL(redirectURI string, state string) string {
    authURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth", c.baseURL, c.realm)

    params := url.Values{}
    params.Set("client_id", c.clientID)
    params.Set("redirect_uri", redirectURI)
    params.Set("response_type", "code")
    params.Set("scope", "openid profile email")
    params.Set("state", state)

    return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// ExchangeCode - exchange authorization code for tokens
func (c *HTTPKeycloakClient) ExchangeCode(ctx context.Context, code string, redirectURI string) (*TokenResponse, error) {
    tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)

    data := url.Values{}
    data.Set("grant_type", "authorization_code")
    data.Set("code", code)
    data.Set("redirect_uri", redirectURI)
    data.Set("client_id", c.clientID)
    data.Set("client_secret", c.clientSecret)

    req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("token exchange failed: %s", resp.Status)
    }

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, err
    }

    return &tokenResp, nil
}

// RefreshToken - refresh access token
func (c *HTTPKeycloakClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
    tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)

    data := url.Values{}
    data.Set("grant_type", "refresh_token")
    data.Set("refresh_token", refreshToken)
    data.Set("client_id", c.clientID)
    data.Set("client_secret", c.clientSecret)

    // ... similar to ExchangeCode
}

// CreateGroup - create Keycloak group (for Workspace)
func (c *HTTPKeycloakClient) CreateGroup(ctx context.Context, name string) (string, error) {
    groupURL := fmt.Sprintf("%s/admin/realms/%s/groups", c.baseURL, c.realm)

    body := map[string]string{"name": name}
    data, _ := json.Marshal(body)

    req, _ := http.NewRequestWithContext(ctx, "POST", groupURL, bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+c.getAdminToken())

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return "", fmt.Errorf("failed to create group: %s", resp.Status)
    }

    // Extract group ID from Location header
    location := resp.Header.Get("Location")
    groupID := location[strings.LastIndex(location, "/")+1:]

    return groupID, nil
}

// AddUserToGroup - add user to group
func (c *HTTPKeycloakClient) AddUserToGroup(ctx context.Context, userID, groupID string) error {
    url := fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s", c.baseURL, c.realm, userID, groupID)

    req, _ := http.NewRequestWithContext(ctx, "PUT", url, nil)
    req.Header.Set("Authorization", "Bearer "+c.getAdminToken())

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("failed to add user to group: %s", resp.Status)
    }

    return nil
}
```

---

## 3. Token Validator

```go
package keycloak

import (
    "context"
    "fmt"
    "github.com/golang-jwt/jwt/v5"
    "github.com/lestrrat-go/jwx/v2/jwk"
)

type TokenValidator struct {
    jwksURL  string
    issuer   string
    audience string
    keySet   jwk.Set
}

func NewTokenValidator(keycloakURL, realm string) *TokenValidator {
    jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", keycloakURL, realm)
    issuer := fmt.Sprintf("%s/realms/%s", keycloakURL, realm)

    return &TokenValidator{
        jwksURL: jwksURL,
        issuer:  issuer,
    }
}

// Validate - validate JWT token
func (v *TokenValidator) Validate(tokenString string) (*Claims, error) {
    // 1. Fetch JWKS (cached)
    if v.keySet == nil {
        keySet, err := jwk.Fetch(context.Background(), v.jwksURL)
        if err != nil {
            return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
        }
        v.keySet = keySet
    }

    // 2. Parse and validate token
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Get key from JWKS
        keyID, ok := token.Header["kid"].(string)
        if !ok {
            return nil, fmt.Errorf("missing kid in token header")
        }

        key, ok := v.keySet.LookupKeyID(keyID)
        if !ok {
            return nil, fmt.Errorf("key not found: %s", keyID)
        }

        var rawKey interface{}
        if err := key.Raw(&rawKey); err != nil {
            return nil, err
        }

        return rawKey, nil
    })

    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    // 3. Extract claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }

    // 4. Verify issuer
    if claims["iss"] != v.issuer {
        return nil, fmt.Errorf("invalid issuer")
    }

    // 5. Map to Claims struct
    return &Claims{
        UserID:   uuid.MustParse(claims["sub"].(string)),
        Username: claims["preferred_username"].(string),
        Email:    claims["email"].(string),
        // ... extract roles, groups
    }, nil
}
```

---

## 4. Integration in Auth Handler

```go
// internal/handler/http/auth_handler.go

type AuthHandler struct {
    keycloakClient keycloak.KeycloakClient
    sessionRepo    redis.SessionRepository
}

func (h *AuthHandler) Login(c echo.Context) error {
    // Generate state for CSRF protection
    state := uuid.New().String()

    // Store state in session
    c.SetCookie(&http.Cookie{
        Name:  "oauth_state",
        Value: state,
        Path:  "/",
    })

    // Redirect to Keycloak
    authURL := h.keycloakClient.GetAuthURL("http://localhost:8080/auth/callback", state)
    return c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) Callback(c echo.Context) error {
    // 1. Verify state
    stateCookie, _ := c.Cookie("oauth_state")
    if stateCookie.Value != c.QueryParam("state") {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid state")
    }

    // 2. Exchange code for tokens
    code := c.QueryParam("code")
    tokens, err := h.keycloakClient.ExchangeCode(c.Request().Context(), code, "http://localhost:8080/auth/callback")
    if err != nil {
        return err
    }

    // 3. Validate token and extract claims
    claims, err := h.keycloakClient.ValidateToken(c.Request().Context(), tokens.AccessToken)
    if err != nil {
        return err
    }

    // 4. Create session
    sessionID := uuid.New().String()
    sessionData := &redis.SessionData{
        UserID:       claims.UserID,
        Username:     claims.Username,
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresAt:    time.Now().Add(24 * time.Hour),
    }

    h.sessionRepo.Save(c.Request().Context(), sessionID, sessionData, 24*time.Hour)

    // 5. Set session cookie
    c.SetCookie(&http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    return c.Redirect(http.StatusFound, "/")
}
```

---

## Тестирование

```go
func TestKeycloakClient_ExchangeCode_Integration(t *testing.T) {
    // Setup Keycloak (testcontainers)
    keycloakURL := testutil.SetupKeycloak(t)

    client := keycloak.NewHTTPKeycloakClient(keycloakURL, "test-realm", "client-id", "secret")

    // Simulate OAuth flow
    // ...
}

func TestTokenValidator_ValidToken(t *testing.T) {
    // Test with valid JWT
}
```

---

## Критерии успеха

- ✅ **OAuth2 flow works end-to-end**
- ✅ **Token validation correct**
- ✅ **Group management works**
- ✅ **User can login/logout**
- ✅ **Session persisted in Redis**
- ✅ **Test coverage >75%**

---

## Следующий шаг

Phase 1 завершена → **Phase 2: Interface Layer**
