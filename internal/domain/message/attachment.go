package message

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment represents a file attachment to a message.
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment creates new attachment
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

// FileID returns the file ID.
func (a Attachment) FileID() uuid.UUID {
	return a.fileID
}

// FileName returns the file name.
func (a Attachment) FileName() string {
	return a.fileName
}

// FileSize returns the file size in bytes.
func (a Attachment) FileSize() int64 {
	return a.fileSize
}

// MimeType returns the MIME type of the file.
func (a Attachment) MimeType() string {
	return a.mimeType
}

// ReconstructAttachment reconstructs attachment from storage.
// Used by repositories for object hydration without business rules validation.
func ReconstructAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) Attachment {
	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}
}
