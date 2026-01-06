# 06: Task Details

**Приоритет:** 🟡 High
**Статус:** ⏳ Не начато
**Зависит от:** [04-chat-ui.md](04-chat-ui.md), [05-kanban-board.md](05-kanban-board.md)

---

## Backend сервисы

### Application Layer — Task Use Cases (`internal/application/task/`)

Для редактирования задач используются те же use cases что и в Kanban Board:

| Use Case | Описание |
|----------|----------|
| `ChangeStatus` | Изменить статус |
| `ChangePriority` | Изменить приоритет |
| `AssignTask` | Назначить исполнителя |
| `UnassignTask` | Снять исполнителя |
| `SetDueDate` | Установить срок |
| `ClearDueDate` | Очистить срок |
| `UpdateTitle` | Обновить заголовок (если есть) |
| `UpdateDescription` | Обновить описание (если есть) |

**Activity Log:** Получение истории изменений задачи через EventStore или отдельный query use case.

---

## Описание

Реализовать детальный просмотр и редактирование задач: sidebar с полной информацией, inline editing всех полей, история активности, связанные сообщения.

---

## Файлы

### Templates

```
web/templates/task/
├── sidebar.html        (~150 LOC) - Full task sidebar
├── form.html           (~100 LOC) - Task edit form
├── activity.html       (~80 LOC) - Activity timeline
└── quick-edit.html     (~50 LOC) - Quick edit popover

web/templates/components/
├── activity_item.html  (~40 LOC) - Activity item
├── user_select.html    (~30 LOC) - User picker component
└── date_picker.html    (~25 LOC) - Date picker component
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+200 LOC) - Task detail handlers
```

---

## Task Sidebar Layout

```
┌─────────────────────────────────────┐
│  Task Details                    ✕  │
├─────────────────────────────────────┤
│                                     │
│  Title                              │
│  ┌─────────────────────────────┐   │
│  │ Implement OAuth             │   │
│  └─────────────────────────────┘   │
│                                     │
│  ─────────────────────────────────  │
│                                     │
│  Status         Priority            │
│  [In Progress▼] [High ▼]            │
│                                     │
│  Assignee                           │
│  [@ alice ▼]                        │
│                                     │
│  Due Date                           │
│  [2026-02-15]                       │
│                                     │
│  ─────────────────────────────────  │
│                                     │
│  Description                        │
│  ┌─────────────────────────────┐   │
│  │ Lorem ipsum dolor sit amet  │   │
│  │ consectetur adipiscing...   │   │
│  └─────────────────────────────┘   │
│                                     │
│  ─────────────────────────────────  │
│                                     │
│  Activity                           │
│  ┌─────────────────────────────┐   │
│  │ ○ alice changed status      │   │
│  │   In Progress → Review      │   │
│  │   2 hours ago               │   │
│  │                             │   │
│  │ ○ bob assigned to alice     │   │
│  │   3 hours ago               │   │
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

---

## Детали реализации

### 1. Task Sidebar (sidebar.html)

```html
{{define "task/sidebar"}}
<aside class="task-sidebar-full"
       id="task-sidebar-{{.Task.ID}}"
       hx-ext="ws"
       ws-connect="/ws?token={{.Token}}">

    <header class="sidebar-header">
        <h3>Task Details</h3>
        <button class="close-btn"
                onclick="closeTaskSidebar()"
                aria-label="Close">
            &times;
        </button>
    </header>

    <div class="sidebar-content">
        <!-- Title (editable) -->
        <div class="field">
            <label>Title</label>
            <div class="editable-field"
                 hx-get="/partials/tasks/{{.Task.ID}}/edit-title"
                 hx-target="this"
                 hx-swap="innerHTML"
                 hx-trigger="click">
                <h4>{{.Task.Title}}</h4>
                <span class="edit-icon">✏️</span>
            </div>
        </div>

        <hr>

        <!-- Status & Priority (inline) -->
        <div class="field-row">
            <div class="field">
                <label>Status</label>
                <select hx-put="/api/v1/tasks/{{.Task.ID}}/status"
                        hx-trigger="change"
                        hx-swap="none"
                        name="status"
                        class="status-select status-{{.Task.Status | lower}}">
                    {{range .Statuses}}
                    <option value="{{.Value}}"
                            {{if eq .Value $.Task.Status}}selected{{end}}>
                        {{.Label}}
                    </option>
                    {{end}}
                </select>
            </div>

            <div class="field">
                <label>Priority</label>
                <select hx-put="/api/v1/tasks/{{.Task.ID}}/priority"
                        hx-trigger="change"
                        hx-swap="none"
                        name="priority"
                        class="priority-select priority-{{.Task.Priority | lower}}">
                    {{range .Priorities}}
                    <option value="{{.Value}}"
                            {{if eq .Value $.Task.Priority}}selected{{end}}>
                        {{.Label}}
                    </option>
                    {{end}}
                </select>
            </div>
        </div>

        <!-- Assignee -->
        <div class="field">
            <label>Assignee</label>
            {{template "user_select" (dict
                "Name" "assignee_id"
                "Selected" .Task.AssigneeID
                "Users" .Participants
                "HxPut" (printf "/api/v1/tasks/%s/assignee" .Task.ID)
                "AllowEmpty" true
                "EmptyLabel" "Unassigned"
            )}}
        </div>

        <!-- Due Date -->
        <div class="field">
            <label>Due Date</label>
            {{template "date_picker" (dict
                "Name" "due_date"
                "Value" .Task.DueDate
                "HxPut" (printf "/api/v1/tasks/%s/due-date" .Task.ID)
                "AllowEmpty" true
            )}}
        </div>

        {{if .Task.DueDate}}
        <div class="due-status {{if .Task.IsOverdue}}overdue{{else if .Task.IsDueSoon}}due-soon{{end}}">
            {{if .Task.IsOverdue}}
                ⚠️ Overdue by {{.Task.OverdueDays}} days
            {{else if .Task.IsDueSoon}}
                ⏰ Due in {{.Task.DaysUntilDue}} days
            {{else}}
                📅 Due {{.Task.DueDate | formatDate}}
            {{end}}
        </div>
        {{end}}

        <hr>

        <!-- Description (editable) -->
        <div class="field">
            <label>Description</label>
            <div class="editable-field description-field"
                 hx-get="/partials/tasks/{{.Task.ID}}/edit-description"
                 hx-target="this"
                 hx-swap="innerHTML"
                 hx-trigger="click">
                {{if .Task.Description}}
                <p>{{.Task.Description}}</p>
                {{else}}
                <p class="text-muted">Click to add description...</p>
                {{end}}
                <span class="edit-icon">✏️</span>
            </div>
        </div>

        <hr>

        <!-- Activity Timeline -->
        <div class="field">
            <label>Activity</label>
            <div id="task-activity-{{.Task.ID}}"
                 class="activity-timeline"
                 hx-get="/partials/tasks/{{.Task.ID}}/activity"
                 hx-trigger="load"
                 hx-swap="innerHTML">
                {{template "loading" (dict "ID" "activity-loading")}}
            </div>
        </div>
    </div>

    <footer class="sidebar-footer">
        <a href="/workspaces/{{.Task.WorkspaceID}}/chats/{{.Task.ChatID}}"
           class="btn outline">
            Open Chat
        </a>
        <button hx-delete="/api/v1/tasks/{{.Task.ID}}"
                hx-confirm="Delete this task? This cannot be undone."
                hx-target="#task-sidebar-{{.Task.ID}}"
                hx-swap="delete"
                class="btn secondary outline">
            Delete Task
        </button>
    </footer>
</aside>

<script>
// Handle real-time updates
document.body.addEventListener('task.updated', function(evt) {
    if (evt.detail.task_id === '{{.Task.ID}}') {
        // Refresh sidebar
        htmx.ajax('GET', '/partials/tasks/{{.Task.ID}}/sidebar', {
            target: '#task-sidebar-{{.Task.ID}}',
            swap: 'outerHTML'
        });
    }
});
</script>

<style>
.task-sidebar-full {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-bottom: 1px solid var(--muted-border-color);
}

.sidebar-header h3 {
    margin: 0;
}

.close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    padding: 0;
    width: auto;
}

.sidebar-content {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
}

.sidebar-footer {
    padding: 1rem;
    border-top: 1px solid var(--muted-border-color);
    display: flex;
    gap: 0.5rem;
}

.field {
    margin-bottom: 1rem;
}

.field label {
    display: block;
    margin-bottom: 0.25rem;
    font-size: 0.875rem;
    color: var(--muted-color);
}

.field-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
}

.editable-field {
    position: relative;
    padding: 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
}

.editable-field:hover {
    background: var(--primary-focus);
}

.editable-field .edit-icon {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    opacity: 0;
    transition: opacity 0.2s;
}

.editable-field:hover .edit-icon {
    opacity: 1;
}

.due-status {
    padding: 0.5rem;
    border-radius: 4px;
    font-size: 0.875rem;
}

.due-status.overdue {
    background: color-mix(in srgb, var(--flowra-danger) 15%, white);
    color: var(--flowra-danger);
}

.due-status.due-soon {
    background: color-mix(in srgb, var(--flowra-warning) 15%, white);
    color: var(--flowra-warning);
}

.activity-timeline {
    max-height: 300px;
    overflow-y: auto;
}
</style>
{{end}}
```

### 2. User Select Component (user_select.html)

```html
{{define "user_select"}}
<select hx-put="{{.HxPut}}"
        hx-trigger="change"
        hx-swap="none"
        name="{{.Name}}"
        class="user-select">
    {{if .AllowEmpty}}
    <option value="" {{if not .Selected}}selected{{end}}>
        {{.EmptyLabel}}
    </option>
    {{end}}
    {{range .Users}}
    <option value="{{.UserID}}"
            {{if eq .UserID $.Selected}}selected{{end}}>
        {{.DisplayName}} (@{{.Username}})
    </option>
    {{end}}
</select>
{{end}}
```

### 3. Date Picker Component (date_picker.html)

```html
{{define "date_picker"}}
<div class="date-picker-wrapper">
    <input type="date"
           name="{{.Name}}"
           value="{{if .Value}}{{.Value | formatDateInput}}{{end}}"
           hx-put="{{.HxPut}}"
           hx-trigger="change"
           hx-swap="none"
           class="date-input">
    {{if and .AllowEmpty .Value}}
    <button type="button"
            class="clear-date"
            hx-put="{{.HxPut}}"
            hx-vals='{"{{.Name}}": ""}'
            hx-swap="none"
            title="Clear date">
        &times;
    </button>
    {{end}}
</div>

<style>
.date-picker-wrapper {
    display: flex;
    gap: 0.25rem;
}

.date-picker-wrapper .date-input {
    flex: 1;
    margin-bottom: 0;
}

.clear-date {
    width: auto;
    padding: 0 0.5rem;
    background: none;
    border: 1px solid var(--muted-border-color);
}
</style>
{{end}}
```

### 4. Activity Timeline (activity.html)

```html
{{define "task/activity"}}
<div class="activity-list">
    {{if .Activities}}
        {{range .Activities}}
            {{template "activity_item" .}}
        {{end}}
    {{else}}
        <p class="text-muted text-center">No activity yet</p>
    {{end}}
</div>
{{end}}

{{define "activity_item"}}
<div class="activity-item">
    <div class="activity-dot"></div>
    <div class="activity-content">
        <div class="activity-header">
            <strong>{{.Actor.Username}}</strong>
            <span class="activity-action">{{.ActionText}}</span>
        </div>

        {{if .Details}}
        <div class="activity-details">
            {{if .OldValue}}
            <span class="old-value">{{.OldValue}}</span>
            <span class="arrow">→</span>
            {{end}}
            <span class="new-value">{{.NewValue}}</span>
        </div>
        {{end}}

        <time class="activity-time text-muted">
            {{.CreatedAt | timeAgo}}
        </time>
    </div>
</div>

<style>
.activity-list {
    position: relative;
    padding-left: 1rem;
}

.activity-list::before {
    content: '';
    position: absolute;
    left: 0.25rem;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--muted-border-color);
}

.activity-item {
    position: relative;
    padding-bottom: 1rem;
    padding-left: 1rem;
}

.activity-dot {
    position: absolute;
    left: -0.75rem;
    top: 0.25rem;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--primary);
    border: 2px solid var(--background-color);
}

.activity-content {
    font-size: 0.875rem;
}

.activity-header {
    margin-bottom: 0.25rem;
}

.activity-details {
    background: var(--card-background-color);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.8rem;
    margin-bottom: 0.25rem;
}

.old-value {
    text-decoration: line-through;
    color: var(--muted-color);
}

.arrow {
    margin: 0 0.25rem;
}

.activity-time {
    font-size: 0.75rem;
}
</style>
{{end}}
```

### 5. Edit Title Form (inline)

```html
{{define "task/edit-title"}}
<form hx-put="/api/v1/tasks/{{.Task.ID}}"
      hx-target="this"
      hx-swap="outerHTML"
      class="edit-title-form">
    <input type="text"
           name="title"
           value="{{.Task.Title}}"
           required
           minlength="3"
           maxlength="200"
           autofocus>
    <div class="edit-actions">
        <button type="submit" class="small">Save</button>
        <button type="button"
                class="small secondary outline"
                hx-get="/partials/tasks/{{.Task.ID}}/title-display"
                hx-target="closest .editable-field"
                hx-swap="innerHTML">
            Cancel
        </button>
    </div>
</form>

<style>
.edit-title-form input {
    margin-bottom: 0.5rem;
}

.edit-actions {
    display: flex;
    gap: 0.5rem;
}
</style>
{{end}}
```

### 6. Edit Description Form (inline)

```html
{{define "task/edit-description"}}
<form hx-put="/api/v1/tasks/{{.Task.ID}}"
      hx-target="this"
      hx-swap="outerHTML"
      class="edit-description-form">
    <textarea name="description"
              rows="4"
              maxlength="2000"
              autofocus>{{.Task.Description}}</textarea>
    <div class="edit-actions">
        <button type="submit" class="small">Save</button>
        <button type="button"
                class="small secondary outline"
                hx-get="/partials/tasks/{{.Task.ID}}/description-display"
                hx-target="closest .editable-field"
                hx-swap="innerHTML">
            Cancel
        </button>
    </div>
</form>
{{end}}
```

---

## Routes

```go
// Task detail partials
partials.GET("/tasks/:task_id/sidebar", h.TaskSidebarPartial)
partials.GET("/tasks/:task_id/activity", h.TaskActivityPartial)
partials.GET("/tasks/:task_id/edit-title", h.TaskEditTitleForm)
partials.GET("/tasks/:task_id/title-display", h.TaskTitleDisplay)
partials.GET("/tasks/:task_id/edit-description", h.TaskEditDescriptionForm)
partials.GET("/tasks/:task_id/description-display", h.TaskDescriptionDisplay)
```

---

## Чеклист

### Templates
- [ ] `task/sidebar.html` - full task sidebar
- [ ] `task/form.html` - task edit form
- [ ] `task/activity.html` - activity timeline
- [ ] `task/quick-edit.html` - quick edit popover
- [ ] `components/activity_item.html` - activity item
- [ ] `components/user_select.html` - user picker
- [ ] `components/date_picker.html` - date picker

### Handlers
- [ ] `TaskSidebarPartial` - sidebar content
- [ ] `TaskActivityPartial` - activity list
- [ ] `TaskEditTitleForm` - inline title edit
- [ ] `TaskEditDescriptionForm` - inline description edit

### Features
- [ ] Sidebar показывает все поля задачи
- [ ] Inline editing title работает
- [ ] Inline editing description работает
- [ ] Status/priority/assignee селекты работают
- [ ] Date picker работает
- [ ] Clear date работает
- [ ] Activity timeline загружается
- [ ] Real-time обновления через WebSocket
- [ ] Delete task с подтверждением

---

## Критерии приёмки

- [ ] Task sidebar открывается из chat view и kanban
- [ ] Все поля редактируются inline
- [ ] Changes сохраняются без перезагрузки страницы
- [ ] Activity показывает историю изменений
- [ ] Overdue задачи выделяются визуально
- [ ] Delete task работает с подтверждением
- [ ] Real-time updates при изменении другими пользователями

---

## Зависимости

### Входящие
- [04-chat-ui.md](04-chat-ui.md) - task sidebar в chat view
- [05-kanban-board.md](05-kanban-board.md) - click на карточку
- **Task Use Cases** — реализованы (`internal/application/task/`)

### Исходящие
- Нет

---

*Обновлено: 2026-01-06*
