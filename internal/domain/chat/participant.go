package chat

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Role представляет роль участника в чате
type Role string

const (
	// RoleMember обычный участник
	RoleMember Role = "member"
	// RoleAdmin администратор чата
	RoleAdmin Role = "admin"
)

// Participant представляет участника чата (value object)
type Participant struct {
	userID   uuid.UUID
	role     Role
	joinedAt time.Time
}

// NewParticipant создает нового участника
func NewParticipant(userID uuid.UUID, role Role) Participant {
	return Participant{
		userID:   userID,
		role:     role,
		joinedAt: time.Now(),
	}
}

// UserID возвращает ID пользователя
func (p Participant) UserID() uuid.UUID { return p.userID }

// Role возвращает роль участника
func (p Participant) Role() Role { return p.role }

// JoinedAt возвращает время присоединения
func (p Participant) JoinedAt() time.Time { return p.joinedAt }

// IsAdmin проверяет, является ли участник администратором
func (p Participant) IsAdmin() bool { return p.role == RoleAdmin }
