# Task 008: Tag System Critical Fixes

**Status**: In Progress (Partially Fixed)
**Priority**: Critical
**Depends on**: Task 007 (Tag-Based Entity Management)
**Created**: 2026-01-20
**Last Tested**: 2026-01-20

---

## Overview

During testing of Task 007 implementation, several critical issues were discovered that prevent the tag-based entity management system from functioning correctly. This task addresses those issues.

---

## Critical Issues Found

### 1. Concurrency Conflict in Tag Executor (CRITICAL)

**Symptom**: Tag commands fail with "concurrent modification detected" error.

**Server Log Evidence**:
```
WARN msg="concurrency conflict in event store" aggregate_id=xxx expected_version=4 current_version=7
WARN msg="concurrency conflict while saving chat events" chat_id=xxx expected_version=4 events_count=1
```

**Bot Response**: "❌ failed to change status: failed to save chat: concurrent modification detected"

**Root Cause**: The tag executor loads the chat aggregate with a stale version when processing tags asynchronously. By the time it tries to save, the aggregate has been modified elsewhere.

**Location**: `internal/application/message/send_message.go` - `processTagsAsync()` function

**Proposed Fix**:
Option A: Load chat fresh within the executor (recommended)
- The executor should load the chat immediately before modifying, not reuse cached version
- Add retry logic with version refresh on conflict

Option B: Process tags synchronously
- Remove async processing, process tags before returning from SendMessageUseCase
- Simpler but blocks the response

**Files to modify**:
- `internal/application/message/send_message.go`
- `internal/domain/tag/executor.go`

---

### 2. Bot Messages Don't Appear in Real-Time

**Symptom**: After sending a message with tags, the bot response only appears after page refresh.

**Expected**: Bot responses should appear via WebSocket in real-time.

**Root Cause**: Bot messages are created in async goroutine, but WebSocket broadcast may not be working for these messages, OR the HTMX swap is not handling the bot message insertion.

**Files to investigate**:
- `internal/application/message/send_message.go` - bot response creation
- `internal/infrastructure/websocket/broadcaster.go` - event broadcast
- `web/static/js/chat.js` - WebSocket message handling
- `web/templates/components/message.html` - message rendering

---

### 3. Bot User Display Issues

**Symptom**: Bot messages display as:
- Username: "User 00000000" instead of "Flowra Bot"
- Avatar: "0" instead of bot icon
- No visual distinction for bot messages

**Root Cause**: The system bot user (UUID: 00000000-0000-0000-0000-000000000001) doesn't exist in the database or has incorrect display name.

**Proposed Fix**:
1. Create system bot user in database with proper display name "Flowra Bot"
2. OR handle bot user specially in templates (check for bot user ID)
3. Add CSS styling for bot messages (TypeBot)

**Files to modify**:
- `cmd/api/container.go` - ensure bot user creation on startup
- `web/templates/components/message.html` - special rendering for bot
- `web/static/css/custom.css` - bot message styling

---

### 4. Task Details Panel Stuck on Loading

**Symptom**: For some chats (particularly Discussions that were converted to Tasks), the "Task Details" panel shows "Loading task details..." indefinitely.

**Example Chat**: "Tag Test Chat" - shows "D" in sidebar (Discussion) but has "TO DO" status and task-like behavior.

**Root Cause**: Data inconsistency between chat type and task read model. The chat may have task data but is still marked as Discussion type.

**Files to investigate**:
- `internal/handler/http/chat_handler.go` - task details loading
- `internal/infrastructure/repository/mongodb/task_read_model_repository.go`

---

### 5. Double Error Emoji in Bot Responses

**Symptom**: Some error messages show "❌ ❌" (double emoji) instead of single.

**Example**: "❌ ❌ No active entity to modify..."

**Location**: `internal/domain/tag/formatter.go` or response generation code

---

## Implementation Checklist

### Phase 1: Fix Concurrency Bug (Critical)

- [x] Investigate `processTagsAsync()` flow and identify version mismatch cause
- [x] Implement optimistic retry with version refresh (partial - needs fresh reload)
- [x] Add logging for debugging concurrency issues
- [ ] Test status change via tags works correctly (fails after first attempt)
- [ ] Test priority change via tags (fails with concurrency error)
- [ ] Test assignee change via tags (fails with concurrency error)

**Note**: Retry mechanism was added but still fails. Root cause identified: executor must reload chat aggregate fresh on each retry, not use cached version.

### Phase 2: Fix Bot Message Display

- [ ] Ensure bot messages are broadcast via WebSocket (saved but not appearing in real-time)
- [ ] Add HTMX handler for bot message insertion (not working, needs page refresh)
- [x] Create/verify system bot user exists in database
- [x] Add proper display name for bot user (shows as "🤖 Flowra Bot")
- [x] Style bot messages differently in CSS

### Phase 3: Fix Data Consistency

- [ ] Investigate Task Details loading failure (deferred)
- [ ] Check chat type vs task read model consistency (deferred)
- [ ] Fix double emoji in error messages (deferred)

---

## Testing Plan

### Manual Testing
1. Open a Task chat
2. Send `#status Done` message
3. Verify:
   - Status changes in header
   - Status changes in Task Details panel
   - Bot response appears immediately (no refresh needed)
   - Bot shows as "Flowra Bot" with proper styling

### Unit Tests
- Add test for concurrent tag processing
- Add test for bot response generation

### Integration Tests
- Full flow: tag message → status change → bot response → WebSocket broadcast

---

## Files Summary

### Priority 1 (Concurrency Fix)
- `internal/application/message/send_message.go`
- `internal/domain/tag/executor.go`
- `internal/domain/tag/chat_usecases.go`

### Priority 2 (Bot Display)
- `cmd/api/container.go`
- `web/templates/components/message.html`
- `web/static/css/custom.css`
- `web/static/js/chat.js`

### Priority 3 (Minor Fixes)
- `internal/domain/tag/formatter.go`
- `internal/handler/http/chat_handler.go`

---

## Verification Results (2026-01-20)

### ✅ Issue #3: Bot User Display - FIXED
- Bot now shows as "🤖 Flowra Bot" with "Bot" tag
- Visual styling correctly distinguishes bot messages
- System bot user is created on startup (FlowraBot)

### ❌ Issue #1: Concurrency Bug - NOT FIXED
- Retry mechanism was added (5 retries with exponential backoff)
- However, retries still fail with same version mismatch
- Root cause: Retry loads chat but gets same stale version (expected_version=4, current_version=7)
- **Problem**: The executor caches the loaded chat, retry doesn't reload fresh
- First command after server restart works, subsequent commands fail

**Evidence from logs**:
```
WARN msg="concurrency conflict in event store" expected_version=4 current_version=7
WARN msg="concurrency conflict while saving chat events" expected_version=4 events_count=1
(repeated 5 times)
msg="domain event" Content="❌ failed after 5 retries: failed to change status..."
```

**Real fix needed**: Each retry must reload the chat aggregate fresh from the repository to get the current version.

### ❌ Issue #2: Bot Messages Real-Time - NOT FIXED
- Bot messages are created and saved to database
- WebSocket broadcast happens but messages don't appear in UI
- Page refresh required to see bot responses
- **Likely cause**: Frontend WebSocket handler doesn't handle bot messages

### ⏸️ Issue #4: Task Details Loading - NOT TESTED
- Deferred, lower priority

### ⏸️ Issue #5: Double Emoji - NOT TESTED
- Deferred, lower priority

---

## Success Criteria

1. ❌ Tag commands (`#status`, `#priority`, `#assignee`, etc.) execute successfully - **FAILS after first attempt due to concurrency error**
2. ❌ Bot responses appear in real-time via WebSocket - **Messages saved but not appearing in real-time, requires page refresh**
3. ✅ Bot user displays with proper name and styling - **DONE: Shows as "🤖 Flowra Bot" with correct styling**
4. ❌ No concurrency errors in server logs - **Still occurring, retry mechanism needs to reload chat fresh**
5. ⏸️ All existing tests pass - **Not yet verified**
