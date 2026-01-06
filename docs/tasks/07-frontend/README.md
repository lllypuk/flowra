# HTMX Frontend

**Цель:** Реализовать минимальный работающий UI на HTMX + Pico CSS
**Статус:** 🔄 В процессе

---

## Обзор

Этот каталог содержит детализированные задачи по разработке фронтенда для Flowra. Frontend построен на принципах progressive enhancement с использованием HTMX для динамики и Pico CSS для стилизации.

### Предварительные требования (Backend)
- ✅ **AuthService** — OAuth2 flow с Keycloak (Login, Logout, RefreshToken)
- ✅ **WorkspaceService** — CRUD workspaces (CreateWorkspace, GetWorkspace, ListUserWorkspaces, UpdateWorkspace, DeleteWorkspace)
- ✅ **MemberService** — управление участниками (AddMember, RemoveMember, UpdateMemberRole, ListMembers, IsOwner)
- ✅ **ChatService** — управление чатами с event sourcing (CreateChat, GetChat, ListChats, RenameChat, DeleteChat, AddParticipant, RemoveParticipant)
- ✅ **Application Layer** — 45+ use cases (chat, message, task, user, workspace, notification)
- ✅ E2E tests проходят

### Tech Stack

| Технология | Версия | Назначение |
|------------|--------|------------|
| **HTMX** | 2.0+ | AJAX без JavaScript |
| **htmx-ext-ws** | 2.0 | WebSocket extension |
| **Pico CSS** | v2 | Classless CSS framework |
| **Go html/template** | stdlib | Server-side rendering |

---

## Структура задач

### Фаза 1: Foundation

| № | Задача | Файл | Приоритет | Статус |
|---|--------|------|-----------|--------|
| 01 | Base Infrastructure | [01-base-infrastructure.md](01-base-infrastructure.md) | 🔴 Critical | ✅ |
| 02 | Auth Pages | [02-auth-pages.md](02-auth-pages.md) | 🔴 Critical | ✅ |

### Фаза 2: Core Features

| № | Задача | Файл | Приоритет | Статус |
|---|--------|------|-----------|--------|
| 03 | Workspace Pages | [03-workspace-pages.md](03-workspace-pages.md) | 🔴 Critical | ✅ |
| 04 | Chat UI | [04-chat-ui.md](04-chat-ui.md) | 🔴 Critical | ⏳ |
| 05 | Kanban Board | [05-kanban-board.md](05-kanban-board.md) | 🟡 High | ⏳ |

### Фаза 3: Task Management & Polish

| № | Задача | Файл | Приоритет | Статус |
|---|--------|------|-----------|--------|
| 06 | Task Details | [06-task-details.md](06-task-details.md) | 🟡 High | ⏳ |
| 07 | Notifications | [07-notifications.md](07-notifications.md) | 🟢 Medium | ⏳ |
| 08 | Polish & Testing | [08-polish.md](08-polish.md) | 🟢 Medium | ⏳ |

---

## Целевая структура файлов

```
web/
├── templates/
│   ├── layout/
│   │   ├── base.html           # HTML5 skeleton + HTMX/Pico
│   │   ├── navbar.html         # Navigation component
│   │   └── footer.html         # Footer component
│   ├── auth/
│   │   ├── login.html          # Login page
│   │   └── callback.html       # OAuth callback
│   ├── workspace/
│   │   ├── list.html           # Workspace list
│   │   ├── create.html         # Create form
│   │   ├── view.html           # Workspace dashboard
│   │   └── members.html        # Member management
│   ├── chat/
│   │   ├── layout.html         # 3-column chat layout
│   │   ├── list.html           # Chat list sidebar
│   │   ├── view.html           # Messages view
│   │   └── create.html         # Create chat form
│   ├── board/
│   │   ├── index.html          # Kanban board
│   │   ├── column.html         # Status column
│   │   └── card.html           # Task card
│   ├── task/
│   │   ├── sidebar.html        # Task details sidebar
│   │   └── form.html           # Task edit form
│   ├── notification/
│   │   ├── dropdown.html       # Navbar dropdown
│   │   └── item.html           # Notification item
│   └── components/
│       ├── message.html        # Chat message
│       ├── message_form.html   # Message input
│       ├── flash.html          # Flash messages
│       ├── loading.html        # Loading indicator
│       └── empty.html          # Empty state
├── static/
│   ├── css/
│   │   └── custom.css          # Custom styles (~200 LOC)
│   └── js/
│       └── app.js              # Utilities (~150 LOC)
└── embed.go                    # go:embed for static files
```

---

## Зависимости между задачами

```
[01 Base Infrastructure]
         │
         ├──> [02 Auth Pages]
         │           │
         │           v
         └──> [03 Workspace Pages]
                     │
         ┌──────────┴──────────┐
         v                     v
[04 Chat UI]          [05 Kanban Board]
         │                     │
         v                     v
[07 Notifications]    [06 Task Details]
         │                     │
         └─────────┬───────────┘
                   v
          [08 Polish & Testing]
```

---

## Ключевые паттерны HTMX

### 1. AJAX Requests

```html
<!-- GET with target -->
<button hx-get="/workspaces"
        hx-target="#workspace-list"
        hx-swap="innerHTML">
    Refresh
</button>

<!-- POST form -->
<form hx-post="/workspaces"
      hx-target="#workspace-list"
      hx-swap="afterbegin">
    <input name="name" required>
    <button type="submit">Create</button>
</form>
```

### 2. Inline Editing

```html
<select hx-put="/tasks/{{.ID}}/status"
        hx-trigger="change"
        name="status">
    <option value="todo">To Do</option>
    <option value="done">Done</option>
</select>
```

### 3. WebSocket

```html
<div hx-ext="ws" ws-connect="/ws?token={{.Token}}">
    <div id="messages" ws-swap="beforeend">
        <!-- Messages appended here -->
    </div>
    <form ws-send>
        <textarea name="content"></textarea>
    </form>
</div>
```

### 4. Loading States

```html
<button hx-get="/data" hx-indicator="#spinner">
    Load
    <span id="spinner" class="htmx-indicator">Loading...</span>
</button>
```

---

## Handler Architecture

### Template Handler

```go
// internal/handler/http/template_handler.go

type TemplateHandler struct {
    templates    *template.Template
    chatService  ChatService
    taskService  TaskService
    // ... other services
}

func NewTemplateHandler(templates *template.Template, ...) *TemplateHandler

// Page handlers (full page render)
func (h *TemplateHandler) Home(c echo.Context) error
func (h *TemplateHandler) LoginPage(c echo.Context) error
func (h *TemplateHandler) WorkspaceList(c echo.Context) error
func (h *TemplateHandler) ChatView(c echo.Context) error
func (h *TemplateHandler) BoardView(c echo.Context) error

// Partial handlers (HTMX fragments)
func (h *TemplateHandler) ChatListPartial(c echo.Context) error
func (h *TemplateHandler) MessagesPartial(c echo.Context) error
func (h *TemplateHandler) TaskCardPartial(c echo.Context) error
```

### Route Groups

```go
// HTML routes (server-side rendering)
html := e.Group("")
html.Use(middleware.HTMLContentType())

html.GET("/", h.Home)
html.GET("/login", h.LoginPage)
html.GET("/workspaces", h.WorkspaceList)
html.GET("/workspaces/:id", h.WorkspaceView)
html.GET("/workspaces/:id/chats/:chat_id", h.ChatView)
html.GET("/workspaces/:id/board", h.BoardView)

// Partials (HTMX fragments)
partials := e.Group("/partials")
partials.GET("/chats", h.ChatListPartial)
partials.GET("/messages/:chat_id", h.MessagesPartial)
partials.GET("/tasks/:id", h.TaskCardPartial)
```

---

## Метрики успеха

### Функциональные требования

- [ ] Пользователь может войти через Keycloak
- [ ] CRUD операции с workspaces через UI
- [ ] Real-time чат с WebSocket
- [ ] Kanban board с drag-n-drop
- [ ] Inline редактирование задач
- [ ] Real-time notifications

### UI/UX Targets

- [ ] Время загрузки страницы < 500ms
- [ ] Работает без JavaScript (degraded mode)
- [ ] Mobile-friendly (responsive)
- [ ] Accessibility: keyboard navigation

### Code Quality

- [ ] Template coverage: 100% страниц
- [ ] E2E tests для всех flows
- [ ] No JavaScript frameworks (только HTMX + vanilla)
- [ ] CSS < 300 LOC

---

## Легенда статусов

- ⏳ — Не начато
- 🔄 — В процессе
- ✅ — Завершено
- ❌ — Заблокировано
- ⏸️ — Приостановлено

---

## Ресурсы

### Документация
- [HTMX Reference](https://htmx.org/reference/)
- [Pico CSS Docs](https://picocss.com/docs/)
- [Go html/template](https://pkg.go.dev/html/template)

### Примеры
- [HTMX Examples](https://htmx.org/examples/)
- [htmx-ext-ws](https://htmx.org/extensions/ws/)

### Внутренние документы
- [API Contracts](../../06-api-contracts.md)
- [Phase 4 Plan](../../roadmap/phase-4/task-4-minimal-frontend.md)

---

*Обновлено: 2026-01-06*
