# PR-11: Make Dev Runtime Full-Stack by Default

## Goal

Run full functionality in development by default (API + worker + infra) so smoke/e2e behavior is deterministic and production-like.

## Why This PR

`make dev` currently starts API only, while outbox is enabled by default. This creates a hidden dependency on worker availability for projection freshness and breaks local smoke assumptions.
The target dev experience is full-stack from one command.

## In Scope

- Make `dev` run all required components for full functionality.
- Keep an optional lightweight mode only as an explicit opt-in.
- Update docs/checklists/Make targets accordingly.
- Add startup log warnings when configuration is inconsistent with expected mode.

## Out of Scope

- Production deployment model changes.
- Replacing outbox pattern.

## Planned Changes

- Set default dev mode to full-stack (`api + worker + required infra`).
- Update Make targets, for example:
  - `make dev` -> full-stack default.
  - `make dev-lite` -> optional API-only mode.
- Keep outbox enabled in the default dev path.
- Update:
  - `README.md`
  - `docs/DEVELOPMENT.md`
  - `tests/e2e/frontend/SMOKE_CHECKLIST.md`
  to reflect exact prerequisites.

## Implementation Steps

1. Implement full-stack `make dev` orchestration.
2. Add optional lightweight mode as explicit alternative.
3. Add runtime logs indicating active mode and required dependencies.
4. Update smoke checklist prerequisites with exact commands.
5. Validate full-stack mode covers all sidebar/board/task flows end-to-end.

## Definition of Done

- `make dev` brings up full functionality without additional manual processes.
- Smoke prerequisites are reproducible on clean local setup.
- Optional lightweight mode is explicit and documented as limited.

## Test Plan

- Run smoke checklist with `make dev` from scratch.
- Verify outbox backlog is actively processed in default dev mode.
- Confirm board/sidebar consistency under default dev mode.

## Risks

- Risk: full-stack dev startup is slower.
- Mitigation: provide `dev-lite` for quick API-only work, but keep full-stack as default.

## Dependencies

- PR-08 and PR-09 are recommended before finalizing docs to avoid documenting broken paths.
