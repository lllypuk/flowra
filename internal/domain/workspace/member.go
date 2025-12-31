package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Role представляет роль участника в рабочем пространстве
type Role string

const (
	// RoleOwner владелец workspace (создатель)
	RoleOwner Role = "owner"
	// RoleAdmin администратор workspace
	RoleAdmin Role = "admin"
	// RoleMember обычный участник
	RoleMember Role = "member"
)

// IsValid проверяет, является ли роль допустимой
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// String возвращает строковое представление роли
func (r Role) String() string {
	return string(r)
}

// Member представляет члена рабочего пространства (value object)
type Member struct {
	userID      uuid.UUID
	workspaceID uuid.UUID
	role        Role
	joinedAt    time.Time
}

// NewMember создает нового члена workspace
func NewMember(userID, workspaceID uuid.UUID, role Role) Member {
	return Member{
		userID:      userID,
		workspaceID: workspaceID,
		role:        role,
		joinedAt:    time.Now(),
	}
}

// ReconstructMember восстанавливает Member из хранилища.
// Используется репозиториями для гидрации объекта без валидации бизнес-правил.
// Все параметры должны быть валидными значениями из хранилища.
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

// UserID возвращает ID пользователя
func (m Member) UserID() uuid.UUID { return m.userID }

// WorkspaceID возвращает ID рабочего пространства
func (m Member) WorkspaceID() uuid.UUID { return m.workspaceID }

// Role возвращает роль участника
func (m Member) Role() Role { return m.role }

// JoinedAt возвращает время присоединения
func (m Member) JoinedAt() time.Time { return m.joinedAt }

// IsOwner проверяет, является ли участник владельцем
func (m Member) IsOwner() bool { return m.role == RoleOwner }

// IsAdmin проверяет, является ли участник администратором (owner или admin)
func (m Member) IsAdmin() bool { return m.role == RoleOwner || m.role == RoleAdmin }

// CanManageMembers проверяет, может ли участник управлять членами workspace
func (m Member) CanManageMembers() bool { return m.IsAdmin() }

// CanInvite проверяет, может ли участник приглашать новых членов
func (m Member) CanInvite() bool { return m.IsAdmin() }
