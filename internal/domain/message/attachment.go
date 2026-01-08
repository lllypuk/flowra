package message

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment represents файловое вложение to сообщению
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment creates новое вложение
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

// FileID returns ID файла
func (a Attachment) FileID() uuid.UUID {
	return a.fileID
}

// FileName returns имя файла
func (a Attachment) FileName() string {
	return a.fileName
}

// FileSize returns size файла in байтах
func (a Attachment) FileSize() int64 {
	return a.fileSize
}

// MimeType returns MIME type файла
func (a Attachment) MimeType() string {
	return a.mimeType
}

// ReconstructAttachment восстанавливает вложение from storage.
// Used by repositories for hydration объекта without validation business rules.
func ReconstructAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) Attachment {
	return Attachment{
		fileID:   fileID,
		fileName: fileName,
		fileSize: fileSize,
		mimeType: mimeType,
	}
}
