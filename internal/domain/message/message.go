package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Message represents message in чате
type Message struct {
	id              uuid.UUID
	chatID          uuid.UUID
	authorID        uuid.UUID
	content         string
	parentMessageID uuid.UUID // for тредов
	createdAt       time.Time
	editedAt        *time.Time
	isDeleted       bool
	deletedAt       *time.Time
	attachments     []Attachment
	reactions       []Reaction
}

// NewMessage creates новое message
func NewMessage(
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
	parentMessageID uuid.UUID,
) (*Message, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if authorID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if content == "" {
		return nil, errs.ErrInvalidInput
	}

	return &Message{
		id:              uuid.NewUUID(),
		chatID:          chatID,
		authorID:        authorID,
		content:         content,
		parentMessageID: parentMessageID,
		createdAt:       time.Now(),
		isDeleted:       false,
		attachments:     make([]Attachment, 0),
		reactions:       make([]Reaction, 0),
	}, nil
}

// Reconstruct восстанавливает message from storage.
// Used by repositories for hydration объекта without validation business rules.
// all parameters должны быть valid values from storage.
func Reconstruct(
	id uuid.UUID,
	chatID uuid.UUID,
	authorID uuid.UUID,
	content string,
	parentMessageID uuid.UUID,
	createdAt time.Time,
	editedAt *time.Time,
	isDeleted bool,
	deletedAt *time.Time,
	attachments []Attachment,
	reactions []Reaction,
) *Message {
	if attachments == nil {
		attachments = make([]Attachment, 0)
	}
	if reactions == nil {
		reactions = make([]Reaction, 0)
	}

	return &Message{
		id:              id,
		chatID:          chatID,
		authorID:        authorID,
		content:         content,
		parentMessageID: parentMessageID,
		createdAt:       createdAt,
		editedAt:        editedAt,
		isDeleted:       isDeleted,
		deletedAt:       deletedAt,
		attachments:     attachments,
		reactions:       reactions,
	}
}

// EditContent редактирует содержимое messages
func (m *Message) EditContent(newContent string, editorID uuid.UUID) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if newContent == "" {
		return errs.ErrInvalidInput
	}
	if !m.CanBeEditedBy(editorID) {
		return errs.ErrForbidden
	}

	m.content = newContent
	now := time.Now()
	m.editedAt = &now
	return nil
}

// Delete мягко удаляет message
func (m *Message) Delete(deleterID uuid.UUID) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if !m.CanBeEditedBy(deleterID) {
		return errs.ErrForbidden
	}

	m.isDeleted = true
	now := time.Now()
	m.deletedAt = &now
	return nil
}

// AddReaction добавляет реакцию
func (m *Message) AddReaction(userID uuid.UUID, emojiCode string) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}
	if m.HasReaction(userID, emojiCode) {
		return errs.ErrAlreadyExists
	}

	reaction, err := NewReaction(userID, emojiCode)
	if err != nil {
		return err
	}

	m.reactions = append(m.reactions, reaction)
	return nil
}

// RemoveReaction удаляет реакцию
func (m *Message) RemoveReaction(userID uuid.UUID, emojiCode string) error {
	if !m.HasReaction(userID, emojiCode) {
		return errs.ErrNotFound
	}

	newReactions := make([]Reaction, 0, len(m.reactions)-1)
	for _, r := range m.reactions {
		if r.UserID() != userID || r.EmojiCode() != emojiCode {
			newReactions = append(newReactions, r)
		}
	}
	m.reactions = newReactions
	return nil
}

// AddAttachment добавляет вложение
func (m *Message) AddAttachment(fileID uuid.UUID, fileName string, fileSize int64, mimeType string) error {
	if m.isDeleted {
		return errs.ErrInvalidState
	}

	attachment, err := NewAttachment(fileID, fileName, fileSize, mimeType)
	if err != nil {
		return err
	}

	m.attachments = append(m.attachments, attachment)
	return nil
}

// HasReaction checks presence реакции от user
func (m *Message) HasReaction(userID uuid.UUID, emojiCode string) bool {
	for _, r := range m.reactions {
		if r.UserID() == userID && r.EmojiCode() == emojiCode {
			return true
		}
	}
	return false
}

// CanBeEditedBy checks, может ли userель редактировать message
func (m *Message) CanBeEditedBy(userID uuid.UUID) bool {
	return m.authorID == userID
}

// IsEdited checks, было ли message отредактировано
func (m *Message) IsEdited() bool {
	return m.editedAt != nil
}

// IsReply checks, is ли message responseом (in треде)
func (m *Message) IsReply() bool {
	return !m.parentMessageID.IsZero()
}

// GetReactionCount returns count реакций specific type
func (m *Message) GetReactionCount(emojiCode string) int {
	count := 0
	for _, r := range m.reactions {
		if r.EmojiCode() == emojiCode {
			count++
		}
	}
	return count
}

// Getters

// ID returns ID messages
func (m *Message) ID() uuid.UUID {
	return m.id
}

// ChatID returns ID chat
func (m *Message) ChatID() uuid.UUID {
	return m.chatID
}

// AuthorID returns ID автора
func (m *Message) AuthorID() uuid.UUID {
	return m.authorID
}

// Content returns содержимое messages
func (m *Message) Content() string {
	return m.content
}

// ParentMessageID returns ID родительского messages (for тредов)
func (m *Message) ParentMessageID() uuid.UUID {
	return m.parentMessageID
}

// CreatedAt returns creation time
func (m *Message) CreatedAt() time.Time {
	return m.createdAt
}

// EditedAt returns time редактирования
func (m *Message) EditedAt() *time.Time {
	return m.editedAt
}

// IsDeleted returns флаг removing
func (m *Message) IsDeleted() bool {
	return m.isDeleted
}

// DeletedAt returns time removing
func (m *Message) DeletedAt() *time.Time {
	return m.deletedAt
}

// Attachments returns копию list вложений
func (m *Message) Attachments() []Attachment {
	attachments := make([]Attachment, len(m.attachments))
	copy(attachments, m.attachments)
	return attachments
}

// Reactions returns копию list реакций
func (m *Message) Reactions() []Reaction {
	reactions := make([]Reaction, len(m.reactions))
	copy(reactions, m.reactions)
	return reactions
}
