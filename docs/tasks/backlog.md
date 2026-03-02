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
- [ ] `P1` PR-11: Make dev runtime full-stack by default.
  - Details: [chat-sot-pr-11-dev-runtime-outbox-contract.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-11-dev-runtime-outbox-contract.md)
- [ ] `P1` PR-12: Add regression coverage for board+sidebar smoke critical path.
  - Details: [chat-sot-pr-12-smoke-regression-coverage.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-12-smoke-regression-coverage.md)

## Deployment Roadmap

- [ ] `P2` PR-13: Move to self-hosted single-image Docker deployment.
  - Details: [chat-sot-pr-13-single-image-selfhost-deployment.md](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-13-single-image-selfhost-deployment.md)
