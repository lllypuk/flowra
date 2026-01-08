package message

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Reaction represents эмоджи реакцию on message
type Reaction struct {
	userID    uuid.UUID
	emojiCode string
	addedAt   time.Time
}

// NewReaction creates New реакцию
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

// UserID returns ID user
func (r Reaction) UserID() uuid.UUID {
	return r.userID
}

// EmojiCode returns код эмоджи
func (r Reaction) EmojiCode() string {
	return r.emojiCode
}

// AddedAt returns time adding реакции
func (r Reaction) AddedAt() time.Time {
	return r.addedAt
}

// ReconstructReaction восстанавливает реакцию from storage.
// Used by repositories for hydration объекта without validation business rules.
func ReconstructReaction(userID uuid.UUID, emojiCode string, addedAt time.Time) Reaction {
	return Reaction{
		userID:    userID,
		emojiCode: emojiCode,
		addedAt:   addedAt,
	}
}
