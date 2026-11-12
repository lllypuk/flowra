# Feed: lead questions and owner answers

Under `/api/v1/feed`, Bearer or session cookie. A question is a `question` record of the published snapshot; the
answer lives in Flowra (`owner_answers`) and is delivered to the lead by `leadfeed deliver`. Design, storage
and delivery: `docs/plans/20261112-owner-question.md`.

| Method | Path | Result |
|---|---|---|
| GET | `/questions` | questions of the published snapshot with the answer and `state`; `waiting` counts those awaiting an answer, they come first |
| GET | `/questions/{qid}` | one question |
| POST | `/questions/{qid}/answer` | body `{"option_id": "...|null", "text": "..."}` |

`state`: `waiting`, `answered` (stored in Flowra, not yet confirmed by the journal), `delivered` (mail accepted by
the lead's pane), `handled` (`resolved` in the journal). `closed_in_session` marks an answer given on a question
the lead already closed.

## Answer

`Idempotency-Key` is required (at most 200 characters). At least one of `option_id` and `text`; `text` is at
most 4000 characters, stored verbatim and escaped on display. An answer is not edited: a correction is a new
question from the lead.

| Code | When |
|---|---|
| 201 | accepted |
| 200 | repeat of the same key and body; checked before the question state, so a lost response is safe to retry |
| 400 | `IDEMPOTENCY_KEY_REQUIRED`, `INVALID_IDEMPOTENCY_KEY` |
| 404 | no such question in the snapshot |
| 409 | `QUESTION_ALREADY_ANSWERED`, `QUESTION_CLOSED`, `IDEMPOTENCY_KEY_REUSED` (same key, other body) |
| 422 | invalid body |

Writes with a cookie must carry `Origin` or `Referer` matching `Host`; Bearer requests skip the check.

## Pages

`/feed/questions` and `/feed/questions/{qid}`. The form posts `application/x-www-form-urlencoded` to
`/feed/questions/{qid}/answer` with `option_id`, `text` and the hidden `idempotency_key`; success, a repeat and
an already-answered or closed question all answer 303 to the question page; an invalid answer re-renders the
page with 422; a reused key is 409. No JavaScript is needed.

## `lead-task` contract

Flowra calls only `export` and `answer`; commands, exit codes (`answer`: 0, 1, 3, 4, 5) and the snapshot
`question` record (`schema: 2`) are in the plan, sections «Контракт `lead-task`» and «Контракт снимка».
