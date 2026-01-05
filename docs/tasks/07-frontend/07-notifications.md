# 07: Notifications

**Приоритет:** 🟢 Medium
**Статус:** ⏳ Не начато
**Период:** 21-23 февраля
**Зависит от:** [04-chat-ui.md](04-chat-ui.md)

---

## Описание

Реализовать систему уведомлений: dropdown в navbar с real-time обновлениями, страница всех уведомлений, mark as read, типы уведомлений (mentions, assignments, status changes).

---

## Файлы

### Templates

```
web/templates/notification/
├── dropdown.html       (~80 LOC) - Navbar dropdown
├── list.html           (~60 LOC) - Full notifications page
├── item.html           (~50 LOC) - Single notification
└── empty.html          (~20 LOC) - Empty state

web/templates/components/
└── notification_badge.html (~15 LOC) - Unread count badge
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+150 LOC) - Notification handlers
```

---

## Notification Types

| Type | Icon | Example |
|------|------|---------|
| `mention` | 💬 | "@alice mentioned you in 'Project Chat'" |
| `assignment` | 👤 | "You were assigned to 'Implement OAuth'" |
| `status_change` | 🔄 | "Task 'Fix Bug' status changed to Done" |
| `comment` | 💭 | "bob replied to 'API Design'" |
| `due_date` | ⏰ | "Task 'Deploy' is due tomorrow" |
| `workspace_invite` | 📨 | "You were invited to 'Engineering Team'" |

---

## Детали реализации

### 1. Notification Badge (notification_badge.html)

```html
{{define "notification_badge"}}
<span id="notification-badge"
      class="notification-badge {{if eq .Count 0}}hidden{{end}}"
      hx-get="/partials/notifications/count"
      hx-trigger="load, every 60s, notification-update from:body"
      hx-swap="outerHTML">
    {{if gt .Count 99}}
        99+
    {{else}}
        {{.Count}}
    {{end}}
</span>

<style>
.notification-badge {
    background: var(--flowra-danger);
    color: white;
    font-size: 0.7rem;
    font-weight: bold;
    padding: 0.15rem 0.4rem;
    border-radius: 10px;
    min-width: 1.2rem;
    text-align: center;
}

.notification-badge.hidden {
    display: none;
}
</style>
{{end}}
```

### 2. Notification Dropdown (dropdown.html)

```html
{{define "notification/dropdown"}}
<details class="notification-dropdown" role="list" dir="rtl">
    <summary aria-haspopup="listbox" role="link">
        <span class="notification-icon">🔔</span>
        {{template "notification_badge" .}}
    </summary>

    <ul role="listbox"
        id="notification-dropdown-list"
        hx-get="/partials/notifications?limit=10"
        hx-trigger="toggle once"
        hx-swap="innerHTML">
        <li class="loading">Loading...</li>
    </ul>
</details>

<style>
.notification-dropdown {
    position: relative;
}

.notification-dropdown summary {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    cursor: pointer;
}

.notification-dropdown ul {
    position: absolute;
    right: 0;
    top: 100%;
    width: 350px;
    max-height: 400px;
    overflow-y: auto;
    background: var(--background-color);
    border: 1px solid var(--muted-border-color);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    padding: 0;
    margin: 0.5rem 0 0 0;
    list-style: none;
    z-index: 1000;
}

.notification-dropdown li.loading {
    padding: 1rem;
    text-align: center;
    color: var(--muted-color);
}
</style>
{{end}}

{{define "notification/dropdown-content"}}
{{if .Notifications}}
    <li class="dropdown-header">
        <span>Notifications</span>
        {{if gt .UnreadCount 0}}
        <button hx-put="/api/v1/notifications/mark-all-read"
                hx-swap="none"
                hx-on::after-request="htmx.trigger(document.body, 'notification-update')"
                class="small outline">
            Mark all read
        </button>
        {{end}}
    </li>

    {{range .Notifications}}
        {{template "notification/dropdown-item" .}}
    {{end}}

    <li class="dropdown-footer">
        <a href="/notifications">View all notifications</a>
    </li>
{{else}}
    <li class="empty-state">
        <span class="empty-icon">🔔</span>
        <p>No notifications</p>
    </li>
{{end}}

<style>
.dropdown-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--muted-border-color);
    font-weight: 600;
}

.dropdown-header button {
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
    margin: 0;
}

.dropdown-footer {
    padding: 0.75rem 1rem;
    border-top: 1px solid var(--muted-border-color);
    text-align: center;
}

.dropdown-footer a {
    font-size: 0.875rem;
}

.empty-state {
    padding: 2rem;
    text-align: center;
    color: var(--muted-color);
}

.empty-icon {
    font-size: 2rem;
    opacity: 0.5;
}
</style>
{{end}}
```

### 3. Dropdown Item (simplified)

```html
{{define "notification/dropdown-item"}}
<li class="notification-item {{if not .ReadAt}}unread{{end}}"
    hx-get="/notifications/{{.ID}}/redirect"
    hx-push-url="true">
    <div class="notification-icon-type">
        {{if eq .Type "mention"}}💬
        {{else if eq .Type "assignment"}}👤
        {{else if eq .Type "status_change"}}🔄
        {{else if eq .Type "comment"}}💭
        {{else if eq .Type "due_date"}}⏰
        {{else if eq .Type "workspace_invite"}}📨
        {{else}}📢
        {{end}}
    </div>
    <div class="notification-content">
        <p class="notification-message">{{.Message}}</p>
        <time class="notification-time">{{.CreatedAt | timeAgo}}</time>
    </div>
    {{if not .ReadAt}}
    <div class="unread-dot"></div>
    {{end}}
</li>

<style>
.notification-item {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    cursor: pointer;
    transition: background 0.2s;
}

.notification-item:hover {
    background: var(--primary-focus);
}

.notification-item.unread {
    background: color-mix(in srgb, var(--primary) 5%, white);
}

.notification-icon-type {
    font-size: 1.25rem;
    flex-shrink: 0;
}

.notification-content {
    flex: 1;
    min-width: 0;
}

.notification-message {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.4;
}

.notification-time {
    font-size: 0.75rem;
    color: var(--muted-color);
}

.unread-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--primary);
    flex-shrink: 0;
}
</style>
{{end}}
```

### 4. Full Notifications Page (list.html)

```html
{{define "notification/list"}}
<div class="notifications-page">
    <header class="page-header">
        <h1>Notifications</h1>

        <div class="header-actions">
            {{if gt .UnreadCount 0}}
            <button hx-put="/api/v1/notifications/mark-all-read"
                    hx-swap="none"
                    hx-on::after-request="location.reload()"
                    class="outline">
                Mark all as read
            </button>
            {{end}}

            <select hx-get="/notifications"
                    hx-target="body"
                    hx-push-url="true"
                    name="filter">
                <option value="" {{if not .Filter}}selected{{end}}>All</option>
                <option value="unread" {{if eq .Filter "unread"}}selected{{end}}>Unread</option>
                <option value="mention" {{if eq .Filter "mention"}}selected{{end}}>Mentions</option>
                <option value="assignment" {{if eq .Filter "assignment"}}selected{{end}}>Assignments</option>
            </select>
        </div>
    </header>

    <div id="notifications-list"
         hx-get="/partials/notifications/list"
         hx-trigger="load"
         hx-swap="innerHTML"
         hx-vals='{"filter": "{{.Filter}}"}'>
        {{template "loading" (dict "ID" "notifications-loading")}}
    </div>
</div>

<style>
.notifications-page {
    max-width: 800px;
    margin: 0 auto;
    padding: 1rem;
}

.page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
}

.header-actions {
    display: flex;
    gap: 0.5rem;
}

.header-actions select {
    margin-bottom: 0;
    width: auto;
}
</style>
{{end}}
```

### 5. Notification List Partial

```html
{{define "notification/list-partial"}}
{{if .Notifications}}
<div class="notification-list">
    {{range .Notifications}}
        {{template "notification/item" .}}
    {{end}}

    {{if .HasMore}}
    <button hx-get="/partials/notifications/list"
            hx-target="this"
            hx-swap="outerHTML"
            hx-vals='{"cursor": "{{.NextCursor}}", "filter": "{{$.Filter}}"}'
            class="load-more outline">
        Load more
    </button>
    {{end}}
</div>
{{else}}
<div class="empty-state">
    <span class="empty-icon">🔔</span>
    <h3>No notifications</h3>
    <p class="text-muted">
        {{if .Filter}}
            No {{.Filter}} notifications found.
        {{else}}
            You're all caught up!
        {{end}}
    </p>
</div>
{{end}}
{{end}}
```

### 6. Full Notification Item (item.html)

```html
{{define "notification/item"}}
<article class="notification-card {{if not .ReadAt}}unread{{end}}"
         id="notification-{{.ID}}">
    <div class="notification-icon-type">
        {{if eq .Type "mention"}}💬
        {{else if eq .Type "assignment"}}👤
        {{else if eq .Type "status_change"}}🔄
        {{else if eq .Type "comment"}}💭
        {{else if eq .Type "due_date"}}⏰
        {{else if eq .Type "workspace_invite"}}📨
        {{else}}📢
        {{end}}
    </div>

    <div class="notification-body">
        <header>
            <strong>{{.Title}}</strong>
            <time>{{.CreatedAt | timeAgo}}</time>
        </header>
        <p>{{.Message}}</p>
        {{if .ResourceType}}
        <a href="{{.ResourceURL}}"
           class="notification-link"
           hx-get="{{.ResourceURL}}"
           hx-push-url="true"
           hx-target="body">
            View {{.ResourceType}}
        </a>
        {{end}}
    </div>

    <div class="notification-actions">
        {{if not .ReadAt}}
        <button hx-put="/api/v1/notifications/{{.ID}}/read"
                hx-target="#notification-{{.ID}}"
                hx-swap="outerHTML"
                class="small outline"
                title="Mark as read">
            ✓
        </button>
        {{end}}
        <button hx-delete="/api/v1/notifications/{{.ID}}"
                hx-target="#notification-{{.ID}}"
                hx-swap="outerHTML swap:0.3s"
                class="small outline secondary"
                title="Delete">
            ✕
        </button>
    </div>
</article>

<style>
.notification-card {
    display: flex;
    gap: 1rem;
    padding: 1rem;
    border-radius: 8px;
    margin-bottom: 0.5rem;
    background: var(--card-background-color);
    transition: background 0.2s;
}

.notification-card.unread {
    background: color-mix(in srgb, var(--primary) 8%, var(--card-background-color));
    border-left: 3px solid var(--primary);
}

.notification-card:hover {
    background: var(--primary-focus);
}

.notification-body {
    flex: 1;
}

.notification-body header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 0.25rem;
}

.notification-body time {
    font-size: 0.75rem;
    color: var(--muted-color);
}

.notification-body p {
    margin: 0 0 0.5rem 0;
    font-size: 0.9rem;
}

.notification-link {
    font-size: 0.85rem;
}

.notification-actions {
    display: flex;
    gap: 0.25rem;
    opacity: 0;
    transition: opacity 0.2s;
}

.notification-card:hover .notification-actions {
    opacity: 1;
}

.notification-actions button {
    padding: 0.25rem 0.5rem;
}
</style>
{{end}}
```

### 7. WebSocket Integration

```javascript
// In app.js or notifications.js

// Handle real-time notifications
document.body.addEventListener('notification.new', function(evt) {
    const notification = evt.detail;

    // Update badge
    htmx.trigger(document.body, 'notification-update');

    // Show toast notification
    showToast(notification.message, notification.type);

    // If dropdown is open, prepend new notification
    const dropdownList = document.getElementById('notification-dropdown-list');
    if (dropdownList && dropdownList.closest('details').open) {
        htmx.ajax('GET', '/partials/notifications?limit=1', {
            target: '#notification-dropdown-list',
            swap: 'afterbegin'
        });
    }
});

function showToast(message, type) {
    const toast = document.createElement('div');
    toast.className = 'toast toast-' + type;
    toast.innerHTML = `
        <span class="toast-icon">${getNotificationIcon(type)}</span>
        <span class="toast-message">${message}</span>
        <button class="toast-close" onclick="this.parentElement.remove()">&times;</button>
    `;

    document.body.appendChild(toast);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 5000);
}

function getNotificationIcon(type) {
    const icons = {
        mention: '💬',
        assignment: '👤',
        status_change: '🔄',
        comment: '💭',
        due_date: '⏰',
        workspace_invite: '📨'
    };
    return icons[type] || '📢';
}
```

### 8. Toast Styles

```css
/* Add to custom.css */

.toast {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: var(--card-background-color);
    border: 1px solid var(--muted-border-color);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 2000;
    animation: slideIn 0.3s ease;
    transition: opacity 0.3s;
}

@keyframes slideIn {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

.toast-icon {
    font-size: 1.25rem;
}

.toast-message {
    flex: 1;
    font-size: 0.9rem;
}

.toast-close {
    background: none;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    padding: 0;
    width: auto;
    opacity: 0.5;
}

.toast-close:hover {
    opacity: 1;
}
```

---

## Routes

```go
// Notifications pages
e.GET("/notifications", h.NotificationsPage, h.RequireAuth)

// Notifications partials
partials.GET("/notifications", h.NotificationsDropdownPartial)
partials.GET("/notifications/count", h.NotificationCountPartial)
partials.GET("/notifications/list", h.NotificationsListPartial)

// Notification redirect (marks as read and redirects)
e.GET("/notifications/:id/redirect", h.NotificationRedirect, h.RequireAuth)
```

---

## Чеклист

### Templates
- [ ] `notification/dropdown.html` - navbar dropdown
- [ ] `notification/list.html` - full page
- [ ] `notification/item.html` - single notification
- [ ] `components/notification_badge.html` - unread count

### Handlers
- [ ] `NotificationsPage` - full page
- [ ] `NotificationsDropdownPartial` - dropdown content
- [ ] `NotificationCountPartial` - badge count
- [ ] `NotificationsListPartial` - list partial
- [ ] `NotificationRedirect` - mark read & redirect

### Features
- [ ] Badge показывает unread count
- [ ] Dropdown открывается с уведомлениями
- [ ] Click на уведомление ведёт к ресурсу
- [ ] Mark as read работает
- [ ] Mark all as read работает
- [ ] Delete notification работает
- [ ] Real-time новые уведомления
- [ ] Toast notifications при новом уведомлении
- [ ] Фильтрация по типу

---

## Критерии приёмки

- [ ] Badge в navbar показывает количество непрочитанных
- [ ] Dropdown показывает последние 10 уведомлений
- [ ] Click на уведомление переходит к связанному ресурсу
- [ ] Уведомление автоматически отмечается прочитанным при click
- [ ] Real-time уведомления появляются без refresh
- [ ] Toast появляется при новом уведомлении
- [ ] Full page с фильтрацией и пагинацией

---

## Зависимости

### Входящие
- [04-chat-ui.md](04-chat-ui.md) - WebSocket connection ✅
- Notification API endpoints

### Исходящие
- Нет

---

*Создано: 2026-01-05*
