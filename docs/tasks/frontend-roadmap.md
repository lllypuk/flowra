# Frontend Development Roadmap

Current frontend status: ~30% (per CLAUDE.md) — core pages exist but many features are stubs or incomplete.

Backend is fully production-ready with 70+ API endpoints. Frontend needs to catch up.

## Task Index

### Priority 1 — Core Gaps (blocking RC)
- [x] [user-profile.md](user-profile.md) — User profile & settings page ✅
- [x] [chat-enhancements.md](chat-enhancements.md) — Mention autocomplete, message editing, reactions ✅
- [x] [notification-improvements.md](notification-improvements.md) — Real-time updates, better UX ✅

### Priority 2 — Feature Completeness
- [x] [workspace-settings.md](workspace-settings.md) — Full workspace admin UI ✅
- [x] [board-enhancements.md](board-enhancements.md) — Task search, filters, bulk operations ✅
- [x] [task-detail-improvements.md](task-detail-improvements.md) — Inline editing polish, activity timeline ✅

### Priority 3 — Quality of Life
- [x] [dark-mode.md](dark-mode.md) — Dark mode toggle UI ✅
- [x] [global-search.md](global-search.md) — Workspace-wide search ✅
- [file-uploads.md](file-uploads.md) — Attachments for messages and tasks

## API Coverage Matrix

Which backend APIs have frontend UI vs which are API-only:

| Domain | API Endpoints | Frontend Coverage | Notes |
|--------|--------------|-------------------|-------|
| Auth | 4 | Full | Login, logout, refresh, callback all wired |
| Workspaces | 8 | Partial | CRUD done, member role editing incomplete |
| Chats | 8 + 7 actions | Mostly done | Actions wired via sidebar, presence works |
| Messages | 4 | Partial | Send/list done, edit/delete UI incomplete |
| Tasks | 8 | Mostly done | Board + sidebar cover most operations |
| Notifications | 5 | Partial | List/mark-read done, real-time push missing |
| Users | 3 | Minimal | GET me used in navbar, no profile page |
| WebSocket | 1 | Connected | Presence + typing work, notification push incomplete |

## Frontend File Inventory

### Templates (web/templates/)
- `layout/` — base.html, navbar.html, footer.html
- `auth/` — login.html, callback.html, logout.html
- `workspace/` — list.html, view.html, members.html, settings.html, create.html, invite.html
- `chat/` — view.html, create.html
- `board/` — index.html, column.html, filters.html
- `task/` — sidebar.html, form.html, create-form.html, edit-title.html, edit-description.html, activity.html
- `notification/` — list.html, dropdown.html, item.html, empty.html, list_partial.html
- `components/` — message.html, message_form.html, task_card.html, user_select.html, date_picker.html, loading.html, flash.html, notification_badge.html, typing.html, empty.html, workspace_card.html, member_row.html, activity_item.html, chat_item.html
- `home.html` — Landing page (standalone)

### Static Assets (web/static/)
- `js/app.js` — Core: HTMX handlers, toasts, WS reconnect, modals, keyboard shortcuts, scroll management
- `js/chat.js` — Chat: textarea resize, typing indicators, tag autocomplete
- `js/board.js` — Board: drag-and-drop, task status API calls
- `css/custom.css` — Global styles, utilities, responsive, accessibility, animations
- `css/board.css` — Board/kanban-specific styles
