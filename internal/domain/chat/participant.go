package chat

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Role represents роль participant in чате
type Role string

const (
	// RoleMember regular участник
	RoleMember Role = "member"
	// RoleAdmin administrator chat
	RoleAdmin Role = "admin"
)

// Participant represents participant chat (value object)
type Participant struct {
	userID   uuid.UUID
	role     Role
	joinedAt time.Time
}

// NewParticipant creates нового participant
func NewParticipant(userID uuid.UUID, role Role) Participant {
	return Participant{
		userID:   userID,
		role:     role,
		joinedAt: time.Now(),
	}
}

// UserID returns ID user
func (p Participant) UserID() uuid.UUID { return p.userID }

// Role returns роль participant
func (p Participant) Role() Role { return p.role }

// JoinedAt returns time присоединения
func (p Participant) JoinedAt() time.Time { return p.joinedAt }

// IsAdmin checks, is ли участник administratorом
func (p Participant) IsAdmin() bool { return p.role == RoleAdmin }
