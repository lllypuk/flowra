# UseCase Implementation Progress Tracker

Используйте этот файл для отслеживания прогресса реализации UseCase слоя.

**Дата начала:** ___________
**Целевая дата завершения:** ___________
**Фактическая дата завершения:** ___________

## Overall Progress

```
[░░░░░░░░░░░░░░░░░░░░] 0% Complete

Phase 1: Architecture         [░░░░░░░░░░] 0/4
Phase 2: Chat UseCases        [░░░░░░░░░░] 0/12
Phase 3: Message UseCases     [░░░░░░░░░░] 0/6
Phase 4: User UseCases        [░░░░░░░░░░] 0/3
Phase 5: Workspace UseCases   [░░░░░░░░░░] 0/5
Phase 6: Notification UseCases[░░░░░░░░░░] 0/4
Phase 7: Integration Testing  [░░░░░░░░░░] 0/5
Phase 8: Tag Integration      [░░░░░░░░░░] 0/3
```

---

## Phase 1: Architecture (Task 01) - 3-4 hours

**Status:** 🔴 Not Started | 🟡 In Progress | 🟢 Complete

### Shared Components

- [ ] `internal/application/shared/interfaces.go`
  - [ ] UseCase interface
  - [ ] Command interface
  - [ ] Query interface
  - [ ] Result structs
  - [ ] EventSourcedResult struct
  - [ ] Validator interface
- [ ] `internal/application/shared/errors.go`
  - [ ] Common errors (validation, authorization, not found, conflict)
  - [ ] ValidationError type
  - [ ] AuthorizationError type
  - [ ] NotFoundError type
- [ ] `internal/application/shared/context.go`
  - [ ] GetUserID / WithUserID
  - [ ] GetWorkspaceID / WithWorkspaceID
  - [ ] GetCorrelationID / WithCorrelationID
  - [ ] GetTraceID / WithTraceID
- [ ] `internal/application/shared/validation.go`
  - [ ] ValidateRequired
  - [ ] ValidateUUID
  - [ ] ValidateMaxLength
  - [ ] ValidateMinLength
  - [ ] ValidateEnum
  - [ ] ValidateDateNotPast
  - [ ] ValidateDateRange

### Tests
- [ ] Unit tests для validation helpers
- [ ] Unit tests для context helpers

**Time spent:** _____ hours

---

## Phase 2: Chat UseCases (Task 02) - 6-8 hours

**Status:** 🔴 Not Started

### Structure Files
- [ ] `commands.go` (12 команд)
- [ ] `queries.go` (3 запроса)
- [ ] `results.go`
- [ ] `errors.go`

### Command UseCases
- [ ] **CreateChatUseCase**
  - [ ] Implementation
  - [ ] Unit tests (success, validation errors, authorization)
  - [ ] Event publishing verification
- [ ] **AddParticipantUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **RemoveParticipantUseCase**
  - [ ] Implementation
  - [ ] Unit tests (cannot remove last admin)
- [ ] **ConvertToTaskUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ConvertToBugUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ConvertToEpicUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ChangeStatusUseCase**
  - [ ] Implementation
  - [ ] Unit tests (status validation per type)
- [ ] **AssignUserUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **SetPriorityUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **SetDueDateUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **RenameChatUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **SetSeverityUseCase**
  - [ ] Implementation
  - [ ] Unit tests (only for Bug type)

### Query UseCases
- [ ] **GetChatUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ListChatsUseCase**
  - [ ] Implementation
  - [ ] Unit tests (pagination, filtering)
- [ ] **ListParticipantsUseCase**
  - [ ] Implementation
  - [ ] Unit tests

**Time spent:** _____ hours

---

## Phase 3: Message UseCases (Task 03) - 5-7 hours

**Status:** 🔴 Not Started

### Structure Files
- [ ] `commands.go`
- [ ] `queries.go`
- [ ] `results.go`
- [ ] `errors.go`

### Command UseCases
- [ ] **SendMessageUseCase**
  - [ ] Implementation
  - [ ] Unit tests (participant check, parent message validation)
  - [ ] Integration with Tag Parser (placeholder)
- [ ] **EditMessageUseCase**
  - [ ] Implementation
  - [ ] Unit tests (author check)
- [ ] **DeleteMessageUseCase**
  - [ ] Implementation
  - [ ] Unit tests (soft delete, author check)
- [ ] **AddReactionUseCase**
  - [ ] Implementation
  - [ ] Unit tests (deduplication)
- [ ] **RemoveReactionUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **AddAttachmentUseCase**
  - [ ] Implementation
  - [ ] Unit tests (file size validation)

### Query UseCases
- [ ] **GetMessageUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ListMessagesUseCase**
  - [ ] Implementation
  - [ ] Unit tests (pagination)
- [ ] **GetThreadUseCase**
  - [ ] Implementation
  - [ ] Unit tests

**Time spent:** _____ hours

---

## Phase 4: User UseCases (Task 04) - 3-4 hours

**Status:** 🔴 Not Started

### Structure Files
- [ ] `commands.go`
- [ ] `queries.go`
- [ ] `results.go`
- [ ] `errors.go`

### Command UseCases
- [ ] **RegisterUserUseCase**
  - [ ] Implementation
  - [ ] Unit tests (username uniqueness)
- [ ] **UpdateProfileUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **PromoteToAdminUseCase**
  - [ ] Implementation
  - [ ] Unit tests (system admin check)

### Query UseCases
- [ ] **GetUserUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **GetUserByUsernameUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ListUsersUseCase**
  - [ ] Implementation
  - [ ] Unit tests (pagination)

**Time spent:** _____ hours

---

## Phase 5: Workspace UseCases (Task 05) - 4-5 hours

**Status:** 🔴 Not Started

### Structure Files
- [ ] `commands.go`
- [ ] `queries.go`
- [ ] `results.go`
- [ ] `errors.go`

### Command UseCases
- [ ] **CreateWorkspaceUseCase**
  - [ ] Implementation
  - [ ] Keycloak group creation
  - [ ] Unit tests
  - [ ] Rollback on failure
- [ ] **UpdateWorkspaceUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **CreateInviteUseCase**
  - [ ] Implementation
  - [ ] Unit tests (token generation, expiration)
- [ ] **AcceptInviteUseCase**
  - [ ] Implementation
  - [ ] Unit tests (validation, max uses, expiration)
  - [ ] Keycloak group membership
- [ ] **RevokeInviteUseCase**
  - [ ] Implementation
  - [ ] Unit tests

### Query UseCases
- [ ] **GetWorkspaceUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ListUserWorkspacesUseCase**
  - [ ] Implementation
  - [ ] Unit tests

### Keycloak Integration
- [ ] Keycloak client interface
- [ ] Mock Keycloak client for tests

**Time spent:** _____ hours

---

## Phase 6: Notification UseCases (Task 06) - 3-4 hours

**Status:** 🔴 Not Started

### Structure Files
- [ ] `commands.go`
- [ ] `queries.go`
- [ ] `results.go`
- [ ] `errors.go`

### Command UseCases
- [ ] **CreateNotificationUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **MarkAsReadUseCase**
  - [ ] Implementation
  - [ ] Unit tests (ownership check)
- [ ] **MarkAllAsReadUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **DeleteNotificationUseCase**
  - [ ] Implementation
  - [ ] Unit tests

### Query UseCases
- [ ] **GetNotificationUseCase**
  - [ ] Implementation
  - [ ] Unit tests
- [ ] **ListNotificationsUseCase**
  - [ ] Implementation
  - [ ] Unit tests (unread filter, pagination)
- [ ] **CountUnreadUseCase**
  - [ ] Implementation
  - [ ] Unit tests

### Event Handlers
- [ ] NotificationEventHandler setup
- [ ] HandleChatCreated
- [ ] HandleUserAssigned
- [ ] HandleStatusChanged
- [ ] HandleMessageSent
- [ ] Event bus subscription setup

**Time spent:** _____ hours

---

## Phase 7: Integration & Testing (Task 07) - 4-5 hours

**Status:** 🔴 Not Started

### Mocks
- [ ] ChatRepository mock
- [ ] MessageRepository mock
- [ ] UserRepository mock
- [ ] WorkspaceRepository mock
- [ ] NotificationRepository mock
- [ ] EventStore mock
- [ ] EventBus mock
- [ ] KeycloakClient mock

### Fixtures
- [ ] ChatBuilder fixture
- [ ] MessageBuilder fixture
- [ ] UserBuilder fixture
- [ ] WorkspaceBuilder fixture
- [ ] NotificationBuilder fixture

### Test Utilities
- [ ] TestSuite helper
- [ ] Context helpers
- [ ] Custom assertions

### Integration Tests
- [ ] Event Bus integration test
- [ ] Chat + Message integration test
- [ ] Workspace + User integration test
- [ ] Notification creation via events test

### E2E Tests
- [ ] Complete Task workflow test
- [ ] Messaging workflow test
- [ ] Workspace invitation workflow test

### CI/CD
- [ ] Test coverage reporting
- [ ] Coverage threshold check (>80%)

**Time spent:** _____ hours

---

## Phase 8: Tag Integration (Task 08) - 2-3 hours

**Status:** 🔴 Not Started

### Refactoring
- [ ] Create ChatUseCases wrapper in tag package
- [ ] Refactor executeCreateTask
- [ ] Refactor executeCreateBug
- [ ] Refactor executeCreateEpic
- [ ] Refactor executeChangeStatus
- [ ] Refactor executeAssignUser (with username resolving)
- [ ] Refactor executeChangePriority
- [ ] Refactor executeSetDueDate
- [ ] Refactor executeChangeTitle
- [ ] Refactor executeSetSeverity
- [ ] Remove publishAndSave method
- [ ] Remove chatRepo dependency

### Integration
- [ ] Integrate tag processing in SendMessageUseCase
- [ ] Async tag processing
- [ ] Error handling for failed tag commands
- [ ] Bot response formatting

### Tests
- [ ] Update executor tests to use mock UseCases
- [ ] Integration test: message with tags → chat updated
- [ ] E2E test: complete tag workflow

### DI Setup
- [ ] Update main.go with ChatUseCases injection
- [ ] Wire tag components correctly

**Time spent:** _____ hours

---

## Code Quality Metrics

### Test Coverage
```
Target: >80% overall

Domain Layer:          ___% (target: >90%)
Application Layer:     ___% (target: >85%)
Integration Tests:     ___% (target: >70%)
```

### Code Review Checklist
- [ ] All UseCases follow the same pattern
- [ ] Error handling is consistent
- [ ] Validation is thorough
- [ ] Authorization checks are present
- [ ] Event publishing is correct
- [ ] Tests cover happy path and error cases
- [ ] Code is properly formatted (gofmt, goimports)
- [ ] No linter warnings (golangci-lint)

### Documentation
- [ ] All UseCases have godoc comments
- [ ] Complex logic is commented
- [ ] README updated with new structure
- [ ] Architecture diagrams are accurate

---

## Issues & Blockers

### Current Blockers
| Date | Issue | Status | Resolution |
|------|-------|--------|------------|
|      |       |        |            |

### Technical Debt
| Date Added | Description | Priority | Planned Resolution |
|------------|-------------|----------|-------------------|
|            |             |          |                   |

---

## Notes & Lessons Learned

### What went well
-

### What could be improved
-

### Tips for next time
-

---

## Sign-off

**Completed by:** ___________
**Reviewed by:** ___________
**Date:** ___________

**Ready for next phase:** ☐ Yes ☐ No

**Next steps:**
- [ ] MongoDB repository implementations
- [ ] HTTP handlers
- [ ] WebSocket handlers
- [ ] Event Bus (Redis) implementation
