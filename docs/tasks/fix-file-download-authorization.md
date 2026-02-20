# Fix: Missing Authorization Check on File Downloads

## Status: Pending

## Severity: High

## Problem

The `Download` handler in `internal/handler/http/file_handler.go:123` only checks that the user
is authenticated (has a valid JWT), but does **not** verify whether the user has permission to
access the requested file. Any authenticated user can download any file uploaded to any chat
by knowing (or guessing) the file ID.

### Current Code Flow

```
GET /api/v1/files/:file_id/:file_name
    → middleware.GetUserID(c)         ← checks authentication only
    → uuid.ParseUUID(fileIDStr)       ← validates UUID format
    → h.storage.Exists(fileID, name)  ← checks file exists on disk
    → c.File(filePath)                ← serves file
    // ❌ No check: does this user have access to a chat containing this file?
```

### Attack Scenario

1. User A uploads a confidential document to a private chat → gets file ID `abc-123`
2. User B (authenticated but not in that chat) constructs URL:
   `GET /api/v1/files/abc-123/document.pdf`
3. Server serves the file — User B reads confidential data

UUIDs are not secret — they can leak via browser history, logs, referrer headers, or
shared links. File security must not depend on UUID unpredictability.

## Approach

Store a mapping from file → message → chat when files are uploaded. On download, verify
the requesting user is a member of the chat that contains the file's message.

## Files to Modify

### 1. New: File Metadata Storage

Option A: **Add a `file_metadata` MongoDB collection** that stores file ownership:

```go
// Collection: file_metadata
{
    "file_id":    "uuid",
    "message_id": "uuid",     // message the file is attached to
    "chat_id":    "uuid",     // chat the message belongs to
    "uploader_id": "uuid",   // user who uploaded the file
    "uploaded_at": "datetime"
}
```

Option B: **Store file metadata in the existing message attachment** — add `ChatID` to the
attachment event and look up chat membership when downloading. This avoids a new collection
but requires loading the message to find the chat.

**Recommended: Option A** — a dedicated collection is simpler, faster to query, and doesn't
require changes to the domain model. The collection is small (one doc per file) and can be
indexed on `file_id`.

### 2. `internal/infrastructure/filestorage/local.go` — Add Metadata Interface

Define a `FileMetadata` struct and a `FileMetadataRepository` interface on the consumer side
(in the handler or a shared package):

```go
// FileMetadata holds ownership information for an uploaded file.
type FileMetadata struct {
    FileID     uuid.UUID
    MessageID  uuid.UUID
    ChatID     uuid.UUID
    UploaderID uuid.UUID
    UploadedAt time.Time
}
```

### 3. `internal/infrastructure/repository/` — New `file_metadata_repository.go`

Implement the MongoDB repository:

```go
type FileMetadataRepository struct {
    collection *mongo.Collection
}

func NewFileMetadataRepository(db *mongo.Database) *FileMetadataRepository {
    return &FileMetadataRepository{
        collection: db.Collection("file_metadata"),
    }
}

func (r *FileMetadataRepository) Save(ctx context.Context, meta FileMetadata) error {
    _, err := r.collection.InsertOne(ctx, bson.M{
        "file_id":     meta.FileID.String(),
        "message_id":  meta.MessageID.String(),
        "chat_id":     meta.ChatID.String(),
        "uploader_id": meta.UploaderID.String(),
        "uploaded_at":  meta.UploadedAt,
    })
    return err
}

func (r *FileMetadataRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*FileMetadata, error) {
    // Find and decode
}
```

### 4. `internal/infrastructure/mongodb/indexes.go` — Add Index

Add an index for `file_metadata` collection:

```go
func GetFileMetadataIndexes() []mongo.IndexModel {
    return []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "file_id", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
    }
}
```

Update `CreateAllIndexes` to include the new indexes.

### 5. `internal/handler/http/file_handler.go` — Add Authorization

Add a membership checker interface and update the handler:

```go
// FileChatMembershipChecker verifies user access to a chat.
type FileChatMembershipChecker interface {
    IsMember(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) (bool, error)
}

// FileMetadataLookup retrieves file ownership metadata.
type FileMetadataLookup interface {
    FindByFileID(ctx context.Context, fileID uuid.UUID) (*FileMetadata, error)
}

type FileHandler struct {
    storage          *filestorage.LocalStorage
    metadataRepo     FileMetadataLookup
    membershipCheck  FileChatMembershipChecker
}
```

Update `Download`:

```go
func (h *FileHandler) Download(c echo.Context) error {
    userID := middleware.GetUserID(c)
    if userID.IsZero() {
        return httpserver.RespondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
    }

    // ... parse fileID and fileName (existing code) ...

    // Authorization: verify user has access to the file's chat
    meta, metaErr := h.metadataRepo.FindByFileID(c.Request().Context(), fileID)
    if metaErr != nil {
        return httpserver.RespondErrorWithCode(
            c, http.StatusNotFound, "FILE_NOT_FOUND", "file not found")
    }

    isMember, memberErr := h.membershipCheck.IsMember(c.Request().Context(), meta.ChatID, userID)
    if memberErr != nil || !isMember {
        return httpserver.RespondErrorWithCode(
            c, http.StatusForbidden, "FORBIDDEN", "you do not have access to this file")
    }

    // ... rest of existing Download logic ...
}
```

### 6. `internal/handler/http/file_handler.go` — Save Metadata on Upload

Update `Upload` to store metadata after saving the file. This requires knowing the `chatID`
and `messageID`. Since files are uploaded before the message is created, there are two options:

**Option A:** Upload returns just the file ID; metadata is created when the message with the
attachment is actually posted. This requires updating the message posting flow to call
`metadataRepo.Save()`.

**Option B:** Accept `chat_id` as a form field during upload and verify membership before
allowing the upload. This is simpler and also prevents unauthorized uploads.

**Recommended: Option B** — add `chat_id` to the upload form:

```go
func (h *FileHandler) Upload(c echo.Context) error {
    // ... existing auth check ...

    chatIDStr := c.FormValue("chat_id")
    if chatIDStr == "" {
        return httpserver.RespondErrorWithCode(
            c, http.StatusBadRequest, "MISSING_CHAT_ID", "chat_id is required")
    }
    chatID, parseErr := uuid.ParseUUID(chatIDStr)
    if parseErr != nil {
        return httpserver.RespondErrorWithCode(
            c, http.StatusBadRequest, "INVALID_CHAT_ID", "invalid chat ID format")
    }

    // Verify user is member of this chat
    isMember, memberErr := h.membershipCheck.IsMember(ctx, chatID, userID)
    if memberErr != nil || !isMember {
        return httpserver.RespondErrorWithCode(
            c, http.StatusForbidden, "FORBIDDEN", "you are not a member of this chat")
    }

    // ... existing file save logic ...

    // Save metadata
    _ = h.metadataRepo.Save(ctx, FileMetadata{
        FileID:     fileID,
        ChatID:     chatID,
        UploaderID: userID,
        UploadedAt: time.Now(),
    })

    // ... return response ...
}
```

### 7. `cmd/api/container.go` — Wire New Dependencies

Update handler initialization to inject the new repository and membership checker:

```go
fileMetadataRepo := repository.NewFileMetadataRepository(db)
fileHandler := httphandler.NewFileHandler(fileStorage, fileMetadataRepo, membershipChecker)
```

### 8. Frontend — Update Upload Form

Update the chat JavaScript (`web/static/js/chat.js` or the upload form template) to include
`chat_id` in the file upload request. Find the file upload `FormData` construction and add:

```javascript
formData.append('chat_id', chatId);
```

## Checklist

- [ ] Create `FileMetadata` struct (handler package or shared)
- [ ] Create `file_metadata_repository.go` in `internal/infrastructure/repository/`
- [ ] Add `file_metadata` index in `internal/infrastructure/mongodb/indexes.go`
- [ ] Define `FileMetadataLookup` and `FileChatMembershipChecker` interfaces in handler
- [ ] Update `FileHandler` struct to accept new dependencies
- [ ] Update `Upload` to accept `chat_id`, verify membership, and save metadata
- [ ] Update `Download` to look up metadata and verify membership
- [ ] Update `NewFileHandler` constructor and wiring in `container.go`
- [ ] Update frontend upload form to send `chat_id`
- [ ] Handle migration: existing files without metadata (option: serve without auth check
      for files uploaded before migration, or backfill metadata from message attachments)
- [ ] Add integration tests for authorized and unauthorized download attempts
- [ ] Run `go test ./...`
- [ ] Run `golangci-lint run`
