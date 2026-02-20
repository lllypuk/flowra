package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestNewAttachment(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		fileID := uuid.NewUUID()
		att, err := task.NewAttachment(fileID, "report.pdf", 1024, "application/pdf")

		require.NoError(t, err)
		assert.Equal(t, fileID, att.FileID())
		assert.Equal(t, "report.pdf", att.FileName())
		assert.Equal(t, int64(1024), att.FileSize())
		assert.Equal(t, "application/pdf", att.MimeType())
	})

	t.Run("zero file ID", func(t *testing.T) {
		_, err := task.NewAttachment(uuid.UUID(""), "file.txt", 100, "text/plain")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty file name", func(t *testing.T) {
		_, err := task.NewAttachment(uuid.NewUUID(), "", 100, "text/plain")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("zero file size", func(t *testing.T) {
		_, err := task.NewAttachment(uuid.NewUUID(), "file.txt", 0, "text/plain")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("negative file size", func(t *testing.T) {
		_, err := task.NewAttachment(uuid.NewUUID(), "file.txt", -1, "text/plain")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty mime type", func(t *testing.T) {
		_, err := task.NewAttachment(uuid.NewUUID(), "file.txt", 100, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

func TestReconstructAttachment(t *testing.T) {
	fileID := uuid.NewUUID()
	att := task.ReconstructAttachment(fileID, "doc.pdf", 2048, "application/pdf")

	assert.Equal(t, fileID, att.FileID())
	assert.Equal(t, "doc.pdf", att.FileName())
	assert.Equal(t, int64(2048), att.FileSize())
	assert.Equal(t, "application/pdf", att.MimeType())
}

func TestAttachment_IsImage(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"JPEG", "image/jpeg", true},
		{"PNG", "image/png", true},
		{"GIF", "image/gif", true},
		{"WebP", "image/webp", true},
		{"SVG", "image/svg+xml", true},
		{"PDF", "application/pdf", false},
		{"Text", "text/plain", false},
		{"ZIP", "application/zip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			att := task.ReconstructAttachment(uuid.NewUUID(), "file", 100, tt.mimeType)
			assert.Equal(t, tt.expected, att.IsImage())
		})
	}
}

// createTestAggregate is a helper that creates a task aggregate in a valid state.
func createTestAggregate(t *testing.T) *task.Aggregate {
	t.Helper()
	agg := task.NewTaskAggregate(uuid.NewUUID())
	err := agg.Create(uuid.NewUUID(), "Test Task", task.TypeTask, task.PriorityMedium, nil, nil, uuid.NewUUID())
	require.NoError(t, err)
	agg.MarkEventsAsCommitted()
	return agg
}

func TestAggregate_AddAttachment(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		agg := createTestAggregate(t)
		fileID := uuid.NewUUID()
		userID := uuid.NewUUID()

		err := agg.AddAttachment(fileID, "report.pdf", 4096, "application/pdf", userID)
		require.NoError(t, err)

		require.Len(t, agg.Attachments(), 1)
		assert.Equal(t, fileID, agg.Attachments()[0].FileID())
		assert.Equal(t, "report.pdf", agg.Attachments()[0].FileName())
		assert.Equal(t, int64(4096), agg.Attachments()[0].FileSize())
		assert.Equal(t, "application/pdf", agg.Attachments()[0].MimeType())

		events := agg.UncommittedEvents()
		require.Len(t, events, 1)
		evt, ok := events[0].(*task.AttachmentAdded)
		require.True(t, ok)
		assert.Equal(t, fileID, evt.FileID)
		assert.Equal(t, "report.pdf", evt.FileName)
		assert.Equal(t, int64(4096), evt.FileSize)
		assert.Equal(t, "application/pdf", evt.MimeType)
		assert.Equal(t, userID, evt.AddedBy)
	})

	t.Run("idempotent - same file twice", func(t *testing.T) {
		agg := createTestAggregate(t)
		fileID := uuid.NewUUID()
		userID := uuid.NewUUID()

		err := agg.AddAttachment(fileID, "file.pdf", 100, "application/pdf", userID)
		require.NoError(t, err)
		agg.MarkEventsAsCommitted()

		err = agg.AddAttachment(fileID, "file.pdf", 100, "application/pdf", userID)
		require.NoError(t, err)

		assert.Len(t, agg.Attachments(), 1, "should not duplicate attachment")
		assert.Empty(t, agg.UncommittedEvents(), "should produce no new events")
	})

	t.Run("multiple different attachments", func(t *testing.T) {
		agg := createTestAggregate(t)
		userID := uuid.NewUUID()

		err := agg.AddAttachment(uuid.NewUUID(), "a.pdf", 100, "application/pdf", userID)
		require.NoError(t, err)
		err = agg.AddAttachment(uuid.NewUUID(), "b.png", 200, "image/png", userID)
		require.NoError(t, err)

		assert.Len(t, agg.Attachments(), 2)
		assert.Len(t, agg.UncommittedEvents(), 2)
	})

	t.Run("error on uninitialized aggregate", func(t *testing.T) {
		agg := task.NewTaskAggregate(uuid.NewUUID())
		err := agg.AddAttachment(uuid.NewUUID(), "file.pdf", 100, "application/pdf", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestAggregate_RemoveAttachment(t *testing.T) {
	t.Run("successful remove", func(t *testing.T) {
		agg := createTestAggregate(t)
		fileID := uuid.NewUUID()
		userID := uuid.NewUUID()

		err := agg.AddAttachment(fileID, "file.pdf", 100, "application/pdf", userID)
		require.NoError(t, err)
		agg.MarkEventsAsCommitted()

		err = agg.RemoveAttachment(fileID, userID)
		require.NoError(t, err)

		assert.Empty(t, agg.Attachments())
		events := agg.UncommittedEvents()
		require.Len(t, events, 1)
		evt, ok := events[0].(*task.AttachmentRemoved)
		require.True(t, ok)
		assert.Equal(t, fileID, evt.FileID)
		assert.Equal(t, userID, evt.RemovedBy)
	})

	t.Run("idempotent - remove non-existent", func(t *testing.T) {
		agg := createTestAggregate(t)

		err := agg.RemoveAttachment(uuid.NewUUID(), uuid.NewUUID())
		require.NoError(t, err)

		assert.Empty(t, agg.UncommittedEvents(), "should produce no events")
	})

	t.Run("remove one of multiple", func(t *testing.T) {
		agg := createTestAggregate(t)
		userID := uuid.NewUUID()
		fileA := uuid.NewUUID()
		fileB := uuid.NewUUID()

		_ = agg.AddAttachment(fileA, "a.pdf", 100, "application/pdf", userID)
		_ = agg.AddAttachment(fileB, "b.png", 200, "image/png", userID)
		agg.MarkEventsAsCommitted()

		err := agg.RemoveAttachment(fileA, userID)
		require.NoError(t, err)

		require.Len(t, agg.Attachments(), 1)
		assert.Equal(t, fileB, agg.Attachments()[0].FileID())
	})

	t.Run("error on uninitialized aggregate", func(t *testing.T) {
		agg := task.NewTaskAggregate(uuid.NewUUID())
		err := agg.RemoveAttachment(uuid.NewUUID(), uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestAggregate_AttachmentReplay(t *testing.T) {
	// Build an aggregate with attachments, then replay all events on a fresh aggregate
	agg := task.NewTaskAggregate(uuid.NewUUID())
	userID := uuid.NewUUID()
	fileA := uuid.NewUUID()
	fileB := uuid.NewUUID()

	_ = agg.Create(uuid.NewUUID(), "Test Task", task.TypeTask, task.PriorityMedium, nil, nil, userID)
	_ = agg.AddAttachment(fileA, "a.pdf", 100, "application/pdf", userID)
	_ = agg.AddAttachment(fileB, "b.png", 200, "image/png", userID)
	_ = agg.RemoveAttachment(fileA, userID)

	allEvents := agg.UncommittedEvents()
	require.Len(t, allEvents, 4) // create + 2 add + 1 remove

	// Replay on a fresh aggregate
	fresh := task.NewTaskAggregate(agg.ID())
	fresh.ReplayEvents(allEvents)

	require.Len(t, fresh.Attachments(), 1)
	assert.Equal(t, fileB, fresh.Attachments()[0].FileID())
	assert.Equal(t, "b.png", fresh.Attachments()[0].FileName())
}
