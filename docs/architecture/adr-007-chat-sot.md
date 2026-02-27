# ADR-007: Chat as Source of Truth for Typed Entities

- Status: Accepted
- Date: 2026-02-27
- Owners: Backend Team
- Related backlog item: [PR-01 Architecture Contract (`Chat = SoT`)](/home/sasha/Project/flowra/docs/tasks/chat-sot-pr-01-architecture-contract.md)

## Context

The current codebase contains historical overlap between Chat and Task write paths. This creates dual-write behavior, inconsistent event ownership, and unclear boundaries for typed entities (`task`, `bug`, `epic`).

The refactor plan requires a single write model before follow-up PRs remove duplicate paths and rebuild projections.

## Decision

`Chat` is the only write aggregate for typed entity business state.

For typed chats, identity is unified:

- `task_id == chat_id`
- `bug_id == chat_id`
- `epic_id == chat_id`

Typed entity state transitions must be emitted from `chat.*` streams only.

## Aggregate Boundaries

- `Chat` owns all mutable business state for typed entities (title, description, status, priority, assignee, due date, and other typed metadata).
- `Task` write aggregate and task-specific command handlers are deprecated and will be removed in subsequent PRs.
- Read models (including task-oriented views) may keep backward-compatible shapes, but they are projections derived from Chat events.

## Event Model Contract

All business write events for typed entities are `chat.*` events. This includes creation, type changes, status updates, priority changes, assignment changes, due date updates, and detail updates.

`task.*` events are frozen:

- no new `task.*` event types;
- no new producers of `task.*` events;
- existing consumers should migrate to `chat.*` contracts.

## Compatibility Strategy (Frontend and Query API)

- Query/read endpoints may continue returning task-oriented DTOs for UI compatibility.
- Projection layer is responsible for mapping typed Chat state into legacy read shapes.
- During transition, adapters are allowed on read side only; write side must remain Chat-only.

## Deprecation Policy for Task Write APIs and Packages

- `internal/application/task` must not receive new write commands or write handlers.
- Task write entry points are deprecated immediately and scheduled for removal by follow-up PRs in the `Chat = SoT` plan.
- Any remaining task package usage should be read-model/query-only until full cleanup.

## Forbidden Patterns

- Direct state writes in Task aggregate for typed entity business state.
- Emitting new `task.*` domain events.
- Adding new direct task write HTTP/WS handlers.
- Adding parallel write flows where both Chat and Task mutate the same business field.

## Consequences

- Positive: one write authority, fewer consistency bugs, and simpler projection architecture.
- Positive: deterministic migration path for removing Task write stack.
- Tradeoff: temporary adapter logic in read layer while compatibility endpoints remain.
- Tradeoff: follow-up PRs must enforce this contract in tests and code review.

## Enforcement Checklist

Use this checklist in each follow-up PR:

- no new write logic in `internal/application/task`;
- no new `task.*` emitted events;
- typed entity write tests assert `chat.*` event emission;
- changed write paths document compatibility impact on read APIs.
