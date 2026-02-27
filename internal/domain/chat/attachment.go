package chat

import (
	"strings"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment represents a file attached to a typed chat (task/bug/epic).
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment creates a validated chat attachment.
func NewAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) (Attachment, error) {
	if fileID.IsZero() {
		return Attachment{}, errs.ErrInvalidInput
	}
	if strings.TrimSpace(fileName) == "" {
		return Attachment{}, errs.ErrInvalidInput
	}
	if fileSize <= 0 {
		return Attachment{}, errs.ErrInvalidInput
	}
	if strings.TrimSpace(mimeType) == "" {
		return Attachment{}, errs.ErrInvalidInput
	}

	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}, nil
}

// ReconstructAttachment creates an attachment from persisted event data.
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
