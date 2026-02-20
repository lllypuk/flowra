# Implementation: Issue 9 — Message Attachments Not Persisted

## Status: Complete

## Problem

`uploadPendingFiles` in `web/templates/components/message_form.html` uploads files to
`/api/v1/files/upload` but never associates the returned `file_id` with the message.
After upload, the code tries to reload the message partial, but the message has no
attachment stored because the `AddAttachmentUseCase` was never called.

The backend side is **already fully implemented**:
- Domain: `message.AddAttachment()` method exists
- Domain: `Attachment` value object exists (`internal/domain/message/attachment.go`)
- Domain: `AttachmentAdded` event exists (`internal/domain/message/events.go`)
- Use case: `AddAttachmentUseCase` exists (`internal/application/message/add_attachment.go`)
- Service: `MessageService.AddAttachment()` wired (`internal/service/message_service.go:174`)
- Container: `AddAttachmentUC` initialized and wired (`cmd/api/container.go:701-704`)
- MongoDB: `attachmentDocument` schema and serialization exist
- Handler: `AttachmentResponse` structs and conversion exist
- Tests: `add_attachment_test.go` covers success, multi-attach, not-author, deleted, etc.

**What's missing:**
1. No HTTP endpoint exposed for `AddAttachment` (no route registered)
2. Frontend never calls the attach endpoint after uploading a file

---

## Implementation Plan

### Step 1: Add HTTP Handler Method for AddAttachment

**File: `internal/handler/http/message_handler.go`**

Add `AddAttachment` to the `MessageService` interface:

```go
// In MessageService interface (line ~88), add:
AddAttachment(ctx context.Context, cmd messageapp.AddAttachmentCommand) (messageapp.Result, error)
```

Add handler method:

```go
// AddAttachment handles POST /api/v1/messages/:id/attachments.
func (h *MessageHandler) AddAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	messageIDStr := c.Param("id")
	messageID, parseErr := uuid.ParseUUID(messageIDStr)
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_MESSAGE_ID", "invalid message ID format")
	}

	var req struct {
		FileID   string `json:"file_id" form:"file_id"`
		FileName string `json:"file_name" form:"file_name"`
		FileSize int64  `json:"file_size" form:"file_size"`
		MimeType string `json:"mime_type" form:"mime_type"`
	}
	if err := c.Bind(&req); err != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	fileID, fileParseErr := uuid.ParseUUID(req.FileID)
	if fileParseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_ID", "invalid file ID format")
	}

	cmd := messageapp.AddAttachmentCommand{
		MessageID: messageID,
		FileID:    fileID,
		FileName:  req.FileName,
		FileSize:  req.FileSize,
		MimeType:  req.MimeType,
		UserID:    userID,
	}

	_, err := h.messageService.AddAttachment(c.Request().Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, messageapp.ErrMessageNotFound):
			return httpserver.RespondErrorWithCode(
				c, http.StatusNotFound, "NOT_FOUND", "message not found")
		case errors.Is(err, messageapp.ErrNotAuthor):
			return httpserver.RespondErrorWithCode(
				c, http.StatusForbidden, "FORBIDDEN", "only message author can add attachments")
		case errors.Is(err, messageapp.ErrMessageDeleted):
			return httpserver.RespondErrorWithCode(
				c, http.StatusBadRequest, "MESSAGE_DELETED", "cannot attach to deleted message")
		default:
			return httpserver.RespondErrorWithCode(
				c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to add attachment")
		}
	}

	return httpserver.RespondOK(c, map[string]string{"status": "attached"})
}
```

### Step 2: Register the Route

**File: `cmd/api/routes.go`**

In `registerMessageRoutes` (line ~161), add after the existing message routes:

```go
r.Auth().POST("/messages/:id/attachments", c.MessageHandler.AddAttachment)
```

Full context — the route section should look like:

```go
if c.MessageHandler != nil {
	messages.POST("", c.MessageHandler.Send)
	messages.GET("", c.MessageHandler.List)

	// Non-workspace-scoped message routes (message ID is globally unique)
	r.Auth().PUT("/messages/:id", c.MessageHandler.Edit)
	r.Auth().DELETE("/messages/:id", c.MessageHandler.Delete)
	r.Auth().POST("/messages/:id/attachments", c.MessageHandler.AddAttachment) // ← NEW
}
```

### Step 3: Update Frontend — `uploadPendingFiles`

**File: `web/templates/components/message_form.html` (line ~256)**

Replace the current `uploadPendingFiles` function with a version that calls the
attach endpoint after each file upload:

```javascript
function uploadPendingFiles(chatId, messageId) {
    var files = window.__pendingFiles[chatId] || [];
    if (files.length === 0) return;

    var remaining = files.length;

    files.forEach(function(file) {
        var formData = new FormData();
        formData.append('file', file);
        formData.append('chat_id', chatId);

        fetch('/api/v1/files/upload', {
            method: 'POST',
            body: formData
        }).then(function(resp) {
            if (!resp.ok) throw new Error('Upload failed');
            return resp.json();
        }).then(function(result) {
            if (result.data && result.data.file_id) {
                // Associate uploaded file with the message
                return fetch('/api/v1/messages/' + messageId + '/attachments', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        file_id: result.data.file_id,
                        file_name: result.data.file_name,
                        file_size: result.data.file_size,
                        mime_type: result.data.mime_type
                    })
                });
            }
        }).then(function() {
            remaining--;
            if (remaining <= 0) {
                // All attachments associated — reload the message
                htmx.ajax('GET', '/partials/messages/' + messageId, {
                    target: '#message-' + messageId,
                    swap: 'outerHTML'
                });
            }
        }).catch(function(err) {
            remaining--;
            console.error('File upload error:', err);
            showToast('Failed to upload ' + file.name, 'error');
        });
    });

    window.__pendingFiles[chatId] = [];
}
```

**Key changes from current code:**
1. After `POST /api/v1/files/upload` succeeds, calls `POST /api/v1/messages/{id}/attachments`
   with `file_id`, `file_name`, `file_size`, `mime_type` from the upload response
2. Tracks remaining file count to reload message only after all attachments are associated
3. Removes the arbitrary `setTimeout(300)` delay — reload triggers after attach completes

---

## Verification

After implementation, verify:
1. `go build ./...` passes
2. `go test ./internal/handler/http/... ./internal/application/message/...` passes
3. Manual test: send a message with attached file → file appears in rendered message after page reload
4. Manual test: try attaching to another user's message → 403 Forbidden

## Checklist

- [x] Add `AddAttachment` to `MessageService` interface in `message_handler.go`
- [x] Add `AddAttachment` handler method in `message_handler.go`
- [x] Register `POST /messages/:id/attachments` route in `routes.go`
- [x] Update `uploadPendingFiles` JS in `message_form.html`
- [x] Add handler tests for AddAttachment endpoint
- [ ] Manual verification
