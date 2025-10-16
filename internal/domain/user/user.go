package user

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// User представляет пользователя системы
type User struct {
	id            uuid.UUID
	username      string
	email         string
	displayName   string
	isSystemAdmin bool
	createdAt     time.Time
	updatedAt     time.Time
}

// NewUser создает нового пользователя
func NewUser(username, email, displayName string) (*User, error) {
	if username == "" {
		return nil, errs.ErrInvalidInput
	}
	if email == "" {
		return nil, errs.ErrInvalidInput
	}

	return &User{
		id:          uuid.NewUUID(),
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
	username, email, displayName string,
	isSystemAdmin bool,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:            id,
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
func (u *User) UpdateProfile(displayName string) error {
	if displayName == "" {
		return errs.ErrInvalidInput
	}
	u.displayName = displayName
	u.updatedAt = time.Now()
	return nil
}

// SetAdmin устанавливает права администратора
func (u *User) SetAdmin(isAdmin bool) {
	u.isSystemAdmin = isAdmin
	u.updatedAt = time.Now()
}
