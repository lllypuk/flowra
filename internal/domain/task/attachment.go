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
