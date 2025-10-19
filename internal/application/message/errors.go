package message

import (
	"errors"
)

var (
	// ErrEmptyContent indicates that message content cannot be empty
	ErrEmptyContent    = errors.New("message content cannot be empty")
	ErrContentTooLong  = errors.New("message content too long")
	ErrInvalidEmoji    = errors.New("invalid emoji")
	ErrInvalidFileSize = errors.New("file size exceeds limit")
	ErrInvalidFileName = errors.New("file name is required")
	ErrInvalidMimeType = errors.New("mime type is required")

	// ErrMessageNotFound indicates that message was not found
	ErrMessageNotFound       = errors.New("message not found")
	ErrChatNotFound          = errors.New("chat not found")
	ErrParentNotFound        = errors.New("parent message not found")
	ErrNotAuthor             = errors.New("user is not the message author")
	ErrMessageDeleted        = errors.New("message is deleted")
	ErrReactionAlreadyExists = errors.New("reaction already exists")
	ErrReactionNotFound      = errors.New("reaction not found")
	ErrParentInDifferentChat = errors.New("parent message is from different chat")

	// ErrNotChatParticipant indicates that user is not a chat participant
	ErrNotChatParticipant = errors.New("user is not a chat participant")
)

const (
	// MaxContentLength максимальная длина сообщения (10k символов)
	MaxContentLength = 10000
	// MaxFileSize максимальный размер файла (10 MB)
	MaxFileSize = 10 << 20
	// DefaultLimit количество сообщений по умолчанию
	DefaultLimit = 50
	// MaxLimit максимальное количество сообщений за раз
	MaxLimit = 100
)
