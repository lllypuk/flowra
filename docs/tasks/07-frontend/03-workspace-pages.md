# 03: Workspace Pages

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
**Период:** 6-9 февраля
**Зависит от:** [02-auth-pages.md](02-auth-pages.md)

---

## Описание

Реализовать страницы управления workspaces: список workspaces пользователя, создание нового workspace, просмотр workspace с sidebar, управление участниками.

---

## Файлы

### Templates

```
web/templates/workspace/
├── list.html           (~100 LOC) - Workspace list (cards)
├── create.html         (~60 LOC) - Create workspace modal/form
├── view.html           (~120 LOC) - Workspace dashboard with sidebar
├── members.html        (~80 LOC) - Member management
└── settings.html       (~70 LOC) - Workspace settings

web/templates/components/
├── workspace_card.html (~40 LOC) - Workspace card component
└── member_row.html     (~30 LOC) - Member list row
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+300 LOC) - Workspace page handlers
```

---

## Детали реализации

### 1. Workspace List (list.html)

```html
{{define "workspace/list"}}
<div class="workspace-list-page">
    <header class="page-header">
        <hgroup>
            <h1>Your Workspaces</h1>
            <p>Select a workspace to get started</p>
        </hgroup>

        <button hx-get="/partials/workspace/create-form"
                hx-target="#modal-container"
                hx-swap="innerHTML">
            + New Workspace
        </button>
    </header>

    <div id="workspace-list"
         class="workspace-grid"
         hx-get="/partials/workspaces"
         hx-trigger="load"
         hx-swap="innerHTML">
        {{template "loading" (dict "ID" "workspace-loading")}}
    </div>

    <!-- Modal container -->
    <div id="modal-container"></div>
</div>

<style>
.workspace-list-page .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 2rem;
}

.workspace-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.5rem;
}
</style>
{{end}}

{{define "workspace/list-partial"}}
{{if .Workspaces}}
    {{range .Workspaces}}
        {{template "workspace_card" .}}
    {{end}}
{{else}}
    {{template "empty" (dict
        "Title" "No workspaces yet"
        "Description" "Create your first workspace to get started"
        "Action" "Create Workspace"
        "ActionURL" "/partials/workspace/create-form"
    )}}
{{end}}
{{end}}
```

### 2. Workspace Card (workspace_card.html)

```html
{{define "workspace_card"}}
<article class="workspace-card"
         hx-get="/workspaces/{{.ID}}"
         hx-push-url="true"
         hx-target="body"
         style="cursor: pointer;">
    <header>
        <hgroup>
            <h3>{{.Name}}</h3>
            {{if .Description}}
            <p>{{.Description | truncate 100}}</p>
            {{end}}
        </hgroup>
    </header>

    <footer>
        <div class="workspace-meta">
            <span title="Members">
                <svg><!-- user icon --></svg>
                {{.MemberCount}} members
            </span>
            {{if gt .UnreadCount 0}}
            <span class="badge" title="Unread messages">
                {{.UnreadCount}}
            </span>
            {{end}}
        </div>
        <small class="text-muted">
            Created {{.CreatedAt | timeAgo}}
        </small>
    </footer>
</article>
{{end}}
```

### 3. Create Workspace Form (create.html)

```html
{{define "workspace/create-form"}}
<dialog open id="create-workspace-modal">
    <article>
        <header>
            <button aria-label="Close"
                    rel="prev"
                    onclick="this.closest('dialog').remove()">
            </button>
            <h3>Create Workspace</h3>
        </header>

        <form hx-post="/api/v1/workspaces"
              hx-target="#workspace-list"
              hx-swap="afterbegin"
              hx-on::after-request="if(event.detail.successful) this.closest('dialog').remove()">

            <label for="name">
                Workspace Name
                <input type="text"
                       id="name"
                       name="name"
                       placeholder="e.g., Engineering Team"
                       required
                       minlength="3"
                       maxlength="100"
                       autofocus>
            </label>

            <label for="description">
                Description (optional)
                <textarea id="description"
                          name="description"
                          placeholder="What is this workspace for?"
                          maxlength="500"
                          rows="3"></textarea>
            </label>

            <footer>
                <button type="button"
                        class="secondary"
                        onclick="this.closest('dialog').remove()">
                    Cancel
                </button>
                <button type="submit">
                    Create Workspace
                </button>
            </footer>
        </form>
    </article>
</dialog>
{{end}}
```

### 4. Workspace View (view.html)

```html
{{define "workspace/view"}}
<div class="workspace-layout">
    <!-- Sidebar -->
    <aside class="workspace-sidebar">
        <header>
            <h2>{{.Workspace.Name}}</h2>
            {{if eq .UserRole "owner"}}
            <a href="/workspaces/{{.Workspace.ID}}/settings"
               title="Settings">
                <svg><!-- settings icon --></svg>
            </a>
            {{end}}
        </header>

        <nav>
            <ul>
                <li>
                    <a href="/workspaces/{{.Workspace.ID}}/chats"
                       {{if eq .ActiveTab "chats"}}aria-current="page"{{end}}>
                        Chats
                        {{if gt .UnreadChats 0}}
                        <span class="badge">{{.UnreadChats}}</span>
                        {{end}}
                    </a>
                </li>
                <li>
                    <a href="/workspaces/{{.Workspace.ID}}/board"
                       {{if eq .ActiveTab "board"}}aria-current="page"{{end}}>
                        Board
                    </a>
                </li>
                <li>
                    <a href="/workspaces/{{.Workspace.ID}}/members"
                       {{if eq .ActiveTab "members"}}aria-current="page"{{end}}>
                        Members
                        <small class="text-muted">({{.Workspace.MemberCount}})</small>
                    </a>
                </li>
            </ul>
        </nav>

        <!-- Quick actions -->
        <div class="sidebar-actions">
            <button hx-get="/partials/chat/create-form?workspace_id={{.Workspace.ID}}"
                    hx-target="#modal-container"
                    hx-swap="innerHTML"
                    class="outline">
                + New Chat
            </button>
        </div>
    </aside>

    <!-- Main content -->
    <main class="workspace-main">
        {{template "content" .}}
    </main>

    <!-- Modal container -->
    <div id="modal-container"></div>
</div>

<style>
.workspace-layout {
    display: grid;
    grid-template-columns: 250px 1fr;
    min-height: calc(100vh - 60px);
}

.workspace-sidebar {
    background: var(--card-background-color);
    border-right: 1px solid var(--muted-border-color);
    padding: 1rem;
}

.workspace-sidebar header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
}

.workspace-sidebar nav ul {
    list-style: none;
    padding: 0;
    margin: 0;
}

.workspace-sidebar nav a {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0.75rem;
    border-radius: 4px;
    text-decoration: none;
}

.workspace-sidebar nav a[aria-current="page"] {
    background: var(--primary-focus);
}

.sidebar-actions {
    margin-top: 2rem;
}

.workspace-main {
    padding: 1.5rem;
    overflow-y: auto;
}

@media (max-width: 768px) {
    .workspace-layout {
        grid-template-columns: 1fr;
    }

    .workspace-sidebar {
        display: none;
    }
}
</style>
{{end}}
```

### 5. Members Page (members.html)

```html
{{define "workspace/members"}}
<div class="members-page">
    <header class="page-header">
        <h2>Members</h2>
        {{if or (eq .UserRole "owner") (eq .UserRole "admin")}}
        <button hx-get="/partials/workspace/{{.Workspace.ID}}/invite-form"
                hx-target="#modal-container"
                hx-swap="innerHTML">
            + Invite Member
        </button>
        {{end}}
    </header>

    <div id="members-list"
         hx-get="/partials/workspace/{{.Workspace.ID}}/members"
         hx-trigger="load"
         hx-swap="innerHTML">
        {{template "loading" (dict "ID" "members-loading")}}
    </div>
</div>
{{end}}

{{define "workspace/members-partial"}}
<table role="grid">
    <thead>
        <tr>
            <th>User</th>
            <th>Role</th>
            <th>Joined</th>
            {{if or (eq $.UserRole "owner") (eq $.UserRole "admin")}}
            <th>Actions</th>
            {{end}}
        </tr>
    </thead>
    <tbody>
        {{range .Members}}
        {{template "member_row" (dict "Member" . "WorkspaceID" $.Workspace.ID "UserRole" $.UserRole "CurrentUserID" $.CurrentUserID)}}
        {{end}}
    </tbody>
</table>
{{end}}
```

### 6. Member Row (member_row.html)

```html
{{define "member_row"}}
<tr id="member-{{.Member.UserID}}">
    <td>
        <div class="member-info">
            {{if .Member.AvatarURL}}
            <img src="{{.Member.AvatarURL}}"
                 alt="{{.Member.Username}}"
                 class="avatar">
            {{else}}
            <div class="avatar avatar-placeholder">
                {{slice .Member.Username 0 1 | upper}}
            </div>
            {{end}}
            <div>
                <strong>{{.Member.DisplayName}}</strong>
                <small class="text-muted">@{{.Member.Username}}</small>
            </div>
        </div>
    </td>
    <td>
        {{if and (or (eq $.UserRole "owner")) (ne .Member.Role "owner") (ne .Member.UserID $.CurrentUserID)}}
        <select hx-put="/api/v1/workspaces/{{$.WorkspaceID}}/members/{{.Member.UserID}}/role"
                hx-target="#member-{{.Member.UserID}}"
                hx-swap="outerHTML"
                name="role">
            <option value="admin" {{if eq .Member.Role "admin"}}selected{{end}}>Admin</option>
            <option value="member" {{if eq .Member.Role "member"}}selected{{end}}>Member</option>
        </select>
        {{else}}
        <span class="role-badge role-{{.Member.Role}}">
            {{.Member.Role | title}}
        </span>
        {{end}}
    </td>
    <td>
        <small>{{.Member.JoinedAt | formatDate}}</small>
    </td>
    {{if or (eq $.UserRole "owner") (eq $.UserRole "admin")}}
    <td>
        {{if and (ne .Member.Role "owner") (ne .Member.UserID $.CurrentUserID)}}
        <button hx-delete="/api/v1/workspaces/{{$.WorkspaceID}}/members/{{.Member.UserID}}"
                hx-target="#member-{{.Member.UserID}}"
                hx-swap="outerHTML swap:1s"
                hx-confirm="Remove {{.Member.Username}} from workspace?"
                class="outline secondary small">
            Remove
        </button>
        {{end}}
    </td>
    {{end}}
</tr>
{{end}}
```

### 7. Handler Implementation

```go
// Workspace list page
func (h *TemplateHandler) WorkspaceList(c echo.Context) error {
    data := map[string]interface{}{
        "Title": "Workspaces",
        "User":  getUserFromContext(c),
    }
    return h.render(c, "layout/base.html", "workspace/list", data)
}

// Workspace list partial (HTMX)
func (h *TemplateHandler) WorkspaceListPartial(c echo.Context) error {
    user := getUserFromContext(c)

    workspaces, err := h.workspaceService.ListByUser(c.Request().Context(), user.ID)
    if err != nil {
        return h.renderError(c, err)
    }

    data := map[string]interface{}{
        "Workspaces": workspaces,
    }
    return h.renderPartial(c, "workspace/list-partial", data)
}

// Workspace view page
func (h *TemplateHandler) WorkspaceView(c echo.Context) error {
    workspaceID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return h.renderNotFound(c)
    }

    user := getUserFromContext(c)

    workspace, err := h.workspaceService.GetByID(c.Request().Context(), workspaceID)
    if err != nil {
        return h.renderNotFound(c)
    }

    member, err := h.memberService.GetMember(c.Request().Context(), workspaceID, user.ID)
    if err != nil {
        return h.renderForbidden(c)
    }

    data := map[string]interface{}{
        "Title":       workspace.Name,
        "User":        user,
        "Workspace":   workspace,
        "UserRole":    member.Role,
        "ActiveTab":   c.QueryParam("tab"),
        "UnreadChats": 0, // TODO: calculate
    }
    return h.render(c, "layout/base.html", "workspace/view", data)
}

// Create workspace form partial
func (h *TemplateHandler) WorkspaceCreateForm(c echo.Context) error {
    return h.renderPartial(c, "workspace/create-form", nil)
}

// Members page
func (h *TemplateHandler) WorkspaceMembers(c echo.Context) error {
    workspaceID, _ := uuid.Parse(c.Param("id"))
    user := getUserFromContext(c)

    workspace, _ := h.workspaceService.GetByID(c.Request().Context(), workspaceID)
    member, _ := h.memberService.GetMember(c.Request().Context(), workspaceID, user.ID)

    data := map[string]interface{}{
        "Title":         "Members - " + workspace.Name,
        "User":          user,
        "Workspace":     workspace,
        "UserRole":      member.Role,
        "CurrentUserID": user.ID,
        "ActiveTab":     "members",
    }
    return h.render(c, "layout/base.html", "workspace/members", data)
}

// Members list partial
func (h *TemplateHandler) WorkspaceMembersPartial(c echo.Context) error {
    workspaceID, _ := uuid.Parse(c.Param("id"))
    user := getUserFromContext(c)

    members, err := h.memberService.ListMembers(c.Request().Context(), workspaceID)
    if err != nil {
        return h.renderError(c, err)
    }

    workspace, _ := h.workspaceService.GetByID(c.Request().Context(), workspaceID)
    member, _ := h.memberService.GetMember(c.Request().Context(), workspaceID, user.ID)

    data := map[string]interface{}{
        "Members":       members,
        "Workspace":     workspace,
        "UserRole":      member.Role,
        "CurrentUserID": user.ID,
    }
    return h.renderPartial(c, "workspace/members-partial", data)
}
```

---

## Routes

```go
// Workspace pages
workspace := e.Group("/workspaces", h.RequireAuth)
workspace.GET("", h.WorkspaceList)
workspace.GET("/:id", h.WorkspaceView)
workspace.GET("/:id/members", h.WorkspaceMembers)
workspace.GET("/:id/settings", h.WorkspaceSettings)

// Workspace partials
partials.GET("/workspaces", h.WorkspaceListPartial)
partials.GET("/workspace/create-form", h.WorkspaceCreateForm)
partials.GET("/workspace/:id/members", h.WorkspaceMembersPartial)
partials.GET("/workspace/:id/invite-form", h.WorkspaceInviteForm)
```

---

## Чеклист

### Templates
- [ ] `workspace/list.html` - список workspaces
- [ ] `workspace/create.html` - форма создания
- [ ] `workspace/view.html` - workspace dashboard
- [ ] `workspace/members.html` - управление участниками
- [ ] `workspace/settings.html` - настройки
- [ ] `components/workspace_card.html` - карточка workspace
- [ ] `components/member_row.html` - строка участника

### Handlers
- [ ] `WorkspaceList` - страница списка
- [ ] `WorkspaceListPartial` - HTMX partial
- [ ] `WorkspaceView` - страница workspace
- [ ] `WorkspaceCreateForm` - форма создания
- [ ] `WorkspaceMembers` - страница участников
- [ ] `WorkspaceMembersPartial` - HTMX partial
- [ ] `WorkspaceSettings` - страница настроек

### Features
- [ ] Список workspaces загружается
- [ ] Создание workspace работает
- [ ] Навигация в workspace работает
- [ ] Управление участниками работает
- [ ] Изменение роли через inline select
- [ ] Удаление участника с подтверждением

---

## Критерии приёмки

- [ ] `/workspaces` отображает список workspaces пользователя
- [ ] Создание workspace добавляет карточку в список
- [ ] Click на карточку открывает workspace
- [ ] Sidebar navigation работает
- [ ] Members page показывает участников
- [ ] Owner может менять роли участников
- [ ] Удаление участника работает
- [ ] Mobile responsive layout

---

## Зависимости

### Входящие
- [02-auth-pages.md](02-auth-pages.md) - authentication ✅
- Workspace API endpoints

### Исходящие
- [04-chat-ui.md](04-chat-ui.md) - workspace context
- [05-kanban-board.md](05-kanban-board.md) - workspace context

---

*Создано: 2026-01-05*
