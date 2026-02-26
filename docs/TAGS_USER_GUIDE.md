# Tag Commands User Guide

This guide explains how to use message tags (commands like `#task` and `#status`) in Flowra chats.

Tags let you create and update work items directly from chat messages.

## Quick Start

Type tags in a chat message and send it. Flowra will:

1. Save your message text (without the tags)
2. Apply valid tags
3. Post a bot response showing what was applied and what failed

Example:

```text
Finished the implementation
#status Done #assignee @none
```

## How Tag Parsing Works

### Basic rules

- Tags start with `#` (for example `#task`, `#status`)
- Tags are parsed only when a line starts with `#`
- If a normal text line appears, tags in the middle of that line are treated as plain text
- You can put multiple tags on one line
- You can also put tags on separate lines

Examples:

```text
#task Implement OAuth #priority High #assignee @alex
```

```text
Fixed and tested locally
#status Done
#priority Low
```

Not parsed as tags (treated as normal text):

```text
I wrote docs #status Done and pushed changes
```

### Tag values (quotes are not special syntax)

A tag value is read as text until the next valid tag or end of line.

Important:

- Quotes are allowed, but they are treated as normal characters
- In practice, most users should enter values without quotes

These are parsed successfully:

```text
#task Implement OAuth flow
```

```text
#task "Implement OAuth flow"
```

## Supported Tags

### Create or convert a chat into a work item

| Tag | Purpose | Example |
|---|---|---|
| `#task <title>` | Create/convert current chat to a Task | `#task Implement OAuth` |
| `#bug <title>` | Create/convert current chat to a Bug | `#bug Login fails on Safari` |
| `#epic <title>` | Create/convert current chat to an Epic | `#epic Q2 onboarding improvements` |

### Update the current work item

| Tag | Purpose | Example | Notes |
|---|---|---|---|
| `#status <value>` | Change status | `#status In Progress` | Allowed values depend on item type |
| `#assignee <@username>` | Assign user | `#assignee @alex` | Use `#assignee @none` or `#assignee` to clear |
| `#priority <value>` | Change priority | `#priority High` | Allowed: `High`, `Medium`, `Low` (not `Critical`) |
| `#due <date>` | Set due date | `#due 2026-03-01` | Use empty value to clear |
| `#title <text>` | Rename current item/chat title | `#title OAuth callback bug` | Requires active item |

### Bug-only tag

| Tag | Purpose | Example | Allowed values |
|---|---|---|---|
| `#severity <value>` | Set bug severity | `#severity Critical` | `Critical`, `Major`, `Minor`, `Trivial` |

### Participant management (chat members)

| Tag | Purpose | Example |
|---|---|---|
| `#invite <@username>` | Add participant to the chat | `#invite @maria` |
| `#remove <@username>` | Remove participant from the chat | `#remove @maria` |

### Chat lifecycle

| Tag | Purpose | Example | Notes |
|---|---|---|---|
| `#close` | Close/archive the current chat | `#close` | Requires active item context |
| `#reopen` | Reopen a closed chat | `#reopen` | |
| `#delete` | Delete chat | `#delete` | Currently not implemented (returns an error) |

## Allowed Status Values by Item Type

Status values are case-insensitive when entered, but must match one of the supported values for the current item type.

### Task statuses

- `To Do`
- `In Progress`
- `Done`

### Bug statuses

- `New`
- `Investigating`
- `Fixed`
- `Verified`

### Epic statuses

- `Planned`
- `In Progress`
- `Completed`

## Date Format (`#due`)

Recommended format:

- `YYYY-MM-DD` (for example `2026-03-01`)

Flowra also accepts common ISO 8601 date/time variants, but `YYYY-MM-DD` is the safest option for everyday use.

Clear due date:

```text
#due
```

## Common Workflows

### 1. Turn a discussion into a task

```text
#task Implement export endpoint
#priority High #assignee @alex #due 2026-03-05
```

### 2. Update a task after finishing work

```text
API endpoint is merged and deployed to staging
#status Done #assignee @none
```

### 3. Report and triage a bug

```text
#bug Login redirects to blank page on Safari
#severity Major #status Investigating #assignee @qa.lead
```

### 4. Add/remove participants during a thread

```text
#invite @designer
```

```text
#remove @designer
```

## Limitations and Validation Rules

### Context requirements

- `#status` requires an active item (Task/Bug/Epic) in the chat
- `#title` is for the current item/chat title
- `#severity` only works in Bug context
- `#close` requires an active item context (not a plain discussion chat)

### Permissions and access

- Tag commands still follow normal Flowra permissions
- Some tags may fail if you do not have permission (for example inviting/removing users or changing chat state)
- The bot response will show the error, but your original message is still posted

### Usernames

- Use `@username` format (for example `@alex`)
- Invalid format is rejected
- If the username does not exist, the tag fails and bot response shows an error

### Partial application (important)

Each tag is validated and applied independently.

This means:

- valid tags can succeed
- invalid tags can fail
- your message is still posted

Example:

```text
#status Done #assignee @unknown #priority High
```

Possible outcome:

- status changed
- priority changed
- assignee failed (`@unknown` not found)

## What Bot Responses Look Like

After sending a tagged message, Flowra posts a bot message with results.

Success examples:

- `✅ Status changed to Done`
- `✅ Assigned to: alex`
- `✅ Due date removed`
- `✅ Chat reopened`

Error examples:

- `❌ no active entity to modify. Create an entity first with #task, #bug, or #epic`
- `❌ invalid assignee format. Use @username`
- `❌ invalid date format. Use ISO 8601: YYYY-MM-DD`
- `❌ invalid priority 'Urgent'. Available: High, Medium, Low`

Warnings may also appear (for example unknown/unsupported tags).

## Troubleshooting

### Tag did not trigger

Check:

- The tag line starts with `#`
- The tag name is supported (`#status`, not `#Status`)
- The tag is not embedded in normal sentence text

### Status tag failed

Check:

- The chat currently has an active item (Task/Bug/Epic)
- The status is valid for that item type

Examples:

- Task: `#status In Progress`
- Bug: `#status Investigating`

### Assignee tag failed

Check:

- Username format is `@username`
- The user exists in Flowra

To remove assignment, use:

- `#assignee @none`
- or `#assignee`

### Due date tag failed

Use:

- `#due 2026-03-01`

Avoid ambiguous formats like:

- `03/01/2026`
- `next Friday`

## Tips

1. Put tags on a new line after your message text for better readability.
2. Use multiple tags in one message to update several fields at once.
3. Start with `#task`, `#bug`, or `#epic` when working in a discussion chat.
4. Watch the bot response to confirm what was actually applied.
