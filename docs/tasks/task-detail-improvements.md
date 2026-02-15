# Task Detail Improvements

**Priority:** 2 (Feature Completeness)
**Status:** Complete

## Context

Task sidebar opens from board card click and provides status/priority/assignee/due-date editing. Inline title and description editing exist as separate templates. Activity timeline template exists. Several polish items remain.

## Available API

- `GET /api/v1/tasks/:task_id` — Full task details
- `PUT /api/v1/tasks/:task_id/status` — Change status
- `PUT /api/v1/tasks/:task_id/assign` — Assign/unassign
- `PUT /api/v1/tasks/:task_id/priority` — Change priority
- `PUT /api/v1/tasks/:task_id/due-date` — Set/clear due date
- `DELETE /api/v1/tasks/:task_id` — Delete
- Chat actions API for linked chat operations

## Deliverables

### Inline Editing Polish
- [x] Title: click to edit, Enter to save, Escape to cancel
- [x] Description: click to edit, show markdown preview
- [x] Smooth transitions between view/edit modes
- [x] Loading state on save (disable inputs, show spinner)
- [x] Error handling with flash messages

### Activity Timeline
- [x] Render activity items from task events (status changes, assignments, etc.)
- [x] Show actor name, action description, timestamp
- [x] Relative timestamps ("2 hours ago")
- [x] Paginate old activity items

### Due Date Improvements
- [x] Calendar date picker (native `<input type="date">` or custom)
- [x] Visual warnings: overdue (red), due soon (yellow), due today (orange)
- [x] Quick date buttons: Today, Tomorrow, Next Week
- [x] Clear due date option

### Linked Chat
- [x] "Open Chat" button navigates to linked chat
- [ ] Show last few messages preview in sidebar
- [ ] Indicate if chat is active/closed

### Task Deletion
- [x] Delete button with confirmation dialog
- [x] After deletion, close sidebar and remove card from board
- [x] HTMX swap or JS removal

## Technical Notes

- Task sidebar templates: `task/sidebar.html`, `task/edit-title.html`, `task/edit-description.html`, `task/activity.html`
- Sidebar opens/closes via JS, loaded via HTMX partial
- Follow existing HTMX pattern for inline edits (hx-put, hx-swap)
- Activity timeline uses EventStore to load domain events and type-asserts to extract old/new values
- UserLookupService resolves actor IDs to display names for activity items
