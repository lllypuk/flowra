# Flowra

A task tracker and messenger for one owner's development agents; later, a small team of people and many agents.

It is not a team chat with a help desk, and it has no SSO: Keycloak was removed
([plan](./docs/plans/completed/20261112-remove-keycloak.md)).

## What works today

- **Owner token sign-in.** `Authorization: Bearer` for the API, a form with an HttpOnly cookie for the web.
  The server stores only SHA-256 digests and refuses to start on a non-loopback address.
- **Lead task feed.** A read projection of the lead's journal: `lead-task export` is imported by `leadfeed` every
  minute. Pages `/feed` and `/feed/tasks/{id}`, JSON under `/api/v1/feed`
  ([plan](./docs/plans/completed/20261112-leadfeed-readonly-slice.md)).
- **Owner questions.** The lead asks through its journal, the owner answers at `/feed/questions` or via
  `POST /api/v1/feed/questions/{qid}/answer`, and `leadfeed deliver` hands the answer back
  ([API](./docs/api/feed-questions.md)).
- **Task actions with a defined outcome.** `Idempotency-Key`; every answer is applied, not applied, or unknown
  ([API](./docs/api/action-endpoints.md)).
- **Tags.** `#status`, `#assignee` and the rest run only from messages of type `user`; system and bot messages are
  never parsed ([guide](./docs/TAGS_USER_GUIDE.md)).

## Kept, not developed

Workspaces and chats (the Chat aggregate, WebSocket, the HTMX board) come from the earlier product. They work and
have tests, but nothing new is built on them. Their docs stay.

## Quick start

Needs Go 1.26+, Docker with Compose, Make.

```bash
token=$(openssl rand -hex 32); echo "$token"
export AUTH_OWNER_TOKEN_SHA256=$(printf %s "$token" | sha256sum | cut -d' ' -f1)
make dev            # MongoDB, Redis, API with the worker, all on loopback
make install-units  # always-on: systemd user units, feed sync every minute
```

| Port | Service |
|---|---|
| `127.0.0.1:8080` | API and web (`/health`, `/login`, `/feed`) |
| `127.0.0.1:27017` | MongoDB |
| `127.0.0.1:6379` | Redis |

Settings: `configs/config.yaml`; after `make install-units`, `~/.config/flowra/`. Details in
[DEPLOYMENT.md](./docs/DEPLOYMENT.md); `make help` lists the other targets.

## Dependencies

MongoDB as a replica set (transactions) and Redis (event bus). Nothing else; see
[min-infra-review.md](./docs/min-infra-review.md).

## Documentation

- [API](./docs/api/README.md), [OpenAPI](./docs/api/openapi.yaml)
- [Architecture](./docs/ARCHITECTURE.md), [Development](./docs/DEVELOPMENT.md), [Deployment](./docs/DEPLOYMENT.md)
- [Contributor and agent guide](./AGENTS.md)

## License

MIT, see [LICENSE](./LICENSE).
