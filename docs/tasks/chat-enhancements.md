# Chat Enhancements

**Priority:** 1 (Core Gap)
**Status:** Pending

## Context

Chat is the core feature. Send/list messages, typing indicators, tag autocomplete, and WebSocket subscriptions all work. However, several expected features from the User Guide are incomplete or missing in the UI.

## 1. @Mention Autocomplete

The message form supports `#tag` autocomplete but `@mention` autocomplete is not implemented.

- [ ] Add `@` trigger in message textarea (same pattern as tag autocomplete)
- [ ] Fetch workspace members list for suggestions (use workspace members API or cache from WS)
- [ ] Show dropdown with user avatars + display names
- [ ] Insert `@username` on selection
- [ ] Style mentions in rendered messages (highlight, link to profile)

**API available:** `GET /api/v1/workspaces/:workspace_id/members` (via workspace members partial) or member data already available in chat context.

## 2. Message Editing

Template `message_edit` exists in `components/message.html` but the full edit flow needs wiring.

- [ ] "Edit" button on own messages triggers inline edit mode
- [ ] Replace message body with textarea pre-filled with original content
- [ ] Save button sends `PUT /api/v1/messages/:id`
- [ ] Cancel button restores original message
- [ ] Show "(edited)" indicator after successful edit
- [ ] HTMX swap to update message in place

**API available:** `PUT /api/v1/messages/:id` — edit message content.

## 3. Message Deletion

- [ ] "Delete" button on own messages with confirmation dialog
- [ ] Send `DELETE /api/v1/messages/:id`
- [ ] Replace message with "This message was deleted" state (already styled in template)
- [ ] Use `hx-confirm` or custom confirm dialog

**API available:** `DELETE /api/v1/messages/:id` — soft-delete.

## 4. Message Reactions (Optional / Lower Priority)

Template includes reactions display but no way to add them.

- [ ] Add reaction button (emoji picker or preset emojis)
- [ ] Toggle reaction on/off
- [ ] Update count in real-time via WebSocket

**Note:** Reactions API may not exist yet — verify backend support before implementing.

## 5. Real-time Message Updates via WebSocket

WebSocket sends `message_posted`, `message_edited` events.

- [ ] Handle `message_edited` event — update message content in DOM
- [ ] Handle `message_deleted` event — update message to deleted state
- [ ] Handle presence updates — show online indicators in chat participant list

## 6. Chat Participant Management

- [ ] "Participants" button in chat header opens modal/sidebar
- [ ] Show list of participants with online status
- [ ] Add participant (search workspace members, POST to API)
- [ ] Remove participant (DELETE to API, with confirmation)

**API available:**
- `POST /api/v1/chats/:id/participants` — add participant
- `DELETE /api/v1/chats/:id/participants/:user_id` — remove participant
- `GET /api/v1/chats/:id/presence` — online status

## Technical Notes

- All changes should follow existing HTMX patterns in chat.js
- Message editing should use `hx-swap="outerHTML"` to replace the message element
- Mention autocomplete can reuse the tag autocomplete dropdown component with minor adaptation
- Test with multiple browser tabs to verify real-time behavior
