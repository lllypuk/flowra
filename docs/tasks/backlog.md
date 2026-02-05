# Backlog

Deferred items from completed tasks.

---

## From Task 007 (Tag-Based Entity Management) ✅

### Frontend Integration for Tag System (Complete)
- [x] System messages rendered differently (compact, inline)
- [x] Group consecutive system messages from same actor
- [x] Show "X changed status to Done" instead of raw tag
- [x] Action buttons in UI POST to action endpoints

### Open Design Questions (Resolved)
- [x] Should system messages be collapsible/expandable in UI? → Always visible
- [x] How to handle rapid consecutive changes? → Deferred (see Future Enhancements)
- [x] Should we allow suppressing system messages? → Never suppress
- [x] How to handle changes made via API by external integrations? → Show integration name
- **Details**: `docs/tasks/007-tag-system-frontend-integration.md`

---

## Future Enhancements

### Batch UI Changes (from Task 007)
- [ ] Implement debounce/batching for rapid UI changes
- [ ] Collect changes within 2-second window
- [ ] Generate combined message: "John changed status to X, priority to Y, and assigned to Z"
- **Details**: `docs/tasks/007-tag-system-frontend-integration.md` (Phase 3.5)

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
