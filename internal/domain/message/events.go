package message

import (
	"time"

	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/uuid"
)

const (
	// EventTypeMessageCreated событие создания сообщения
	EventTypeMessageCreated = "message.created"
	// EventTypeMessageEdited событие редактирования сообщения
	EventTypeMessageEdited = "message.edited"
	// EventTypeMessageDeleted событие удаления сообщения
	EventTypeMessageDeleted = "message.deleted"
	// EventTypeMessageReactionAdded событие добавления реакции
	EventTypeMessageReactionAdded = "message.reaction.added"
	// EventTypeMessageReactionRemoved событие удаления реакции
	EventTypeMessageReactionRemoved = "message.reaction.removed"
	// EventTypeMessageAttachmentAdded событие добавления вложения
	EventTypeMessageAttachmentAdded = "message.attachment.added"
)

// Created событие создания сообщения
type Created struct {
	event.BaseEvent

	ChatID          uuid.UUID
	AuthorID        uuid.UUID
	Content         string
	ParentMessageID uuid.UUID
	CreatedAt       time.Time
}

// NewCreated создает событие Created
func NewCreated(
	messageID uuid.UUID,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
	parentMessageID uuid.UUID,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:       event.NewBaseEvent(EventTypeMessageCreated, messageID.String(), "Message", 1, metadata),
		ChatID:          chatID,
		AuthorID:        authorID,
		Content:         content,
		ParentMessageID: parentMessageID,
		CreatedAt:       time.Now(),
	}
}

// Edited событие редактирования сообщения
type Edited struct {
	event.BaseEvent

	NewContent string
	EditedAt   time.Time
}

// NewEdited создает событие Edited
func NewEdited(messageID uuid.UUID, newContent string, version int, metadata event.Metadata) *Edited {
	return &Edited{
		BaseEvent:  event.NewBaseEvent(EventTypeMessageEdited, messageID.String(), "Message", version, metadata),
		NewContent: newContent,
		EditedAt:   time.Now(),
	}
}

// Deleted событие удаления сообщения
type Deleted struct {
	event.BaseEvent

	DeletedBy uuid.UUID
	DeletedAt time.Time
}

// NewDeleted создает событие Deleted
func NewDeleted(messageID uuid.UUID, deletedBy uuid.UUID, version int, metadata event.Metadata) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeMessageDeleted, messageID.String(), "Message", version, metadata),
		DeletedBy: deletedBy,
		DeletedAt: time.Now(),
	}
}

// ReactionAdded событие добавления реакции
type ReactionAdded struct {
	event.BaseEvent

	UserID    uuid.UUID
	EmojiCode string
	AddedAt   time.Time
}

// NewReactionAdded создает событие ReactionAdded
func NewReactionAdded(
	messageID uuid.UUID,
	userID uuid.UUID,
	emojiCode string,
	version int,
	metadata event.Metadata,
) *ReactionAdded {
	return &ReactionAdded{
		BaseEvent: event.NewBaseEvent(EventTypeMessageReactionAdded, messageID.String(), "Message", version, metadata),
		UserID:    userID,
		EmojiCode: emojiCode,
		AddedAt:   time.Now(),
	}
}

// ReactionRemoved событие удаления реакции
type ReactionRemoved struct {
	event.BaseEvent

	UserID    uuid.UUID
	EmojiCode string
	RemovedAt time.Time
}

// NewReactionRemoved создает событие ReactionRemoved
func NewReactionRemoved(
	messageID uuid.UUID,
	userID uuid.UUID,
	emojiCode string,
	version int,
	metadata event.Metadata,
) *ReactionRemoved {
	return &ReactionRemoved{
		BaseEvent: event.NewBaseEvent(
			EventTypeMessageReactionRemoved,
			messageID.String(),
			"Message",
			version,
			metadata,
		),
		UserID:    userID,
		EmojiCode: emojiCode,
		RemovedAt: time.Now(),
	}
}

// AttachmentAdded событие добавления вложения
type AttachmentAdded struct {
	event.BaseEvent

	FileID   uuid.UUID
	FileName string
	FileSize int64
	MimeType string
	AddedAt  time.Time
}

// NewAttachmentAdded создает событие AttachmentAdded
func NewAttachmentAdded(
	messageID uuid.UUID,
	fileID uuid.UUID,
	fileName string,
	fileSize int64,
	mimeType string,
	version int,
	metadata event.Metadata,
) *AttachmentAdded {
	return &AttachmentAdded{
		BaseEvent: event.NewBaseEvent(
			EventTypeMessageAttachmentAdded,
			messageID.String(),
			"Message",
			version,
			metadata,
		),
		FileID:   fileID,
		FileName: fileName,
		FileSize: fileSize,
		MimeType: mimeType,
		AddedAt:  time.Now(),
	}
}
