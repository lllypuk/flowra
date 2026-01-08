package user

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Command базовый interface commands
type Command interface {
	CommandName() string
}

// RegisterUserCommand - регистрация user
type RegisterUserCommand struct {
	ExternalID  string // ID from внешней системы аутентификации (Keycloak, Auth0, etc.)
	Username    string
	Email       string
	DisplayName string
}

func (c RegisterUserCommand) CommandName() string { return "RegisterUser" }

// UpdateProfileCommand - update профиля
type UpdateProfileCommand struct {
	UserID      uuid.UUID
	DisplayName *string // опционально
	Email       *string // опционально
}

func (c UpdateProfileCommand) CommandName() string { return "UpdateProfile" }

// PromoteToAdminCommand - повышение before admin
type PromoteToAdminCommand struct {
	UserID     uuid.UUID
	PromotedBy uuid.UUID // должен быть system admin
}

func (c PromoteToAdminCommand) CommandName() string { return "PromoteToAdmin" }
