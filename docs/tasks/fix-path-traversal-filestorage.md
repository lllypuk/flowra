# Fix: Path Traversal Protection in File Storage

## Status: Complete

## Severity: Critical (Defense-in-Depth)

## Problem

`LocalStorage.FilePath()` in `internal/infrastructure/filestorage/local.go:56` constructs the
file path without verifying that the resulting path stays within the `baseDir`:

```go
func (s *LocalStorage) FilePath(fileID uuid.UUID, fileName string) string {
    ext := filepath.Ext(fileName)
    return filepath.Join(s.baseDir, fileID.String()+ext)
}
```

Currently, `uuid.ParseUUID()` uses `google/uuid.Parse()` which strictly validates UUID format,
so the `fileID` component itself cannot contain path traversal sequences like `../`. This limits
the immediate attack surface.

However, the `fileName` parameter (from URL path `:file_name` in `file_handler.go:136`) is
user-controlled and passed without sanitization. While `filepath.Ext()` only extracts the
extension, there is no defense-in-depth guarantee that the resulting path remains inside
`baseDir`. If UUID parsing logic ever changes, or if `FilePath` is called from new code with
a different `fileID` source, path traversal becomes exploitable immediately.

The same issue applies to `Save()`, `Delete()`, and `Exists()` which all build paths from
user-influenced inputs.

### Attack Scenario (if UUID validation weakens)

1. Attacker requests `GET /api/v1/files/../../../etc/passwd/x`
2. If `fileID` bypasses UUID validation, `FilePath` joins `baseDir` with `../../../etc/passwd`
3. `c.File(filePath)` in `file_handler.go:163` serves arbitrary file from the filesystem

## Files to Modify

### 1. `internal/infrastructure/filestorage/local.go`

Add a `safePath` helper that validates the resolved path stays inside `baseDir`.
Apply it in `FilePath`, and return an error instead of just the path string.
Alternatively, keep `FilePath` signature but add a separate `ValidatePath` method.

**Recommended approach** — add validation directly in `FilePath` and change its signature to
return an error:

```go
import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/lllypuk/flowra/internal/domain/uuid"
)

// FilePath returns the full path to a stored file.
// Returns an error if the resolved path escapes the base directory.
func (s *LocalStorage) FilePath(fileID uuid.UUID, fileName string) (string, error) {
    ext := filepath.Ext(fileName)
    fullPath := filepath.Join(s.baseDir, fileID.String()+ext)
    cleanPath := filepath.Clean(fullPath)

    if !strings.HasPrefix(cleanPath, s.baseDir+string(filepath.Separator)) && cleanPath != s.baseDir {
        return "", fmt.Errorf("path traversal detected: resolved path is outside base directory")
    }

    return cleanPath, nil
}
```

**Impact on callers** — the following methods use `FilePath` and must be updated:

- `Delete()` (line 62) — propagate the error
- `Exists()` (line 71) — return `false` on error
- `file_handler.go:148` (`Download`) — return 400 on error
- `file_handler.go:143` (`Download` → `Exists`) — already returns bool, no change needed

Also sanitize `fileName` in `Save()` to prevent malicious extensions:

```go
func (s *LocalStorage) Save(reader io.Reader, originalName string) (uuid.UUID, error) {
    fileID := uuid.NewUUID()
    ext := filepath.Ext(filepath.Base(originalName)) // Use Base to strip directory components
    storedName := fileID.String() + ext

    filePath := filepath.Join(s.baseDir, storedName)
    cleanPath := filepath.Clean(filePath)

    if !strings.HasPrefix(cleanPath, s.baseDir+string(filepath.Separator)) {
        return "", fmt.Errorf("invalid file name: resolved path is outside base directory")
    }
    // ... rest unchanged
}
```

### 2. `internal/handler/http/file_handler.go`

Update `Download` handler to handle the new error from `FilePath`:

```go
filePath, pathErr := h.storage.FilePath(fileID, fileName)
if pathErr != nil {
    return httpserver.RespondErrorWithCode(
        c, http.StatusBadRequest, "INVALID_PATH", "invalid file path")
}
```

Also add `fileName` sanitization in `Download` to strip directory components:

```go
fileName := filepath.Base(c.Param("file_name"))
if fileName == "." || fileName == "" {
    return httpserver.RespondErrorWithCode(
        c, http.StatusBadRequest, "INVALID_FILE_NAME", "file name is required")
}
```

### 3. `internal/infrastructure/filestorage/local_test.go`

Add test cases:

- `TestFilePath_PathTraversal` — verify `FilePath` returns error for `../` sequences in fileID
- `TestFilePath_MaliciousFileName` — verify `FilePath` returns error for `../../etc/passwd.jpg`
- `TestSave_MaliciousFileName` — verify `Save` strips directory components from file name
- `TestFilePath_NormalCase` — verify normal operation still works

## Checklist

- [x] Add `safePath` validation to `FilePath` (change signature to return error)
- [x] Sanitize `originalName` in `Save` with `filepath.Base()`
- [x] Update `Delete` to propagate `FilePath` error
- [x] Update `Exists` to return false on `FilePath` error
- [x] Update `file_handler.go` Download to handle `FilePath` error
- [x] Sanitize `fileName` URL param in Download handler with `filepath.Base()`
- [x] Add unit tests for path traversal scenarios
- [x] Run `go test ./internal/infrastructure/filestorage/...`
- [x] Run `go test ./internal/handler/http/...`
- [x] Run `golangci-lint run`
