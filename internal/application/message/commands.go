package message

import (
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// SendMessageCommand - отправка сообщения
type SendMessageCommand struct {
	ChatID          uuid.UUID
	Content         string
	AuthorID        uuid.UUID
	ParentMessageID uuid.UUID // для replies, zero UUID если не reply
}

// CommandName возвращает имя команды
func (c SendMessageCommand) CommandName() string { return "SendMessage" }

// EditMessageCommand - редактирование сообщения
type EditMessageCommand struct {
	MessageID uuid.UUID
	Content   string
	EditorID  uuid.UUID // должен совпадать с AuthorID
}

// CommandName возвращает имя команды
func (c EditMessageCommand) CommandName() string { return "EditMessage" }

// DeleteMessageCommand - удаление сообщения
type DeleteMessageCommand struct {
	MessageID uuid.UUID
	DeletedBy uuid.UUID // должен совпадать с AuthorID
}

// CommandName возвращает имя команды
func (c DeleteMessageCommand) CommandName() string { return "DeleteMessage" }

// AddReactionCommand - добавление реакции
type AddReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName возвращает имя команды
func (c AddReactionCommand) CommandName() string { return "AddReaction" }

// RemoveReactionCommand - удаление реакции
type RemoveReactionCommand struct {
	MessageID uuid.UUID
	Emoji     string
	UserID    uuid.UUID
}

// CommandName возвращает имя команды
func (c RemoveReactionCommand) CommandName() string { return "RemoveReaction" }

// AddAttachmentCommand - добавление вложения
type AddAttachmentCommand struct {
	MessageID uuid.UUID
	FileID    uuid.UUID
	FileName  string
	FileSize  int64
	MimeType  string
	UserID    uuid.UUID // должен совпадать с AuthorID
}

// CommandName возвращает имя команды
func (c AddAttachmentCommand) CommandName() string { return "AddAttachment" }
