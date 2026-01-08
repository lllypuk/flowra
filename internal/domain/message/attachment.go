package message

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment represents faylovoe attachment to soobscheniyu
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

// FileID returns ID fayla
func (a Attachment) FileID() uuid.UUID {
	return a.fileID
}

// FileName returns imya fayla
func (a Attachment) FileName() string {
	return a.fileName
}

// FileSize returns size fayla in baytah
func (a Attachment) FileSize() int64 {
	return a.fileSize
}

// MimeType returns MIME type fayla
func (a Attachment) MimeType() string {
	return a.mimeType
}

// ReconstructAttachment reconstructs attachment from save.
// Used by repositories for hydration obekta without validation business rules.
func ReconstructAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) Attachment {
	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}
}
