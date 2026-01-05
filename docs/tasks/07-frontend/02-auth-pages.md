# 02: Auth Pages

**Приоритет:** 🔴 Critical
**Статус:** ✅ Завершено
**Период:** 4-5 февраля
**Зависит от:** [01-base-infrastructure.md](01-base-infrastructure.md)

---

## Описание

Реализовать страницы аутентификации: login page с редиректом на Keycloak, OAuth callback обработка, и logout flow. Интеграция с существующим Auth API.

---

## Файлы

### Templates

```
web/templates/auth/
├── login.html          (~50 LOC) - Login page
├── callback.html       (~30 LOC) - OAuth callback processing
└── logout.html         (~25 LOC) - Logout confirmation
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+150 LOC) - Auth page handlers
```

---

## Auth Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   /login    │────>│  Keycloak   │────>│  /callback  │
│   (page)    │     │  (OAuth)    │     │  (handler)  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               v
                                        ┌─────────────┐
                                        │ /workspaces │
                                        │  (redirect) │
                                        └─────────────┘
```

---

## Детали реализации

### 1. Login Page (login.html)

```html
{{define "auth/login"}}
<article class="login-container">
    <header>
        <hgroup>
            <h1>Welcome to Flowra</h1>
            <p>Chat-first task management for teams</p>
        </hgroup>
    </header>

    <div class="login-content">
        {{if .Error}}
        <article class="flash flash-error">
            {{.Error}}
        </article>
        {{end}}

        <p class="text-center text-muted">
            Sign in with your organization account
        </p>

        <a href="{{.AuthURL}}"
           role="button"
           class="login-button">
            Sign in with SSO
        </a>

        <hr>

        <footer class="text-center text-muted">
            <small>
                By signing in, you agree to our
                <a href="/terms">Terms of Service</a> and
                <a href="/privacy">Privacy Policy</a>
            </small>
        </footer>
    </div>
</article>

<style>
.login-container {
    max-width: 400px;
    margin: 4rem auto;
}

.login-content {
    padding: 2rem;
}

.login-button {
    width: 100%;
    display: block;
    text-align: center;
}
</style>
{{end}}
```

### 2. Callback Page (callback.html)

```html
{{define "auth/callback"}}
<article class="callback-container">
    <header>
        <h2>Signing you in...</h2>
    </header>

    <div class="text-center">
        <span aria-busy="true"></span>
        <p class="text-muted">Please wait while we complete the sign-in process.</p>
    </div>

    {{if .Error}}
    <article class="flash flash-error">
        <h4>Sign-in failed</h4>
        <p>{{.Error}}</p>
        <a href="/login">Try again</a>
    </article>
    {{else}}
    <!-- Auto-redirect via meta refresh as fallback -->
    <meta http-equiv="refresh" content="0;url={{.RedirectURL}}">
    {{end}}
</article>

<style>
.callback-container {
    max-width: 400px;
    margin: 4rem auto;
    text-align: center;
}
</style>
{{end}}
```

### 3. Logout Confirmation (logout.html)

```html
{{define "auth/logout"}}
<article class="logout-container">
    <header>
        <h2>Sign Out</h2>
    </header>

    <p>Are you sure you want to sign out?</p>

    <footer>
        <form hx-post="/auth/logout"
              hx-redirect="/">
            <div class="grid">
                <a href="javascript:history.back()"
                   role="button"
                   class="secondary outline">
                    Cancel
                </a>
                <button type="submit">
                    Sign Out
                </button>
            </div>
        </form>
    </footer>
</article>

<style>
.logout-container {
    max-width: 400px;
    margin: 4rem auto;
}
</style>
{{end}}
```

### 4. Handler Implementation

```go
// internal/handler/http/template_handler.go

// LoginPage renders the login page with Keycloak auth URL
func (h *TemplateHandler) LoginPage(c echo.Context) error {
    // Check if already logged in
    if user := getUserFromContext(c); user != nil {
        return c.Redirect(http.StatusFound, "/workspaces")
    }

    // Build Keycloak auth URL
    state := generateState()
    setStateCookie(c, state)

    authURL := h.authService.GetAuthURL(state, getRedirectURI(c))

    data := map[string]interface{}{
        "Title":   "Login",
        "AuthURL": authURL,
        "Error":   c.QueryParam("error"),
    }

    return h.render(c, "layout/base.html", "auth/login", data)
}

// AuthCallback handles OAuth callback from Keycloak
func (h *TemplateHandler) AuthCallback(c echo.Context) error {
    code := c.QueryParam("code")
    state := c.QueryParam("state")
    errorParam := c.QueryParam("error")

    // Check for OAuth error
    if errorParam != "" {
        return h.renderCallback(c, "", errorParam)
    }

    // Validate state
    expectedState := getStateCookie(c)
    if state != expectedState {
        return h.renderCallback(c, "", "Invalid state parameter")
    }

    // Exchange code for tokens
    tokens, err := h.authService.ExchangeCode(c.Request().Context(), code, getRedirectURI(c))
    if err != nil {
        return h.renderCallback(c, "", "Failed to authenticate: "+err.Error())
    }

    // Set session cookie
    setSessionCookie(c, tokens.AccessToken, tokens.ExpiresIn)

    // Get or create user
    user, err := h.authService.GetOrCreateUser(c.Request().Context(), tokens.AccessToken)
    if err != nil {
        return h.renderCallback(c, "", "Failed to get user info")
    }

    // Redirect to intended destination or default
    redirectURL := getRedirectCookie(c)
    if redirectURL == "" {
        redirectURL = "/workspaces"
    }
    clearRedirectCookie(c)

    return h.renderCallback(c, redirectURL, "")
}

func (h *TemplateHandler) renderCallback(c echo.Context, redirectURL, errorMsg string) error {
    data := map[string]interface{}{
        "Title":       "Signing In",
        "RedirectURL": redirectURL,
        "Error":       errorMsg,
    }
    return h.render(c, "layout/base.html", "auth/callback", data)
}

// LogoutPage renders logout confirmation
func (h *TemplateHandler) LogoutPage(c echo.Context) error {
    data := map[string]interface{}{
        "Title": "Sign Out",
        "User":  getUserFromContext(c),
    }
    return h.render(c, "layout/base.html", "auth/logout", data)
}

// Logout handles the logout action
func (h *TemplateHandler) Logout(c echo.Context) error {
    // Clear session cookie
    clearSessionCookie(c)

    // Optionally: call Keycloak logout endpoint
    // h.authService.Logout(c.Request().Context(), getSessionToken(c))

    // For HTMX requests, return redirect header
    if c.Request().Header.Get("HX-Request") == "true" {
        c.Response().Header().Set("HX-Redirect", "/")
        return c.NoContent(http.StatusOK)
    }

    return c.Redirect(http.StatusFound, "/")
}
```

### 5. Cookie Helpers

```go
// internal/handler/http/auth_cookies.go

const (
    sessionCookieName  = "flowra_session"
    stateCookieName    = "flowra_state"
    redirectCookieName = "flowra_redirect"
)

func setSessionCookie(c echo.Context, token string, expiresIn int) {
    cookie := &http.Cookie{
        Name:     sessionCookieName,
        Value:    token,
        Path:     "/",
        MaxAge:   expiresIn,
        HttpOnly: true,
        Secure:   c.Scheme() == "https",
        SameSite: http.SameSiteLaxMode,
    }
    c.SetCookie(cookie)
}

func getSessionCookie(c echo.Context) string {
    cookie, err := c.Cookie(sessionCookieName)
    if err != nil {
        return ""
    }
    return cookie.Value
}

func clearSessionCookie(c echo.Context) {
    cookie := &http.Cookie{
        Name:     sessionCookieName,
        Value:    "",
        Path:     "/",
        MaxAge:   -1,
        HttpOnly: true,
    }
    c.SetCookie(cookie)
}

func setStateCookie(c echo.Context, state string) {
    cookie := &http.Cookie{
        Name:     stateCookieName,
        Value:    state,
        Path:     "/",
        MaxAge:   300, // 5 minutes
        HttpOnly: true,
        Secure:   c.Scheme() == "https",
        SameSite: http.SameSiteLaxMode,
    }
    c.SetCookie(cookie)
}

func getStateCookie(c echo.Context) string {
    cookie, err := c.Cookie(stateCookieName)
    if err != nil {
        return ""
    }
    return cookie.Value
}

func generateState() string {
    b := make([]byte, 16)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}

func setRedirectCookie(c echo.Context, url string) {
    cookie := &http.Cookie{
        Name:     redirectCookieName,
        Value:    url,
        Path:     "/",
        MaxAge:   300,
        HttpOnly: true,
    }
    c.SetCookie(cookie)
}

func getRedirectCookie(c echo.Context) string {
    cookie, err := c.Cookie(redirectCookieName)
    if err != nil {
        return ""
    }
    return cookie.Value
}

func clearRedirectCookie(c echo.Context) {
    cookie := &http.Cookie{
        Name:   redirectCookieName,
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    }
    c.SetCookie(cookie)
}
```

### 6. Routes Registration

```go
// Add to RegisterRoutes in template_handler.go

func (h *TemplateHandler) RegisterRoutes(e *echo.Echo) {
    // ... existing routes ...

    // Auth pages
    e.GET("/login", h.LoginPage)
    e.GET("/auth/callback", h.AuthCallback)
    e.GET("/logout", h.LogoutPage)
    e.POST("/auth/logout", h.Logout)
}
```

---

## Auth Middleware Integration

```go
// Middleware to check authentication for protected pages
func (h *TemplateHandler) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := getSessionCookie(c)
        if token == "" {
            // Save intended destination
            setRedirectCookie(c, c.Request().URL.Path)
            return c.Redirect(http.StatusFound, "/login")
        }

        // Validate token
        user, err := h.authService.ValidateToken(c.Request().Context(), token)
        if err != nil {
            clearSessionCookie(c)
            setRedirectCookie(c, c.Request().URL.Path)
            return c.Redirect(http.StatusFound, "/login?error=session_expired")
        }

        // Set user in context
        c.Set("user", user)

        return next(c)
    }
}
```

---

## Чеклист

### Templates
- [x] `auth/login.html` - Login page с SSO button
- [x] `auth/callback.html` - OAuth callback processing
- [x] `auth/logout.html` - Logout confirmation

### Handlers
- [x] `LoginPage` - отображает login page
- [x] `AuthCallback` - обрабатывает OAuth callback
- [x] `LogoutPage` - отображает logout confirmation
- [x] `Logout` - выполняет logout

### Cookies
- [x] Session cookie management
- [x] State cookie for CSRF protection
- [x] Redirect cookie for return URL

### Integration
- [x] Keycloak auth URL генерируется корректно (mock mode)
- [x] OAuth callback обрабатывается
- [x] Session создаётся при успешном login
- [x] Logout очищает session
- [x] Protected routes редиректят на login

---

## Критерии приёмки

- [x] `/login` отображает страницу входа
- [x] Click на "Sign in with SSO" редиректит на Keycloak (mock mode)
- [x] После успешной авторизации пользователь попадает на `/workspaces`
- [x] `/logout` отображает подтверждение
- [x] После logout session удалён
- [x] Попытка доступа к protected route редиректит на login
- [x] После login пользователь возвращается на исходную страницу

---

## Зависимости

### Входящие
- [01-base-infrastructure.md](01-base-infrastructure.md) - base layout ✅
- Auth API endpoint (`/api/v1/auth/*`)
- Keycloak configured

### Исходящие
- [03-workspace-pages.md](03-workspace-pages.md) - требует аутентификации
- Все protected pages используют `RequireAuth` middleware

---

## Безопасность

### CSRF Protection
- State parameter в OAuth flow
- SameSite cookie attribute

### Session Security
- HttpOnly cookies
- Secure flag в production
- Session expiration

### XSS Prevention
- Все данные экранируются в templates
- Content-Security-Policy header (optional)

---

## Заметки

- Keycloak URL конфигурируется через environment variables
- При ошибке OAuth пользователь видит понятное сообщение
- Session token хранится в HttpOnly cookie, не в localStorage
- HTMX logout использует `hx-redirect` для SPA-like experience

---

*Создано: 2026-01-05*
