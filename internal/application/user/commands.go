package user

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Command базовый интерфейс команд
type Command interface {
	CommandName() string
}

// RegisterUserCommand - регистрация пользователя
type RegisterUserCommand struct {
	ExternalID  string // ID из внешней системы аутентификации (Keycloak, Auth0, etc.)
	Username    string
	Email       string
	DisplayName string
}

func (c RegisterUserCommand) CommandName() string { return "RegisterUser" }

// UpdateProfileCommand - обновление профиля
type UpdateProfileCommand struct {
	UserID      uuid.UUID
	DisplayName *string // опционально
	Email       *string // опционально
}

func (c UpdateProfileCommand) CommandName() string { return "UpdateProfile" }

// PromoteToAdminCommand - повышение до admin
type PromoteToAdminCommand struct {
	UserID     uuid.UUID
	PromotedBy uuid.UUID // должен быть system admin
}

func (c PromoteToAdminCommand) CommandName() string { return "PromoteToAdmin" }
