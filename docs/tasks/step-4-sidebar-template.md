# Step 4: Update Task Sidebar Template

## Status: Complete

## Goal

Redirect all field-change HTMX calls in `web/templates/task/sidebar.html` from the direct
`PUT /api/v1/.../tasks/:id/*` endpoints to the new `POST .../tasks/:id/actions/*` endpoints,
and remove the Activity Timeline section from the sidebar.

## File to Change

**`web/templates/task/sidebar.html`**

## Changes Overview

| Change | Location |
|--------|----------|
| Status select — `hx-put` → `hx-post`, new URL | Lines 36–38 |
| Priority select — `hx-put` → `hx-post`, new URL | Lines 54–56 |
| Assignee `user_select` — pass `HxPost` instead of `HxPut` | Lines 74–81 |
| Due date `date_picker` — pass `HxPost` instead of `HxPut` | Lines 87–92 |
| Quick-date JS buttons — update `setQuickDate` function | Lines 111–117, 210–248 |
| Remove Activity Timeline block | Lines 182–192 |
| Remove `.activity-timeline` CSS | Lines 417–420 |

---

## Change 1: Status Select

### Current (lines 36–42)

```html
<select hx-put="/api/v1/tasks/{{.Task.ID}}/status"
        hx-trigger="change"
        hx-swap="none"
        hx-indicator="#status-save-indicator"
        name="status"
        class="status-select status-{{.Task.Status | lower}}">
```

### Updated

```html
<select hx-post="/api/v1/workspaces/{{.Task.WorkspaceID}}/tasks/{{.Task.ID}}/actions/status"
        hx-trigger="change"
        hx-swap="none"
        hx-indicator="#status-save-indicator"
        name="status"
        class="status-select status-{{.Task.Status | lower}}">
```

---

## Change 2: Priority Select

### Current (lines 54–59)

```html
<select hx-put="/api/v1/tasks/{{.Task.ID}}/priority"
        hx-trigger="change"
        hx-swap="none"
        hx-indicator="#priority-save-indicator"
        name="priority"
        class="priority-select priority-{{.Task.Priority | lower}}">
```

### Updated

```html
<select hx-post="/api/v1/workspaces/{{.Task.WorkspaceID}}/tasks/{{.Task.ID}}/actions/priority"
        hx-trigger="change"
        hx-swap="none"
        hx-indicator="#priority-save-indicator"
        name="priority"
        class="priority-select priority-{{.Task.Priority | lower}}">
```

---

## Change 3: Assignee (`user_select` component)

The `user_select` component currently accepts only `HxPut`. After Step 5 adds `HxPost`
support to the component, change the parameter here from `HxPut` to `HxPost`.

### Current (lines 74–81)

```html
{{template "components/user_select" (dict
    "Name" "assignee_id"
    "Selected" .Task.AssigneeID
    "Users" .Participants
    "HxPut" (printf "/api/v1/tasks/%s/assignee" .Task.ID)
    "AllowEmpty" true
    "EmptyLabel" "Unassigned"
)}}
```

### Updated

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

---

## Change 4: Due Date (`date_picker` component)

The `date_picker` component currently accepts only `HxPut`. After Step 5 adds `HxPost`
support to the component, change the parameter here from `HxPut` to `HxPost`.

### Current (lines 87–92)

```html
{{template "components/date_picker" (dict
    "Name" "due_date"
    "Value" .Task.DueDate
    "HxPut" (printf "/api/v1/tasks/%s/due-date" .Task.ID)
    "AllowEmpty" true
)}}
```

### Updated

```html
{{template "components/date_picker" (dict
    "Name" "due_date"
    "Value" .Task.DueDate
    "HxPost" (printf "/api/v1/workspaces/%s/tasks/%s/actions/due-date" .Task.WorkspaceID .Task.ID)
    "AllowEmpty" true
)}}
```

---

## Change 5: Quick Date Buttons

### Current HTML (lines 111–117)

```html
<div class="quick-dates">
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', 0)">Today</button>
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', 1)">Tomorrow</button>
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', 7)">Next Week</button>
</div>
```

### Updated HTML

Add `WorkspaceID` as the second argument:

```html
<div class="quick-dates">
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', '{{.Task.WorkspaceID}}', 0)">Today</button>
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', '{{.Task.WorkspaceID}}', 1)">Tomorrow</button>
    <button type="button" class="btn-quick-date small outline"
            onclick="setQuickDate('{{.Task.ID}}', '{{.Task.WorkspaceID}}', 7)">Next Week</button>
</div>
```

### Current JS Function

The `setQuickDate` function does not currently exist in the template's `<script>` block.
The quick-date buttons are non-functional stubs. Replace with a working implementation:

### Updated JS (in the `<script>` block)

```javascript
function setQuickDate(taskId, workspaceId, daysFromNow) {
    var date = new Date();
    date.setDate(date.getDate() + daysFromNow);
    var dateStr = date.toISOString().split('T')[0];

    var url = '/api/v1/workspaces/' + workspaceId + '/tasks/' + taskId + '/actions/due-date';
    htmx.ajax('POST', url, {
        values: { due_date: dateStr },
        swap: 'none'
    });
}
```

---

## Change 6: Remove Activity Timeline Block

### Current (lines 182–192)

```html
<!-- Activity Timeline -->
<div class="field">
    <label>Activity</label>
    <div id="task-activity-{{.Task.ID}}"
         class="activity-timeline"
         hx-get="/partials/tasks/{{.Task.ID}}/activity"
         hx-trigger="load"
         hx-swap="innerHTML">
        {{template "components/loading" (dict "ID" "activity-loading")}}
    </div>
</div>
```

### Updated

Remove the entire block. The `<hr>` separator before it (line 180) and the description
block after it (starting at line 121 with its own `<hr>` at line 138) remain untouched.
Check the template structure to confirm the correct `<hr>` tags remain.

The sidebar's `<hr>` structure before this change:

```
<hr>  ← after due-date section (line 119)
<!-- Description -->
<hr>  ← after description (line 138)
<!-- Attachments -->
<hr>  ← after attachments (line 180)  ← REMOVE BOTH this HR and the activity block
<!-- Activity Timeline -->  ← REMOVE
```

After removing the activity block, also remove the `<hr>` at line 180 (there is no longer
a section following it that needs the separator).

---

## Change 7: Remove `.activity-timeline` CSS

### Current (lines 417–420 in the `<style>` block)

```css
.activity-timeline {
    max-height: 300px;
    overflow-y: auto;
}
```

### Updated

Remove these 4 lines. The class is no longer used in the template.

---

## Verification

After all changes, the sidebar should:

1. **Status change** → sends `POST .../actions/status` with `name="status"` form field
2. **Priority change** → sends `POST .../actions/priority` with `name="priority"` form field
3. **Assignee change** → sends `POST .../actions/assignee` with `name="assignee_id"` form field
4. **Due date change** → sends `POST .../actions/due-date` with `name="due_date"` form field
5. **Quick date button** → calls `setQuickDate(taskId, workspaceId, days)` → HTMX POST
6. **No activity section** → sidebar ends with Attachments

All HTMX requests use `hx-swap="none"` so no DOM swapping occurs on response.
The sidebar refreshes only when the WebSocket `task.updated` event arrives, which fires
after the tag processor completes the asynchronous update.

## Dependency

Changes 3 and 4 (assignee and due date) depend on Step 5 (component updates).
Do not apply Changes 3 and 4 until `user_select.html` and `date_picker.html` support
the `HxPost` parameter.
