# Implementation: Issue 8 — Task Attachments Not Persisted

## Status: Complete

## Problem

`uploadTaskFile` in `web/templates/task/sidebar.html` uploads files to `/api/v1/files/upload`
and refreshes the sidebar, but the uploaded `file_id` is never associated with the task.
The `file_metadata` collection only stores `chat_id` and `uploader_id` — there is no `task_id`.
After refresh, the sidebar shows no new attachment because the task has no attachment data.

`removeTaskAttachment` only removes the DOM element without any backend call.

**Existing infrastructure that can be reused:**
- File upload endpoint: `POST /api/v1/files/upload` (works, stores file on disk + metadata in MongoDB)
- `FileMetadataEntry`: has `FileID`, `ChatID`, `UploaderID`, `UploadedAt`
- `LocalStorage`: stores files with UUID-based names, provides `Save/Delete/FilePath`
- Message attachment pattern: complete implementation to use as reference

---

## Architecture Decision

Tasks use event sourcing. Two options for storing attachments:

**Option A — Event-sourced on Task Aggregate** (recommended):
- Add `AttachmentAdded`/`AttachmentRemoved` events to task domain
- Add `attachments` field to the task aggregate
- Add to the task read model and `taskReadModelDocument`
- Follows existing patterns (`StatusChanged`, `PriorityChanged`, etc.)
- Full audit trail for attachments

**Option B — Separate collection (not recommended)**:
- Store in `task_attachments` collection with `task_id` + `file_id`
- Avoids modifying the event-sourced aggregate
- But breaks the event sourcing pattern and loses audit trail

**Decision: Option A** — follows established project patterns.

---

## Implementation Plan

### Step 1: Add Attachment Value Object to Task Domain

**File: `internal/domain/task/attachment.go`** (new file)

```go
package task

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment represents a file attached to a task.
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment creates a validated task attachment.
func NewAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) (Attachment, error) {
	if fileID.IsZero() {
		return Attachment{}, errs.ErrInvalidInput
	}
	if fileName == "" {
		return Attachment{}, errs.ErrInvalidInput
	}
	if fileSize <= 0 {
		return Attachment{}, errs.ErrInvalidInput
	}
	if mimeType == "" {
		return Attachment{}, errs.ErrInvalidInput
	}
	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}, nil
}

// ReconstructAttachment creates an Attachment from persisted data (no validation).
func ReconstructAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) Attachment {
	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}
}

func (a Attachment) FileID() uuid.UUID { return a.fileID }
func (a Attachment) FileName() string  { return a.fileName }
func (a Attachment) FileSize() int64   { return a.fileSize }
func (a Attachment) MimeType() string  { return a.mimeType }

// IsImage returns true if the attachment is an image type.
func (a Attachment) IsImage() bool {
	switch a.mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml":
		return true
	default:
		return false
	}
}
```

### Step 2: Add Events to Task Domain

**File: `internal/domain/task/events.go`** (add to existing file)

Add two new event types and constants:

```go
const (
	EventTypeAttachmentAdded   = "task.attachment.added"
	EventTypeAttachmentRemoved = "task.attachment.removed"
)

// AttachmentAdded is raised when a file is attached to a task.
type AttachmentAdded struct {
	event.BaseEvent
	FileID   uuid.UUID `json:"file_id"`
	FileName string    `json:"file_name"`
	FileSize int64     `json:"file_size"`
	MimeType string    `json:"mime_type"`
	AddedBy  uuid.UUID `json:"added_by"`
}

// NewAttachmentAdded creates a new AttachmentAdded event.
func NewAttachmentAdded(
	aggregateID uuid.UUID,
	fileID uuid.UUID,
	fileName string,
	fileSize int64,
	mimeType string,
	addedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *AttachmentAdded {
	return &AttachmentAdded{
		BaseEvent: event.NewBaseEvent(
			aggregateID,
			EventTypeAttachmentAdded,
			version,
			metadata,
		),
		FileID:   fileID,
		FileName: fileName,
		FileSize: fileSize,
		MimeType: mimeType,
		AddedBy:  addedBy,
	}
}

// AttachmentRemoved is raised when a file is detached from a task.
type AttachmentRemoved struct {
	event.BaseEvent
	FileID    uuid.UUID `json:"file_id"`
	RemovedBy uuid.UUID `json:"removed_by"`
}

// NewAttachmentRemoved creates a new AttachmentRemoved event.
func NewAttachmentRemoved(
	aggregateID uuid.UUID,
	fileID uuid.UUID,
	removedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *AttachmentRemoved {
	return &AttachmentRemoved{
		BaseEvent: event.NewBaseEvent(
			aggregateID,
			EventTypeAttachmentRemoved,
			version,
			metadata,
		),
		FileID:    fileID,
		RemovedBy: removedBy,
	}
}
```

### Step 3: Add Aggregate Methods to Task

**File: `internal/domain/task/task.go`** (modify existing file)

Add `attachments` field to Aggregate struct:

```go
type Aggregate struct {
	// ... existing fields ...
	attachments []Attachment // ← add this field
}
```

Add getter:

```go
// Attachments returns the task's file attachments.
func (a *Aggregate) Attachments() []Attachment {
	return a.attachments
}
```

Add command methods:

```go
// AddAttachment attaches a file to the task.
func (a *Aggregate) AddAttachment(
	fileID uuid.UUID, fileName string, fileSize int64, mimeType string, addedBy uuid.UUID,
) error {
	// Check for duplicate
	for _, att := range a.attachments {
		if att.FileID() == fileID {
			return nil // idempotent
		}
	}

	evt := NewAttachmentAdded(
		a.id, fileID, fileName, fileSize, mimeType, addedBy,
		a.version+1,
		event.Metadata{UserID: addedBy.String()},
	)
	a.applyChange(evt)
	a.uncommittedEvents = append(a.uncommittedEvents, evt)
	return nil
}

// RemoveAttachment detaches a file from the task.
func (a *Aggregate) RemoveAttachment(fileID uuid.UUID, removedBy uuid.UUID) error {
	found := false
	for _, att := range a.attachments {
		if att.FileID() == fileID {
			found = true
			break
		}
	}
	if !found {
		return nil // idempotent
	}

	evt := NewAttachmentRemoved(
		a.id, fileID, removedBy,
		a.version+1,
		event.Metadata{UserID: removedBy.String()},
	)
	a.applyChange(evt)
	a.uncommittedEvents = append(a.uncommittedEvents, evt)
	return nil
}
```

Add cases to `applyChange`:

```go
func (a *Aggregate) applyChange(evt event.DomainEvent) {
	switch e := evt.(type) {
	// ... existing cases ...
	case *AttachmentAdded:
		a.attachments = append(a.attachments, ReconstructAttachment(
			e.FileID, e.FileName, e.FileSize, e.MimeType,
		))
	case *AttachmentRemoved:
		filtered := make([]Attachment, 0, len(a.attachments))
		for _, att := range a.attachments {
			if att.FileID() != e.FileID {
				filtered = append(filtered, att)
			}
		}
		a.attachments = filtered
	}
	a.version++
	a.appliedEventCounts++
}
```

### Step 4: Add Application Layer Commands and Use Cases

**File: `internal/application/task/commands.go`** (add to existing file)

```go
// AddAttachmentCommand attaches a file to a task.
type AddAttachmentCommand struct {
	TaskID   uuid.UUID
	FileID   uuid.UUID
	FileName string
	FileSize int64
	MimeType string
	AddedBy  uuid.UUID
}

func (c AddAttachmentCommand) CommandName() string { return "AddTaskAttachment" }

// RemoveAttachmentCommand detaches a file from a task.
type RemoveAttachmentCommand struct {
	TaskID    uuid.UUID
	FileID    uuid.UUID
	RemovedBy uuid.UUID
}

func (c RemoveAttachmentCommand) CommandName() string { return "RemoveTaskAttachment" }
```

**File: `internal/application/task/add_attachment.go`** (new file)

```go
package task

import (
	"context"
	"fmt"
)

// AddAttachmentUseCase handles adding attachments to tasks.
type AddAttachmentUseCase struct {
	taskRepo CommandRepository
}

// NewAddAttachmentUseCase creates a new AddAttachmentUseCase.
func NewAddAttachmentUseCase(taskRepo CommandRepository) *AddAttachmentUseCase {
	return &AddAttachmentUseCase{taskRepo: taskRepo}
}

// Execute adds an attachment to a task.
func (uc *AddAttachmentUseCase) Execute(
	ctx context.Context,
	cmd AddAttachmentCommand,
) (TaskResult, error) {
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	task, err := uc.taskRepo.Load(ctx, cmd.TaskID)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to load task: %w", err)
	}

	if addErr := task.AddAttachment(
		cmd.FileID, cmd.FileName, cmd.FileSize, cmd.MimeType, cmd.AddedBy,
	); addErr != nil {
		return TaskResult{}, addErr
	}

	events := task.UncommittedEvents()
	if len(events) == 0 {
		return TaskResult{
			TaskID:  task.ID(),
			Version: task.Version(),
			Success: true,
			Message: "attachment already exists",
		}, nil
	}

	if saveErr := uc.taskRepo.Save(ctx, task); saveErr != nil {
		return TaskResult{}, fmt.Errorf("failed to save task: %w", saveErr)
	}

	return TaskResult{
		TaskID:  task.ID(),
		Version: task.Version(),
		Events:  events,
		Success: true,
		Message: "attachment added",
	}, nil
}

func (uc *AddAttachmentUseCase) validate(cmd AddAttachmentCommand) error {
	if cmd.TaskID.IsZero() {
		return fmt.Errorf("task_id is required")
	}
	if cmd.FileID.IsZero() {
		return fmt.Errorf("file_id is required")
	}
	if cmd.FileName == "" {
		return fmt.Errorf("file_name is required")
	}
	if cmd.FileSize <= 0 {
		return fmt.Errorf("file_size must be positive")
	}
	if cmd.MimeType == "" {
		return fmt.Errorf("mime_type is required")
	}
	if cmd.AddedBy.IsZero() {
		return fmt.Errorf("added_by is required")
	}
	return nil
}
```

**File: `internal/application/task/remove_attachment.go`** (new file)

```go
package task

import (
	"context"
	"fmt"
)

// RemoveAttachmentUseCase handles removing attachments from tasks.
type RemoveAttachmentUseCase struct {
	taskRepo CommandRepository
}

// NewRemoveAttachmentUseCase creates a new RemoveAttachmentUseCase.
func NewRemoveAttachmentUseCase(taskRepo CommandRepository) *RemoveAttachmentUseCase {
	return &RemoveAttachmentUseCase{taskRepo: taskRepo}
}

// Execute removes an attachment from a task.
func (uc *RemoveAttachmentUseCase) Execute(
	ctx context.Context,
	cmd RemoveAttachmentCommand,
) (TaskResult, error) {
	if cmd.TaskID.IsZero() {
		return TaskResult{}, fmt.Errorf("task_id is required")
	}
	if cmd.FileID.IsZero() {
		return TaskResult{}, fmt.Errorf("file_id is required")
	}
	if cmd.RemovedBy.IsZero() {
		return TaskResult{}, fmt.Errorf("removed_by is required")
	}

	task, err := uc.taskRepo.Load(ctx, cmd.TaskID)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to load task: %w", err)
	}

	if removeErr := task.RemoveAttachment(cmd.FileID, cmd.RemovedBy); removeErr != nil {
		return TaskResult{}, removeErr
	}

	events := task.UncommittedEvents()
	if len(events) == 0 {
		return TaskResult{
			TaskID:  task.ID(),
			Version: task.Version(),
			Success: true,
			Message: "attachment not found",
		}, nil
	}

	if saveErr := uc.taskRepo.Save(ctx, task); saveErr != nil {
		return TaskResult{}, fmt.Errorf("failed to save task: %w", saveErr)
	}

	return TaskResult{
		TaskID:  task.ID(),
		Version: task.Version(),
		Events:  events,
		Success: true,
		Message: "attachment removed",
	}, nil
}
```

### Step 5: Update Task Read Model

**File: `internal/application/task/repository.go`** (modify existing `ReadModel`)

Add `Attachments` field:

```go
type ReadModel struct {
	// ... existing fields ...
	Attachments []AttachmentReadModel // ← add this
}

// AttachmentReadModel represents an attachment in the task read model.
type AttachmentReadModel struct {
	FileID   uuid.UUID
	FileName string
	FileSize int64
	MimeType string
}
```

### Step 6: Update MongoDB Task Repository

**File: `internal/infrastructure/repository/mongodb/task_repository.go`**

Add attachment document struct (near `taskReadModelDocument`):

```go
type taskAttachmentDocument struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FileSize int64  `bson:"file_size"`
	MimeType string `bson:"mime_type"`
}
```

Add to `taskReadModelDocument`:

```go
type taskReadModelDocument struct {
	// ... existing fields ...
	Attachments []taskAttachmentDocument `bson:"attachments,omitempty"` // ← add this
}
```

Update `updateReadModel` method to include attachments:

```go
func (r *MongoTaskRepository) updateReadModel(ctx context.Context, task *taskdomain.Aggregate) error {
	// ... existing field serialization ...

	// Convert attachments
	var attachmentDocs []taskAttachmentDocument
	for _, a := range task.Attachments() {
		attachmentDocs = append(attachmentDocs, taskAttachmentDocument{
			FileID:   a.FileID().String(),
			FileName: a.FileName(),
			FileSize: a.FileSize(),
			MimeType: a.MimeType(),
		})
	}
	// Include in update document:
	// "attachments": attachmentDocs
}
```

Update `documentToReadModel` (or equivalent converter) to deserialize attachments:

```go
func documentToReadModel(doc taskReadModelDocument) *taskapp.ReadModel {
	rm := &taskapp.ReadModel{
		// ... existing fields ...
	}
	for _, a := range doc.Attachments {
		fileID, _ := uuid.ParseUUID(a.FileID)
		rm.Attachments = append(rm.Attachments, taskapp.AttachmentReadModel{
			FileID:   fileID,
			FileName: a.FileName,
			FileSize: a.FileSize,
			MimeType: a.MimeType,
		})
	}
	return rm
}
```

Also register `AttachmentAdded`/`AttachmentRemoved` in the event deserialization map (if the
event store uses type-based registration). Check `internal/infrastructure/eventstore/` for
an `init()` or registration function and add the new event types.

### Step 7: Add HTTP Endpoints

**File: `internal/handler/http/task_handler.go`** (modify existing)

Add methods to `TaskService` interface:

```go
type TaskService interface {
	// ... existing methods ...
	AddAttachment(ctx context.Context, cmd taskapp.AddAttachmentCommand) (taskapp.TaskResult, error)
	RemoveAttachment(ctx context.Context, cmd taskapp.RemoveAttachmentCommand) (taskapp.TaskResult, error)
}
```

Add handler methods:

```go
// AddAttachment handles POST /api/v1/workspaces/:workspace_id/tasks/:task_id/attachments.
func (h *TaskHandler) AddAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskID, parseErr := uuid.ParseUUID(c.Param("task_id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
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

	cmd := taskapp.AddAttachmentCommand{
		TaskID:   taskID,
		FileID:   fileID,
		FileName: req.FileName,
		FileSize: req.FileSize,
		MimeType: req.MimeType,
		AddedBy:  userID,
	}

	_, err := h.taskService.AddAttachment(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to add attachment")
	}

	return httpserver.RespondOK(c, map[string]string{"status": "attached"})
}

// RemoveAttachment handles DELETE /api/v1/workspaces/:workspace_id/tasks/:task_id/attachments/:file_id.
func (h *TaskHandler) RemoveAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return httpserver.RespondErrorWithCode(
			c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	taskID, parseErr := uuid.ParseUUID(c.Param("task_id"))
	if parseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID format")
	}

	fileID, fileParseErr := uuid.ParseUUID(c.Param("file_id"))
	if fileParseErr != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusBadRequest, "INVALID_FILE_ID", "invalid file ID format")
	}

	cmd := taskapp.RemoveAttachmentCommand{
		TaskID:    taskID,
		FileID:    fileID,
		RemovedBy: userID,
	}

	_, err := h.taskService.RemoveAttachment(c.Request().Context(), cmd)
	if err != nil {
		return httpserver.RespondErrorWithCode(
			c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to remove attachment")
	}

	return httpserver.RespondOK(c, map[string]string{"status": "removed"})
}
```

### Step 8: Register Routes

**File: `cmd/api/routes.go`** (in `registerTaskRoutes`)

Add after existing task routes (line ~215):

```go
tasks.POST("/:task_id/attachments", c.TaskHandler.AddAttachment)
tasks.DELETE("/:task_id/attachments/:file_id", c.TaskHandler.RemoveAttachment)
```

### Step 9: Wire Use Cases in Container

**File: `cmd/api/container.go`**

Add fields to Container:

```go
AddTaskAttachmentUC    *taskapp.AddAttachmentUseCase
RemoveTaskAttachmentUC *taskapp.RemoveAttachmentUseCase
```

Initialize in the task setup section (near other task use cases):

```go
c.AddTaskAttachmentUC = taskapp.NewAddAttachmentUseCase(c.TaskRepo)
c.RemoveTaskAttachmentUC = taskapp.NewRemoveAttachmentUseCase(c.TaskRepo)
```

Update `fullTaskServiceAdapter` to implement the new methods:

```go
func (a *fullTaskServiceAdapter) AddAttachment(
	ctx context.Context,
	cmd taskapp.AddAttachmentCommand,
) (taskapp.TaskResult, error) {
	uc := taskapp.NewAddAttachmentUseCase(a.taskRepo)
	return uc.Execute(ctx, cmd)
}

func (a *fullTaskServiceAdapter) RemoveAttachment(
	ctx context.Context,
	cmd taskapp.RemoveAttachmentCommand,
) (taskapp.TaskResult, error) {
	uc := taskapp.NewRemoveAttachmentUseCase(a.taskRepo)
	return uc.Execute(ctx, cmd)
}
```

### Step 10: Update Task Sidebar View Data

**File: `internal/handler/http/task_detail_template_handler.go`**

Update `TaskDetailViewData` to include attachments:

```go
type TaskDetailViewData struct {
	// ... existing fields ...
	Attachments []TaskAttachmentViewData // ← add this
}

type TaskAttachmentViewData struct {
	FileID   string
	FileName string
	FileSize int64
	MimeType string
	URL      string
	IsImage  bool
}
```

In the handler method that builds `TaskDetailViewData`, populate attachments from the read model:

```go
for _, a := range readModel.Attachments {
	viewData.Attachments = append(viewData.Attachments, TaskAttachmentViewData{
		FileID:   a.FileID.String(),
		FileName: a.FileName,
		FileSize: a.FileSize,
		MimeType: a.MimeType,
		URL:      fmt.Sprintf("/api/v1/files/%s/%s", a.FileID.String(), url.PathEscape(a.FileName)),
		IsImage:  strings.HasPrefix(a.MimeType, "image/"),
	})
}
```

### Step 11: Update Frontend — `uploadTaskFile`

**File: `web/templates/task/sidebar.html`** (line ~248)

Replace the current `uploadTaskFile` function:

```javascript
function uploadTaskFile(file, taskId, chatId) {
    if (file.size > 10 * 1024 * 1024) {
        showToast('File "' + file.name + '" exceeds 10 MB limit', 'error');
        return;
    }
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
            // Associate uploaded file with the task
            var workspaceId = '{{.Task.WorkspaceID}}';
            return fetch('/api/v1/workspaces/' + workspaceId + '/tasks/' + taskId + '/attachments', {
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
        showToast('File uploaded: ' + file.name, 'success');
        // Refresh sidebar to show the new attachment
        htmx.ajax('GET', '/partials/tasks/' + taskId + '/sidebar', {
            target: '#task-sidebar-' + taskId,
            swap: 'outerHTML'
        });
    }).catch(function(err) {
        console.error('Task file upload error:', err);
        showToast('Failed to upload ' + file.name, 'error');
    });
}
```

### Step 12: Update Frontend — `removeTaskAttachment`

**File: `web/templates/task/sidebar.html`** (line ~278)

Replace the current `removeTaskAttachment` function:

```javascript
function removeTaskAttachment(fileId, fileName) {
    if (!confirm('Remove attachment "' + fileName + '"?')) return;
    var workspaceId = '{{.Task.WorkspaceID}}';
    var taskId = '{{.Task.ID}}';
    fetch('/api/v1/workspaces/' + workspaceId + '/tasks/' + taskId + '/attachments/' + fileId, {
        method: 'DELETE'
    }).then(function(resp) {
        if (!resp.ok) throw new Error('Remove failed');
        var el = document.getElementById('task-att-' + fileId);
        if (el) el.remove();
        showToast('Attachment removed', 'success');
    }).catch(function(err) {
        console.error('Remove attachment error:', err);
        showToast('Failed to remove attachment', 'error');
    });
}
```

### Step 13: Update Chat Task Sidebar (if applicable)

**File: `web/templates/chat/task-sidebar.html`**

If this sidebar also shows attachments, update it with the same pattern. Check whether
this template references `removeTaskAttachment` or `uploadTaskFile` and apply the same changes.

---

## Event Store Registration

If the event store uses a type registry to deserialize events (check
`internal/infrastructure/eventstore/` for `Register`, `init()`, or a type map), add:

```go
Register(task.EventTypeAttachmentAdded, &task.AttachmentAdded{})
Register(task.EventTypeAttachmentRemoved, &task.AttachmentRemoved{})
```

Without this, the event store will fail to replay `AttachmentAdded`/`AttachmentRemoved`
events, breaking aggregate reconstruction.

---

## Verification

After implementation:
1. `go build ./...` passes
2. `go test ./internal/domain/task/... ./internal/application/task/... ./internal/handler/http/...` passes
3. Manual test: open task sidebar → upload file → sidebar refreshes → file appears
4. Manual test: click remove on attachment → confirm → attachment disappears + backend record removed
5. Manual test: close and reopen sidebar → attachments still visible (persisted)

## Checklist

- [x] Create `internal/domain/task/attachment.go` (value object)
- [x] Add `AttachmentAdded`/`AttachmentRemoved` events to `events.go`
- [x] Add `attachments` field and methods to task aggregate (`task.go`)
- [x] Add `applyChange` cases for new events
- [x] Add command structs to `commands.go`
- [x] Create `internal/application/task/add_attachment.go` (use case)
- [x] Create `internal/application/task/remove_attachment.go` (use case)
- [x] Add `AttachmentReadModel` to `ReadModel` in `repository.go`
- [x] Update MongoDB `taskReadModelDocument` with attachments
- [x] Update `updateReadModel` to persist attachments
- [x] Update document-to-read-model converter
- [x] Register new event types in event store (if type registry is used)
- [x] Add `AddAttachment`/`RemoveAttachment` to `TaskService` interface
- [x] Add handler methods to `TaskHandler`
- [x] Register routes in `routes.go`
- [x] Wire use cases in `container.go`
- [x] Update `fullTaskServiceAdapter` with new methods
- [x] Update `TaskDetailViewData` with attachments
- [x] Populate attachments in sidebar handler
- [x] Update `uploadTaskFile` JS to call attach endpoint
- [x] Update `removeTaskAttachment` JS to call detach endpoint
- [x] Update chat task sidebar template if needed
- [x] Add unit tests for domain methods
- [x] Add unit tests for use cases
- [x] Add handler tests
- [ ] Manual verification
