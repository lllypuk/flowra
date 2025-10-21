package user

import (
	"time"

	"github.com/flowra/flowra/internal/domain/errs"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// User представляет пользователя системы
type User struct {
	id            uuid.UUID
	externalID    string // ID из внешней системы аутентификации (Keycloak, Auth0, etc.)
	username      string
	email         string
	displayName   string
	isSystemAdmin bool
	createdAt     time.Time
	updatedAt     time.Time
}

// NewUser создает нового пользователя
func NewUser(externalID, username, email, displayName string) (*User, error) {
	if externalID == "" {
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
		externalID:  externalID,
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
	externalID, username, email, displayName string,
	isSystemAdmin bool,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:            id,
		externalID:    externalID,
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

// ExternalID возвращает ID пользователя во внешней системе аутентификации
func (u *User) ExternalID() string {
	return u.externalID
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
