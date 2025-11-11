# Task 4: Minimal Frontend (HTMX + Pico CSS)

**Приоритет:** 🟡 MEDIUM
**Время:** 2-3 недели
**Зависимости:** Phase 3 завершена (API работает)

---

## Объединяет Tasks

- Task 4.1: Base Templates & Components
- Task 4.2: Core Pages (Auth, Workspace, Chat, Kanban, Notifications)
- Task 4.3: Static Assets (CSS, JS)

---

## Цель

Создать минимальный работающий UI для:
- Authentication (Keycloak login/logout)
- Workspace management
- Chat view с real-time messages
- Kanban board для tasks
- Notifications

---

## Tech Stack

- **HTMX 2.0** - динамика без JS
- **Pico CSS v2** - минималистичный CSS framework
- **WebSocket** - real-time updates
- **Go templates** - server-side rendering

---

## Структура файлов

```
web/
├── templates/
│   ├── layout.html              (base layout)
│   ├── auth/
│   │   ├── login.html           (login page)
│   │   └── callback.html        (OAuth callback)
│   ├── workspace/
│   │   ├── list.html            (workspace list)
│   │   ├── create.html          (create workspace form)
│   │   └── view.html            (workspace view)
│   ├── chat/
│   │   ├── view.html            (chat view with messages)
│   │   ├── list.html            (chat list sidebar)
│   │   └── create.html          (create chat form)
│   ├── board/
│   │   └── index.html           (kanban board)
│   └── notification/
│       └── list.html            (notifications dropdown)
├── components/
│   ├── navbar.html
│   ├── chat_list.html
│   ├── message.html
│   ├── task_card.html
│   └── notification_item.html
└── static/
    ├── css/
    │   └── custom.css           (custom styles)
    └── js/
        └── app.js               (utilities: WebSocket, autocomplete)
```

---

## 1. Base Layout (layout.html)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }} - Flowra</title>

    <!-- Pico CSS -->
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
    <link rel="stylesheet" href="/static/css/custom.css">

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.0"></script>
    <script src="https://unpkg.com/htmx-ext-ws@2.0.0/ws.js"></script>
</head>
<body>
    {{ template "navbar" . }}

    <main class="container">
        {{ template "content" . }}
    </main>

    {{ template "footer" . }}

    <script src="/static/js/app.js"></script>
</body>
</html>
```

---

## 2. Chat View (chat/view.html)

```html
{{ define "content" }}
<div class="chat-layout" hx-ext="ws" ws-connect="/ws?token={{ .Token }}">
    <!-- Sidebar: Chat List -->
    <aside class="chat-sidebar">
        {{ template "chat_list" .Chats }}
    </aside>

    <!-- Main: Messages -->
    <section class="chat-main">
        <header>
            <h2>{{ .Chat.Title }}</h2>
            <span class="badge">{{ .Chat.Type }}</span>
        </header>

        <div class="messages" id="messages">
            {{ range .Messages }}
                {{ template "message" . }}
            {{ end }}
        </div>

        <!-- Message Input -->
        <form hx-post="/api/v1/chats/{{ .Chat.ID }}/messages"
              hx-target="#messages"
              hx-swap="beforeend"
              hx-on::after-request="this.reset()">
            <textarea name="content"
                      placeholder="Type a message... Use #createTask to create tasks"
                      rows="3"></textarea>
            <button type="submit">Send</button>
        </form>
    </section>

    <!-- Sidebar: Task Info (if Task/Bug/Epic) -->
    {{ if .Chat.IsTask }}
    <aside class="task-sidebar">
        <h3>Task Details</h3>

        <label>Status</label>
        <select hx-put="/api/v1/chats/{{ .Chat.ID }}/status"
                hx-trigger="change"
                name="status">
            <option value="New">New</option>
            <option value="InProgress" {{ if eq .Chat.Status "InProgress" }}selected{{ end }}>In Progress</option>
            <option value="Done">Done</option>
        </select>

        <label>Assignee</label>
        <select hx-put="/api/v1/chats/{{ .Chat.ID }}/assignee" name="assignee_id">
            <option value="">Unassigned</option>
            {{ range .Participants }}
            <option value="{{ .UserID }}">{{ .Username }}</option>
            {{ end }}
        </select>

        <label>Priority</label>
        <select hx-put="/api/v1/chats/{{ .Chat.ID }}/priority" name="priority">
            <option value="Low">Low</option>
            <option value="Medium">Medium</option>
            <option value="High">High</option>
        </select>
    </aside>
    {{ end }}
</div>

<script>
// Real-time message updates
document.body.addEventListener("chat.message.posted", function(e) {
    const msg = e.detail;
    if (msg.chat_id === "{{ .Chat.ID }}") {
        htmx.ajax("GET", "/api/v1/messages/" + msg.message_id, {
            target: "#messages",
            swap: "beforeend"
        });
    }
});

// Typing indicator
let typingTimeout;
document.querySelector('textarea[name="content"]').addEventListener('input', function() {
    clearTimeout(typingTimeout);

    // Send typing event
    // ws.send(JSON.stringify({type: "chat.typing", chat_id: "{{ .Chat.ID }}"}));

    typingTimeout = setTimeout(() => {
        // Stop typing
    }, 1000);
});
</script>
{{ end }}
```

---

## 3. Kanban Board (board/index.html)

```html
{{ define "content" }}
<div class="kanban-board">
    <h1>{{ .WorkspaceName }} - Board</h1>

    <div class="board-columns">
        {{ range .Columns }}
        <div class="column" data-status="{{ .Status }}">
            <h3>{{ .Title }} <span class="badge">{{ .Count }}</span></h3>

            <div class="cards"
                 hx-post="/api/v1/tasks/move"
                 hx-trigger="drop"
                 ondrop="drop(event)"
                 ondragover="allowDrop(event)">
                {{ range .Tasks }}
                    {{ template "task_card" . }}
                {{ end }}
            </div>
        </div>
        {{ end }}
    </div>
</div>

<style>
.kanban-board {
    padding: 1rem;
}

.board-columns {
    display: flex;
    gap: 1rem;
    overflow-x: auto;
}

.column {
    min-width: 300px;
    background: var(--card-background-color);
    padding: 1rem;
    border-radius: 8px;
}

.cards {
    min-height: 400px;
}

.task-card {
    background: white;
    padding: 1rem;
    margin-bottom: 0.5rem;
    border-radius: 4px;
    cursor: grab;
    border-left: 4px solid var(--primary);
}

.task-card[data-priority="High"] {
    border-left-color: #e53e3e;
}

.task-card.dragging {
    opacity: 0.5;
}
</style>

<script>
function allowDrop(ev) {
    ev.preventDefault();
}

function drag(ev) {
    ev.dataTransfer.setData("taskId", ev.target.dataset.taskId);
}

function drop(ev) {
    ev.preventDefault();
    const taskId = ev.dataTransfer.getData("taskId");
    const newStatus = ev.target.closest('.column').dataset.status;

    // Update task status
    htmx.ajax("PUT", `/api/v1/chats/${taskId}/status`, {
        values: { status: newStatus }
    });
}
</script>
{{ end }}
```

---

## 4. Components

### Message Component (message.html)

```html
{{ define "message" }}
<article class="message" id="message-{{ .ID }}">
    <header>
        <strong>{{ .SentBy.Username }}</strong>
        <time datetime="{{ .CreatedAt }}">{{ .CreatedAt | formatTime }}</time>
    </header>

    <p>{{ .Content }}</p>

    {{ if .Reactions }}
    <div class="reactions">
        {{ range .Reactions }}
        <button class="reaction"
                hx-delete="/api/v1/messages/{{ $.ID }}/reactions/{{ .Emoji }}"
                hx-trigger="click">
            {{ .Emoji }} {{ .Count }}
        </button>
        {{ end }}
    </div>
    {{ end }}

    <footer>
        <button hx-get="/api/v1/messages/{{ .ID }}/thread"
                hx-target="#thread-{{ .ID }}"
                hx-swap="innerHTML">
            {{ .ThreadCount }} replies
        </button>
    </footer>

    <div id="thread-{{ .ID }}"></div>
</article>
{{ end }}
```

### Task Card Component (task_card.html)

```html
{{ define "task_card" }}
<div class="task-card"
     data-task-id="{{ .ID }}"
     data-priority="{{ .Priority }}"
     draggable="true"
     ondragstart="drag(event)">
    <h4>{{ .Title }}</h4>

    {{ if .AssignedTo }}
    <p class="assignee">👤 {{ .AssignedTo.Username }}</p>
    {{ end }}

    <div class="task-meta">
        <span class="priority-badge priority-{{ .Priority }}">{{ .Priority }}</span>
        {{ if .DueDate }}
        <span class="due-date">📅 {{ .DueDate | formatDate }}</span>
        {{ end }}
    </div>
</div>
{{ end }}
```

---

## 5. Custom CSS (custom.css)

```css
:root {
    --primary-color: #0066cc;
    --success-color: #10b981;
    --danger-color: #ef4444;
    --warning-color: #f59e0b;
}

.chat-layout {
    display: grid;
    grid-template-columns: 250px 1fr 300px;
    gap: 1rem;
    height: calc(100vh - 80px);
}

.chat-sidebar {
    overflow-y: auto;
}

.chat-main {
    display: flex;
    flex-direction: column;
}

.messages {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
}

.message {
    margin-bottom: 1rem;
    padding: 0.5rem;
    border-left: 3px solid var(--primary-color);
}

.priority-High { background: var(--danger-color); color: white; }
.priority-Medium { background: var(--warning-color); color: white; }
.priority-Low { background: var(--success-color); color: white; }
```

---

## 6. JavaScript Utilities (app.js)

```javascript
// Tag autocomplete
function initTagAutocomplete() {
    const textarea = document.querySelector('textarea[name="content"]');
    if (!textarea) return;

    textarea.addEventListener('input', function(e) {
        const value = e.target.value;
        const cursorPos = e.target.selectionStart;

        // Detect # at cursor position
        if (value[cursorPos - 1] === '#') {
            showTagSuggestions(e.target);
        }
    });
}

function showTagSuggestions(textarea) {
    const suggestions = [
        '#createTask',
        '#createBug',
        '#createEpic',
        '#setStatus',
        '#assign',
        '#setPriority',
        '#setDueDate'
    ];

    // Show autocomplete dropdown
    // ... implementation
}

// WebSocket reconnect
let ws;
function connectWebSocket(token) {
    ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

    ws.onclose = function() {
        setTimeout(() => connectWebSocket(token), 3000);
    };

    ws.onmessage = function(e) {
        const msg = JSON.parse(e.data);
        document.body.dispatchEvent(new CustomEvent(msg.type, { detail: msg.data }));
    };
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    initTagAutocomplete();
});
```

---

## Handler для Templates

```go
// internal/handler/http/template_handler.go

type TemplateHandler struct {
    templates *template.Template
}

func (h *TemplateHandler) RenderChatView(c echo.Context) error {
    chatID := c.Param("chatId")

    // Fetch data
    chat, _ := h.getChatUC.Execute(ctx, GetChatQuery{ChatID: chatID})
    messages, _ := h.listMessagesUC.Execute(ctx, ListMessagesQuery{ChatID: chatID})

    data := map[string]interface{}{
        "Title":    chat.Title,
        "Chat":     chat,
        "Messages": messages,
        "Token":    getAccessToken(c),
    }

    return h.templates.ExecuteTemplate(c.Response(), "chat/view.html", data)
}
```

---

## Критерии успеха

- ✅ **Pico CSS загружается**
- ✅ **HTMX работает (AJAX requests)**
- ✅ **WebSocket real-time updates**
- ✅ **Users can login/logout**
- ✅ **Chat view functional**
- ✅ **Kanban board drag-n-drop works**
- ✅ **Notifications real-time**

---

## MVP Release Ready! 🎉

После Phase 4 → Production deployment
