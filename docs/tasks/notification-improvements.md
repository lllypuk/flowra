# Notification Improvements

**Priority:** 1 (Core Gap)
**Status:** Complete

## Context

Notifications page exists with list/filter/mark-read functionality. Navbar has a notification bell with badge. However, real-time push and overall UX needs work.

## Available API

- `GET /api/v1/notifications` — List with optional `unread=true` filter, pagination
- `GET /api/v1/notifications/unread/count` — Unread count for badge
- `PUT /api/v1/notifications/:id/read` — Mark single as read
- `PUT /api/v1/notifications/mark-all-read` — Mark all as read
- `DELETE /api/v1/notifications/:id` — Delete notification
- WebSocket event: `notification` — Real-time notification push

## Deliverables

### Real-time Notification Push
- [x] Handle WebSocket `notification` event in app.js
- [x] Increment badge count on new notification
- [x] Show toast notification with message preview
- [x] If notification dropdown is open, prepend new notification to list
- [ ] Play subtle sound or browser notification (with user permission) — *Not implemented (optional)*

### Navbar Dropdown Improvements
- [x] Lazy-load dropdown content on first open (already partially done)
- [x] Add "View all" link to full notifications page
- [x] Mark notification as read on click (before navigating)
- [x] Show notification type icon (mention, assignment, status change)
- [x] Truncate long notification text with ellipsis

### Notification List Page
- [x] Improve filter dropdown styling (All / Unread / Mentions / Assignments)
- [x] Add pagination or infinite scroll for long lists
- [x] Click notification to navigate to relevant item (chat message, task, etc.)
- [x] Delete individual notifications with swipe or button
- [x] Empty state when no notifications match filter

### Badge Count
- [x] Poll unread count on page load (fallback if WS disconnected)
- [x] Update badge in real-time via WebSocket
- [x] Hide badge when count is 0
- [x] Animate badge on increment

## Technical Notes

- WebSocket already connects in app.js with reconnect logic
- Notification dropdown uses HTMX lazy loading (`hx-trigger="toggle"`)
- Use existing toast system (`window.showToast()`) for notification popups
- Badge element already exists in navbar template
