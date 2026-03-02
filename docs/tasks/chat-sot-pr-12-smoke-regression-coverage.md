# PR-12: Add Regression Coverage for Board + Sidebar Smoke Path

## Goal

Prevent repeat regressions by automating critical board/sidebar consistency checks from smoke flow.

## Why This PR

The bug surfaced only in manual smoke: typed chat was visible in sidebar and timeline but missing on board. Existing tests did not assert this cross-view consistency.

## In Scope

- Add integration/e2e coverage for:
  - create typed chat from chats page.
  - mutate status/priority/assignee/due date from chat sidebar actions.
  - verify board card existence and consistency after reload.
- Add assertion that no duplicates are created in sidebar/board after mutations.

## Out of Scope

- Full frontend visual regression suite.
- Non-typed discussion chat scenarios.

## Planned Changes

- Extend frontend e2e tests in `tests/e2e/frontend`.
- Add API/integration test for read-model consistency between chat and board queries.
- Add helper assertions for final state parity:
  - sidebar status == board column.
  - priority/assignee/due-date parity after reload.

## Implementation Steps

1. Implement scenario fixture setup (workspace + user + typed chat).
2. Add end-to-end test that follows smoke sequence.
3. Add duplicate-detection assertions for sidebar list and board cards.
4. Add regression test for reload persistence.
5. Hook test into `make test-e2e-frontend` or dedicated smoke target.

## Definition of Done

- Automated test fails on the current regression and passes after fixes.
- Smoke critical path has coverage for create + mutate + board parity.
- CI includes the new regression test path.

## Test Plan

- Run targeted test locally.
- Run full `make test-e2e-frontend`.
- Re-run manual smoke once to confirm parity with checklist expectations.

## Risks

- Risk: frontend e2e flakiness from async updates.
- Mitigation: deterministic waits on server responses/state selectors; avoid timing-only sleeps.

## Dependencies

- PR-08, PR-09, PR-10, PR-11.
