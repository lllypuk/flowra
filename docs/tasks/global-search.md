# Global Search

**Priority:** 3 (Quality of Life)
**Status:** Complete ✅

## Context

The navbar has a keyboard shortcut (Ctrl+K / Cmd+K) that focuses a "quick search" element, but no search functionality is implemented. Board has text filter for task titles. A workspace-wide search would improve navigation.

## Deliverables

### Search Modal (Cmd+K)
- [x] Command palette style modal (centered overlay)
- [x] Text input with auto-focus
- [x] Search across: chats (by name), tasks (by title)
- [x] Show results grouped by type with icons
- [x] Keyboard navigation: arrow keys to select, Enter to open
- [x] Escape to close
- [x] Debounced search (300ms) to avoid excessive requests

### Search Backend
- [x] Verify if search API endpoints exist, or if client-side filtering is sufficient
- [x] If no search API: use existing list endpoints with name/title query params
- [x] Chat list: `GET /api/v1/workspaces/:id/chats` — fetched client-side
- [x] Task list: `GET /api/v1/workspaces/:id/tasks` — fetched client-side

### Result Actions
- [x] Click chat result → navigate to chat view
- [x] Click task result → navigate to board with task param
- [ ] Click member result → navigate to member profile (deferred — member data uses placeholder names)
- [ ] Recent searches (optional, stored in localStorage) (deferred)

## Technical Notes

- Reuse modal/dialog pattern from app.js
- Cmd+K shortcut already registered in app.js — wire it to search modal
- Consider client-side search first (if workspace data is small enough)
- No new backend endpoints may be needed initially
