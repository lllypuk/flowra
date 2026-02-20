# Fix: Remaining Issues from PR #26 Review

## Status: Pending

## Overview

Four issues identified in the PR #26 code review that require implementation.
Issue #11 (file download authorization) is already complete — see `fix-file-download-authorization.md`.

---

## Issue 8: Task Attachments Not Persisted

### Problem

`uploadTaskFile` in `web/templates/task/sidebar.html` uploads a file to `/api/v1/files/upload`
and then refreshes the sidebar via HTMX, but the upload response `file_id` is never associated
with the task. After refresh, the sidebar shows no new attachment because there is no server-side
record of the file belonging to this task.

Similarly, `removeTaskAttachment` only removes the DOM element without making a server-side
delete/unlink call:

```javascript
function removeTaskAttachment(fileId, fileName) {
    if (!confirm('Remove attachment "' + fileName + '"?')) return;
    // For now just remove the element visually
    var el = document.getElementById('task-att-' + fileId);
    if (el) el.remove();
}
```

### Required Changes

**Backend:**

1. Add a `POST /api/v1/workspaces/:workspace_id/chats/:chat_id/tasks/:task_id/attachments`
   endpoint that accepts `{ file_id: "uuid" }` and persists the association.
2. Add a `DELETE /api/v1/workspaces/:workspace_id/chats/:chat_id/tasks/:task_id/attachments/:file_id`
   endpoint that removes the association (and optionally deletes the stored file).
3. Add `Attachments []AttachmentRef` to the task read model (or load them from `file_metadata`
   filtered by `task_id`).
4. Expose attachments in the task detail partial response.

**Frontend (`web/templates/task/sidebar.html`):**

Update `uploadTaskFile` to call the attach endpoint after upload:

```javascript
.then(function(result) {
    if (result.data && result.data.file_id) {
        return fetch('/api/v1/workspaces/' + workspaceId +
                     '/chats/' + chatId + '/tasks/' + taskId + '/attachments', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ file_id: result.data.file_id })
        });
    }
}).then(function() {
    htmx.ajax('GET', '/partials/tasks/' + taskId + '/sidebar', { ... });
})
```

Update `removeTaskAttachment` to call the delete endpoint:

```javascript
function removeTaskAttachment(fileId, taskId, workspaceId, chatId, fileName) {
    if (!confirm('Remove attachment "' + fileName + '"?')) return;
    fetch('/api/v1/workspaces/' + workspaceId +
          '/chats/' + chatId + '/tasks/' + taskId + '/attachments/' + fileId, {
        method: 'DELETE'
    }).then(function() {
        var el = document.getElementById('task-att-' + fileId);
        if (el) el.remove();
    });
}
```

### Checklist

- [ ] Design task attachment storage (new collection or extend `file_metadata` with `task_id`)
- [ ] Add `AttachTaskFile` use case or extend existing task use cases
- [ ] Add attach/detach HTTP handler methods
- [ ] Register routes in `cmd/api/routes.go`
- [ ] Wire handler in `cmd/api/container.go`
- [ ] Update task read model / sidebar partial to include attachments
- [ ] Update `uploadTaskFile` JS to call attach endpoint after upload
- [ ] Update `removeTaskAttachment` JS to call detach endpoint
- [ ] Add tests

---

## Issue 9: Message Attachments Not Persisted

### Problem

`uploadPendingFiles` in `web/templates/components/message_form.html` uploads files to
`/api/v1/files/upload` but never associates the returned `file_id` with the message.
After upload, the code reloads the message via HTMX, but the message model has no
attachments stored, so nothing new appears:

```javascript
.then(function(result) {
    if (result.data) {
        // Reload the message to show attachment
        setTimeout(function() {
            htmx.ajax('GET', '/partials/messages/' + messageId, { ... });
        }, 300);
        // ❌ Never calls an endpoint to associate file_id with messageId
    }
})
```

### Required Changes

**Backend:**

1. Add a `POST /api/v1/messages/:message_id/attachments` endpoint (or extend the
   message-send request to accept `attachment_ids[]`):
   - Preferred: include uploaded `file_id` values in the original send-message request
     to avoid a separate round-trip.
   - Alternative: a dedicated attach endpoint called after upload completes.
2. Store attachment references on the `Message` aggregate / read model.
3. Return attachments in the message partial response so the template can render them.

**Frontend (`web/templates/components/message_form.html`):**

Option A — attach after upload:

```javascript
.then(function(result) {
    if (result.data && result.data.file_id) {
        return fetch('/api/v1/messages/' + messageId + '/attachments', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ file_id: result.data.file_id })
        });
    }
}).then(function() {
    htmx.ajax('GET', '/partials/messages/' + messageId, { ... });
})
```

Option B — collect file IDs before sending, include in the message form as hidden inputs.
This keeps the flow as a single request but requires the file to be pre-uploaded before
the message form is submitted.

### Checklist

- [ ] Decide on attach approach (post-send endpoint vs. IDs in send request)
- [ ] Add `AddAttachment` use case / extend `PostMessage` use case
- [ ] Add HTTP endpoint and register route
- [ ] Wire handler in `cmd/api/container.go`
- [ ] Update message read model to include attachments
- [ ] Update message partial template to render attachments
- [ ] Update `uploadPendingFiles` JS to call attach endpoint after each upload
- [ ] Add tests

---

## Issue 10: File Storage Directory Hardcoded in container.go — ✅ COMPLETE

### Problem

`cmd/api/container.go:1604` initializes file storage with a hardcoded string
instead of reading from the application config:

```go
fileStorage, fileErr := filestorage.NewLocalStorage("uploads")
```

The `configs/config.yaml` already defines `uploads.dir` and `uploads.max_file_size`,
but the `Config` struct has no corresponding field, so these settings are never loaded.

### Required Changes

**`internal/config/config.go`:**

Add `UploadConfig` struct and include it in `Config`:

```go
// UploadConfig holds file upload configuration.
type UploadConfig struct {
    Dir         string `yaml:"dir"          env:"UPLOADS_DIR"`
    MaxFileSize int64  `yaml:"max_file_size" env:"UPLOADS_MAX_FILE_SIZE"`
}

type Config struct {
    // ... existing fields ...
    Uploads UploadConfig `yaml:"uploads"`
}
```

**`cmd/api/container.go`:**

Replace the hardcoded string with the config value:

```go
uploadDir := c.Config.Uploads.Dir
if uploadDir == "" {
    uploadDir = "uploads" // fallback default
}
fileStorage, fileErr := filestorage.NewLocalStorage(uploadDir)
```

Also propagate `MaxFileSize` to the file handler so it uses the configured limit
instead of the hardcoded `maxUploadSize` constant in `file_handler.go`.

### Checklist

- [x] Add `UploadConfig` struct to `internal/config/config.go`
- [x] Add `Uploads UploadConfig` field to `Config`
- [x] Update `cmd/api/container.go` to use `c.Config.Uploads.Dir`
- [x] Pass `c.Config.Uploads.MaxFileSize` to the file handler (replace hardcoded limit)
- [ ] Update `configs/config.dev.yaml` / `configs/config.prod.yaml` if needed
- [ ] Add config validation (e.g. dir must not be empty)
- [ ] Update config tests

---

## Issue 12: Misleading hx-confirm Copy for Workspace Delete — ✅ COMPLETE (Option A)

### Problem

The Delete Workspace button in `web/templates/workspace/settings.html:204` uses
`hx-confirm` with text that tells the user to "type the workspace name to confirm",
but HTMX's built-in `hx-confirm` only shows a simple browser `window.confirm()` dialog
with OK/Cancel — there is no text input field.

```html
hx-confirm="Are you sure you want to delete '{{.Data.Workspace.Name}}'?
            This action cannot be undone. Type the workspace name to confirm."
```

The instruction to type the workspace name is never shown as an input and cannot be
acted on, making the copy misleading.

### Resolution

Applied **Option A** — updated `hx-confirm` text to match actual behavior by removing
the "Type the workspace name to confirm" instruction and replacing with a clear warning
about permanent data loss.

### Checklist

- [x] Choose Option A (quick fix) or Option B (modal)
- [x] Update `web/templates/workspace/settings.html`
- [ ] ~~If Option B: add JS functions inside the template `<script>` block~~ (N/A — Option A chosen)
- [ ] Test the confirmation flow manually
