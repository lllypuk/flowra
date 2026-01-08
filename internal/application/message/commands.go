package message

import (
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// SendMessageCommand - sendа messages
type SendMessageCommand struct {
	ChatID          uuid.UUID
	Content         string
	AuthorID        uuid.UUID
	ParentMessageID uuid.UUID // for replies, zero UUID if not reply
}

// CommandName returns command name
func (c SendMessageCommand) CommandName() string { return "SendMessage" }

// EditMessageCommand - редактирование messages
type EditMessageCommand struct {
	MessageID uuid.UUID
	Content   string
	EditorID  uuid.UUID // должен совпадать с AuthorID
}

// CommandName returns command name
func (c EditMessageCommand) CommandName() string { return "EditMessage" }

// DeleteMessageCommand - deletion messages
type DeleteMessageCommand struct {
	MessageID uuid.UUID
	DeletedBy uuid.UUID // должен совпадать с AuthorID
}

// CommandName returns command name
func (c DeleteMessageCommand) CommandName() string { return "DeleteMessage" }

// AddReactionCommand - adding реакции
type AddReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName returns command name
func (c AddReactionCommand) CommandName() string { return "AddReaction" }

// RemoveReactionCommand - deletion реакции
type RemoveReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName returns command name
func (c RemoveReactionCommand) CommandName() string { return "RemoveReaction" }

// AddAttachmentCommand - adding вложения
type AddAttachmentCommand struct {
	MessageID uuid.UUID
	FileID    uuid.UUID
	FileName  string
	FileSize  int64
	MimeType  string
	UserID    uuid.UUID // должен совпадать с AuthorID
}

// CommandName returns command name
func (c AddAttachmentCommand) CommandName() string { return "AddAttachment" }
