# Step 5: Add `HxPost` Support to Reusable Components

## Status: Complete

## Goal

The `user_select` and `date_picker` components currently only support `hx-put` via the
`HxPut` template parameter. Add `HxPost` parameter support so that callers can choose
between PUT and POST HTTP methods. This is needed so `sidebar.html` can route assignee
and due date changes through the task action endpoints (`POST .../actions/*`).

## Files to Change

- `web/templates/components/user_select.html`
- `web/templates/components/date_picker.html`

---

## Change 1: `user_select.html`

### Current Implementation

```html
{{define "components/user_select"}}
<select hx-put="{{.HxPut}}"
        hx-trigger="change"
        hx-swap="none"
        name="{{.Name}}"
        class="user-select">
    {{if .AllowEmpty}}
    <option value="" {{if not .Selected}}selected{{end}}>
        {{.EmptyLabel}}
    </option>
    {{end}}
    {{range .Users}}
    <option value="{{.UserID}}"
            {{if eq .UserID $.Selected}}selected{{end}}>
        {{.DisplayName}} (@{{.Username}})
    </option>
    {{end}}
</select>
{{end}}
```

### Updated Implementation

Use a conditional to choose between `hx-post` and `hx-put` based on which parameter
is provided by the caller. If `HxPost` is set, use POST; otherwise fall back to PUT.

```html
{{define "components/user_select"}}
<select {{if .HxPost}}hx-post="{{.HxPost}}"{{else}}hx-put="{{.HxPut}}"{{end}}
        hx-trigger="change"
        hx-swap="none"
        name="{{.Name}}"
        class="user-select">
    {{if .AllowEmpty}}
    <option value="" {{if not .Selected}}selected{{end}}>
        {{.EmptyLabel}}
    </option>
    {{end}}
    {{range .Users}}
    <option value="{{.UserID}}"
            {{if eq .UserID $.Selected}}selected{{end}}>
        {{.DisplayName}} (@{{.Username}})
    </option>
    {{end}}
</select>
{{end}}
```

### How Callers Use It

**With POST (task sidebar — new):**
```html
{{template "components/user_select" (dict
    "Name" "assignee_id"
    "Selected" .Task.AssigneeID
    "Users" .Participants
    "HxPost" (printf "/api/v1/workspaces/%s/tasks/%s/actions/assignee" .Task.WorkspaceID .Task.ID)
    "AllowEmpty" true
    "EmptyLabel" "Unassigned"
)}}
```

**With PUT (existing chat sidebar — unchanged):**
```html
{{template "components/user_select" (dict
    "Name" "assignee_id"
    "Selected" .Chat.AssigneeID
    "Users" .Participants
    "HxPut" "/api/v1/chats/some-id/actions/assignee"
    "AllowEmpty" true
    "EmptyLabel" "Unassigned"
)}}
```

Existing callers that pass `HxPut` continue to work unchanged because the conditional
falls through to `hx-put` when `HxPost` is empty.

---

## Change 2: `date_picker.html`

### Current Implementation

```html
{{define "components/date_picker"}}
<div class="date-picker-wrapper">
    <input type="date"
           name="{{.Name}}"
           value="{{if .Value}}{{.Value | formatDateInput}}{{end}}"
           hx-put="{{.HxPut}}"
           hx-trigger="change"
           hx-swap="none"
           class="date-input">
    {{if and .AllowEmpty .Value}}
    <button type="button"
            class="clear-date"
            hx-put="{{.HxPut}}"
            hx-vals='{"{{.Name}}": ""}'
            hx-swap="none"
            title="Clear date">
        &times;
    </button>
    {{end}}
</div>

<style>
.date-picker-wrapper {
    display: flex;
    gap: 0.25rem;
}

.date-picker-wrapper .date-input {
    flex: 1;
    margin-bottom: 0;
}

.clear-date {
    width: auto;
    padding: 0 0.5rem;
    background: none;
    border: 1px solid var(--muted-border-color);
}
</style>
{{end}}
```

### Updated Implementation

Two elements use the URL: the `<input>` and the clear `<button>`. Both need the conditional.

```html
{{define "components/date_picker"}}
{{$url := .HxPut}}{{if .HxPost}}{{$url = .HxPost}}{{end}}
{{$method := "hx-put"}}{{if .HxPost}}{{$method = "hx-post"}}{{end}}
<div class="date-picker-wrapper">
    <input type="date"
           name="{{.Name}}"
           value="{{if .Value}}{{.Value | formatDateInput}}{{end}}"
           {{$method}}="{{$url}}"
           hx-trigger="change"
           hx-swap="none"
           class="date-input">
    {{if and .AllowEmpty .Value}}
    <button type="button"
            class="clear-date"
            {{$method}}="{{$url}}"
            hx-vals='{"{{.Name}}": ""}'
            hx-swap="none"
            title="Clear date">
        &times;
    </button>
    {{end}}
</div>

<style>
.date-picker-wrapper {
    display: flex;
    gap: 0.25rem;
}

.date-picker-wrapper .date-input {
    flex: 1;
    margin-bottom: 0;
}

.clear-date {
    width: auto;
    padding: 0 0.5rem;
    background: none;
    border: 1px solid var(--muted-border-color);
}
</style>
{{end}}
```

### Template Variable Logic

Go templates do not support inline ternary expressions. The pattern used is:

```html
{{$url := .HxPut}}{{if .HxPost}}{{$url = .HxPost}}{{end}}
{{$method := "hx-put"}}{{if .HxPost}}{{$method = "hx-post"}}{{end}}
```

This sets `$url` and `$method` at the top of the template, then uses them in both the
`<input>` and the clear `<button>`:

```html
{{$method}}="{{$url}}"
```

This renders as either `hx-put="/some/url"` or `hx-post="/some/url"` depending on
which parameter was passed.

### How Callers Use It

**With POST (task sidebar — new):**
```html
{{template "components/date_picker" (dict
    "Name" "due_date"
    "Value" .Task.DueDate
    "HxPost" (printf "/api/v1/workspaces/%s/tasks/%s/actions/due-date" .Task.WorkspaceID .Task.ID)
    "AllowEmpty" true
)}}
```

**With PUT (existing usage — unchanged):**
```html
{{template "components/date_picker" (dict
    "Name" "due_date"
    "Value" .Chat.DueDate
    "HxPut" "/api/v1/chats/some-id/actions/due-date"
    "AllowEmpty" true
)}}
```

---

## Audit: All Current Callers

Before making these changes, search for all callers of both components to confirm
no existing usage is broken. The `HxPut` fallback ensures backward compatibility,
but verify the search results:

```
Grep: pattern="components/user_select", glob="**/*.html"
Grep: pattern="components/date_picker", glob="**/*.html"
```

All callers that currently pass `HxPut` will continue to work unchanged.
Only `sidebar.html` will start passing `HxPost`.

---

## Go Template Variable Assignment Note

The `{{$var = value}}` syntax (variable reassignment) requires Go 1.11+. The project
uses Go 1.25+ per `CLAUDE.md`, so this is safe.

The template renderer is configured with `html/template` (standard library), which
supports this syntax.
