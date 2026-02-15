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

### ~~4. Notification dropdown stuck on "Loading..."~~ ✅ FIXED
- **Fix applied**: Changed `hx-trigger="toggle once"` to `hx-trigger="intersect once"` on the `<ul>` element in `web/templates/layout/navbar.html:48`. The `toggle` event fires on `<details>`, not on its child `<ul>`, so HTMX never triggered the request. Using `intersect` triggers when the dropdown list becomes visible.
- **Verified**: Build passes, no regressions.

### ~~5. Participants modal shows "No participants yet" despite participant count showing "1"~~ ✅ FIXED
- **Fix applied**: Implemented `loadParticipants()` in `chat_template_handler.go` to load real participant data from `ChatTemplateService.GetChat()`. Added `UserProfileLookup` interface and injected it into `ChatTemplateHandler` to resolve participant usernames and display names from the user repository.
- **Files changed**: `internal/handler/http/chat_template_handler.go`, `cmd/api/container.go`

### ~~6. User Profile page shows hardcoded mock data~~ ✅ FIXED
- **Fix applied**: Replaced mock data in `UserProfile` handler with real user lookup via `UserProfileLookup` service. Returns 404 if user is not found instead of showing fake data.
- **Files changed**: `internal/handler/http/template_handler.go`

### ~~7. Members page shows truncated user ID instead of real username~~ ✅ FIXED
- **Fix applied**: Added `resolveMemberView()` helper to `TemplateHandler` that resolves actual user details via `UserProfileLookup`. Updated `WorkspaceMembersPartial`, `WorkspaceMembersOptionsPartial`, and `TransferOwnershipForm` to use it. Also updated `boardMemberServiceAdapter` in `container.go` to resolve real usernames from the user repository.
- **Files changed**: `internal/handler/http/template_handler.go`, `cmd/api/container.go`

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
