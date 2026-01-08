package message

import (
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// SendMessageCommand - send messages
type SendMessageCommand struct {
	ChatID          uuid.UUID
	Content         string
	AuthorID        uuid.UUID
	ParentMessageID uuid.UUID // for replies, zero UUID if not reply
}

// CommandName returns command name
func (c SendMessageCommand) CommandName() string { return "SendMessage" }

// EditMessageCommand - edit messages
type EditMessageCommand struct {
	MessageID uuid.UUID
	Content   string
	EditorID  uuid.UUID // must match AuthorID
}

// CommandName returns command name
func (c EditMessageCommand) CommandName() string { return "EditMessage" }

// DeleteMessageCommand - delete messages
type DeleteMessageCommand struct {
	MessageID uuid.UUID
	DeletedBy uuid.UUID // must match AuthorID
}

// CommandName returns command name
func (c DeleteMessageCommand) CommandName() string { return "DeleteMessage" }

// AddReactionCommand - add reactions
type AddReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName returns command name
func (c AddReactionCommand) CommandName() string { return "AddReaction" }

// RemoveReactionCommand - remove reactions
type RemoveReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName returns command name
func (c RemoveReactionCommand) CommandName() string { return "RemoveReaction" }

// AddAttachmentCommand - add attachments
type AddAttachmentCommand struct {
	MessageID uuid.UUID
	FileID    uuid.UUID
	FileName  string
	FileSize  int64
	MimeType  string
	UserID    uuid.UUID // must match AuthorID
}

// CommandName returns command name
func (c AddAttachmentCommand) CommandName() string { return "AddAttachment" }
