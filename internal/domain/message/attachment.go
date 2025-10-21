package message

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Attachment представляет файловое вложение к сообщению
type Attachment struct {
	fileID   uuid.UUID
	fileName string
	fileSize int64
	mimeType string
}

// NewAttachment создает новое вложение
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

// FileID возвращает ID файла
func (a Attachment) FileID() uuid.UUID {
	return a.fileID
}

// FileName возвращает имя файла
func (a Attachment) FileName() string {
	return a.fileName
}

// FileSize возвращает размер файла в байтах
func (a Attachment) FileSize() int64 {
	return a.fileSize
}

// MimeType возвращает MIME тип файла
func (a Attachment) MimeType() string {
	return a.mimeType
}
