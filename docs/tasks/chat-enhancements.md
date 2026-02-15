# Chat Enhancements

**Priority:** 1 (Core Gap)
**Status:** Complete ✅

## Context

Chat is the core feature. Send/list messages, typing indicators, tag autocomplete, and WebSocket subscriptions all work. However, several expected features from the User Guide are incomplete or missing in the UI.

## 1. @Mention Autocomplete ✅

The message form now supports `@mention` autocomplete alongside tag autocomplete.

- [x] Add `@` trigger in message textarea (same pattern as tag autocomplete)
- [x] Fetch workspace members list for suggestions (uses workspace members API)
- [x] Show dropdown with user avatars + display names
- [x] Insert `@username` on selection
- [ ] Style mentions in rendered messages (optional: requires server-side parsing)

**API available:** `GET /partials/workspace/:id/members-options` provides member data.

**Implementation:** `web/static/js/chat.js` + `web/static/css/custom.css`

## 2. Message Editing ✅

Template `message_edit` exists in `components/message.html` and the full edit flow is wired.

- [x] "Edit" button on own messages triggers inline edit mode
- [x] Replace message body with textarea pre-filled with original content
- [x] Save button sends `PUT /api/v1/messages/:id`
- [x] Cancel button restores original message
- [x] Show "(edited)" indicator after successful edit
- [x] HTMX swap to update message in place
- [x] Fixed API routes to support direct `/messages/:id` endpoints

**API available:** `PUT /api/v1/messages/:id` — edit message content.

**Implementation:** Templates already complete. Route fix in `cmd/api/routes.go`.

## 3. Message Deletion ✅

- [x] "Delete" button on own messages with confirmation dialog
- [x] Send `DELETE /api/v1/messages/:id`
- [x] Replace message with "This message was deleted" state (already styled in template)
- [x] Use `hx-confirm` for confirmation
- [x] Fixed API routes to support direct `/messages/:id` endpoints

**API available:** `DELETE /api/v1/messages/:id` — soft-delete.

**Implementation:** Templates already complete. Route fix in `cmd/api/routes.go`.

## 4. Message Reactions (Optional / Lower Priority)

Template includes reactions display but no way to add them.

- [ ] Add reaction button (emoji picker or preset emojis)
- [ ] Toggle reaction on/off
- [ ] Update count in real-time via WebSocket

**Note:** Reactions API may not exist yet — verify backend support before implementing.

## 5. Real-time Message Updates via WebSocket ✅

WebSocket sends `message_posted`, `message_edited`, `message_deleted` events.

- [x] Handle `message_edited` event — update message content in DOM
- [x] Handle `message_deleted` event — remove message from DOM
- [x] Handle presence updates — show online indicators in chat participant list

**Implementation:** Already complete in `web/templates/chat/view.html`.

## 6. Chat Participant Management ✅

- [x] "Participants" button in chat header opens modal
- [x] Show list of participants with online status
- [x] Add participant (search workspace members, POST to API)
- [x] Remove participant (DELETE to API, with confirmation)
- [x] Display role badges (creator, admin, member)
- [x] Real-time online presence indicators

**API available:**
- `POST /api/v1/chats/:id/participants` — add participant
- `DELETE /api/v1/chats/:id/participants/:user_id` — remove participant
- `GET /api/v1/chats/:id/presence` — online status

**Implementation:** Already complete in `web/templates/chat/participants.html`.

## Technical Notes

- All changes should follow existing HTMX patterns in chat.js
- Message editing should use `hx-swap="outerHTML"` to replace the message element
- Mention autocomplete can reuse the tag autocomplete dropdown component with minor adaptation
- Test with multiple browser tabs to verify real-time behavior
