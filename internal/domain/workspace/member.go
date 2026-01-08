package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Role represents роль participant in workspaceем пространстве
type Role string

const (
	// RoleOwner владелец workspace (создатель)
	RoleOwner Role = "owner"
	// RoleAdmin administrator workspace
	RoleAdmin Role = "admin"
	// RoleMember regular участник
	RoleMember Role = "member"
)

// IsValid checks, is ли роль acceptable
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// String returns строковое view роли
func (r Role) String() string {
	return string(r)
}

// Member represents члена workspace пространства (value object)
type Member struct {
	userID      uuid.UUID
	workspaceID uuid.UUID
	role        Role
	joinedAt    time.Time
}

// NewMember creates нового члена workspace
func NewMember(userID, workspaceID uuid.UUID, role Role) Member {
	return Member{
		userID:      userID,
		workspaceID: workspaceID,
		role:        role,
		joinedAt:    time.Now(),
	}
}

// ReconstructMember восстанавливает Member from storage.
// Used by repositories for hydration объекта without validation business rules.
// all parameters должны быть valid values from storage.
func ReconstructMember(
	userID uuid.UUID,
	workspaceID uuid.UUID,
	role Role,
	joinedAt time.Time,
) Member {
	return Member{
		userID:      userID,
		workspaceID: workspaceID,
		role:        role,
		joinedAt:    joinedAt,
	}
}

// UserID returns ID user
func (m Member) UserID() uuid.UUID { return m.userID }

// WorkspaceID returns ID workspace пространства
func (m Member) WorkspaceID() uuid.UUID { return m.workspaceID }

// Role returns роль participant
func (m Member) Role() Role { return m.role }

// JoinedAt returns time присоединения
func (m Member) JoinedAt() time.Time { return m.joinedAt }

// IsOwner checks, is ли участник владельцем
func (m Member) IsOwner() bool { return m.role == RoleOwner }

// IsAdmin checks, is ли участник administratorом (owner or admin)
func (m Member) IsAdmin() bool { return m.role == RoleOwner || m.role == RoleAdmin }

// CanManageMembers checks, может ли участник управлять членами workspace
func (m Member) CanManageMembers() bool { return m.IsAdmin() }

// CanInvite checks, может ли участник приглашать New членов
func (m Member) CanInvite() bool { return m.IsAdmin() }

// WithRole returns копию Member с новой ролью (immutable update)
func (m Member) WithRole(role Role) Member {
	return Member{
		userID:      m.userID,
		workspaceID: m.workspaceID,
		role:        role,
		joinedAt:    m.joinedAt,
	}
}
