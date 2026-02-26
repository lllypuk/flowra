# Tag System Load Testing (k6)

This folder contains a repeatable `k6` load test for tag-heavy message flows, with special focus on:

- tag parsing + async processing,
- optimistic-lock retry pressure on shared typed chats,
- bot response creation observability (best-effort polling mode),
- latency/error regression tracking across runs.

## What It Tests

The script sends messages to:

- `POST /api/v1/workspaces/:workspace_id/chats/:chat_id/messages`

using tag-heavy payloads such as:

- `#status Done`
- `#status Done #priority High`
- `#status Completed` (intentional invalid tag value, should still save user message and produce bot error message)

This targets the real API path (HTTP + middleware + app layer + async tag handling).

## Prerequisites

- `k6` installed locally
- Docker services running (`mongo`, `redis`, `keycloak`) and app started (`make dev` or equivalent)
- A valid bearer token for a real user in your local/dev environment

Recommended startup:

```bash
make docker-up
make dev
```

## Authentication (Required)

Set `AUTH_TOKEN` to a valid access token.

Practical local/dev approach (recommended):

1. Login in the browser to local Flowra (`http://localhost:8080`)
2. Open DevTools -> Network
3. Open any `/api/v1/...` request
4. Copy the `Authorization: Bearer <token>` value
5. Export token for `k6`

Example:

```bash
export AUTH_TOKEN='<paste access token here>'
```

## Fastest Way To Run (Auto Setup)

Use `AUTO_SETUP=true` to let the script create a workspace and typed chats (Task chats) before the load profile starts.

Smoke (shared contention):

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e AUTO_SETUP=true \
  -e PROFILE=smoke \
  -e MODE=shared \
  tests/load/tag-system/k6-tag-message-flow.js
```

Moderate (distributed chats + bot lag polling):

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e AUTO_SETUP=true \
  -e AUTO_SETUP_CHAT_COUNT=30 \
  -e PROFILE=moderate \
  -e MODE=distributed \
  -e POLL_BOT=true \
  tests/load/tag-system/k6-tag-message-flow.js
```

High contention (same chat, retry pressure):

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e AUTO_SETUP=true \
  -e AUTO_SETUP_CHAT_COUNT=1 \
  -e PROFILE=high \
  -e MODE=shared \
  -e INVALID_RATIO=0.02 \
  tests/load/tag-system/k6-tag-message-flow.js
```

Burst profile:

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e AUTO_SETUP=true \
  -e PROFILE=burst \
  -e MODE=shared \
  tests/load/tag-system/k6-tag-message-flow.js
```

## Manual Setup Mode (Pre-created Data)

If you want stable comparison across runs, reuse the same workspace/chat IDs:

- `WORKSPACE_ID` (required)
- `SHARED_CHAT_ID` (for shared mode), or
- `CHAT_IDS` (comma-separated list for distributed mode)

Example:

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e WORKSPACE_ID='<workspace-uuid>' \
  -e SHARED_CHAT_ID='<task-chat-uuid>' \
  -e PROFILE=high \
  -e MODE=shared \
  tests/load/tag-system/k6-tag-message-flow.js
```

Distributed mode example:

```bash
k6 run \
  -e AUTH_TOKEN="$AUTH_TOKEN" \
  -e WORKSPACE_ID='<workspace-uuid>' \
  -e CHAT_IDS='<chat-1>,<chat-2>,<chat-3>,<chat-4>' \
  -e PROFILE=moderate \
  -e MODE=distributed \
  -e POLL_BOT=true \
  tests/load/tag-system/k6-tag-message-flow.js
```

## Profiles

- `smoke`: quick validation (default)
- `moderate`: steady load
- `high`: contention-heavy concurrency
- `burst`: short spikes and ramp-down

You can override profile values:

- `VUS`, `DURATION` for constant-VU profiles
- `STAGES` (JSON array) for burst/ramping profiles

## Traffic Modes (Contention Strategy)

- `MODE=shared`: all virtual users hit the same typed chat (max optimistic-lock contention)
- `MODE=distributed`: virtual users spread across `CHAT_IDS` (or auto-created set) for throughput measurements

## Bot Response Lag (Optional)

Enable best-effort bot observation polling:

- `POLL_BOT=true`

The script polls:

- `GET /api/v1/workspaces/:workspace_id/chats/:chat_id/messages`

and searches for a matching bot message (`type == "bot"` + expected content fragment).

Important limitation:

- In `MODE=shared`, bot-lag matching is approximate under high concurrency (another VU may produce a similar bot response).
- For cleaner lag data, prefer `MODE=distributed` with enough chats (e.g. `AUTO_SETUP_CHAT_COUNT >= VUs`).

## Metrics Captured

HTTP / request metrics (k6 built-in + custom):

- `http_req_failed`
- `http_req_duration`
- `tag_send_latency_ms`
- `tag_send_success_rate`
- `tag_http_2xx`, `tag_http_4xx`, `tag_http_5xx`
- `tag_auth_errors`, `tag_validation_errors`, `tag_timeout_errors`, `tag_network_errors`

Bot observability metrics (optional polling mode):

- `tag_bot_observed_rate`
- `tag_bot_lag_ms`

## Default Thresholds (Conservative Initial Values)

- `http_req_failed < 10%`
- `http_req_duration p95 < 2000ms`, `p99 < 5000ms`
- `tag_send_success_rate > 90%`
- `tag_send_latency_ms p95 < 1500ms`, `p99 < 4000ms`
- If `POLL_BOT=true`: `tag_bot_observed_rate > 60%`, `tag_bot_lag_ms p95 < 5000ms`

Adjust thresholds after collecting a stable local baseline on your machine.

## Useful Environment Variables

- `BASE_URL` (default: `http://127.0.0.1:8080`)
- `AUTH_TOKEN` (required)
- `PROFILE` (`smoke|moderate|high|burst`)
- `MODE` (`shared|distributed`)
- `AUTO_SETUP` (`true|false`)
- `AUTO_SETUP_CHAT_COUNT` (default: `20`)
- `WORKSPACE_ID`, `SHARED_CHAT_ID`, `CHAT_IDS`
- `POLL_BOT` (`true|false`)
- `BOT_POLL_TIMEOUT_MS` (default: `5000`)
- `BOT_POLL_INTERVAL_MS` (default: `100`)
- `BOT_SEARCH_LIMIT` (default: `30`)
- `INVALID_RATIO` (default: `0.05`)
- `MULTI_TAG_RATIO` (default: `0.20`)
- `THINK_MS` (default: `0`)
- `REQUEST_TIMEOUT` (default: `10s`)
- `SUMMARY_JSON` (optional path for raw k6 summary JSON)

## Interpreting Results

Use `MODE=shared` + `PROFILE=high` to answer:

- Are retries under contention causing latency spikes (`p95`/`p99`)?
- Do non-2xx responses increase materially?
- Does `tag_send_success_rate` degrade as concurrency rises?

Use `MODE=distributed` + `POLL_BOT=true` to answer:

- Does bot response observation lag grow significantly?
- Is `tag_bot_observed_rate` stable (within known best-effort limitations)?

When comparing runs:

- Keep `PROFILE`, `MODE`, `AUTO_SETUP_CHAT_COUNT`, and `INVALID_RATIO` constant
- Prefer running on an idle machine
- Save results with `SUMMARY_JSON`

## Notes / Limitations

- This load test is not part of `make test` / CI by default.
- `AUTO_SETUP` creates fresh data each run (good for convenience, less ideal for apples-to-apples comparisons).
- `AUTH_TOKEN` acquisition is intentionally external to the script.

