# 04: Chat UI

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
**Зависит от:** [03-workspace-pages.md](03-workspace-pages.md)

---

## Backend сервисы

### ChatService (`internal/service/chat_service.go`)

| Метод | Описание |
|-------|----------|
| `CreateChat(ctx, cmd)` | Создать чат |
| `GetChat(ctx, query)` | Получить чат по ID |
| `ListChats(ctx, query)` | Список чатов workspace |
| `RenameChat(ctx, cmd)` | Переименовать чат |
| `AddParticipant(ctx, cmd)` | Добавить участника |
| `RemoveParticipant(ctx, cmd)` | Удалить участника |
| `DeleteChat(ctx, chatID, deletedBy)` | Soft delete через event sourcing |

**Особенности ChatService:**
- Использует event sourcing для операций
- `loadAggregate()` / `saveAggregate()` для работы с Chat aggregate

### Application Layer Use Cases

**Message Use Cases** (`internal/application/message/`):
- `SendMessage` — отправка сообщения
- `EditMessage` — редактирование сообщения
- `DeleteMessage` — удаление сообщения
- `AddReaction` — добавление реакции
- `RemoveReaction` — удаление реакции

---

## Описание

Реализовать интерфейс чата: 3-колоночный layout (chat list, messages, task sidebar), real-time сообщения через WebSocket, отправка сообщений с поддержкой тегов, typing indicators.

---

## Файлы

### Templates

```
web/templates/chat/
├── layout.html         (~150 LOC) - 3-column chat layout
├── list.html           (~80 LOC) - Chat list sidebar
├── view.html           (~120 LOC) - Messages view
├── create.html         (~70 LOC) - Create chat form
└── participants.html   (~60 LOC) - Participants panel

web/templates/components/
├── message.html        (~60 LOC) - Single message
├── message_form.html   (~50 LOC) - Message input with tags
├── chat_item.html      (~40 LOC) - Chat list item
└── typing.html         (~20 LOC) - Typing indicator
```

### Static Assets

```
web/static/js/
└── chat.js             (~150 LOC) - Chat-specific JS (typing, autocomplete)
```

### Go Code

```
internal/handler/http/
└── template_handler.go  (+400 LOC) - Chat page handlers
```

---

## Layout Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           Navbar                                │
├────────────┬────────────────────────────────────┬───────────────┤
│            │                                    │               │
│  Chat List │          Messages                  │  Task/Info    │
│  (250px)   │          (flex)                    │  Sidebar      │
│            │                                    │  (300px)      │
│  - Search  │  ┌────────────────────────────┐   │               │
│  - Filters │  │ Messages (scroll)          │   │  - Status     │
│  - Items   │  │                            │   │  - Assignee   │
│            │  │                            │   │  - Priority   │
│            │  └────────────────────────────┘   │  - Due Date   │
│            │                                    │               │
│            │  ┌────────────────────────────┐   │  - Activity   │
│            │  │ Message Input              │   │               │
│            │  └────────────────────────────┘   │               │
│            │                                    │               │
└────────────┴────────────────────────────────────┴───────────────┘
```

---

## Детали реализации

### 1. Chat Layout (layout.html)

```html
{{define "chat/layout"}}
<div class="chat-layout"
     hx-ext="ws"
     ws-connect="/ws?token={{.Token}}">

    <!-- Chat List Sidebar -->
    <aside class="chat-sidebar">
        <header>
            <input type="search"
                   placeholder="Search chats..."
                   hx-get="/partials/chats/search"
                   hx-trigger="input changed delay:300ms"
                   hx-target="#chat-list"
                   name="q">
        </header>

        <nav id="chat-list"
             hx-get="/partials/workspace/{{.Workspace.ID}}/chats"
             hx-trigger="load"
             hx-swap="innerHTML">
            {{template "loading" (dict "ID" "chat-list-loading")}}
        </nav>

        <footer>
            <button hx-get="/partials/chat/create-form?workspace_id={{.Workspace.ID}}"
                    hx-target="#modal-container"
                    class="outline small">
                + New Chat
            </button>
        </footer>
    </aside>

    <!-- Main Content -->
    <main class="chat-main">
        {{if .Chat}}
            {{template "chat/view" .}}
        {{else}}
            {{template "chat/empty" .}}
        {{end}}
    </main>

    <!-- Task Sidebar (for task/bug/epic chats) -->
    {{if and .Chat .Chat.IsTaskChat}}
    <aside class="task-sidebar">
        {{template "chat/task-sidebar" .}}
    </aside>
    {{end}}

    <!-- Modal container -->
    <div id="modal-container"></div>
</div>

<style>
.chat-layout {
    display: grid;
    grid-template-columns: 250px 1fr;
    height: calc(100vh - 60px);
}

.chat-layout.with-task-sidebar {
    grid-template-columns: 250px 1fr 300px;
}

.chat-sidebar {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--muted-border-color);
    background: var(--card-background-color);
}

.chat-sidebar header {
    padding: 1rem;
    border-bottom: 1px solid var(--muted-border-color);
}

.chat-sidebar nav {
    flex: 1;
    overflow-y: auto;
}

.chat-sidebar footer {
    padding: 1rem;
    border-top: 1px solid var(--muted-border-color);
}

.chat-main {
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.task-sidebar {
    border-left: 1px solid var(--muted-border-color);
    background: var(--card-background-color);
    padding: 1rem;
    overflow-y: auto;
}

@media (max-width: 1024px) {
    .chat-layout {
        grid-template-columns: 1fr;
    }

    .chat-sidebar,
    .task-sidebar {
        display: none;
    }
}
</style>
{{end}}
```

### 2. Chat List Item (chat_item.html)

```html
{{define "chat_item"}}
<a href="/workspaces/{{.WorkspaceID}}/chats/{{.ID}}"
   class="chat-item {{if eq .ID $.ActiveChatID}}active{{end}}"
   hx-get="/partials/chats/{{.ID}}"
   hx-target=".chat-main"
   hx-push-url="true">

    <div class="chat-item-icon">
        {{if eq .Type "task"}}
            <span class="type-icon type-task" title="Task">T</span>
        {{else if eq .Type "bug"}}
            <span class="type-icon type-bug" title="Bug">B</span>
        {{else if eq .Type "epic"}}
            <span class="type-icon type-epic" title="Epic">E</span>
        {{else}}
            <span class="type-icon type-discussion" title="Discussion">D</span>
        {{end}}
    </div>

    <div class="chat-item-content">
        <div class="chat-item-title">
            {{.Title | truncate 30}}
        </div>
        {{if .LastMessage}}
        <div class="chat-item-preview text-muted">
            {{.LastMessage.Author.Username}}: {{.LastMessage.Content | truncate 40}}
        </div>
        {{end}}
    </div>

    <div class="chat-item-meta">
        {{if gt .UnreadCount 0}}
        <span class="badge">{{.UnreadCount}}</span>
        {{end}}
        <small class="text-muted">{{.UpdatedAt | timeAgo}}</small>
    </div>
</a>
{{end}}
```

### 3. Messages View (view.html)

```html
{{define "chat/view"}}
<div class="chat-view" id="chat-{{.Chat.ID}}">
    <!-- Chat Header -->
    <header class="chat-header">
        <div class="chat-title">
            <h2>{{.Chat.Title}}</h2>
            {{if .Chat.IsTaskChat}}
            <span class="status-badge status-{{.Chat.Status | lower}}">
                {{.Chat.Status}}
            </span>
            {{end}}
        </div>

        <div class="chat-actions">
            <button hx-get="/partials/chats/{{.Chat.ID}}/participants"
                    hx-target="#modal-container"
                    class="outline small"
                    title="Participants">
                <svg><!-- users icon --></svg>
                {{.Chat.ParticipantCount}}
            </button>
        </div>
    </header>

    <!-- Messages Container -->
    <div class="messages-container"
         id="messages-{{.Chat.ID}}"
         hx-get="/partials/chats/{{.Chat.ID}}/messages"
         hx-trigger="load"
         hx-swap="innerHTML">
        {{template "loading" (dict "ID" "messages-loading")}}
    </div>

    <!-- Typing Indicator -->
    <div id="typing-indicator-{{.Chat.ID}}" class="typing-indicator hidden">
        <span class="typing-dots">
            <span></span><span></span><span></span>
        </span>
        <span id="typing-users"></span> is typing...
    </div>

    <!-- Message Input -->
    {{template "message_form" .}}
</div>

<script>
// Scroll to bottom on load
document.addEventListener('htmx:afterSwap', function(evt) {
    if (evt.detail.target.id === 'messages-{{.Chat.ID}}') {
        scrollToBottom('messages-{{.Chat.ID}}');
    }
});

// Handle incoming WebSocket messages
document.body.addEventListener('chat.message.posted', function(evt) {
    const msg = evt.detail;
    if (msg.chat_id === '{{.Chat.ID}}') {
        htmx.ajax('GET', '/partials/messages/' + msg.message_id, {
            target: '#messages-{{.Chat.ID}}',
            swap: 'beforeend'
        }).then(function() {
            scrollToBottom('messages-{{.Chat.ID}}');
        });
    }
});

// Handle typing indicator
document.body.addEventListener('chat.typing', function(evt) {
    const data = evt.detail;
    if (data.chat_id === '{{.Chat.ID}}' && data.user_id !== '{{.User.ID}}') {
        showTypingIndicator(data.username);
    }
});
</script>
{{end}}
```

### 4. Message Component (message.html)

```html
{{define "message"}}
<article class="message {{if .IsSystemMessage}}system-message{{end}}"
         id="message-{{.ID}}">
    {{if not .IsSystemMessage}}
    <div class="message-avatar">
        {{if .Author.AvatarURL}}
        <img src="{{.Author.AvatarURL}}" alt="{{.Author.Username}}">
        {{else}}
        <div class="avatar-placeholder">
            {{slice .Author.Username 0 1 | upper}}
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="message-content">
        {{if not .IsSystemMessage}}
        <header class="message-header">
            <strong>{{.Author.DisplayName}}</strong>
            <small class="text-muted">
                @{{.Author.Username}}
            </small>
            <time datetime="{{.CreatedAt}}" class="text-muted">
                {{.CreatedAt | formatTime}}
            </time>
            {{if .EditedAt}}
            <small class="text-muted">(edited)</small>
            {{end}}
        </header>
        {{end}}

        <div class="message-body">
            {{.Content | renderMarkdown | safeHTML}}
        </div>

        {{if .Tags}}
        <div class="message-tags">
            {{range .Tags}}
            <span class="tag tag-{{.Key}}">
                #{{.Key}} {{.Value}}
            </span>
            {{end}}
        </div>
        {{end}}

        {{if and (not .IsSystemMessage) (eq .Author.ID $.User.ID)}}
        <footer class="message-actions">
            <button hx-get="/partials/messages/{{.ID}}/edit"
                    hx-target="#message-{{.ID}}"
                    hx-swap="outerHTML"
                    class="small outline">
                Edit
            </button>
            <button hx-delete="/api/v1/messages/{{.ID}}"
                    hx-target="#message-{{.ID}}"
                    hx-swap="outerHTML"
                    hx-confirm="Delete this message?"
                    class="small outline secondary">
                Delete
            </button>
        </footer>
        {{end}}
    </div>
</article>
{{end}}
```

### 5. Message Form (message_form.html)

```html
{{define "message_form"}}
<form class="message-form"
      ws-send
      hx-on::ws-after-send="this.reset(); scrollToBottom('messages-{{.Chat.ID}}')">

    <input type="hidden" name="type" value="chat.send">
    <input type="hidden" name="chat_id" value="{{.Chat.ID}}">

    <div class="message-input-wrapper">
        <textarea name="content"
                  id="message-input-{{.Chat.ID}}"
                  placeholder="Type a message... Use # for tags, @ for mentions"
                  rows="1"
                  required
                  oninput="autoResize(this); handleTyping('{{.Chat.ID}}')"
                  onkeydown="if(event.key==='Enter' && !event.shiftKey) { event.preventDefault(); this.form.requestSubmit(); }"></textarea>

        <!-- Tag autocomplete dropdown -->
        <div id="tag-autocomplete" class="autocomplete-dropdown hidden">
            <ul>
                <li data-tag="#createTask">Create Task</li>
                <li data-tag="#createBug">Create Bug</li>
                <li data-tag="#setStatus">Set Status</li>
                <li data-tag="#assign">Assign to</li>
                <li data-tag="#setPriority">Set Priority</li>
                <li data-tag="#setDueDate">Set Due Date</li>
            </ul>
        </div>
    </div>

    <button type="submit" title="Send message">
        <svg><!-- send icon --></svg>
        Send
    </button>
</form>

<style>
.message-form {
    display: flex;
    gap: 0.5rem;
    padding: 1rem;
    border-top: 1px solid var(--muted-border-color);
    background: var(--background-color);
}

.message-input-wrapper {
    flex: 1;
    position: relative;
}

.message-form textarea {
    width: 100%;
    resize: none;
    min-height: 2.5rem;
    max-height: 10rem;
    margin-bottom: 0;
}

.autocomplete-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    right: 0;
    background: var(--card-background-color);
    border: 1px solid var(--muted-border-color);
    border-radius: 4px;
    box-shadow: 0 -4px 6px rgba(0, 0, 0, 0.1);
    max-height: 200px;
    overflow-y: auto;
}

.autocomplete-dropdown ul {
    list-style: none;
    margin: 0;
    padding: 0;
}

.autocomplete-dropdown li {
    padding: 0.5rem 1rem;
    cursor: pointer;
}

.autocomplete-dropdown li:hover,
.autocomplete-dropdown li.active {
    background: var(--primary-focus);
}

.hidden {
    display: none !important;
}
</style>
{{end}}
```

### 6. Task Sidebar (task-sidebar partial)

```html
{{define "chat/task-sidebar"}}
<div class="task-details">
    <h3>Task Details</h3>

    <!-- Status -->
    <div class="field">
        <label>Status</label>
        <select hx-put="/api/v1/tasks/{{.Task.ID}}/status"
                hx-trigger="change"
                name="status">
            <option value="todo" {{if eq .Task.Status "todo"}}selected{{end}}>To Do</option>
            <option value="in_progress" {{if eq .Task.Status "in_progress"}}selected{{end}}>In Progress</option>
            <option value="review" {{if eq .Task.Status "review"}}selected{{end}}>Review</option>
            <option value="done" {{if eq .Task.Status "done"}}selected{{end}}>Done</option>
        </select>
    </div>

    <!-- Assignee -->
    <div class="field">
        <label>Assignee</label>
        <select hx-put="/api/v1/tasks/{{.Task.ID}}/assignee"
                hx-trigger="change"
                name="assignee_id">
            <option value="">Unassigned</option>
            {{range .Participants}}
            <option value="{{.UserID}}"
                    {{if eq .UserID $.Task.AssigneeID}}selected{{end}}>
                {{.Username}}
            </option>
            {{end}}
        </select>
    </div>

    <!-- Priority -->
    <div class="field">
        <label>Priority</label>
        <select hx-put="/api/v1/tasks/{{.Task.ID}}/priority"
                hx-trigger="change"
                name="priority">
            <option value="low" {{if eq .Task.Priority "low"}}selected{{end}}>Low</option>
            <option value="medium" {{if eq .Task.Priority "medium"}}selected{{end}}>Medium</option>
            <option value="high" {{if eq .Task.Priority "high"}}selected{{end}}>High</option>
            <option value="critical" {{if eq .Task.Priority "critical"}}selected{{end}}>Critical</option>
        </select>
    </div>

    <!-- Due Date -->
    <div class="field">
        <label>Due Date</label>
        <input type="date"
               hx-put="/api/v1/tasks/{{.Task.ID}}/due-date"
               hx-trigger="change"
               name="due_date"
               value="{{if .Task.DueDate}}{{.Task.DueDate | formatDateInput}}{{end}}">
    </div>

    <hr>

    <!-- Activity -->
    <div class="task-activity">
        <h4>Activity</h4>
        <div id="task-activity"
             hx-get="/partials/tasks/{{.Task.ID}}/activity"
             hx-trigger="load"
             hx-swap="innerHTML">
            {{template "loading" (dict "ID" "activity-loading")}}
        </div>
    </div>
</div>
{{end}}
```

### 7. Chat JavaScript (chat.js)

```javascript
/**
 * Chat-specific JavaScript
 */

// Auto-resize textarea
function autoResize(textarea) {
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 160) + 'px';
}

// Scroll to bottom of container
function scrollToBottom(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.scrollTop = element.scrollHeight;
    }
}

// Typing indicator
let typingTimeout;
function handleTyping(chatId) {
    clearTimeout(typingTimeout);

    // Send typing event via WebSocket
    const ws = htmx.find('[hx-ext="ws"]').__htmx_ws;
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            type: 'chat.typing',
            chat_id: chatId
        }));
    }

    typingTimeout = setTimeout(() => {
        // Typing stopped
    }, 1000);
}

function showTypingIndicator(username) {
    const indicator = document.querySelector('.typing-indicator');
    const usersSpan = document.getElementById('typing-users');

    if (indicator && usersSpan) {
        usersSpan.textContent = username;
        indicator.classList.remove('hidden');

        // Hide after 3 seconds
        setTimeout(() => {
            indicator.classList.add('hidden');
        }, 3000);
    }
}

// Tag autocomplete
document.addEventListener('DOMContentLoaded', function() {
    const inputs = document.querySelectorAll('.message-form textarea');

    inputs.forEach(function(input) {
        input.addEventListener('input', function(e) {
            handleTagAutocomplete(e.target);
        });

        input.addEventListener('keydown', function(e) {
            handleAutocompleteNavigation(e);
        });
    });
});

function handleTagAutocomplete(textarea) {
    const value = textarea.value;
    const cursorPos = textarea.selectionStart;
    const textBeforeCursor = value.substring(0, cursorPos);

    // Check if user just typed #
    const hashMatch = textBeforeCursor.match(/#(\w*)$/);

    const dropdown = textarea.parentElement.querySelector('.autocomplete-dropdown');

    if (hashMatch) {
        const filter = hashMatch[1].toLowerCase();
        const items = dropdown.querySelectorAll('li');
        let hasVisible = false;

        items.forEach(function(item) {
            const tag = item.dataset.tag.toLowerCase();
            if (tag.includes(filter) || filter === '') {
                item.style.display = '';
                hasVisible = true;
            } else {
                item.style.display = 'none';
            }
        });

        if (hasVisible) {
            dropdown.classList.remove('hidden');
        } else {
            dropdown.classList.add('hidden');
        }
    } else {
        dropdown.classList.add('hidden');
    }
}

function handleAutocompleteNavigation(e) {
    const dropdown = e.target.parentElement.querySelector('.autocomplete-dropdown');
    if (dropdown.classList.contains('hidden')) return;

    const items = Array.from(dropdown.querySelectorAll('li:not([style*="display: none"])'));
    const active = dropdown.querySelector('li.active');
    let index = items.indexOf(active);

    switch (e.key) {
        case 'ArrowDown':
            e.preventDefault();
            if (active) active.classList.remove('active');
            index = (index + 1) % items.length;
            items[index].classList.add('active');
            break;

        case 'ArrowUp':
            e.preventDefault();
            if (active) active.classList.remove('active');
            index = index <= 0 ? items.length - 1 : index - 1;
            items[index].classList.add('active');
            break;

        case 'Enter':
        case 'Tab':
            if (active) {
                e.preventDefault();
                insertTag(e.target, active.dataset.tag);
                dropdown.classList.add('hidden');
            }
            break;

        case 'Escape':
            dropdown.classList.add('hidden');
            break;
    }
}

function insertTag(textarea, tag) {
    const value = textarea.value;
    const cursorPos = textarea.selectionStart;
    const textBeforeCursor = value.substring(0, cursorPos);
    const textAfterCursor = value.substring(cursorPos);

    // Replace the partial # input with the full tag
    const newText = textBeforeCursor.replace(/#\w*$/, tag + ' ') + textAfterCursor;
    textarea.value = newText;

    // Move cursor after the tag
    const newCursorPos = textBeforeCursor.replace(/#\w*$/, tag + ' ').length;
    textarea.setSelectionRange(newCursorPos, newCursorPos);
    textarea.focus();
}

// WebSocket message handler for HTMX
document.body.addEventListener('htmx:wsAfterMessage', function(evt) {
    try {
        const msg = JSON.parse(evt.detail.message);
        document.body.dispatchEvent(new CustomEvent(msg.type, { detail: msg.data }));
    } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
    }
});
```

---

## Routes

```go
// Chat pages
chat := workspace.Group("/:workspace_id/chats", h.RequireWorkspaceAccess)
chat.GET("", h.ChatLayout)
chat.GET("/:chat_id", h.ChatView)

// Chat partials
partials.GET("/workspace/:workspace_id/chats", h.ChatListPartial)
partials.GET("/chats/:chat_id", h.ChatViewPartial)
partials.GET("/chats/:chat_id/messages", h.MessagesPartial)
partials.GET("/messages/:message_id", h.SingleMessagePartial)
partials.GET("/messages/:message_id/edit", h.MessageEditForm)
partials.GET("/chats/:chat_id/participants", h.ParticipantsPartial)
partials.GET("/chat/create-form", h.ChatCreateForm)
```

---

## Чеклист

### Templates
- [ ] `chat/layout.html` - 3-column layout
- [ ] `chat/list.html` - chat list sidebar
- [ ] `chat/view.html` - messages view
- [ ] `chat/create.html` - create chat form
- [ ] `components/message.html` - message component
- [ ] `components/message_form.html` - message input
- [ ] `components/chat_item.html` - chat list item
- [ ] `components/typing.html` - typing indicator

### JavaScript
- [ ] `chat.js` - typing, autocomplete, scroll

### Handlers
- [ ] `ChatLayout` - chat page with layout
- [ ] `ChatView` - specific chat
- [ ] `ChatListPartial` - chat list
- [ ] `MessagesPartial` - messages list
- [ ] `SingleMessagePartial` - single message
- [ ] `MessageEditForm` - edit message form

### Features
- [ ] Chat list загружается
- [ ] Click на chat показывает сообщения
- [ ] Отправка сообщений работает
- [ ] Real-time сообщения через WebSocket
- [ ] Typing indicator работает
- [ ] Tag autocomplete работает
- [ ] Edit/delete сообщений работает
- [ ] Task sidebar для task chats

---

## Критерии приёмки

- [ ] 3-column layout отображается корректно
- [ ] Chat list показывает чаты workspace
- [ ] Сообщения загружаются при выборе чата
- [ ] Новые сообщения появляются в real-time
- [ ] Typing indicator показывается
- [ ] Tags (#createTask etc.) autocomplete работает
- [ ] Task sidebar показывает и обновляет task details
- [ ] Mobile responsive (single column)

---

## Зависимости

### Входящие
- [03-workspace-pages.md](03-workspace-pages.md) - workspace context
- **ChatService** — реализован (`internal/service/chat_service.go`)
- **Message Use Cases** — реализованы (`internal/application/message/`)
- WebSocket server

### Исходящие
- [05-kanban-board.md](05-kanban-board.md) - task updates sync
- [07-notifications.md](07-notifications.md) - message notifications

---

## WebSocket Events

### Client → Server

```json
// Send message
{"type": "chat.send", "chat_id": "...", "content": "Hello!"}

// Typing indicator
{"type": "chat.typing", "chat_id": "..."}

// Subscribe to chat
{"type": "subscribe.chat", "chat_id": "..."}
```

### Server → Client

```json
// New message
{"type": "chat.message.posted", "data": {"message_id": "...", "chat_id": "...", ...}}

// Message edited
{"type": "chat.message.edited", "data": {"message_id": "...", ...}}

// Message deleted
{"type": "chat.message.deleted", "data": {"message_id": "..."}}

// Typing indicator
{"type": "chat.typing", "data": {"chat_id": "...", "user_id": "...", "username": "..."}}
```

---

*Обновлено: 2026-01-06*
