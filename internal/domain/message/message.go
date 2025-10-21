package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Message представляет сообщение в чате
type Message struct {
	id              uuid.UUID
	chatID          uuid.UUID
	authorID        uuid.UUID
	content         string
	parentMessageID uuid.UUID // для тредов
	createdAt       time.Time
	editedAt        *time.Time
	isDeleted       bool
	deletedAt       *time.Time
	attachments     []Attachment
	reactions       []Reaction
}

// NewMessage создает новое сообщение
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

// EditContent редактирует содержимое сообщения
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

// Delete мягко удаляет сообщение
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

// HasReaction проверяет наличие реакции от пользователя
func (m *Message) HasReaction(userID uuid.UUID, emojiCode string) bool {
	for _, r := range m.reactions {
		if r.UserID() == userID && r.EmojiCode() == emojiCode {
			return true
		}
	}
	return false
}

// CanBeEditedBy проверяет, может ли пользователь редактировать сообщение
func (m *Message) CanBeEditedBy(userID uuid.UUID) bool {
	return m.authorID == userID
}

// IsEdited проверяет, было ли сообщение отредактировано
func (m *Message) IsEdited() bool {
	return m.editedAt != nil
}

// IsReply проверяет, является ли сообщение ответом (в треде)
func (m *Message) IsReply() bool {
	return !m.parentMessageID.IsZero()
}

// GetReactionCount возвращает количество реакций определенного типа
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

// ID возвращает ID сообщения
func (m *Message) ID() uuid.UUID {
	return m.id
}

// ChatID возвращает ID чата
func (m *Message) ChatID() uuid.UUID {
	return m.chatID
}

// AuthorID возвращает ID автора
func (m *Message) AuthorID() uuid.UUID {
	return m.authorID
}

// Content возвращает содержимое сообщения
func (m *Message) Content() string {
	return m.content
}

// ParentMessageID возвращает ID родительского сообщения (для тредов)
func (m *Message) ParentMessageID() uuid.UUID {
	return m.parentMessageID
}

// CreatedAt возвращает время создания
func (m *Message) CreatedAt() time.Time {
	return m.createdAt
}

// EditedAt возвращает время редактирования
func (m *Message) EditedAt() *time.Time {
	return m.editedAt
}

// IsDeleted возвращает флаг удаления
func (m *Message) IsDeleted() bool {
	return m.isDeleted
}

// DeletedAt возвращает время удаления
func (m *Message) DeletedAt() *time.Time {
	return m.deletedAt
}

// Attachments возвращает копию списка вложений
func (m *Message) Attachments() []Attachment {
	attachments := make([]Attachment, len(m.attachments))
	copy(attachments, m.attachments)
	return attachments
}

// Reactions возвращает копию списка реакций
func (m *Message) Reactions() []Reaction {
	reactions := make([]Reaction, len(m.reactions))
	copy(reactions, m.reactions)
	return reactions
}
