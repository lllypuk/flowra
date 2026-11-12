# Flowra Owner's Guide

The web UI of the lead feed: tasks, their threads, and the lead's questions. The pages are in Russian; the labels below are translated. Product overview: [README](../README.md). Phone access: [DEPLOYMENT.md](./DEPLOYMENT.md#phone).

## Signing in

1. Open `/login`, paste the access token, press **Sign in**.
2. On success the server sets the `flowra_session` cookie (HttpOnly, SameSite Lax, 30 days by default, `AUTH_SESSION_TTL`) and redirects to the page you asked for, else to `/feed`.

The cookie holds the token itself; the server compares its SHA-256 with the stored digests.

- A wrong token: the form reappears with "Invalid token." (HTTP 401). The attempt is logged with the IP, not the token.
- "The sign-in form expired": the form carries a one-time state tied to a cookie; reload `/login` and retry.
- An unauthenticated request for a page redirects to `/login`; a token the server no longer accepts clears the cookie and does the same.
- `/login` with a cookie already set redirects to `/workspaces`, not to the feed; open `/feed` directly.

Tokens are issued by whoever runs the server ([DEPLOYMENT.md](./DEPLOYMENT.md)).

## Feed: `/feed`

Tasks from the lead's journal, read only. Times are in the server's time zone.

Under the [snapshot banner](#snapshot-banner) tasks are grouped by who is expected to act:

| Group | Statuses |
|---|---|
| Waiting on the owner | `wait_user` ("waits for you") |
| Waiting on the lead | `question` ("question to the lead"), `review` ("with the lead for review") |
| With the agent | `queued`, `work` |
| Unknown | any status the page does not know; shown as is |
| Closed | `done` ("accepted"), `cancelled` |

A row shows the id, project, title and the lead's note, the status (label and the source status beside it) and the time in that status. The age is counted to the snapshot time, not to now.

If the feed holds questions, a link "Lead's questions" shows how many wait for an answer. A feed longer than one page shows a note that only part is listed.

## Task thread: `/feed/tasks/{id}`

- Facts: project, agent, status, **waiting on** (who the lead's journal says the task waits on: lead, agent, owner, or "nobody"), time in status, creation time, number of returns.
- "Time and transitions" (collapsed): time spent in queue, work, waiting on the lead, waiting on the owner; the status history with the record that caused each change.
- Records, oldest first: a kind (letter, question, report, decision, observation), the author, the time, the record id and where the lead found it. A question record links to its page.

Markers in dashed boxes mean the lead's export guessed, not read:

- "derived" next to a transition or an author: inferred, not written in the journal.
- "text from transcript": the text is not in the journal; it was found in the agent's transcript by the event time and may be the wrong one.
- "No text: ..." in italics: the text could not be found; the reason follows.

Long threads are paged ("Next"). After the feed is republished an old page link restarts the thread from the top.

## Questions: `/feed/questions`

The list shows questions of the current snapshot, waiting ones first, then newest first: id, state, time asked, task, first line.

A question page (`/feed/questions/{qid}`) shows:

- the text, the task and the time asked;
- **scope**: what the answer applies to; check it before answering;
- options, if the lead offered any, and **the lead's recommendation** with the reason. The recommended option is marked in the form but not preselected;
- the answer form: an option and/or text up to 4000 characters. An option or a non-empty text is required; an invalid answer shows the form again with your text and an error.

An answer cannot be edited or withdrawn. If it was wrong, the lead has to ask a new question. Pressing the button twice, or an answer to a question that is already answered or closed, shows the current state of the question instead of an error.

States as the page shows them:

| State | Meaning |
|---|---|
| waiting for an answer | No answer anywhere, the lead has not closed it. The only state with a form |
| answered, the lead has not received it yet | Flowra holds your answer; the lead's journal has not confirmed it. Normal for up to a minute after sending: delivery goes through `leadfeed deliver` and the next import |
| handed to the lead | The journal confirmed your answer and the lead's mailbox accepted it |
| handled | The lead closed the question and your answer, if any, is confirmed. Closed without an answer: the page says so and shows the lead's note; "answered in the session" means you replied outside Flowra |

"Answered, ... question closed in the session": the lead closed the question before your answer was confirmed; the answer is still passed on. An answer marked "after the question was closed" arrived late.

Delivery and the state table: [feed-questions.md](./api/feed-questions.md).

## Snapshot banner

At the top of every feed page. The feed is a snapshot, and the banner says how far to trust it.

- **Snapshot from ... (N ago)**: when the lead's journal was exported; also when it was published here and its revision. "The feed is empty: no snapshot imported yet" before the first import.
- **Snapshot is stale**: older than 5 minutes; the sync does not keep up. The banner turns orange. Question pages show only this and the snapshot time.
- **Last import failed (outcome: error)**: the latest import did not pass; the previous snapshot is still shown.
- **Answers without the lead's confirmation: N, the oldest ...**: your answers the snapshot does not confirm yet. Older than 5 minutes: orange, "the answer does not reach the lead". An error from the delivery command is shown next to it ("Answer delivery: ...").

Sync and delivery units: [DEPLOYMENT.md](./DEPLOYMENT.md).

## Kept from the earlier product

Workspaces, chats, the HTMX board, invites and notifications still work but are not developed here and are not described. Tags: [TAGS_USER_GUIDE.md](./TAGS_USER_GUIDE.md).
