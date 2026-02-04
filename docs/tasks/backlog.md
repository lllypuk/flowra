# Backlog

Deferred items from completed tasks.

---

## From Task 004 (Outbox Pattern)

### Add Outbox Metrics and Monitoring
- [x] Add Prometheus metrics for outbox lag
- [x] Dashboard for outbox backlog visualization
- Priority: Low
- **Details**: `docs/tasks/004-outbox-metrics-monitoring.md`
- **Completed**: 2026-02-04

---

## From Task 007 (Tag-Based Entity Management)

### Frontend Integration for Tag System
- [ ] System messages rendered differently (compact, inline)
- [ ] Group consecutive system messages from same actor
- [ ] Show "X changed status to Done" instead of raw tag
- [ ] Action buttons in UI POST to action endpoints
- Priority: Medium

### Open Design Questions
- [ ] Should system messages be collapsible/expandable in UI?
- [ ] How to handle rapid consecutive changes (batch into single message)?
- [ ] Should we allow suppressing system messages for automated/bulk operations?
- [ ] How to handle changes made via API by external integrations?
- Priority: Low
- **Details**: `docs/tasks/007-tag-system-frontend-integration.md`

---

## From Task 008 (Tag System Fixes)

**Note**: Task 008 is complete. Critical fixes (concurrency bug, bot real-time messages) were implemented 2026-01-21.
See `docs/tasks/008-tag-system-fixes.md` for details.

### Remaining Items
- [x] Investigate Task Details loading failure → **Fixed in Task 009**
- [x] Check chat type vs task read model consistency → **Fixed in Task 009**
- [x] Fix double emoji in error messages → **Verified: Not an issue** (formatter works correctly)
- [ ] Fix Cyrillic comments in tag system files (35+ comments to translate)
- Priority: Low

---

## From Frontend Testing (2026-02-04)

### Task 009: Chat Task Details "Resource Not Found"
- [ ] Add WorkspaceID to Task ReadModel (deferred - using chatInfo instead)
- [x] Implement participants loading for assignee dropdown
- [x] Improve error handling (return HTML, not 404)
- [x] Handle Discussion chats (don't load task sidebar)
- Priority: Medium
- **Completed**: 2026-02-04
- **Details**: `docs/tasks/009-chat-task-details-resource-not-found.md`

### Task 010: Notifications Dropdown Loading Stuck
- [x] Add error recovery to notification handler
- [x] Create error state template for dropdown
- [x] Add retry mechanism on error
- [x] Verify template bundle includes required partials
- Priority: Medium
- **Completed**: 2026-02-04
- **Details**: `docs/tasks/010-notifications-dropdown-loading-stuck.md`

---

## Future Enhancements

### WebSocket Improvements
- [ ] Connection status indicator in UI
- [ ] Reconnection logic with exponential backoff
- [ ] Presence indicators (who's online in chat)
- **Details**: `docs/tasks/011-websocket-improvements.md`

### Testing
- [ ] Add integration tests for concurrent tag processing
- [ ] Add E2E tests for bot response flow
- [ ] Load testing for tag system under high concurrency

### Documentation
- [ ] API documentation for action endpoints
- [ ] WebSocket protocol documentation
- [ ] Tag system user guide
