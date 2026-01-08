package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Role represents role participant in workspace prostranstve
type Role string

const (
	// RoleOwner vladelets workspace (sozdatel)
	RoleOwner Role = "owner"
	// RoleAdmin administrator workspace
	RoleAdmin Role = "admin"
	// RoleMember regular uchastnik
	RoleMember Role = "member"
)

// IsValid checks, is li role acceptable
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// String returns strokovoe view roli
func (r Role) String() string {
	return string(r)
}

// Member represents chlena workspace prostranstva (value object)
type Member struct {
	userID      uuid.UUID
	workspaceID uuid.UUID
	role        Role
	joinedAt    time.Time
}

// NewMember creates novogo chlena workspace
func NewMember(userID, workspaceID uuid.UUID, role Role) Member {
	return Member{
		userID:      userID,
		workspaceID: workspaceID,
		role:        role,
		joinedAt:    time.Now(),
	}
}

// ReconstructMember reconstructs Member from save.
// Used by repositories for hydration obekta without validation business rules.
// all parameters dolzhny byt valid values from save.
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

// WorkspaceID returns ID workspace prostranstva
func (m Member) WorkspaceID() uuid.UUID { return m.workspaceID }

// Role returns role participant
func (m Member) Role() Role { return m.role }

// JoinedAt returns time prisoedineniya
func (m Member) JoinedAt() time.Time { return m.joinedAt }

// IsOwner checks, is li uchastnik vladeltsem
func (m Member) IsOwner() bool { return m.role == RoleOwner }

// IsAdmin checks, is li uchastnik administrator (owner or admin)
func (m Member) IsAdmin() bool { return m.role == RoleOwner || m.role == RoleAdmin }

// CanManageMembers checks, mozhet li uchastnik upravlyat chlenami workspace
func (m Member) CanManageMembers() bool { return m.IsAdmin() }

// CanInvite checks, mozhet li uchastnik priglashat New chlenov
func (m Member) CanInvite() bool { return m.IsAdmin() }

// WithRole returns kopiyu Member s novoy role (immutable update)
func (m Member) WithRole(role Role) Member {
	return Member{
		userID:      m.userID,
		workspaceID: m.workspaceID,
		role:        role,
		joinedAt:    m.joinedAt,
	}
}
