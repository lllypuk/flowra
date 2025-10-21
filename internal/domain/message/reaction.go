package message

import (
	"time"

	"github.com/flowra/flowra/internal/domain/errs"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// Reaction представляет эмоджи реакцию на сообщение
type Reaction struct {
	userID    uuid.UUID
	emojiCode string
	addedAt   time.Time
}

// NewReaction создает новую реакцию
func NewReaction(userID uuid.UUID, emojiCode string) (Reaction, error) {
	if userID.IsZero() {
		return Reaction{}, errs.ErrInvalidInput
	}
	if emojiCode == "" {
		return Reaction{}, errs.ErrInvalidInput
	}

	return Reaction{
		userID:    userID,
		emojiCode: emojiCode,
		addedAt:   time.Now(),
	}, nil
}

// UserID возвращает ID пользователя
func (r Reaction) UserID() uuid.UUID {
	return r.userID
}

// EmojiCode возвращает код эмоджи
func (r Reaction) EmojiCode() string {
	return r.emojiCode
}

// AddedAt возвращает время добавления реакции
func (r Reaction) AddedAt() time.Time {
	return r.addedAt
}
