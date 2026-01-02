package message

import (
	"net/http"
)

// appError is a helper type that implements httpserver.HTTPError interface.
type appError struct {
	msg        string
	httpStatus int
	httpCode   string
	httpMsg    string
}

func (e *appError) Error() string       { return e.msg }
func (e *appError) HTTPStatus() int     { return e.httpStatus }
func (e *appError) HTTPCode() string    { return e.httpCode }
func (e *appError) HTTPMessage() string { return e.httpMsg }

var (
	// ErrEmptyContent indicates that message content cannot be empty
	ErrEmptyContent = &appError{
		msg:        "message content cannot be empty",
		httpStatus: http.StatusBadRequest,
		httpCode:   "EMPTY_CONTENT",
		httpMsg:    "message content cannot be empty",
	}
	ErrContentTooLong = &appError{
		msg:        "message content too long",
		httpStatus: http.StatusBadRequest,
		httpCode:   "CONTENT_TOO_LONG",
		httpMsg:    "message content is too long",
	}
	ErrInvalidEmoji = &appError{
		msg:        "invalid emoji",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_EMOJI",
		httpMsg:    "invalid emoji",
	}
	ErrInvalidFileSize = &appError{
		msg:        "file size exceeds limit",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_FILE_SIZE",
		httpMsg:    "file size exceeds limit",
	}
	ErrInvalidFileName = &appError{
		msg:        "file name is required",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_FILE_NAME",
		httpMsg:    "file name is required",
	}
	ErrInvalidMimeType = &appError{
		msg:        "mime type is required",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_MIME_TYPE",
		httpMsg:    "mime type is required",
	}

	// ErrMessageNotFound indicates that message was not found
	ErrMessageNotFound = &appError{
		msg:        "message not found",
		httpStatus: http.StatusNotFound,
		httpCode:   "MESSAGE_NOT_FOUND",
		httpMsg:    "message not found",
	}
	ErrChatNotFound = &appError{
		msg:        "chat not found",
		httpStatus: http.StatusNotFound,
		httpCode:   "CHAT_NOT_FOUND",
		httpMsg:    "chat not found",
	}
	ErrParentNotFound = &appError{
		msg:        "parent message not found",
		httpStatus: http.StatusBadRequest,
		httpCode:   "PARENT_NOT_FOUND",
		httpMsg:    "parent message not found",
	}
	ErrNotAuthor = &appError{
		msg:        "user is not the message author",
		httpStatus: http.StatusForbidden,
		httpCode:   "NOT_AUTHOR",
		httpMsg:    "only message author can edit",
	}
	ErrMessageDeleted = &appError{
		msg:        "message is deleted",
		httpStatus: http.StatusGone,
		httpCode:   "MESSAGE_DELETED",
		httpMsg:    "message is deleted",
	}
	ErrReactionAlreadyExists = &appError{
		msg:        "reaction already exists",
		httpStatus: http.StatusConflict,
		httpCode:   "REACTION_ALREADY_EXISTS",
		httpMsg:    "reaction already exists",
	}
	ErrReactionNotFound = &appError{
		msg:        "reaction not found",
		httpStatus: http.StatusNotFound,
		httpCode:   "REACTION_NOT_FOUND",
		httpMsg:    "reaction not found",
	}
	ErrParentInDifferentChat = &appError{
		msg:        "parent message is from different chat",
		httpStatus: http.StatusBadRequest,
		httpCode:   "PARENT_DIFFERENT_CHAT",
		httpMsg:    "parent message is from different chat",
	}

	// ErrNotChatParticipant indicates that user is not a chat participant
	ErrNotChatParticipant = &appError{
		msg:        "user is not a chat participant",
		httpStatus: http.StatusForbidden,
		httpCode:   "NOT_PARTICIPANT",
		httpMsg:    "not a participant of this chat",
	}
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
