# Backlog

Deferred items from completed tasks.

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
