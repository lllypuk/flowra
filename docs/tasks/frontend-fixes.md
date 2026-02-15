# Frontend Fixes

Issues discovered during testing of all frontend-roadmap.md deliverables.

## Priority: Critical (Blocking)

### ~~1. Message template crash — `.Author.UserID` field mismatch~~ ✅ FIXED
- **Fix applied**: Changed `.Author.UserID` to `.Author.ID` in `web/templates/components/message.html:21`.
- **Verified**: Messages render correctly in all chats.

### ~~2. User Settings page — form fields are empty, Account Created shows error~~ ✅ FIXED
- **Fix applied**: Rewrote `UserSettings` handler to always build a PascalCase map from `getUserView()` result, instead of passing the raw lowercase-keyed auth context map.
- **Verified**: All fields (Username, Display Name, Email, User ID, Account Created, Last Updated) display correctly. Avatar shows initial "T".

### ~~3. Notifications page renders blank (0 bytes response)~~ ✅ FIXED
- **Fix applied**: Two issues fixed in `web/templates/notification/list.html`:
  1. Template define name `"notification/list"` didn't match handler's lookup `"notification/list.html"` — added `.html` suffix to define.
  2. `{{if not .Data.Filter}}` failed because `not` expects bool but Filter is a string — changed to `{{if eq .Data.Filter ""}}`.
- **Verified**: Page renders with title, filter dropdown, and empty state message.

## Priority: High

### 4. Notification dropdown stuck on "Loading..."
- **Symptom**: Clicking the notification bell opens dropdown but content stays as "Loading..." spinner indefinitely.
- **Root cause**: The `<ul>` element has `hx-trigger="toggle once"` but the `toggle` event fires on the `<details>` element, not on child elements. The `<ul>` inside `<details>` never receives a `toggle` event, so HTMX never fires the request to `/partials/notifications?limit=10`.
- **Fix**: Move `hx-get` and `hx-trigger` attributes to the `<details>` element, or use a different trigger like `hx-trigger="intersect once"` on the `<ul>`, or use JS to manually trigger on details toggle.
- **Files**: `web/templates/layout/navbar.html:44-53`

### 5. Participants modal shows "No participants yet" despite participant count showing "1"
- **Symptom**: Chat header shows participant count "1", but clicking it opens a modal showing "No participants yet" with only a "Close" button.
- **Root cause**: The participants partial endpoint likely returns empty data, or the HTMX request to load participants fails silently. The modal is also missing the "Add participant" search/invite functionality described in chat-enhancements.md.
- **Fix**: Debug the `/partials/chats/:chat_id/participants` endpoint to ensure it returns participants. Add the participant add/remove UI.
- **Files**: `internal/handler/http/chat_template_handler.go` (ParticipantsPartial), `web/templates/components/` (participants template)

### 6. User Profile page shows hardcoded mock data
- **Symptom**: `/users/:id` page shows "User Name", "@usera9c9f2ad", "user@example.com" — all hardcoded.
- **Root cause**: The `UserProfile` handler returns mock data with comment "For now, return a mock user profile — In a full implementation, this would fetch from UserService".
- **Fix**: Replace mock data with actual user lookup from `UserService` or user repository.
- **Files**: `internal/handler/http/template_handler.go:558-600`

### 7. Members page shows truncated user ID instead of real username
- **Symptom**: The member row shows "User a9c9f2ad @usera9c9f2ad" instead of actual display name and username. Same issue appears in the board assignee filter dropdown ("usera9c9f2ad").
- **Root cause**: The member data uses a generated/truncated user ID as display name and username instead of querying real user details from the user repository.
- **Fix**: Ensure workspace member queries resolve actual user details (display name, username) from the user store.
- **Files**: `internal/handler/http/template_handler.go` (WorkspaceMembers handler), `internal/handler/http/board_template_handler.go` (assignee filter)

## Priority: Medium

### 8. JavaScript error: `updateOnlineCount is not defined`
- **Symptom**: Console error on chat pages.
- **Root cause**: A function call to `updateOnlineCount()` exists in JS code but the function is not defined.
- **Fix**: Define the `updateOnlineCount` function or remove the call.
- **Files**: `web/static/js/chat.js` or `web/static/js/app.js`

### 9. JavaScript error: `Cannot read properties of undefined (reading 'close')`
- **Symptom**: Console error, likely when trying to close a modal or dialog that doesn't exist.
- **Root cause**: Code attempts to call `.close()` on an undefined element reference.
- **Fix**: Add null check before calling `.close()`.
- **Files**: `web/static/js/app.js` or `web/static/js/chat.js`

### 10. Chat presence endpoint returns 404
- **Symptom**: `GET /api/v1/chats/:id/presence` returns 404.
- **Root cause**: The presence endpoint is not registered in routes, or the handler is not implemented.
- **Fix**: Register the presence route or remove the client-side call if presence is handled via WebSocket only.
- **Files**: `cmd/api/routes.go`, `web/static/js/chat.js`

### 11. Navbar username disappears after re-login via SSO redirect
- **Symptom**: After SSO redirect login (not first login), the username button in navbar is empty (no text). Shows up correctly on subsequent navigations.
- **Root cause**: The `getUserView` may return incomplete data during the redirect callback before the full user context is established.
- **Fix**: Ensure the auth callback handler fully populates user context before redirect.
- **Files**: `internal/handler/http/auth_middleware.go`, `internal/handler/http/template_handler.go`
