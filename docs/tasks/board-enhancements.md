# Board & Task List Enhancements

**Priority:** 2 (Feature Completeness)
**Status:** Complete

## Context

Kanban board works with drag-and-drop between columns, task cards with priority/type indicators, and a filter bar. Task sidebar opens on card click with status/priority/assignee/due-date editing. Several enhancements are needed for full feature coverage.

## Available API

- `GET /api/v1/workspaces/:workspace_id/tasks` — List with filters (status, assignee, priority, type) and pagination
- `PUT /api/v1/tasks/:task_id/status` — Change status
- `PUT /api/v1/tasks/:task_id/assign` — Assign/unassign
- `PUT /api/v1/tasks/:task_id/priority` — Change priority
- `PUT /api/v1/tasks/:task_id/due-date` — Set/clear due date
- `DELETE /api/v1/tasks/:task_id` — Delete task

## Deliverables

### Board Filters
- [x] Verify all filter dropdowns work (type, assignee, priority)
- [x] Text search filter — client-side filtering by task title
- [x] "Clear filters" button
- [x] Persist filter state in URL query params or sessionStorage
- [x] Show active filter count badge

### Task Creation from Board
- [x] "New Task" button opens modal (already partially exists)
- [x] Quick-create form: title, type, priority, assignee
- [x] After creation, card appears in correct column without page reload
- [x] HTMX swap to prepend card to target column

### Real-time Board Updates
- [x] Handle WebSocket `task_updated` event
- [x] Move card between columns when status changes (from another user)
- [x] Update card content when priority/assignee changes
- [x] Add new card when task created by another user
- [x] Remove card when task deleted by another user

### Load More / Pagination
- [x] "Load more" button at bottom of columns with many tasks
- [x] Cursor-based pagination for task loading
- [x] Show total count per column in header

### Board View Options (Optional)
- [x] Compact card view (title + priority only)
- [x] Sort within columns (by priority, due date, created date)

## Technical Notes

- Drag-and-drop in board.js uses native HTML5 API
- Real-time updates need WebSocket event handlers in board.js
- Filters already have HTML in `board/filters.html` — verify they send requests
- Task creation modal likely in `task/create-form.html`
