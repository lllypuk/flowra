# 08: Polish & Testing

**Приоритет:** 🟢 Medium
**Статус:** ✅ Завершено
**Зависит от:** Все предыдущие задачи (01-07)

---

## Описание

Финальная доработка UI: исправление багов, улучшение UX, accessibility, responsive design, E2E тесты для фронтенда, performance optimization.

---

## Области работы

### 1. Bug Fixes & UX Improvements
### 2. Accessibility (a11y)
### 3. Responsive Design
### 4. Performance Optimization
### 5. E2E Frontend Tests
### 6. Documentation

---

## 1. Bug Fixes & UX Improvements

### Типичные проблемы для проверки

```
□ Flash messages исчезают корректно
□ Loading states показываются везде
□ Error states информативны
□ Empty states имеют call-to-action
□ Модальные окна закрываются по Escape
□ Формы сохраняют состояние при ошибке
□ WebSocket reconnect работает
□ Scroll position сохраняется при navigation
□ Back button работает корректно
□ Deep links работают
```

### UX Improvements Checklist

```
□ Добавить confirmation dialogs где нужно
□ Добавить undo для destructive actions
□ Улучшить feedback при actions
□ Добавить keyboard shortcuts
□ Улучшить form validation messages
□ Добавить progress indicators для long operations
□ Улучшить error recovery
```

---

## 2. Accessibility (a11y)

### WCAG 2.1 AA Compliance

#### Keyboard Navigation

```html
<!-- Все интерактивные элементы должны быть focusable -->
<button tabindex="0">Click me</button>

<!-- Skip links -->
<a href="#main-content" class="skip-link">Skip to main content</a>

<!-- Focus trap в модальных окнах -->
<dialog aria-modal="true">
    <!-- Focus должен оставаться внутри -->
</dialog>
```

#### ARIA Labels

```html
<!-- Кнопки с иконками -->
<button aria-label="Close" title="Close">&times;</button>

<!-- Состояния загрузки -->
<div aria-busy="true" aria-live="polite">Loading...</div>

<!-- Уведомления -->
<div role="alert" aria-live="assertive">Error message</div>

<!-- Навигация -->
<nav aria-label="Main navigation">
    <ul role="menubar">...</ul>
</nav>
```

#### Color Contrast

```css
/* Минимальный контраст 4.5:1 для текста */
:root {
    --text-color: #1a1a1a;       /* На белом фоне */
    --muted-color: #6b7280;      /* 4.5:1 minimum */
    --link-color: #0066cc;       /* Контрастный */
}

/* Не полагаться только на цвет */
.error {
    color: var(--flowra-danger);
    border-left: 3px solid var(--flowra-danger); /* + visual indicator */
}
```

### A11y Testing Checklist

```
□ Keyboard-only navigation работает
□ Screen reader объявляет контент корректно
□ Focus visible на всех элементах
□ Color contrast >= 4.5:1
□ Images имеют alt text
□ Forms имеют labels
□ Error messages связаны с inputs
□ Modals trap focus
□ Dynamic content объявляется
```

---

## 3. Responsive Design

### Breakpoints

```css
/* Mobile first approach */
:root {
    --breakpoint-sm: 576px;
    --breakpoint-md: 768px;
    --breakpoint-lg: 1024px;
    --breakpoint-xl: 1280px;
}

/* Base: Mobile */
.chat-layout {
    display: flex;
    flex-direction: column;
}

/* Tablet */
@media (min-width: 768px) {
    .chat-layout {
        display: grid;
        grid-template-columns: 250px 1fr;
    }
}

/* Desktop */
@media (min-width: 1024px) {
    .chat-layout.with-sidebar {
        grid-template-columns: 250px 1fr 300px;
    }
}
```

### Mobile-specific Features

```html
<!-- Mobile navigation -->
<nav class="mobile-nav">
    <button class="hamburger" aria-label="Open menu">
        ☰
    </button>
</nav>

<!-- Swipe gestures for kanban -->
<div class="board-columns" data-swipe="horizontal">
    <!-- Columns -->
</div>

<!-- Bottom sheet for actions -->
<div class="bottom-sheet" role="dialog">
    <!-- Action buttons -->
</div>
```

### Responsive Checklist

```
□ Mobile: Single column layout
□ Mobile: Hamburger menu
□ Mobile: Touch-friendly buttons (44px min)
□ Mobile: No horizontal scroll
□ Tablet: 2-column layout
□ Desktop: 3-column layout
□ All: Text readable without zoom
□ All: Forms usable on all devices
```

---

## 4. Performance Optimization

### Loading Performance

```html
<!-- Preload critical assets -->
<link rel="preload" href="/static/css/custom.css" as="style">
<link rel="preload" href="https://unpkg.com/htmx.org@2.0.0" as="script">

<!-- Lazy load images -->
<img src="placeholder.jpg"
     data-src="actual-image.jpg"
     loading="lazy"
     alt="...">

<!-- Defer non-critical JS -->
<script src="/static/js/app.js" defer></script>
```

### HTMX Optimization

```html
<!-- Использовать hx-boost для SPA-like navigation -->
<body hx-boost="true">

<!-- Preload часто используемых partial -->
<link rel="preload" href="/partials/notifications" as="fetch">

<!-- Limit swap scope -->
<div hx-get="/data" hx-select=".result-only">
```

### Caching Strategy

```go
// Static assets with cache headers
e.Static("/static", "web/static", middleware.WithCacheControl("public, max-age=31536000"))

// API responses with etag
e.GET("/api/v1/workspaces", handler, middleware.WithETag())

// HTML pages no-cache
e.GET("/workspaces", handler, middleware.WithNoCache())
```

### Performance Checklist

```
□ First Contentful Paint < 1.5s
□ Time to Interactive < 3s
□ Static assets cached
□ Images optimized
□ CSS/JS minified in production
□ Gzip enabled
□ No render-blocking resources
□ Lazy loading for images
```

---

## 5. E2E Frontend Tests

### Test Framework

```go
// tests/e2e/frontend_test.go

//go:build e2e

package e2e

import (
    "testing"

    "github.com/playwright-community/playwright-go"
)

func TestFrontend_LoginFlow(t *testing.T) {
    pw, _ := playwright.Run()
    browser, _ := pw.Chromium.Launch()
    page, _ := browser.NewPage()

    // Navigate to login
    page.Goto("http://localhost:8080/login")

    // Click SSO button
    page.Click("text=Sign in with SSO")

    // Complete Keycloak login
    page.Fill("#username", "testuser")
    page.Fill("#password", "password")
    page.Click("#kc-login")

    // Verify redirect to workspaces
    page.WaitForURL("**/workspaces")

    // Check user menu
    expect(page.Locator(".user-menu")).ToContainText("testuser")
}

func TestFrontend_CreateWorkspace(t *testing.T) {
    page := loginAsTestUser(t)

    // Click create button
    page.Click("text=+ New Workspace")

    // Fill form
    page.Fill("input[name=name]", "Test Workspace")
    page.Fill("textarea[name=description]", "Test description")

    // Submit
    page.Click("text=Create Workspace")

    // Verify workspace appears
    expect(page.Locator(".workspace-card")).ToContainText("Test Workspace")
}

func TestFrontend_ChatRealtime(t *testing.T) {
    // Open two browser contexts
    user1Page := loginAsUser(t, "alice")
    user2Page := loginAsUser(t, "bob")

    // Both open same chat
    chatURL := "/workspaces/test/chats/test-chat"
    user1Page.Goto(chatURL)
    user2Page.Goto(chatURL)

    // User1 sends message
    user1Page.Fill("textarea[name=content]", "Hello from Alice!")
    user1Page.Press("textarea[name=content]", "Enter")

    // Verify User2 sees message (real-time)
    expect(user2Page.Locator(".message").Last()).ToContainText("Hello from Alice!")
}

func TestFrontend_KanbanDragDrop(t *testing.T) {
    page := loginAsTestUser(t)
    page.Goto("/workspaces/test/board")

    // Find task card
    taskCard := page.Locator(".task-card").First()
    doneColumn := page.Locator("[data-status=done] .column-cards")

    // Drag to Done column
    taskCard.DragTo(doneColumn)

    // Verify status updated
    expect(page.Locator("[data-status=done] .task-card")).ToHaveCount(1)
}
```

### Test Scenarios

```
Auth:
□ Login via Keycloak
□ Logout
□ Session expiry handling
□ Protected route redirect

Workspace:
□ Create workspace
□ Edit workspace name
□ Add/remove members
□ Delete workspace

Chat:
□ Create chat
□ Send message
□ Real-time message delivery
□ Edit message
□ Delete message
□ Typing indicator

Board:
□ View kanban board
□ Drag and drop
□ Filter by type/assignee
□ Real-time updates

Task:
□ Create task from chat
□ Edit task details
□ Change status via dropdown
□ View activity

Notifications:
□ Receive notification
□ Mark as read
□ Click to navigate
```

---

## 6. Documentation

### User Guide

```markdown
# Flowra User Guide

## Getting Started
1. Login with your organization SSO
2. Create or join a workspace
3. Start chatting!

## Features

### Chat
- Send messages with Markdown support
- Use tags like #createTask to create tasks
- @mention users to notify them

### Kanban Board
- Drag tasks between columns to change status
- Click a card to see task details
- Filter by type, assignee, or priority

### Keyboard Shortcuts
- `Ctrl+K` - Quick search
- `Ctrl+Enter` - Send message
- `Escape` - Close modal
```

### Developer Guide

```markdown
# Frontend Development Guide

## Tech Stack
- HTMX 2.0 for dynamic updates
- Pico CSS v2 for styling
- Go html/template for SSR

## Directory Structure
web/
├── templates/     # HTML templates
├── static/        # CSS, JS assets
└── embed.go       # Static file embedding

## Adding a New Page
1. Create template in `web/templates/`
2. Add handler in `template_handler.go`
3. Register route in `RegisterRoutes()`
4. Add tests

## HTMX Patterns
- Use `hx-get` for loading content
- Use `hx-post` for form submissions
- Use `hx-swap` to control where content goes
- Use `ws-connect` for WebSocket
```

---

## Чеклист

### Bug Fixes
- [ ] Все известные баги исправлены
- [ ] Edge cases обработаны
- [ ] Error recovery работает

### Accessibility
- [ ] Keyboard navigation
- [ ] Screen reader support
- [ ] Color contrast
- [ ] ARIA labels

### Responsive
- [ ] Mobile layout
- [ ] Tablet layout
- [ ] Desktop layout
- [ ] Touch-friendly

### Performance
- [ ] FCP < 1.5s
- [ ] Assets cached
- [ ] Images optimized
- [ ] No layout shifts

### Testing
- [ ] E2E tests для основных flows
- [ ] Cross-browser testing
- [ ] Mobile device testing

### Documentation
- [ ] User guide
- [ ] Developer guide
- [ ] API documentation updated

---

## Критерии приёмки

- [ ] Все E2E тесты проходят
- [ ] Lighthouse score > 90 (Performance, Accessibility, Best Practices)
- [ ] Работает в Chrome, Firefox, Safari
- [ ] Работает на iOS и Android
- [ ] Нет critical/high severity багов
- [ ] Документация актуальна

---

## Browser Support

| Browser | Version | Status |
|---------|---------|--------|
| Chrome | 90+ | ✅ Primary |
| Firefox | 88+ | ✅ Supported |
| Safari | 14+ | ✅ Supported |
| Edge | 90+ | ✅ Supported |
| Mobile Chrome | Latest | ✅ Supported |
| Mobile Safari | iOS 14+ | ✅ Supported |

---

## Definition of Done

Фронтенд считается завершённым когда:

1. **Функциональность**
   - Все страницы реализованы
   - Все features работают
   - Real-time updates работают

2. **Качество**
   - E2E tests проходят
   - Нет known bugs
   - Performance acceptable

3. **UX**
   - Responsive design
   - Accessible
   - Intuitive navigation

4. **Documentation**
   - User guide готов
   - Developer guide готов

---

*Обновлено: 2026-01-06*
