# Backlog

This file tracks non-urgent work that is valuable but not yet scheduled.

## Conventions

- Status: use checkboxes (`[ ]` pending, `[x]` complete).
- Priority: use `P1` (high), `P2` (medium), `P3` (low).
- Keep tasks actionable and outcome-focused.
- Do not add time estimates.
- Backward compatibility is not required for this roadmap; breaking changes and clean rewrites are allowed.

## Chat = SoT Refactor Plan (No Migrations)

Assumption: service is not in production yet. We can make breaking changes and reset data between PRs.

- [x] `P1` PR-01: Architecture Contract (`Chat = SoT`) and scope freeze.
  - Details: [chat-sot-pr-01-architecture-contract.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-01-architecture-contract.md)
  - ADR: [adr-007-chat-sot.md](/home/sasha/Project/flowra/docs/architecture/adr-007-chat-sot.md)
- [x] `P1` PR-02: Remove duplicate entity creation paths.
  - Details: [chat-sot-pr-02-remove-duplicate-creation.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-02-remove-duplicate-creation.md)
- [x] `P1` PR-03: Unify all writes through Chat command pipeline.
  - Details: [chat-sot-pr-03-unify-write-path.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-03-unify-write-path.md)
- [x] `P1` PR-04: Rebuild projections from `chat.*` events only.
  - Details: [chat-sot-pr-04-chat-projections.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-04-chat-projections.md)
- [x] `P1` PR-05: Move task-only fields (attachments and details) into Chat domain.
  - Details: [chat-sot-pr-05-move-task-fields-to-chat.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-05-move-task-fields-to-chat.md)
- [x] `P1` PR-06: Remove Task aggregate write stack.
  - Details: [chat-sot-pr-06-remove-task-write-stack.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-06-remove-task-write-stack.md)
- [x] `P1` PR-07: Data reset tooling, test hardening, and final cleanup.
  - Details: [chat-sot-pr-07-reset-tests-cleanup.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-07-reset-tests-cleanup.md)

## Board + Chat Sidebar Smoke Stabilization

Assumption: we optimize for a clean implementation, not for compatibility with legacy internals.

- [x] `P1` PR-08: Force task projection sync for chat-driven typed mutations.
  - Details: [chat-sot-pr-08-chat-driven-task-projection-sync.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-08-chat-driven-task-projection-sync.md)
- [x] `P1` PR-09: Unify read-model collection names (`task_read_model` vs `tasks_read_model`).
  - Details: [chat-sot-pr-09-unify-read-model-collection-names.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-09-unify-read-model-collection-names.md)
- [x] `P1` PR-10: Fix nullable field cleanup in chat read model (`$unset` for assignee/due date).
  - Details: [chat-sot-pr-10-chat-read-model-nullable-unset.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-10-chat-read-model-nullable-unset.md)
- [x] `P1` PR-11: Make dev runtime full-stack by default.
  - Details: [chat-sot-pr-11-dev-runtime-outbox-contract.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-11-dev-runtime-outbox-contract.md)
- [x] `P1` PR-12: Add regression coverage for board+sidebar smoke critical path.
  - Details: [chat-sot-pr-12-smoke-regression-coverage.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-12-smoke-regression-coverage.md)

## Deployment Roadmap

- [ ] `P2` PR-13: Move to self-hosted single-image Docker deployment.
  - Details: [chat-sot-pr-13-single-image-selfhost-deployment.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-13-single-image-selfhost-deployment.md)

## Review Follow-Ups: `refactoring/chat-sot` vs `main` (2026-03-02)

- [x] `P1` PR-14: Make board typed-chat creation failure-safe and projection-consistent.
  - Scope: `boardChatCreatorAdapter.CreateChat` in `cmd/api/container.go`.
  - Done when: partial failures after `CreateChat` cannot leave board state stale; projection rebuild/repair path is guaranteed.
- [x] `P1` PR-15: Decouple system tag execution from request lifecycle in `SendMessageUseCase`.
  - Scope: `internal/application/message/send_message.go`.
  - Done when: system tag side effects are resilient to HTTP context cancellation and method naming/behavior are aligned.
- [x] `P1` PR-16: Reuse a single `ChatToTaskReadModelProjector` instance across API container wiring.
  - Scope: `cmd/api/container.go`.
  - Done when: one shared projector is injected into all consumers instead of per-call/per-handler instantiation.
- [x] `P1` PR-17: Restore task assignment notifications for chat-driven events.
  - Scope: `internal/infrastructure/eventbus/handlers.go` and handler registration.
  - Done when: assignee changes emit `task.assigned` notifications with regression coverage.
- [x] `P1` PR-18: Restore assignee existence validation in chat assignment write path.
  - Scope: chat assign flow (`AssignUserUseCase` / service adapters).
  - Done when: assigning a non-existent user fails with deterministic domain/application error.
- [x] `P2` PR-19: Stop constructing chat use cases on every task command call.
  - Scope: `fullTaskServiceAdapter` in `cmd/api/container.go`.
  - Done when: use cases are initialized once in adapter construction and reused.
- [x] `P2` PR-20: Unify mutation behavior in `TaskHandler` (`ActionService` vs direct `TaskService` branch).
  - Scope: `internal/handler/http/task_handler.go`.
  - Done when: both execution paths have the same side effects and API semantics (including system-message behavior).
- [x] `P1` PR-21: Escape user-visible modal error content to prevent HTML injection.
  - Scope: `modalError` in `internal/handler/http/chat_template_handler.go`.
  - Done when: error message rendering uses safe escaping.
- [x] `P1` PR-22: Sync task projection after `ActionService.Close/Reopen`.
  - Scope: `internal/service/action_service.go`.
  - Done when: close/reopen updates are immediately reflected in `tasks_read_model`.
- [x] `P3` PR-23: Remove duplicate aggregate ID collection logic.
  - Scope: `internal/infrastructure/projector/chat_to_task_read_model_projector.go`.
  - Done when: projector reuses shared `getAllAggregateIDsByType`.
- [x] `P3` PR-24: Deduplicate `logDevRuntimeMode` helper.
  - Scope: `cmd/api/main.go`, `cmd/worker/main.go`.
  - Done when: one shared helper is used by both binaries.
- [x] `P3` PR-25: Define and enforce `TaskResult.Events` contract.
  - Scope: `internal/application/task/results.go` and task service adapters.
  - Done when: field is either correctly populated or explicitly removed/deprecated with tests.
- [x] `P2` PR-26: Add concurrent-write regression coverage for one chat aggregate.
  - Scope: integration tests (`tests/integration`).
  - Done when: race-prone concurrent status/assignee/priority operations are covered and deterministic.
- [x] `P2` PR-27: Add failure-recovery tests for partial create/setup and projection rebuild errors.
  - Scope: integration tests around board/task creation flow.
  - Done when: failures after chat creation do not leave permanently inconsistent board/task read model.
- [x] `P2` PR-28: Add conversion-cycle projection consistency tests (`task -> bug -> epic -> discussion`).
  - Scope: projector/integration tests.
  - Done when: `chats_read_model` and `tasks_read_model` stay consistent across type flips.
- [x] `P2` PR-29: Add typed-chat deletion projection regression test.
  - Scope: projector/integration tests.
  - Done when: deleting typed chats reliably removes stale `tasks_read_model` docs.
- [x] `P2` PR-30: Add notification regression tests for chat-driven assignment/status flows.
  - Scope: eventbus notification handler tests.
  - Done when: assignment/status notifications are asserted for current `chat.*` event model.
- [x] `P2` PR-31: Add startup warning when legacy read-model collections contain data.
  - Scope: startup/bootstrap checks.
  - Done when: non-empty `chat_read_model`/`task_read_model` are detected and logged with reset guidance.
- [x] `P3` PR-32: Document handling of legacy `task.*` events in Chat=SoT era.
  - Scope: docs/architecture or operations docs.
  - Done when: behavior is explicitly documented for operators and developers.

## Review Follow-Ups: Part 2 (2026-03-02)

- [x] `P1` PR-33: Make `userRepo` mandatory in `NewAssignUserUseCase` to prevent silent assignee validation bypass.
  - Scope: `internal/application/chat/assign_user.go` and all call sites/tests constructing `AssignUserUseCase`.
  - Done when: constructor requires `appcore.UserRepository` (non-variadic), missing dependency fails at compile time, and tests use explicit stub/mock repositories.
- [x] `P1` PR-34: Add failure-safe projection sync in `fullTaskServiceAdapter.CreateTask`.
  - Scope: `cmd/api/container.go` (`fullTaskServiceAdapter.CreateTask` flow).
  - Done when: partial failures after typed chat creation (`setPriority`/`assignUser`/`setDueDate`) still trigger projection repair/sync via defer-style finalization, keeping `tasks_read_model` consistent.
