package user

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// User представляет пользователя системы
type User struct {
	id            uuid.UUID
	keycloakID    string // ID пользователя в Keycloak
	username      string
	email         string
	displayName   string
	isSystemAdmin bool
	createdAt     time.Time
	updatedAt     time.Time
}

// NewUser создает нового пользователя
func NewUser(keycloakID, username, email, displayName string) (*User, error) {
	if keycloakID == "" {
		return nil, errs.ErrInvalidInput
	}
	if username == "" {
		return nil, errs.ErrInvalidInput
	}
	if email == "" {
		return nil, errs.ErrInvalidInput
	}

	return &User{
		id:          uuid.NewUUID(),
		keycloakID:  keycloakID,
		username:    username,
		email:       email,
		displayName: displayName,
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}, nil
}

// Reconstruct восстанавливает пользователя из хранилища
func Reconstruct(
	id uuid.UUID,
	keycloakID, username, email, displayName string,
	isSystemAdmin bool,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:            id,
		keycloakID:    keycloakID,
		username:      username,
		email:         email,
		displayName:   displayName,
		isSystemAdmin: isSystemAdmin,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

// Getters

// ID возвращает ID пользователя
func (u *User) ID() uuid.UUID {
	return u.id
}

// KeycloakID возвращает ID пользователя в Keycloak
func (u *User) KeycloakID() string {
	return u.keycloakID
}

// Username возвращает имя пользователя
func (u *User) Username() string {
	return u.username
}

// Email возвращает email пользователя
func (u *User) Email() string {
	return u.email
}

// DisplayName возвращает отображаемое имя
func (u *User) DisplayName() string {
	return u.displayName
}

// IsSystemAdmin возвращает флаг системного администратора
func (u *User) IsSystemAdmin() bool {
	return u.isSystemAdmin
}

// CreatedAt возвращает время создания
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt возвращает время последнего обновления
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// UpdateProfile обновляет профиль пользователя
func (u *User) UpdateProfile(displayName *string, email *string) error {
	updated := false

	if displayName != nil && *displayName != "" {
		u.displayName = *displayName
		updated = true
	}

	if email != nil && *email != "" {
		u.email = *email
		updated = true
	}

	if !updated {
		return errs.ErrInvalidInput
	}

	u.updatedAt = time.Now()
	return nil
}

// SetAdmin устанавливает права администратора
func (u *User) SetAdmin(isAdmin bool) {
	u.isSystemAdmin = isAdmin
	u.updatedAt = time.Now()
}
