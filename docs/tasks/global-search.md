# Global Search

**Priority:** 3 (Quality of Life)
**Status:** Pending

## Context

The navbar has a keyboard shortcut (Ctrl+K / Cmd+K) that focuses a "quick search" element, but no search functionality is implemented. Board has text filter for task titles. A workspace-wide search would improve navigation.

## Deliverables

### Search Modal (Cmd+K)
- [ ] Command palette style modal (centered overlay)
- [ ] Text input with auto-focus
- [ ] Search across: chats (by name), tasks (by title), members (by name)
- [ ] Show results grouped by type with icons
- [ ] Keyboard navigation: arrow keys to select, Enter to open
- [ ] Escape to close
- [ ] Debounced search (300ms) to avoid excessive requests

### Search Backend
- [ ] Verify if search API endpoints exist, or if client-side filtering is sufficient
- [ ] If no search API: use existing list endpoints with name/title query params
- [ ] Chat list: `GET /api/v1/workspaces/:id/chats` may support name filter
- [ ] Task list: `GET /api/v1/workspaces/:id/tasks` may support title filter

### Result Actions
- [ ] Click chat result → navigate to chat view
- [ ] Click task result → open task sidebar on board
- [ ] Click member result → navigate to member profile
- [ ] Recent searches (optional, stored in localStorage)

## Technical Notes

- Reuse modal/dialog pattern from app.js
- Cmd+K shortcut already registered in app.js — wire it to search modal
- Consider client-side search first (if workspace data is small enough)
- No new backend endpoints may be needed initially
