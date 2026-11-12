# Flowra Architecture

Flowra is a task tracker and messenger for one owner's development agents (later, a small team of people and many
agents). Product scope: [README](../README.md). Contributor rules: [AGENTS.md](../AGENTS.md).

## Contents

1. [Components](#components)
2. [Lead Feed](#lead-feed)
3. [Owner Questions and Answers](#owner-questions-and-answers)
4. [Sync and Delivery](#sync-and-delivery)
5. [Contract with the Lead's Journal](#contract-with-the-leads-journal)
6. [Authentication](#authentication)
7. [Kept from the Earlier Product (Not Developed)](#kept-from-the-earlier-product-not-developed)
8. [Technology Stack](#technology-stack)
9. [Architectural Decisions](#architectural-decisions)

---

## Components

```
 lead's journal ──lead-task export──▶ leadfeed import ──▶ MongoDB: leadfeed_* (projection)
       ▲                                                          │ read in one snapshot
       │ lead-task answer                                         ▼
 leadfeed deliver ◀── MongoDB: owner_answers ◀── API (Echo): /feed pages, /api/v1/feed
                                                      ▲
                                  owner (browser, Bearer token), agents
```

| Part | What it is |
|---|---|
| `cmd/api` | HTTP server (Echo): JSON API, HTMX pages, WebSocket; runs the worker loops with `FLOWRA_WORKER=true` |
| `cmd/worker` | Worker loops alone: outbox publisher and repair of read models (`internal/worker`) |
| `cmd/leadfeed` | `import`, `deliver`, `status`, `fail`; talks to MongoDB directly, not through the API |
| `scripts/leadfeed-sync.sh` | One sync run; started every minute by the systemd timer from `scripts/units` |
| MongoDB | Replica set: the feed is written and read in transactions |
| Redis | Event bus (pub/sub), rate-limit store, dead-letter list |

Layers inside `internal/`: `domain` (aggregates, events), `application` (use cases and services), `infrastructure`
(MongoDB, Redis, event store), `handler/http` and `handler/websocket`, `middleware`. Dependencies point inward;
interfaces are declared by the consumer (ADR-006). The lead feed follows this: `application/leadfeed` defines the
`Reader`, `Publisher`, `AnswerStore` and `AnswerSender` interfaces, MongoDB implements them in
`infrastructure/repository/mongodb`.

---

## Lead Feed

A read projection of the lead's task journal: `internal/application/leadfeed`, collections `leadfeed_tasks`,
`leadfeed_records`, `leadfeed_state`. It is not part of the Chat write model and never goes through tag parsing
(boundary rules: [AGENTS.md](../AGENTS.md#lead-feed-boundary)).

**Snapshot.** `lead-task export` writes JSON lines: a header, tasks, records, a footer. `ParseSnapshot` accepts schema
1 and 2 and rejects the whole snapshot when:

- the footer is missing or the header says `complete: false`;
- the content does not match the `revision` in the header;
- a `question` record appears in schema 1.

The source writes schema 2 only when the export holds a question, so a build that predates questions rejects the
snapshot instead of silently dropping them. `source`, `epoch` and `revision` name one immutable content set; a new
`epoch` means another journal installation.

**Importer.** `Importer.Import` reads at most `DefaultMaxSnapshotBytes` (32 MiB) into memory, validates, and only
then publishes. Every attempt, successful or not, is recorded in its own document in `leadfeed_state`; it never moves
the published revision or `as_of`. An attempt is detached from the request context, so an interrupted import still
leaves its record.

**Publication.** `MongoLeadFeedRepository.Publish` replaces tasks, records and state in one transaction (snapshot
read concern, majority write concern). The freshness check on `as_of` lives inside that transaction because the
driver may re-run it:

| Snapshot against the published one | Result |
|---|---|
| older `as_of` | `ErrSnapshotStale`, rejected |
| same source, epoch and revision | `refreshed`: only `as_of` and `published_at` move; the watermark must advance, because a late older snapshot is judged by it |
| same `as_of`, other content | `ErrSnapshotConflict`, rejected |
| otherwise | `published`; `PreviousEpoch` is set when the epoch changed |

An unknown transaction commit becomes `ErrCommitUnknown`: the feed holds the old or the new snapshot in full, which
one is not known. `leadfeed import` exits with 3 for it.

**Read.** `QuestionService` and the query service read through `Reader.Read`, one snapshot-isolated transaction, so a
list, its `State` and the tasks it shows come from the same revision. The callback may run more than once and must
rebuild its result each time. List cursors carry the revision; a republish between pages gives
`ErrRevisionChanged` and paging restarts.

Source statuses are stored and returned verbatim. `StatusLabel` and `StatusGroup` are display-only; an unknown status
is shown as is and grouped `unknown`.

Routes: [docs/api/README.md](./api/README.md), questions in [docs/api/feed-questions.md](./api/feed-questions.md).

---

## Owner Questions and Answers

The lead asks through its journal (a `question` record in the snapshot); the owner answers in Flowra; the answer
goes back through `lead-task`. API, codes and states: [docs/api/feed-questions.md](./api/feed-questions.md).

**Storage.** An answer is stored in `owner_answers`, outside the projection: an import replaces `leadfeed_*` and
never touches it, and nothing writes answers into `leadfeed_*`. The document holds everything needed to deliver
without a snapshot (`OwnerAnswer`).

**Identity.** `(source, epoch, qid)` is the document `_id`; the feed revision is deliberately not part of it, so an
answer survives republication. One answer per question follows from the unique `_id`. A second unique index is on
`command_id`, the idempotency key. Answers of another epoch are kept and counted but never delivered.

**Accepting an answer** (`QuestionService.Answer`), in this order:

1. Key and body checks (`Idempotency-Key` required; an option or a text; text at most 4000 characters).
2. Lookup by key, in any epoch. Same key and same body fingerprint: return the original answer (`repeated`), whatever
   the question became since. Same key, other body: `ErrIdempotencyKeyReused`.
3. Only then the question state in one snapshot: unknown option, already answered (stored or in the snapshot),
   closed by the lead.
4. `SaveAnswer` inserts; a concurrent request with the same key that wins the insert is a repeat, not a conflict.

The key goes first so that a response lost on the way can be retried safely after the question moved on.

**Confirmation.** Flowra does not mark an answer delivered itself. The lead's journal records the answer under the
`answer_id` Flowra generated, and the next snapshot shows it with `delivered: true`. `isConfirmed` accepts only the
same `answer_id` marked delivered: a closed question or another answer is not confirmation. Until then the state is
`answered`.

---

## Sync and Delivery

`scripts/leadfeed-sync.sh`, one run:

1. Takes `flock` on a lock file (waits up to `LEADFEED_SYNC_LOCK_WAIT` seconds, 90 by default), so runs do not
   overlap.
2. `lead-task export` into a temp file. The import starts only after the exporter finished with success: a footer
   proves the content is whole, not that the exporter ended well. On export failure `leadfeed fail` records the
   reason and nothing is imported.
3. `leadfeed import <file>`.
4. `leadfeed deliver --lead-task PATH` runs whatever happened to the import: an answer must not wait for a healthy
   export.
5. The exit code is non-zero if the export, the import or the delivery failed.

`Deliverer.Run` reads the published feed in one snapshot, then for every stored answer of the current epoch that
the snapshot does not confirm calls `lead-task answer <qid>` with the answer JSON on stdin (`execSender`, bounded
by a timeout). Delivery is at-least-once: repeated calls are safe because the journal checks `answer_id`. An answer
whose question is not in the published snapshot, or whose record changed, is counted as failed and not sent. The
outcome of the run (`sent`, `failed`, `unconfirmed`, oldest unconfirmed time, other-epoch count, last error) is
stored apart from the import attempt and shown by `leadfeed status`.

`sent` counts calls the journal accepted; the snapshot confirms them one import later. `leadfeed deliver` exits with
1 if any answer failed.

---

## Contract with the Lead's Journal

The journal is the source of tasks and questions. Flowra never writes it except through `lead-task answer`,
which only `leadfeed deliver` invokes; the only other call is `lead-task export`.

| Call | Made by | Failure |
|---|---|---|
| `lead-task export` | `leadfeed-sync.sh` | nothing is imported, the attempt is recorded as failed |
| `lead-task answer <qid>` | `leadfeed deliver` | the answer stays unconfirmed and is retried on the next run |

A non-zero exit of `lead-task answer` is not interpreted by Flowra beyond "not delivered"; the exit codes and the
payload fields are defined in the contract section of the design plan, which is design history, not the source of
behaviour: [lead-task contract](./plans/20261112-owner-question.md#контракт-lead-task). What Flowra sends is
`answerPayload` in `internal/application/leadfeed/delivery.go`.

---

## Authentication

Flowra authenticates requests itself: the server holds SHA-256 digests of owner tokens and resolves every valid
token to the owner user.

- **Tokens, not sessions.** `Authorization: Bearer <token>` for API and agents; the web form `POST /login` checks
  the token the same way and stores it in the HttpOnly `flowra_session` cookie.
- **Digests only.** `auth.tokens[{name, sha256}]` and `AUTH_OWNER_TOKEN_SHA256`; tokens are named so one can be
  dropped without changing the others. The check is a constant-time comparison.
- **Loopback only.** `Config.ValidateAuth` (called by `cmd/api`) refuses a non-loopback `server.host`: a token is a
  bearer secret over plain HTTP.
- **No fallbacks.** Without a validator a request is rejected, never served as a mock user.
- **Cookie writes.** A write authenticated by cookie must carry `Origin` or `Referer` matching `Host`
  (`RequireSameOriginForCookie`); Bearer requests skip the check.
- **WebSocket.** `/api/v1/ws` sits behind the same middleware. `websocket.Handler.getUserID` also reads a `token`
  query parameter when the context has no user; the request log redacts that parameter.

### Several Users: What Is Missing

The model "several named tokens, different users" is possible but not built:

- a token has no identity of its own: it needs `tokens[].user {username, email}` and an `ExternalUserID` taken from
  the token instead of the constant `owner`, plus a uniqueness check between tokens;
- `userResolver` falls back to an existing user with the same `username`; with several identities a token with a
  colliding username would silently sign in as that user, so this must be refused;
- `AcceptInviteUseCase` is not wired to HTTP and grants no membership; people can be added only as existing users
  through the workspace invite form, and a token can be issued only by editing the config and restarting;
- a team on several machines needs network access with TLS, which loopback-only does not give.

---

## Kept from the Earlier Product (Not Developed)

Workspaces, chats, tags, WebSocket and the HTMX board come from the earlier chat product. They work and have
tests; nothing new is built on them. The one rule that still binds new work is Chat=SoT
([ADR-007](./architecture/adr-007-chat-sot.md)): for typed entities (`task`, `bug`, `epic`) the Chat aggregate is
the only write model and emits `chat.*` events only; `tasks_read_model` is a projection.

### Domain

| Aggregate / entity | Where |
|---|---|
| Chat (also Task, Bug, Epic as typed chats) | `internal/domain/chat` |
| Message | `internal/domain/message` |
| Notification | `internal/domain/notification` |
| User, Workspace | `internal/domain/user`, `internal/domain/workspace` |

Workspace roles are enforced by `middleware.RequireWorkspaceAdmin` / `RequireWorkspaceOwner` on the
workspace-scoped routes (`cmd/api/routes.go`).

### Task actions and tags

Field writes from HTTP actions run in the request through `chatapp.CommandRunner` (per-chat lock, bounded retry with
reload, `Idempotency-Key` stored with the events) and answer with a definite outcome
([docs/api/action-endpoints.md](./api/action-endpoints.md)). Tags (`#status`, `#assignee`, ...) are the other
route, from chat messages: only messages of type `user` are parsed, tags run after the response and have no
outcome, and a valid tag applies even when another in the same message fails. Syntax, values and errors:
[docs/TAGS_USER_GUIDE.md](./TAGS_USER_GUIDE.md).

### Event flow

```
Command ─▶ Chat aggregate ─▶ event store (events) + outbox ─▶ outbox worker ─▶ Redis pub/sub
                                                                                   │
                                          projection handlers · notification handler · WebSocket hub
```

- **Event store.** Collection `events`: `aggregate_id`, `aggregate_type`, `event_type`, `version`, `data`,
  `metadata`, `occurred_at`; optional `command_id` and `command_fp` for idempotent commands. Unique indexes on
  `(aggregate_id, version)` and on the aggregate's command id.
- **Outbox.** Events also go to the `outbox` collection; the outbox worker polls it (`outbox.*` config: interval,
  batch, retries, cleanup) and publishes to the Redis bus, channel `events:<event type>` by default. Delivery to
  subscribers is at-most-once on the Redis side; the outbox is what makes publication retryable.
- **Read models.** `chats_read_model` and `tasks_read_model`, written by projectors only with a version not older
  than the stored one (`mongodb.UpsertNewer`). The repair worker rebuilds a model from the event store using the
  `repair_queue` collection. Legacy `chat_read_model` and `task_read_model` are not authoritative (`make reset-data`
  cleans them).
- **Dead letters.** `DeadLetterHandler` keeps failed events in the Redis list `events:dead_letter`; the health check
  reports it.

### WebSocket

Handler in `internal/handler/websocket`, hub in `internal/infrastructure/websocket`. HTMX 2 pages open the socket
through the `ws` extension; client-side pitfalls are in [AGENTS.md](../AGENTS.md). Protocol:
[docs/api/websocket-protocol.md](./api/websocket-protocol.md). Authentication: see above.

### Board and pages

Server-rendered HTMX templates in `web/templates` with Pico CSS (`internal/handler/http/*template*`). Frontend
notes: [FRONTEND_DEV_GUIDE.md](./FRONTEND_DEV_GUIDE.md).

---

## Technology Stack

| Technology | Purpose |
|---|---|
| Go 1.26+, Echo v4 | Backend, HTTP |
| MongoDB 6+ (Go driver v2), replica set | Primary store, feed transactions |
| Redis 7+ (go-redis v9) | Event bus, rate limiting |
| gorilla/websocket | WebSocket (kept) |
| HTMX 2, Pico CSS v2, vanilla JS modules | Web pages |
| Docker Compose | Local and self-hosted runtime |

---

## Architectural Decisions

Records written as the system grew. ADR-001 to ADR-003, ADR-005 and ADR-006 date from the earlier product and still
describe the code; ADR-004 replaced the earlier identity provider.

### ADR-001: MongoDB for Primary Storage

**Decision:** MongoDB is the primary database; the document model fits aggregates.

**Consequences:** consistency across documents is managed in the application, except where a transaction is
used (feed publication and feed reads), which is why a replica set is required.

### ADR-002: Event-Driven Architecture

**Decision:** components communicate through domain events (outbox, Redis bus).

**Consequences:** eventual consistency between write and read models; failed events land in the dead-letter list.
The feed does not use it: it is a snapshot import.

### ADR-003: Redis for Pub/Sub and Rate Limiting

**Decision:** Redis carries the event bus and the rate-limit store.

**Consequences:** one more infrastructure component, needed by the kept chat parts; the feed does not need it.

### ADR-004: Owner Tokens for Authentication

**Context:** the service is for one owner and agents, at most a small team. Keycloak and its Postgres cost about
770 MB and were not started in practice. Owner decision of 12.11.2026: remove Keycloak
([plan](./plans/completed/20261112-remove-keycloak.md)).

**Decision:** accept only named owner tokens configured as SHA-256 digests; every token resolves to the owner
identity. Keycloak, OAuth login, user and group sync and the mock login paths are removed.

**Consequences:**
- no external identity provider; the server listens on loopback only;
- several people cannot sign in as different users yet (see [Authentication](#authentication)).

### ADR-005: CQRS

**Decision:** separate read models for queries; write side through aggregates. The feed is a pure read side with its
own write path (import) and a separate answer store.

### ADR-006: Interface Declaration on Consumer Side

**Decision:** interfaces live where they are used, so consumers define their dependencies and tests use small fakes.

### ADR-007: Chat as Source of Truth for Typed Entities

See [adr-007-chat-sot.md](./architecture/adr-007-chat-sot.md).

---

*Last updated: November 2026*
