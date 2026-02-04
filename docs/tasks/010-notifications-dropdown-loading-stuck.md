# Task 010: Notifications Dropdown Stuck on "Loading..."

**Status**: Complete
**Priority**: Medium
**Completed**: 2026-02-04
**Depends on**: None
**Created**: 2026-02-04
**Discovered**: Frontend testing with agent-browser

---

## Overview

When clicking the notifications bell icon in the navbar, the dropdown opens but displays "Loading..." indefinitely. Notifications never load and the dropdown remains stuck.

---

## Symptoms

1. Click notifications bell icon in navbar
2. Dropdown opens with "Loading..." text
3. Never transitions to actual notifications list
4. No error messages shown to user
5. No JavaScript console errors (silent failure)

**Screenshot evidence**: `/tmp/12-notifications.png` from frontend testing session

---

## Technical Analysis

### HTMX Request Flow

```
User clicks notifications bell
  ↓
navbar.html <details> element opens
  ↓
HTMX GET: /partials/notifications?limit=10
  (hx-trigger="toggle once")
  ↓
NotificationsDropdownPartial() handler
  ├─ Fetch notifications from service
  ├─ Call renderPartial("notification/dropdown-content", data)
  │   ├─ Template lookup
  │   ├─ Template execution
  │   └─ Return error if failed
  └─ Error not properly handled for HTMX
      ↓
HTMX receives non-HTML error response
  ↓
innerHTML not replaced → "Loading..." persists
```

### Template Trigger

**File**: `web/templates/layout/navbar.html` (lines 38-48)

```html
<details role="list" class="dropdown-notifications">
    <summary aria-haspopup="listbox" role="button" class="secondary">
        <!-- Bell icon -->
    </summary>
    <ul role="listbox"
        hx-get="/partials/notifications?limit=10"
        hx-trigger="toggle once"
        hx-swap="innerHTML">
        <li>Loading...</li>
    </ul>
</details>
```

### Handler Implementation

**File**: `internal/handler/http/notification_template_handler.go` (lines 125-176)

```go
func (h *NotificationTemplateHandler) NotificationsDropdownPartial(c echo.Context) error {
    userID, ok := c.Get("user_id").(uuid.UUID)
    if !ok {
        return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
    }

    query := notificationapp.ListQuery{
        UserID: userID,
        Limit:  10,
        Offset: 0,
    }

    result, err := h.notificationService.ListNotifications(c.Request().Context(), query)
    if err != nil {
        h.logger.Error("failed to list notifications", slog.String("error", err.Error()))
        // Returns empty list on service error - GOOD
        return h.renderPartial(c, "notification/dropdown-content", NotificationListData{
            Notifications: []NotificationViewData{},
            UnreadCount:   0,
        })
    }

    // ... build data ...

    return h.renderPartial(c, "notification/dropdown-content", data)
}
```

### Template Definition

**File**: `web/templates/notification/dropdown.html` (line 54)

```html
{{define "notification/dropdown-content"}}
    <!-- Template content -->
{{end}}
```

### Render Partial Method

**File**: `internal/handler/http/notification_template_handler.go` (lines 308-316)

```go
func (h *NotificationTemplateHandler) renderPartial(c echo.Context, templateName string, data any) error {
    if h.renderer == nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "template renderer not configured")
    }

    c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
    return h.renderer.Render(c.Response().Writer, templateName, data, c)
}
```

---

## Root Causes

### 1. Template Render Error Not Caught (HIGH)

If `renderPartial()` fails (template not found, execution error), the error propagates to Echo's error handler which returns a non-HTML response. HTMX can't process this.

**File**: `internal/handler/http/template_handler.go` (lines 139-179)

```go
func (r *TemplateRenderer) Render(...) error {
    tmpl := r.templates.Lookup(name)
    if tmpl == nil {
        r.logger.Error("template not found", slog.String("name", name))
        return fmt.Errorf("template %q not found", name)  // Returns error
    }

    if err := tmpl.Execute(w, data); err != nil {
        r.logger.Error("template execution failed", ...)
        return err  // Returns error - NOT HTML!
    }
    return nil
}
```

### 2. No HTMX-Friendly Error Response (HIGH)

When template rendering fails, HTMX expects HTML content. Instead it receives:
- HTTP 500 error
- Error message in text/plain
- HTMX can't swap this into the DOM

### 3. Silent Failure - No User Feedback (MEDIUM)

Unlike the chat task details (which shows toast), the notifications dropdown fails silently because:
- HTMX `hx-trigger="toggle once"` only fires once
- No retry mechanism
- No error indicator in the dropdown

### 4. Possible Template Not Loaded (LOW)

The template `notification/dropdown-content` might not be included in the template bundle during initialization.

---

## Affected Files

| File | Line | Issue |
|------|------|-------|
| `web/templates/layout/navbar.html` | 38-48 | HTMX trigger definition |
| `internal/handler/http/notification_template_handler.go` | 125-176 | Handler implementation |
| `internal/handler/http/notification_template_handler.go` | 308-316 | renderPartial method |
| `internal/handler/http/template_handler.go` | 139-179 | Render method |
| `web/templates/notification/dropdown.html` | 54 | Template definition |

---

## Implementation Plan

### Phase 1: Add Error Recovery to Handler

```go
func (h *NotificationTemplateHandler) NotificationsDropdownPartial(c echo.Context) error {
    // ... existing code ...

    err := h.renderPartial(c, "notification/dropdown-content", data)
    if err != nil {
        h.logger.Error("failed to render notifications", slog.String("error", err.Error()))
        // Return fallback HTML for HTMX
        return c.HTML(http.StatusOK, `<li class="error-state">Failed to load notifications</li>`)
    }
    return nil
}
```

### Phase 2: Create Error State Template

- [ ] Create `notification/dropdown-error.html` partial
- [ ] Show user-friendly message: "Could not load notifications"
- [ ] Add "Retry" button with hx-get to retry loading

### Phase 3: Add Retry Mechanism

Update navbar.html to add retry on error:

```html
<ul role="listbox"
    hx-get="/partials/notifications?limit=10"
    hx-trigger="toggle once"
    hx-swap="innerHTML"
    hx-target="this"
    hx-on::response-error="this.innerHTML = '<li>Failed to load. <a href=\'#\' hx-get=\'/partials/notifications\' hx-swap=\'innerHTML\' hx-target=\'closest ul\'>Retry</a></li>'">
```

### Phase 4: Verify Template Loading

- [ ] Check template bundle includes `notification/dropdown-content`
- [ ] Add startup validation for critical templates
- [ ] Log warning if template missing during initialization

---

## Testing Plan

### Manual Testing

1. Start fresh server
2. Login as testuser
3. Click notifications bell
4. Verify notifications load (or show empty state)
5. Create a notification (e.g., assign task to user)
6. Refresh and verify notification appears

### Error Testing

1. Stop notification service (if separate)
2. Click notifications bell
3. Verify error message appears (not stuck on Loading)
4. Verify retry option works

### Edge Cases

1. User with no notifications - show "No notifications"
2. User not authenticated - handle 401 gracefully
3. Template render error - show error state

---

## Success Criteria

1. [x] Notifications dropdown loads successfully
2. [x] Empty state shows "No notifications" message
3. [x] Error state shows user-friendly message
4. [x] Retry mechanism available on error
5. [x] No silent failures - always show feedback
