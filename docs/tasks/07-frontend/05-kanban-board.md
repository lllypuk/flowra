# 05: Kanban Board

**Приоритет:** 🟡 High
**Статус:** ✅ Завершено
**Зависит от:** [03-workspace-pages.md](03-workspace-pages.md)

---

## Backend сервисы

### Application Layer — Task Use Cases (`internal/application/task/`)

| Use Case | Описание |
|----------|----------|
| `CreateTask` | Создать задачу |
| `GetTask` | Получить задачу по ID |
| `ListTasks` | Список задач (с фильтрами) |
| `ChangeStatus` | Изменить статус |
| `ChangePriority` | Изменить приоритет |
| `ChangeSeverity` | Изменить severity |
| `AssignTask` | Назначить исполнителя |
| `UnassignTask` | Снять исполнителя |
| `SetDueDate` | Установить срок |
| `ClearDueDate` | Очистить срок |
| `DeleteTask` | Удалить задачу |

**Статусы задач:** `todo`, `in_progress`, `review`, `done`

**Примечание:** Может потребоваться создание `TaskService` wrapper (по аналогии с `ChatService`) для удобства использования в handlers.

---

## Описание

Реализовать канбан-доску: колонки по статусам, карточки задач с drag-n-drop, фильтрация и сортировка, real-time обновления при изменении статуса.

---

## Файлы

### Templates

```
web/templates/board/
├── index.html          (~120 LOC) - Kanban board layout
├── column.html         (~60 LOC) - Status column
├── card.html           (~50 LOC) - Task card
└── filters.html        (~40 LOC) - Filter controls

web/templates/components/
└── task_card.html      (~60 LOC) - Reusable task card
```

### Static Assets

```
web/static/
├── css/
│   └── board.css       (~100 LOC) - Board-specific styles
└── js/
    └── board.js        (~120 LOC) - Drag-n-drop logic
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+200 LOC) - Board handlers
```

---

## Board Layout

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Filters: [Type ▼] [Assignee ▼] [Priority ▼]    [Search...]  [+ Task]  │
├─────────────────┬─────────────────┬─────────────────┬───────────────────┤
│    To Do (5)    │  In Progress (3) │    Review (2)   │     Done (12)     │
├─────────────────┼─────────────────┼─────────────────┼───────────────────┤
│ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────┐   │
│ │ Task Card   │ │ │ Task Card   │ │ │ Task Card   │ │ │ Task Card   │   │
│ │ - Title     │ │ │             │ │ │             │ │ │             │   │
│ │ - Assignee  │ │ └─────────────┘ │ └─────────────┘ │ └─────────────┘   │
│ │ - Priority  │ │ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────┐   │
│ │ - Due Date  │ │ │ Task Card   │ │ │ Task Card   │ │ │ Task Card   │   │
│ └─────────────┘ │ │             │ │ │             │ │ │             │   │
│ ┌─────────────┐ │ └─────────────┘ │ └─────────────┘ │ └─────────────┘   │
│ │ Task Card   │ │                 │                 │                   │
│ └─────────────┘ │                 │                 │                   │
│        ⋮        │        ⋮        │        ⋮        │        ⋮          │
└─────────────────┴─────────────────┴─────────────────┴───────────────────┘
```

---

## Детали реализации

### 1. Board Index (index.html)

```html
{{define "board/index"}}
<div class="board-page">
    <!-- Header with filters -->
    <header class="board-header">
        <div class="board-title">
            <h1>Board</h1>
            <span class="task-count">{{.TotalTasks}} tasks</span>
        </div>

        {{template "board/filters" .}}

        <button hx-get="/partials/task/create-form?workspace_id={{.Workspace.ID}}"
                hx-target="#modal-container"
                class="primary">
            + New Task
        </button>
    </header>

    <!-- Kanban Board -->
    <div class="board-container"
         hx-ext="ws"
         ws-connect="/ws?token={{.Token}}">

        <div class="board-columns"
             id="board-columns"
             hx-get="/partials/workspace/{{.Workspace.ID}}/board"
             hx-trigger="load"
             hx-swap="innerHTML"
             hx-vals='{{.FilterParams | json}}'>
            {{template "loading" (dict "ID" "board-loading")}}
        </div>
    </div>

    <!-- Modal container -->
    <div id="modal-container"></div>
</div>

<script src="/static/js/board.js"></script>

<style>
.board-page {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 60px);
    padding: 1rem;
}

.board-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
}

.board-title {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
}

.board-title h1 {
    margin: 0;
}

.task-count {
    color: var(--muted-color);
}

.board-container {
    flex: 1;
    overflow: hidden;
}

.board-columns {
    display: flex;
    gap: 1rem;
    height: 100%;
    overflow-x: auto;
    padding-bottom: 1rem;
}
</style>
{{end}}
```

### 2. Filters (filters.html)

```html
{{define "board/filters"}}
<div class="board-filters">
    <select name="type"
            hx-get="/partials/workspace/{{.Workspace.ID}}/board"
            hx-target="#board-columns"
            hx-include="[name='assignee'], [name='priority'], [name='search']">
        <option value="">All Types</option>
        <option value="task" {{if eq .Filters.Type "task"}}selected{{end}}>Tasks</option>
        <option value="bug" {{if eq .Filters.Type "bug"}}selected{{end}}>Bugs</option>
        <option value="epic" {{if eq .Filters.Type "epic"}}selected{{end}}>Epics</option>
    </select>

    <select name="assignee"
            hx-get="/partials/workspace/{{.Workspace.ID}}/board"
            hx-target="#board-columns"
            hx-include="[name='type'], [name='priority'], [name='search']">
        <option value="">All Assignees</option>
        <option value="unassigned" {{if eq .Filters.Assignee "unassigned"}}selected{{end}}>
            Unassigned
        </option>
        <option value="me" {{if eq .Filters.Assignee "me"}}selected{{end}}>
            Assigned to me
        </option>
        {{range .Members}}
        <option value="{{.UserID}}" {{if eq .UserID $.Filters.Assignee}}selected{{end}}>
            {{.Username}}
        </option>
        {{end}}
    </select>

    <select name="priority"
            hx-get="/partials/workspace/{{.Workspace.ID}}/board"
            hx-target="#board-columns"
            hx-include="[name='type'], [name='assignee'], [name='search']">
        <option value="">All Priorities</option>
        <option value="critical" {{if eq .Filters.Priority "critical"}}selected{{end}}>Critical</option>
        <option value="high" {{if eq .Filters.Priority "high"}}selected{{end}}>High</option>
        <option value="medium" {{if eq .Filters.Priority "medium"}}selected{{end}}>Medium</option>
        <option value="low" {{if eq .Filters.Priority "low"}}selected{{end}}>Low</option>
    </select>

    <input type="search"
           name="search"
           placeholder="Search tasks..."
           value="{{.Filters.Search}}"
           hx-get="/partials/workspace/{{.Workspace.ID}}/board"
           hx-target="#board-columns"
           hx-trigger="input changed delay:300ms"
           hx-include="[name='type'], [name='assignee'], [name='priority']">
</div>

<style>
.board-filters {
    display: flex;
    gap: 0.5rem;
    flex: 1;
}

.board-filters select,
.board-filters input {
    margin-bottom: 0;
    width: auto;
}

.board-filters input[type="search"] {
    min-width: 200px;
}
</style>
{{end}}
```

### 3. Column (column.html)

```html
{{define "board/column"}}
<div class="board-column"
     id="column-{{.Status}}"
     data-status="{{.Status}}">

    <header class="column-header">
        <h3>
            <span class="status-dot status-{{.Status | lower}}"></span>
            {{.Title}}
        </h3>
        <span class="column-count">{{.Count}}</span>
    </header>

    <div class="column-cards"
         data-status="{{.Status}}"
         ondrop="handleDrop(event)"
         ondragover="handleDragOver(event)"
         ondragleave="handleDragLeave(event)">

        {{range .Tasks}}
            {{template "task_card" .}}
        {{end}}

        {{if gt .TotalCount .Count}}
        <button hx-get="/partials/workspace/{{$.WorkspaceID}}/board/{{.Status}}/more"
                hx-target="this"
                hx-swap="outerHTML"
                hx-vals='{"offset": {{.Count}}}'
                class="load-more outline small">
            Load more ({{sub .TotalCount .Count}} remaining)
        </button>
        {{end}}
    </div>
</div>

<style>
.board-column {
    flex: 0 0 300px;
    display: flex;
    flex-direction: column;
    background: var(--card-background-color);
    border-radius: 8px;
    max-height: 100%;
}

.column-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-bottom: 1px solid var(--muted-border-color);
}

.column-header h3 {
    margin: 0;
    font-size: 1rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
}

.status-dot.status-todo { background: var(--muted-color); }
.status-dot.status-in_progress { background: var(--flowra-primary); }
.status-dot.status-review { background: var(--flowra-warning); }
.status-dot.status-done { background: var(--flowra-success); }

.column-count {
    background: var(--muted-border-color);
    padding: 0.125rem 0.5rem;
    border-radius: 10px;
    font-size: 0.875rem;
}

.column-cards {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
    min-height: 100px;
}

.column-cards.drag-over {
    background: var(--primary-focus);
}

.load-more {
    width: 100%;
    margin-top: 0.5rem;
}
</style>
{{end}}
```

### 4. Task Card (task_card.html)

```html
{{define "task_card"}}
<article class="task-card"
         id="task-{{.ID}}"
         data-task-id="{{.ID}}"
         data-priority="{{.Priority}}"
         draggable="true"
         ondragstart="handleDragStart(event)"
         ondragend="handleDragEnd(event)"
         hx-get="/workspaces/{{.WorkspaceID}}/chats/{{.ChatID}}"
         hx-push-url="true"
         hx-target="body">

    <!-- Type indicator -->
    <div class="card-type type-{{.Type}}">
        {{if eq .Type "bug"}}
            <span title="Bug">B</span>
        {{else if eq .Type "epic"}}
            <span title="Epic">E</span>
        {{else}}
            <span title="Task">T</span>
        {{end}}
    </div>

    <!-- Title -->
    <h4 class="card-title">{{.Title | truncate 60}}</h4>

    <!-- Meta info -->
    <div class="card-meta">
        {{if .Assignee}}
        <span class="card-assignee" title="{{.Assignee.DisplayName}}">
            {{if .Assignee.AvatarURL}}
            <img src="{{.Assignee.AvatarURL}}" alt="" class="avatar-tiny">
            {{else}}
            <span class="avatar-tiny avatar-placeholder">
                {{slice .Assignee.Username 0 1 | upper}}
            </span>
            {{end}}
            {{.Assignee.Username}}
        </span>
        {{end}}

        {{if .DueDate}}
        <span class="card-due {{if .IsOverdue}}overdue{{end}}" title="Due date">
            {{.DueDate | formatDate}}
        </span>
        {{end}}
    </div>

    <!-- Priority indicator -->
    <div class="card-priority priority-{{.Priority | lower}}"
         title="{{.Priority}} priority">
    </div>
</article>

<style>
.task-card {
    background: var(--background-color);
    border-radius: 6px;
    padding: 0.75rem;
    margin-bottom: 0.5rem;
    cursor: grab;
    position: relative;
    border-left: 4px solid transparent;
    transition: transform 0.1s, box-shadow 0.1s;
}

.task-card:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.task-card.dragging {
    opacity: 0.5;
    cursor: grabbing;
}

.task-card[data-priority="critical"] { border-left-color: #dc2626; }
.task-card[data-priority="high"] { border-left-color: #f59e0b; }
.task-card[data-priority="medium"] { border-left-color: #3b82f6; }
.task-card[data-priority="low"] { border-left-color: #10b981; }

.card-type {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    width: 20px;
    height: 20px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: bold;
}

.card-type.type-task { background: #dbeafe; color: #1d4ed8; }
.card-type.type-bug { background: #fee2e2; color: #dc2626; }
.card-type.type-epic { background: #f3e8ff; color: #9333ea; }

.card-title {
    margin: 0 0 0.5rem 0;
    font-size: 0.9rem;
    font-weight: 500;
    padding-right: 1.5rem;
}

.card-meta {
    display: flex;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: var(--muted-color);
}

.card-assignee {
    display: flex;
    align-items: center;
    gap: 0.25rem;
}

.avatar-tiny {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    font-size: 0.6rem;
}

.card-due.overdue {
    color: var(--flowra-danger);
}

.card-priority {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 3px;
    border-radius: 0 0 6px 6px;
}
</style>
{{end}}
```

### 5. Board JavaScript (board.js)

```javascript
/**
 * Kanban Board Drag and Drop
 */

let draggedTask = null;

function handleDragStart(event) {
    draggedTask = event.target;
    event.target.classList.add('dragging');

    // Set data for the drag operation
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', event.target.dataset.taskId);

    // Create a drag image
    const ghost = event.target.cloneNode(true);
    ghost.style.opacity = '0.8';
    ghost.style.position = 'absolute';
    ghost.style.top = '-1000px';
    document.body.appendChild(ghost);
    event.dataTransfer.setDragImage(ghost, 0, 0);
    setTimeout(() => ghost.remove(), 0);
}

function handleDragEnd(event) {
    event.target.classList.remove('dragging');
    draggedTask = null;

    // Remove all drag-over states
    document.querySelectorAll('.column-cards.drag-over').forEach(col => {
        col.classList.remove('drag-over');
    });
}

function handleDragOver(event) {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';

    const column = event.currentTarget;
    column.classList.add('drag-over');

    // Find insertion point
    const cards = [...column.querySelectorAll('.task-card:not(.dragging)')];
    const afterElement = getDragAfterElement(column, event.clientY);

    if (afterElement) {
        column.insertBefore(draggedTask, afterElement);
    } else {
        // Find the load-more button if exists
        const loadMore = column.querySelector('.load-more');
        if (loadMore) {
            column.insertBefore(draggedTask, loadMore);
        } else {
            column.appendChild(draggedTask);
        }
    }
}

function handleDragLeave(event) {
    // Only remove drag-over if leaving the column entirely
    const rect = event.currentTarget.getBoundingClientRect();
    if (
        event.clientX < rect.left ||
        event.clientX > rect.right ||
        event.clientY < rect.top ||
        event.clientY > rect.bottom
    ) {
        event.currentTarget.classList.remove('drag-over');
    }
}

function handleDrop(event) {
    event.preventDefault();

    const column = event.currentTarget;
    column.classList.remove('drag-over');

    const taskId = event.dataTransfer.getData('text/plain');
    const newStatus = column.dataset.status;
    const taskCard = document.getElementById('task-' + taskId);

    if (!taskCard) return;

    const oldStatus = taskCard.closest('.column-cards').dataset.status;

    // If status changed, update via API
    if (oldStatus !== newStatus) {
        updateTaskStatus(taskId, newStatus, taskCard);
    }
}

function getDragAfterElement(container, y) {
    const draggableElements = [
        ...container.querySelectorAll('.task-card:not(.dragging)')
    ];

    return draggableElements.reduce((closest, child) => {
        const box = child.getBoundingClientRect();
        const offset = y - box.top - box.height / 2;

        if (offset < 0 && offset > closest.offset) {
            return { offset: offset, element: child };
        } else {
            return closest;
        }
    }, { offset: Number.NEGATIVE_INFINITY }).element;
}

function updateTaskStatus(taskId, newStatus, taskCard) {
    // Show loading state
    taskCard.style.opacity = '0.5';

    // Update via HTMX
    htmx.ajax('PUT', '/api/v1/tasks/' + taskId + '/status', {
        values: { status: newStatus },
        target: '#task-' + taskId,
        swap: 'outerHTML'
    }).then(function() {
        // Update column counts
        updateColumnCounts();
    }).catch(function(err) {
        console.error('Failed to update task status:', err);
        // Revert position on error
        taskCard.style.opacity = '1';
        // Could show error toast here
    });
}

function updateColumnCounts() {
    document.querySelectorAll('.board-column').forEach(column => {
        const count = column.querySelectorAll('.task-card').length;
        const countEl = column.querySelector('.column-count');
        if (countEl) {
            countEl.textContent = count;
        }
    });
}

// Real-time updates via WebSocket
document.body.addEventListener('task.updated', function(evt) {
    const data = evt.detail;

    // If status changed, move the card
    if (data.changes && data.changes.status) {
        const taskCard = document.getElementById('task-' + data.task_id);
        if (taskCard) {
            const newColumn = document.querySelector(
                `.column-cards[data-status="${data.changes.status.new}"]`
            );
            if (newColumn) {
                newColumn.appendChild(taskCard);
                updateColumnCounts();
            }
        }
    }

    // Refresh task card if other fields changed
    if (data.changes && !data.changes.status) {
        htmx.ajax('GET', '/partials/tasks/' + data.task_id + '/card', {
            target: '#task-' + data.task_id,
            swap: 'outerHTML'
        });
    }
});
```

---

## Routes

```go
// Board pages
board := workspace.Group("/:workspace_id/board", h.RequireWorkspaceAccess)
board.GET("", h.BoardIndex)

// Board partials
partials.GET("/workspace/:workspace_id/board", h.BoardPartial)
partials.GET("/workspace/:workspace_id/board/:status/more", h.BoardColumnMore)
partials.GET("/tasks/:task_id/card", h.TaskCardPartial)
partials.GET("/task/create-form", h.TaskCreateForm)
```

---

## Чеклист

### Templates
- [x] `board/index.html` - board page layout
- [x] `board/column.html` - status column
- [x] `board/card.html` - task card (реализовано как `components/task_card.html`)
- [x] `board/filters.html` - filter controls
- [x] `components/task_card.html` - reusable card

### JavaScript
- [x] `board.js` - drag-n-drop logic

### CSS
- [x] `board.css` - board styles

### Handlers
- [x] `BoardIndex` - board page
- [x] `BoardPartial` - columns partial
- [x] `BoardColumnMore` - load more tasks
- [x] `TaskCardPartial` - single card

### Features
- [x] Columns отображаются с задачами
- [x] Drag-n-drop работает
- [x] Статус обновляется при drop
- [x] Фильтрация работает
- [x] Поиск работает
- [x] Real-time обновления через WS
- [x] "Load more" для длинных колонок

---

## Критерии приёмки

- [x] Kanban board отображает 4 колонки (To Do, In Progress, Review, Done)
- [x] Карточки можно перетаскивать между колонками
- [x] При drop статус задачи обновляется
- [x] Фильтры по типу/assignee/priority работают
- [x] Поиск фильтрует карточки
- [x] Click на карточку открывает чат задачи
- [x] Real-time обновления при изменении другими пользователями
- [x] Mobile: горизонтальный scroll

---

## Зависимости

### Входящие
- [03-workspace-pages.md](03-workspace-pages.md) - workspace context
- **Task Use Cases** — реализованы (`internal/application/task/`)
- WebSocket for real-time updates

### Исходящие
- [06-task-details.md](06-task-details.md) - task editing via sidebar

---

## Performance Considerations

- Загружаем только первые N задач в каждой колонке (default: 20)
- "Load more" для ленивой загрузки остальных
- Drag-n-drop использует native HTML5 API (не библиотеки)
- WebSocket updates только для видимых задач (workspace subscription)

---

*Обновлено: 2026-01-15*

**Статус:** ✅ Завершено
