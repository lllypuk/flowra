# PR-01: Architecture Contract (`Chat = SoT`) and Scope Freeze

## Goal

Formally lock architecture to `Chat = source of truth` for all typed entity state (`task`, `bug`, `epic`) and prevent new code from introducing parallel write paths.

## Why This PR

Without a hard contract, future PRs can accidentally keep dual-write behavior (`Chat` and `Task` both mutate state), which is the main source of complexity and inconsistency.

## In Scope

- Add ADR describing final target model and boundaries.
- Define identity rule: `task_id == chat_id` for typed chats.
- Define event model contract: all business write events come from `chat.*` streams.
- Define deprecation policy for `Task` write APIs and packages.
- Add a short implementation checklist to `CLAUDE.md` or architecture docs for contributors.

## Out of Scope

- Runtime logic changes.
- Route removals.
- Projection rewrites.

## Planned Changes

- Add ADR file in `docs/architecture/` with:
  - aggregate boundaries (`Chat` owns write state for typed entities)
  - event taxonomy (`chat.created`, `chat.type_changed`, `chat.status_changed`, and so on)
  - compatibility strategy for frontend and query API
- Add short cross-reference from task backlog to ADR.
- Add contributor notes:
  - no new writes in `internal/application/task`.
  - no new `task.*` domain events.

## Implementation Steps

1. Create ADR and circulate it for review.
2. Add explicit accepted decision and consequences section.
3. Add "forbidden patterns" list:
   - direct state write in `Task` aggregate
   - emitting new `task.*` events
   - adding new direct task write handlers
4. Link ADR from [backlog.md](/home/sasha/Project/flowra/docs/tasks/backlog.md).

## Definition of Done

- ADR merged and referenced in backlog.
- Team has single accepted rule: `Chat` is the only write model.
- No open ambiguity about identity, events, and ownership.

## Test Plan

- Documentation-only PR: no runtime tests required.
- Validate links and markdown rendering.

## Risks

- Risk: decision not enforced in next PRs.
- Mitigation: explicitly add checklist in each subsequent PR template.

## Dependencies

- None.
