package filestorage_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/filestorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *filestorage.LocalStorage {
	t.Helper()
	dir := t.TempDir()
	storage, err := filestorage.NewLocalStorage(dir)
	require.NoError(t, err)
	return storage
}

func TestNewLocalStorage(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "subdir", "uploads")
		storage, err := filestorage.NewLocalStorage(dir)
		require.NoError(t, err)
		require.NotNil(t, storage)

		info, statErr := os.Stat(dir)
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		dir := t.TempDir()
		storage, err := filestorage.NewLocalStorage(dir)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})
}

func TestLocalStorage_Save(t *testing.T) {
	t.Run("saves file and returns non-zero ID", func(t *testing.T) {
		storage := newTestStorage(t)
		content := "hello world"

		fileID, err := storage.Save(strings.NewReader(content), "test.txt")
		require.NoError(t, err)
		assert.False(t, fileID.IsZero())
	})

	t.Run("file exists on disk after save", func(t *testing.T) {
		storage := newTestStorage(t)
		content := "file content"

		fileID, err := storage.Save(strings.NewReader(content), "document.pdf")
		require.NoError(t, err)

		assert.True(t, storage.Exists(fileID, "document.pdf"))
	})

	t.Run("preserves file content", func(t *testing.T) {
		storage := newTestStorage(t)
		content := "preserved content"

		fileID, err := storage.Save(strings.NewReader(content), "data.txt")
		require.NoError(t, err)

		path, pathErr := storage.FilePath(fileID, "data.txt")
		require.NoError(t, pathErr)
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, content, string(data))
	})

	t.Run("preserves file extension", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("img"), "photo.png")
		require.NoError(t, err)

		path, pathErr := storage.FilePath(fileID, "photo.png")
		require.NoError(t, pathErr)
		assert.Equal(t, ".png", filepath.Ext(path))
	})

	t.Run("strips directory components from original name", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("data"), "../../etc/passwd.txt")
		require.NoError(t, err)

		assert.True(t, storage.Exists(fileID, "passwd.txt"))
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		storage := newTestStorage(t)

		id1, err1 := storage.Save(strings.NewReader("a"), "a.txt")
		require.NoError(t, err1)
		id2, err2 := storage.Save(strings.NewReader("b"), "b.txt")
		require.NoError(t, err2)

		assert.NotEqual(t, id1, id2)
	})

	t.Run("handles file without extension", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("data"), "Makefile")
		require.NoError(t, err)
		assert.True(t, storage.Exists(fileID, "Makefile"))
	})
}

func TestLocalStorage_FilePath(t *testing.T) {
	t.Run("returns path with file ID and extension", func(t *testing.T) {
		storage := newTestStorage(t)
		fileID := uuid.UUID("test-id-123")

		path, err := storage.FilePath(fileID, "report.pdf")
		require.NoError(t, err)
		assert.Contains(t, path, "test-id-123.pdf")
	})

	t.Run("uses extension from file name", func(t *testing.T) {
		storage := newTestStorage(t)
		fileID := uuid.UUID("abc")

		path, err := storage.FilePath(fileID, "image.jpg")
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(path, "abc.jpg"))
	})
}

func TestFilePath_PathTraversal(t *testing.T) {
	t.Run("rejects traversal via malicious file name extension", func(t *testing.T) {
		storage := newTestStorage(t)
		fileID := uuid.UUID("../../etc/passwd")

		_, err := storage.FilePath(fileID, "x.jpg")
		assert.Error(t, err)
	})
}

func TestFilePath_MaliciousFileName(t *testing.T) {
	t.Run("rejects traversal via malicious fileID", func(t *testing.T) {
		storage := newTestStorage(t)

		// filepath.Ext only takes the last extension, so "../../../etc/passwd" has no ext.
		// Test that a traversal attempt in fileID is caught.
		_, err := storage.FilePath(uuid.UUID("../../../etc/passwd"), "x")
		assert.Error(t, err)
	})
}

func TestSave_MaliciousFileName(t *testing.T) {
	t.Run("strips directory components and saves safely", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("content"), "../../etc/shadow.txt")
		require.NoError(t, err)

		// File was saved with the base name only
		assert.True(t, storage.Exists(fileID, "shadow.txt"))
	})
}

func TestFilePath_NormalCase(t *testing.T) {
	t.Run("normal file path works correctly", func(t *testing.T) {
		storage := newTestStorage(t)
		fileID, err := storage.Save(strings.NewReader("ok"), "document.pdf")
		require.NoError(t, err)

		path, pathErr := storage.FilePath(fileID, "document.pdf")
		require.NoError(t, pathErr)
		assert.True(t, strings.HasSuffix(path, ".pdf"))
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	t.Run("deletes existing file", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("to delete"), "temp.txt")
		require.NoError(t, err)
		require.True(t, storage.Exists(fileID, "temp.txt"))

		deleteErr := storage.Delete(fileID, "temp.txt")
		require.NoError(t, deleteErr)
		assert.False(t, storage.Exists(fileID, "temp.txt"))
	})

	t.Run("no error when file does not exist", func(t *testing.T) {
		storage := newTestStorage(t)

		err := storage.Delete(uuid.UUID("nonexistent"), "gone.txt")
		assert.NoError(t, err)
	})
}

func TestLocalStorage_Exists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		storage := newTestStorage(t)

		fileID, err := storage.Save(strings.NewReader("exists"), "file.txt")
		require.NoError(t, err)

		assert.True(t, storage.Exists(fileID, "file.txt"))
	})

	t.Run("returns false for non-existing file", func(t *testing.T) {
		storage := newTestStorage(t)

		assert.False(t, storage.Exists(uuid.UUID("no-such"), "missing.txt"))
	})
}
