# Frontend Development Guide

This guide covers frontend development for Flowra using HTMX, Pico CSS, and Go templates.

## Tech Stack

| Technology | Version | Purpose |
|------------|---------|---------|
| HTMX | 2.0 | Dynamic content updates without JavaScript |
| Pico CSS | v2 | Minimal CSS framework |
| Go html/template | stdlib | Server-side rendering |
| WebSocket | HTMX ext | Real-time updates |

## Directory Structure

```
web/
├── templates/           # HTML templates
│   ├── layout/         # Base layout templates
│   │   ├── base.html   # Main HTML5 template
│   │   ├── navbar.html # Navigation bar (dark mode toggle, global search)
│   │   └── footer.html # Footer
│   ├── components/     # Reusable HTMX components
│   │   ├── message.html         # Chat message rendering
│   │   ├── message_form.html    # Message input + file upload
│   │   ├── task_card.html       # Kanban task card
│   │   ├── user_search_results.html  # User autocomplete results
│   │   ├── user_select.html     # User picker component
│   │   ├── date_picker.html     # Date picker
│   │   └── member_row.html      # Workspace member row
│   ├── auth/          # Authentication pages
│   ├── workspace/     # Workspace pages (list, settings, invite, transfer, members)
│   ├── chat/          # Chat pages
│   │   ├── view.html            # Main chat view
│   │   ├── task-sidebar.html    # Task sidebar in chat (see note below)
│   │   └── participants.html    # Participant list
│   ├── board/         # Kanban board
│   │   └── filters.html         # Board filter panel
│   ├── task/          # Task detail views
│   │   ├── sidebar.html         # Task sidebar on board (see note below)
│   │   ├── activity.html        # Activity timeline
│   │   ├── create-form.html     # Task creation form
│   │   ├── edit-title.html      # Inline title editing
│   │   └── edit-description.html # Inline description editing
│   ├── user/          # User pages
│   │   ├── profile.html         # User profile view
│   │   └── settings.html        # User settings
│   ├── notification/  # Notification components
│   └── home.html      # Landing page (standalone, no base.html)
├── static/             # Static assets
│   ├── css/
│   │   ├── custom.css  # Custom styles (includes dark mode vars)
│   │   └── board.css   # Board-specific styles
│   └── js/
│       ├── app.js      # Core JavaScript (IIFE + guard pattern)
│       ├── chat.js     # Chat: typing indicators, autocomplete, presence (IIFE + guard)
│       └── board.js    # Kanban drag-and-drop, real-time updates (guard flag)
└── embed.go            # Go embed for static files
```

> **Task Sidebar Note:** There are two separate task sidebar templates that must be kept in sync:
> - `web/templates/task/sidebar.html` — used by the board/task detail panel (loaded as HTMX partial)
> - `web/templates/chat/task-sidebar.html` — used in the chat view right panel

## Adding a New Page

### 1. Create the Template

Create a new template in the appropriate directory:

```html
{{define "my-page"}}
{{template "base" .}}
{{define "content"}}
<article>
    <header>
        <h1>{{.Title}}</h1>
    </header>

    <section>
        <!-- Page content -->
    </section>
</article>
{{end}}
{{end}}
```

### 2. Create the Handler

Add a handler in `internal/handler/http/`:

```go
// MyPageHandler renders the my page template.
func (h *TemplateHandler) MyPage(c echo.Context) error {
    data := PageData{
        Title: "My Page",
        User:  h.getUserView(c),
        // Add page-specific data
    }
    return h.render(c, "my-page", data)
}
```

### 3. Register the Route

Add the route in `routes.go`:

```go
// In registerPageRoutes()
protected.GET("/my-page", c.TemplateHandler.MyPage)
```

### 4. Add Tests

Create tests for your handler:

```go
func TestMyPage_Renders(t *testing.T) {
    // ... test implementation
}
```

## HTMX Patterns

### Loading Content Dynamically

```html
<!-- Load content on page load -->
<div hx-get="/partials/data"
     hx-trigger="load"
     hx-swap="innerHTML">
    <div class="loading-spinner">
        <div class="spinner"></div>
        <span>Loading...</span>
    </div>
</div>
```

### Form Submission

```html
<form hx-post="/api/items"
      hx-swap="beforeend"
      hx-target="#items-list"
      hx-on::after-request="this.reset()">
    <input type="text" name="name" required>
    <button type="submit" data-loading-text="Saving...">
        Save
    </button>
</form>
```

### Real-time Updates with WebSocket

See also:

- `docs/api/websocket-protocol.md` for the current server/client message protocol, event envelope, and reconnection notes
- `docs/api/action-endpoints.md` for UI action routes often used together with real-time updates

```html
<div hx-ext="ws"
     ws-connect="/api/v1/ws?token={{.Token}}"
     ws-send>

    <!-- Messages container -->
    <div id="messages"
         hx-swap-oob="beforeend:#messages">
    </div>

    <!-- Message input -->
    <form ws-send>
        <textarea name="content"></textarea>
        <button type="submit">Send</button>
    </form>
</div>
```

### Infinite Scroll

```html
<div id="items-list">
    {{range .Items}}
    <div class="item">{{.Name}}</div>
    {{end}}

    {{if .HasMore}}
    <button hx-get="/partials/items?offset={{.NextOffset}}"
            hx-swap="outerHTML"
            hx-trigger="revealed">
        Load More
    </button>
    {{end}}
</div>
```

### Confirmation Dialogs

```html
<button hx-delete="/api/items/123"
        data-confirm="Are you sure you want to delete this item?"
        hx-swap="outerHTML swap:1s"
        hx-target="closest .item">
    Delete
</button>
```

### Loading States

```html
<button hx-post="/api/action"
        data-loading-text="Processing..."
        hx-indicator=".spinner">
    Submit
    <span class="spinner htmx-indicator"></span>
</button>
```

## Component Patterns

### Flash Messages

```html
{{define "flash"}}
{{if .Flash}}
<article class="flash flash-{{.Flash.Type}}" role="alert">
    <button class="close" aria-label="Dismiss">&times;</button>
    {{.Flash.Message}}
</article>
{{end}}
{{end}}
```

### Empty State

```html
<div class="empty-state">
    <div class="empty-icon">📭</div>
    <h3>No items yet</h3>
    <p>Create your first item to get started.</p>
    <button hx-get="/partials/create-form"
            hx-target="#modal-container">
        Create Item
    </button>
</div>
```

### Loading Skeleton

```html
<div class="skeleton">
    <div class="skeleton-line"></div>
    <div class="skeleton-line medium"></div>
    <div class="skeleton-line short"></div>
</div>
```

### Error State

```html
<div class="error-state" role="alert">
    <div class="error-icon">⚠️</div>
    <p class="error-message">Failed to load data</p>
    <button class="retry-btn"
            hx-get="/partials/data"
            hx-target="closest .error-state"
            hx-swap="outerHTML">
        Try Again
    </button>
</div>
```

## Styling Guidelines

### Using CSS Variables

```css
/* Use Pico CSS variables */
.my-component {
    background: var(--pico-background-color);
    color: var(--pico-color);
    border: 1px solid var(--pico-muted-border-color);
}

/* Use Flowra custom variables */
.status-success {
    color: var(--flowra-success);
}
```

### Dark Mode

Dark mode is toggled via the `data-theme` attribute on `<html>`. Pico CSS handles the color switch automatically. Custom styles use CSS variables that respect both themes. The user's preference is persisted in `localStorage` and applied on page load via a script in `base.html`.

### Responsive Design

```css
/* Mobile first approach */
.my-layout {
    display: flex;
    flex-direction: column;
}

/* Tablet and up */
@media (min-width: 768px) {
    .my-layout {
        flex-direction: row;
    }
}

/* Desktop */
@media (min-width: 1024px) {
    .my-layout {
        max-width: 1200px;
        margin: 0 auto;
    }
}
```

### Utility Classes

```html
<!-- Spacing -->
<div class="mt-2 mb-1 p-2">...</div>

<!-- Flexbox -->
<div class="flex items-center justify-between gap-2">...</div>

<!-- Grid -->
<div class="grid grid-cols-3">...</div>

<!-- Visibility -->
<div class="hide-mobile show-desktop">...</div>
```

## Accessibility Checklist

When creating new components:

- [ ] Add proper ARIA labels to interactive elements
- [ ] Ensure keyboard navigation works
- [ ] Provide visible focus states
- [ ] Include alt text for images
- [ ] Associate labels with form inputs
- [ ] Use semantic HTML elements
- [ ] Test with screen reader
- [ ] Check color contrast (4.5:1 minimum)

### ARIA Examples

```html
<!-- Button with icon -->
<button aria-label="Close dialog" title="Close">
    <span aria-hidden="true">&times;</span>
</button>

<!-- Loading state -->
<div aria-busy="true" aria-live="polite">
    Loading...
</div>

<!-- Alert -->
<div role="alert" aria-live="assertive">
    Error: Something went wrong
</div>

<!-- Navigation -->
<nav aria-label="Main navigation">
    <ul role="menubar">...</ul>
</nav>
```

## Testing

### Unit Tests

```go
func TestTemplateHandler_RenderPage(t *testing.T) {
    handler := NewTemplateHandler(renderer, logger, nil, nil)

    req := httptest.NewRequest(http.MethodGet, "/page", nil)
    rec := httptest.NewRecorder()
    c := echo.New().NewContext(req, rec)

    err := handler.MyPage(c)
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, rec.Code)
}
```

### E2E Tests

```go
func TestFrontend_PageLoads(t *testing.T) {
    page := suite.newPage(t)
    defer page.Close()

    _, err := page.Goto(baseURL + "/my-page")
    require.NoError(t, err)

    element := page.Locator("h1")
    visible, err := element.IsVisible()
    require.True(t, visible)
}
```

## Performance Tips

1. **Use hx-swap wisely** - Prefer `innerHTML` over `outerHTML` when possible
2. **Lazy load content** - Use `hx-trigger="revealed"` for below-fold content
3. **Cache static assets** - Static files are cached with max-age headers
4. **Minimize DOM updates** - Use `hx-select` to swap only needed content
5. **Defer non-critical JS** - All JS files use `defer` attribute

## Debugging

### HTMX Debug Mode

Enable in browser console:

```javascript
htmx.logAll();
```

### Check WebSocket Connection

HTMX v2 stores the WebSocket connection differently from v1:

```javascript
// ✅ HTMX v2 — correct path to the socket
var el = document.querySelector('[hx-ext*="ws"]');
var internalData = el['htmx-internal-data'];
var wsWrapper = internalData && internalData.webSocket;
var socket = wsWrapper && wsWrapper.socket;
if (socket) {
    console.log('readyState:', socket.readyState); // 0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED
}

// ❌ HTMX v1 — broken in v2
element.__htmx_ws; // null in HTMX v2
```

WebSocket events dispatched on `document.body` contain a `data` field with PascalCase Go field names. Always handle both styles for robustness:

```javascript
document.body.addEventListener("chat.message.posted", function(evt) {
    var msg = evt.detail;
    var chatId = msg.ChatID || msg.chat_id;
    var messageId = msg.aggregate_id || msg.message_id;
});
```

### Template Rendering Issues

Check the server logs for template parsing errors:

```bash
go run cmd/api/main.go 2>&1 | grep -i template
```

## JavaScript Patterns

### IIFE + Guard for Script Files

All JS files that may be loaded via `hx-boost` navigation must follow the IIFE + guard pattern to prevent double-initialization:

```javascript
(function() {
    if (window.__myJsLoaded) return;
    window.__myJsLoaded = true;

    // Functions called from templates must be on window:
    window.myFunction = function() { ... };

    // Internal helpers stay private:
    function helperFunction() { ... }
})();
```

Without this guard, `hx-boost` navigation re-evaluates `<script>` tags and causes `Identifier has already been declared` errors for top-level `const`/`let` declarations.

### Echo Handler Struct Tags

Request structs must include **both** `json:` and `form:` tags. HTMX sends requests as `application/x-www-form-urlencoded`; without the `form:` tag, Echo's `Bind()` silently ignores form fields.

```go
// ✅ Supports both JSON API calls and HTMX form submissions
var req struct {
    Status string `json:"status" form:"status"`
}

// ❌ HTMX form data will not be bound (req.Status stays empty → 400)
var req struct {
    Status string `json:"status"`
}
```

### File Upload Pattern

File uploads use a two-step flow:

1. POST file to `/api/v1/files/upload` → get file ID
2. POST file ID to `/api/v1/messages/{id}/attachments` (or task equivalent)
3. Re-fetch the message partial via `outerHTML` swap to display the attachment

### Chat Action Route URLs

Chat action routes are workspace-scoped. Always include `workspace_id`:

```
✅ POST /api/v1/workspaces/:workspace_id/chats/:id/actions/status
❌ POST /api/v1/chats/:id/actions/status  (returns 404)
```

## Common Issues

### HTMX Not Triggering

- Check the element has proper `hx-*` attributes
- Verify the endpoint returns HTML
- Check for JavaScript errors in console

### Styles Not Applying

- Check CSS class names match
- Verify CSS file is loaded (Network tab)
- Check for CSS specificity issues

### WebSocket Not Connecting

- Verify the token is valid
- Check WebSocket endpoint URL
- Look for connection errors in console
