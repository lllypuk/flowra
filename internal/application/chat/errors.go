package chat

import (
	"errors"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// Validation errors
var (
	// ErrInvalidChatType indicates an invalid chat type was provided
	ErrInvalidChatType = errors.New("invalid chat type")
	// ErrInvalidStatus indicates an invalid status for the chat type
	ErrInvalidStatus = errors.New("invalid status for chat type")
	// ErrInvalidPriority indicates an invalid priority value
	ErrInvalidPriority = errors.New("invalid priority")
	// ErrInvalidSeverity indicates an invalid severity value
	ErrInvalidSeverity = errors.New("invalid severity")
	// ErrInvalidRole indicates an invalid participant role
	ErrInvalidRole = errors.New("invalid participant role")
	// ErrTitleRequired indicates that a title is required for typed chats
	ErrTitleRequired = errors.New("title is required for typed chats")
)

// Business logic errors
var (
	// ErrChatNotFound indicates the requested chat was not found
	ErrChatNotFound = errors.New("chat not found")
	// ErrUserNotParticipant indicates the user is not a participant
	ErrUserNotParticipant = errors.New("user is not a participant")
	// ErrUserAlreadyParticipant indicates the user is already a participant
	ErrUserAlreadyParticipant = errors.New("user is already a participant")
	// ErrCannotRemoveLastAdmin indicates cannot remove the last admin
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
	// ErrCannotRemoveCreator indicates cannot remove the chat creator
	ErrCannotRemoveCreator = errors.New("cannot remove the chat creator")
	// ErrNotAdmin indicates the user is not an admin
	ErrNotAdmin = errors.New("user is not an admin")
	// ErrCannotConvertType indicates the chat type cannot be converted
	ErrCannotConvertType = errors.New("cannot convert chat type")
	// ErrSeverityOnlyForBugs indicates severity can only be set on bugs
	ErrSeverityOnlyForBugs = errors.New("severity can only be set on bugs")
	// ErrCannotModifyDiscussion indicates cannot modify properties of discussion chat
	ErrCannotModifyDiscussion = errors.New("cannot modify properties of discussion chat")
)

// Authorization errors
var (
	// ErrNotAuthorized indicates the user is not authorized
	ErrNotAuthorized = appcore.ErrUnauthorized
	// ErrForbidden indicates the action is forbidden
	ErrForbidden = appcore.ErrForbidden
)
