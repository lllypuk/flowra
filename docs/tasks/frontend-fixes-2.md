# Frontend Fixes — Round 2

Issues discovered during browser testing of all frontend-roadmap.md deliverables (2026-02-15).

## Priority: Critical (Blocking)

### ~~1. User Settings save fails — PUT /api/v1/users/me returns 501 Not Implemented~~ **[DONE]**
- **Where**: `/settings` page → "Save Changes" button
- **Symptom**: Error alert "User service not available" appears above the form. Two duplicate "Server error. Please try again later." toast notifications appear.
- **Root cause**: The `PUT /api/v1/users/me` API endpoint returns HTTP 501 (Not Implemented). The handler likely has a stub or the user update use case is not wired.
- **Console error**: `Response Status Error Code 501 from /api/v1/users/me`
- **Expected**: Saving display name and email should succeed and reflect changes.
- **Fix**: Wired `UserHandler` in `container.go` with `userServiceAdapter` delegating to real use cases.

### ~~2. Message Edit button fails — GET /partials/messages/{id}/edit returns 500~~ **[DONE]**
- **Where**: Any chat → click "Edit" button on a message
- **Symptom**: Multiple "Server error. Please try again later." toast notifications appear. No inline editor is shown.
- **Root cause**: The `GET /partials/messages/{id}/edit` endpoint returns HTTP 500 Internal Server Error.
- **Console error**: `Response Status Error Code 500 from /partials/messages/{id}/edit`
- **Expected**: Clicking Edit should show an inline text editor for the message content.
- **Fix**: Replaced `{{slice .Author.Username 0 1 | upper}}` with safe `{{initials}}` call in `message_edit` template.

### ~~3. Message author shows "User a9c9f2ad" instead of username~~ **[DONE]**
- **Where**: All chat message views (`/workspaces/{id}/chats/{id}`)
- **Symptom**: Message headers show "User a9c9f2ad" (truncated UUID) and "@a9c9f2ad" instead of "testuser" and "@testuser".
- **Root cause**: The message component template is not resolving the user's display name/username from the user repository. It falls back to a generated name from the first 8 chars of the UUID.
- **Files to check**: `web/templates/components/message.html`, message handler that populates author data.
- **Expected**: Messages should display the actual username and display name.
- **Fix**: Added `userLookup.GetUser()` call in `convertMessageToView()` to resolve author names from DB.

### ~~4. Task activity shows "Unknown" for all actors~~ **[DONE]**
- **Where**: Task detail sidebar → Activity section (e.g., `/workspaces/{id}/chats/{chat_id}` for task chats)
- **Symptom**: All activity entries show **Unknown** instead of the user who performed the action (e.g., "Unknown changed status In Review → Done").
- **Root cause**: The activity template or handler does not resolve user IDs to display names.
- **Files to check**: `web/templates/task/activity.html`, activity handler/partial.
- **Expected**: Activity entries should show the actual username (e.g., "testuser changed status...").
- **Fix**: `resolveUsername()` already uses `userLookup`; root cause was `getUserView()` not populating user context — fixed via robust `getUserView()` in `TaskDetailTemplateHandler`.

## Priority: High

### ~~5. Global search returns no results~~ **[DONE]**
- **Where**: Press Ctrl+K → type any query (e.g., "Test", "Task")
- **Symptom**: Always shows "No results for [query]" even though matching chats and tasks exist (e.g., "Test Task 1", "Test Chat from Browser", "Test bug").
- **Root cause**: The search dialog fetches data from `/api/v1/workspaces/{id}/chats` and `/api/v1/workspaces/{id}/tasks` (both return 200), but the client-side filtering/matching logic appears broken. The API responses may be in a format the JS code doesn't expect, or the search matching function has a bug.
- **Files to check**: `web/static/js/app.js` (search modal logic), check how the fetched JSON is parsed and filtered.
- **Expected**: Searching "Test" should return "Test Task 1", "Test Chat from Browser", "Test bug", etc.
- **Fix**: Fixed JSON response parsing — API wraps data in `{ "data": {...} }`, JS now checks `data.data.chats` with fallback.

### ~~6. Board search does not filter tasks visually~~ **[DONE]**
- **Where**: Board page → "Search tasks..." input field
- **Symptom**: Typing "Test" in the search box shows a "1" badge (indicating 1 active filter) but all 3 task cards remain visible across all columns. Tasks that don't match the search term ("ddddd sd sfd", "sd fsd few e we") are not hidden.
- **Root cause**: The board partial endpoint receives the search param (`?search=Test`) but the server-side filtering doesn't apply the search to task names, or the HTMX response still returns all tasks.
- **Files to check**: Board handler that processes `search` query param, task repository query logic.
- **Expected**: Only tasks matching the search term should be displayed.
- **Fix**: Added `filterTasksBySearch()` for post-fetch case-insensitive title matching in `buildColumns()` and `BoardColumnMore()`.

## Priority: Medium

### ~~7. Duplicate toast notifications on errors~~ **[DONE]**
- **Where**: Multiple locations (user settings save, message edit, etc.)
- **Symptom**: When an HTMX request fails, two identical "Server error. Please try again later." toast notifications appear simultaneously.
- **Root cause**: Likely double event handling — both the HTMX `htmx:responseError` event and another error handler are both creating toasts for the same error.
- **Files to check**: `web/static/js/app.js` (toast/error handling logic).
- **Expected**: Only one toast notification per error.
- **Fix**: Added toast deduplication — same message within 2 seconds is skipped.

### ~~8. Chat participants loading error~~ **[DONE]**
- **Where**: Server logs when opening a task chat
- **Symptom**: `failed to load participants: validation failed: validation error on field 'requestedBy': must be a valid UUID`
- **Root cause**: The `requestedBy` field is not being set to a valid UUID when loading chat participants. Likely the user ID is empty or malformed in the request context.
- **Expected**: Participants should load without validation errors.
- **Fix**: Added `userID` variadic parameter to `loadParticipants()` and pass `RequestedBy` in `GetChatQuery`. Updated all 3 call sites.

### ~~9. JS error: "Cannot read properties of undefined (reading 'close')"~~ **[DONE]**
- **Where**: Board page, triggered when interacting with modals/dialogs
- **Symptom**: JS console error `Cannot read properties of undefined (reading 'close')`.
- **Note**: This was reportedly fixed in round 1 (issue #9) but still reproduces on the board page. The fix may be incomplete or there's a different code path triggering it.
- **Files to check**: `web/static/js/app.js`, `web/static/js/board.js` — dialog close handlers.
- **Fix**: Added `window.closeTaskForm` function that closes the nearest open `<dialog>`.

### ~~10. Navbar username missing on Board page~~ **[DONE]**
- **Where**: Board page (`/workspaces/{id}/board`)
- **Symptom**: The user button in the top-right navbar shows empty text (no username displayed), while on other pages it shows "testuser".
- **Root cause**: The Board page template may use a different layout or the user context is not being passed correctly to the navbar partial on this page.
- **Files to check**: `web/templates/layout/navbar.html`, board page template, board handler.
- **Expected**: Username should always be visible in the navbar.
- **Fix**: Replaced minimal `getUserView()` in `BoardTemplateHandler` and `TaskDetailTemplateHandler` with robust implementation that extracts username/email/display name from context.
