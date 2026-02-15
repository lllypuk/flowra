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

### ~~8. JavaScript error: `updateOnlineCount is not defined`~~ ✅ FIXED
- **Fix applied**: Exposed `updateOnlineCount` function on `window` in `web/static/js/chat.js` so it's accessible from HTML template inline handlers.
- **Files changed**: `web/static/js/chat.js`

### ~~9. JavaScript error: `Cannot read properties of undefined (reading 'close')`~~ ✅ FIXED
- **Fix applied**: Added null check before calling `.close()` on `this.closest('dialog')` result in keyboard shortcuts help button (app.js).
- **Files changed**: `web/static/js/app.js`

### ~~10. Chat presence endpoint returns 404~~ ✅ FIXED
- **Fix applied**: Removed REST API calls to non-existent `/api/v1/chats/:id/presence` endpoint from `participants.html`. Replaced with a call to `window.updateOnlineCount()` which uses WebSocket-based presence state already tracked in chat.js.
- **Files changed**: `web/templates/chat/participants.html`

### ~~11. Navbar username disappears after re-login via SSO redirect~~ ✅ FIXED
- **Fix applied**: Enhanced `getUserView` in `template_handler.go` to fall back to individual middleware context keys (`middleware.GetUsername`, `middleware.GetEmail`) when the "user" context map is missing or has empty fields. Also ensures `DisplayName` is always populated by falling back to username or email.
- **Files changed**: `internal/handler/http/template_handler.go`
