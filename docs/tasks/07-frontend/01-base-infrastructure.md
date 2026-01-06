# 01: Base Infrastructure

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
**Зависит от:** Backend API готов

---

## Описание

Создать базовую инфраструктуру для HTMX фронтенда: layout templates, CSS framework подключение, JavaScript utilities, и template handler в Go.

---

## Файлы

### Templates

```
web/templates/
├── layout/
│   ├── base.html           (~80 LOC) - HTML5 skeleton
│   ├── navbar.html         (~60 LOC) - Navigation bar
│   └── footer.html         (~20 LOC) - Footer
└── components/
    ├── flash.html          (~30 LOC) - Flash messages
    ├── loading.html        (~15 LOC) - HTMX loading indicator
    └── empty.html          (~20 LOC) - Empty state placeholder
```

### Static Assets

```
web/static/
├── css/
│   └── custom.css          (~100 LOC) - Base custom styles
└── js/
    └── app.js              (~50 LOC) - Base utilities
```

### Go Code

```
web/
└── embed.go                (~20 LOC) - go:embed for static files

internal/handler/http/
├── template_handler.go     (~200 LOC) - Template rendering handler
└── template_funcs.go       (~100 LOC) - Custom template functions
```

---

## Детали реализации

### 1. Base Layout (base.html)

```html
<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Flowra</title>

    <!-- Pico CSS -->
    <link rel="stylesheet"
          href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">

    <!-- Custom CSS -->
    <link rel="stylesheet" href="/static/css/custom.css">

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.0"
            integrity="sha384-..."
            crossorigin="anonymous"></script>

    <!-- HTMX WebSocket Extension -->
    <script src="https://unpkg.com/htmx-ext-ws@2.0.0/ws.js"></script>
</head>
<body>
    {{template "navbar" .}}

    <main class="container">
        {{template "flash" .}}
        {{template "content" .}}
    </main>

    {{template "footer" .}}

    <script src="/static/js/app.js"></script>
</body>
</html>
```

### 2. Navbar (navbar.html)

```html
{{define "navbar"}}
<nav class="container-fluid">
    <ul>
        <li><a href="/" class="contrast"><strong>Flowra</strong></a></li>
    </ul>

    {{if .User}}
    <ul>
        <li>
            <a href="/workspaces">Workspaces</a>
        </li>
        <li>
            <!-- Notifications dropdown -->
            <details role="list" dir="rtl">
                <summary aria-haspopup="listbox" role="link">
                    <span id="notification-badge"
                          hx-get="/partials/notifications/count"
                          hx-trigger="load, every 30s"
                          hx-swap="innerHTML">
                    </span>
                    Notifications
                </summary>
                <ul role="listbox"
                    hx-get="/partials/notifications"
                    hx-trigger="toggle"
                    hx-swap="innerHTML">
                    <li>Loading...</li>
                </ul>
            </details>
        </li>
        <li>
            <details role="list" dir="rtl">
                <summary aria-haspopup="listbox" role="link">
                    {{.User.Username}}
                </summary>
                <ul role="listbox">
                    <li><a href="/settings">Settings</a></li>
                    <li>
                        <a href="#"
                           hx-post="/auth/logout"
                           hx-redirect="/">Logout</a>
                    </li>
                </ul>
            </details>
        </li>
    </ul>
    {{else}}
    <ul>
        <li><a href="/login" role="button">Login</a></li>
    </ul>
    {{end}}
</nav>
{{end}}
```

### 3. Flash Messages (flash.html)

```html
{{define "flash"}}
{{if .Flash}}
<div id="flash-messages">
    {{range .Flash.Success}}
    <article class="flash flash-success" role="alert">
        <button class="close"
                onclick="this.parentElement.remove()"
                aria-label="Close">&times;</button>
        {{.}}
    </article>
    {{end}}

    {{range .Flash.Error}}
    <article class="flash flash-error" role="alert">
        <button class="close"
                onclick="this.parentElement.remove()"
                aria-label="Close">&times;</button>
        {{.}}
    </article>
    {{end}}

    {{range .Flash.Info}}
    <article class="flash flash-info" role="alert">
        <button class="close"
                onclick="this.parentElement.remove()"
                aria-label="Close">&times;</button>
        {{.}}
    </article>
    {{end}}
</div>
{{end}}
{{end}}
```

### 4. Loading Indicator (loading.html)

```html
{{define "loading"}}
<div class="htmx-indicator" id="{{.ID}}">
    <span aria-busy="true">Loading...</span>
</div>
{{end}}
```

### 5. Custom CSS (custom.css)

```css
/* ===== Variables ===== */
:root {
    --flowra-primary: #0066cc;
    --flowra-success: #10b981;
    --flowra-warning: #f59e0b;
    --flowra-danger: #ef4444;
}

/* ===== Flash Messages ===== */
.flash {
    position: relative;
    padding-right: 2.5rem;
    margin-bottom: 1rem;
}

.flash .close {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1.25rem;
    padding: 0;
    margin: 0;
    width: auto;
}

.flash-success {
    background-color: color-mix(in srgb, var(--flowra-success) 15%, white);
    border-left: 4px solid var(--flowra-success);
}

.flash-error {
    background-color: color-mix(in srgb, var(--flowra-danger) 15%, white);
    border-left: 4px solid var(--flowra-danger);
}

.flash-info {
    background-color: color-mix(in srgb, var(--flowra-primary) 15%, white);
    border-left: 4px solid var(--flowra-primary);
}

/* ===== HTMX Loading ===== */
.htmx-indicator {
    display: none;
}

.htmx-request .htmx-indicator,
.htmx-request.htmx-indicator {
    display: inline-block;
}

/* ===== Utility Classes ===== */
.text-center { text-align: center; }
.text-right { text-align: right; }
.text-muted { color: var(--muted-color); }

.mt-1 { margin-top: 0.5rem; }
.mt-2 { margin-top: 1rem; }
.mb-1 { margin-bottom: 0.5rem; }
.mb-2 { margin-bottom: 1rem; }

.hidden { display: none !important; }
```

### 6. Base JavaScript (app.js)

```javascript
/**
 * Flowra Frontend Utilities
 */

// Auto-hide flash messages after 5 seconds
document.addEventListener('DOMContentLoaded', function() {
    const flashMessages = document.querySelectorAll('.flash');
    flashMessages.forEach(function(flash) {
        setTimeout(function() {
            flash.style.opacity = '0';
            setTimeout(function() {
                flash.remove();
            }, 300);
        }, 5000);
    });
});

// HTMX event handlers
document.body.addEventListener('htmx:beforeSwap', function(evt) {
    // Handle 422 validation errors
    if (evt.detail.xhr.status === 422) {
        evt.detail.shouldSwap = true;
        evt.detail.isError = false;
    }
});

// Handle HTMX errors
document.body.addEventListener('htmx:responseError', function(evt) {
    console.error('HTMX Error:', evt.detail);
    // Could show a toast notification here
});

// Scroll to bottom utility (for chat)
function scrollToBottom(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.scrollTop = element.scrollHeight;
    }
}
```

### 7. Template Handler (template_handler.go)

```go
package http

import (
    "embed"
    "html/template"
    "io/fs"
    "net/http"

    "github.com/labstack/echo/v4"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type TemplateHandler struct {
    templates *template.Template
}

func NewTemplateHandler() (*TemplateHandler, error) {
    // Parse all templates
    tmpl, err := template.New("").
        Funcs(templateFuncs()).
        ParseFS(templateFS, "templates/**/*.html")
    if err != nil {
        return nil, err
    }

    return &TemplateHandler{
        templates: tmpl,
    }, nil
}

// RegisterRoutes registers HTML routes
func (h *TemplateHandler) RegisterRoutes(e *echo.Echo) {
    // Serve static files
    staticSub, _ := fs.Sub(staticFS, "static")
    e.StaticFS("/static", staticSub)

    // HTML pages
    e.GET("/", h.Home)
    e.GET("/login", h.LoginPage)
}

// Home renders the home page
func (h *TemplateHandler) Home(c echo.Context) error {
    data := map[string]interface{}{
        "Title": "Home",
        "User":  getUserFromContext(c),
    }
    return h.render(c, "layout/base.html", "home", data)
}

// LoginPage renders the login page
func (h *TemplateHandler) LoginPage(c echo.Context) error {
    data := map[string]interface{}{
        "Title": "Login",
    }
    return h.render(c, "layout/base.html", "auth/login", data)
}

// render executes a template with layout
func (h *TemplateHandler) render(c echo.Context, layout, content string, data map[string]interface{}) error {
    // Clone template to add content
    tmpl, err := h.templates.Clone()
    if err != nil {
        return err
    }

    // Define content block
    _, err = tmpl.New("content").Parse(
        `{{template "` + content + `" .}}`,
    )
    if err != nil {
        return err
    }

    c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
    return tmpl.ExecuteTemplate(c.Response(), layout, data)
}

func getUserFromContext(c echo.Context) interface{} {
    // Get user from middleware context
    return c.Get("user")
}
```

### 8. Template Functions (template_funcs.go)

```go
package http

import (
    "html/template"
    "time"
)

func templateFuncs() template.FuncMap {
    return template.FuncMap{
        // Time formatting
        "formatTime": func(t time.Time) string {
            return t.Format("15:04")
        },
        "formatDate": func(t time.Time) string {
            return t.Format("Jan 2, 2006")
        },
        "formatDateTime": func(t time.Time) string {
            return t.Format("Jan 2, 2006 15:04")
        },
        "timeAgo": func(t time.Time) string {
            diff := time.Since(t)
            switch {
            case diff < time.Minute:
                return "just now"
            case diff < time.Hour:
                return fmt.Sprintf("%dm ago", int(diff.Minutes()))
            case diff < 24*time.Hour:
                return fmt.Sprintf("%dh ago", int(diff.Hours()))
            default:
                return t.Format("Jan 2")
            }
        },

        // String helpers
        "truncate": func(s string, n int) string {
            if len(s) <= n {
                return s
            }
            return s[:n] + "..."
        },

        // Conditional helpers
        "eq": func(a, b interface{}) bool {
            return a == b
        },
        "ne": func(a, b interface{}) bool {
            return a != b
        },

        // Safe HTML (use with caution)
        "safeHTML": func(s string) template.HTML {
            return template.HTML(s)
        },
    }
}
```

---

## Embed Files (embed.go)

```go
// web/embed.go
package web

import "embed"

//go:embed templates
var TemplatesFS embed.FS

//go:embed static
var StaticFS embed.FS
```

---

## Чеклист

### Templates
- [ ] `layout/base.html` - HTML5 skeleton с HTMX/Pico
- [ ] `layout/navbar.html` - Navigation с user menu
- [ ] `layout/footer.html` - Simple footer
- [ ] `components/flash.html` - Flash messages
- [ ] `components/loading.html` - HTMX loading indicator
- [ ] `components/empty.html` - Empty state

### Static Assets
- [ ] `css/custom.css` - Base styles
- [ ] `js/app.js` - Base utilities

### Go Code
- [ ] `embed.go` - Embed static files
- [ ] `template_handler.go` - Base handler
- [ ] `template_funcs.go` - Template functions

### Integration
- [ ] Templates загружаются без ошибок
- [ ] Static files доступны по `/static/*`
- [ ] Navbar отображается корректно
- [ ] Flash messages работают
- [ ] HTMX подключен и работает
- [ ] Pico CSS применяется

---

## Критерии приёмки

- [ ] `go build` проходит без ошибок
- [ ] `/` возвращает HTML страницу
- [ ] `/static/css/custom.css` доступен
- [ ] HTMX `hx-get` работает
- [ ] Pico CSS стили применяются
- [ ] Navbar адаптивен на mobile
- [ ] Flash messages автоматически скрываются

---

## Зависимости

### Входящие
- Backend API работает
- Echo server настроен

### Исходящие
- [02-auth-pages.md](02-auth-pages.md) - использует base layout
- Все остальные страницы используют base.html

---

## Заметки

- Pico CSS используется как classless framework — минимум собственных классов
- HTMX подключается через CDN для простоты, в production можно включить в bundle
- `go:embed` позволяет собрать все assets в один binary
- Template functions добавляют удобные хелперы для форматирования

---

*Обновлено: 2026-01-06*
