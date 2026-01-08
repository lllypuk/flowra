package user

import "github.com/lllypuk/flowra/internal/domain/uuid"

// Command bazovyy interface commands
type Command interface {
	CommandName() string
}

// RegisterUserCommand - registratsiya user
type RegisterUserCommand struct {
	ExternalID  string // ID from vneshney sistemy autentifikatsii (Keycloak, Auth0, etc.)
	Username    string
	Email       string
	DisplayName string
}

func (c RegisterUserCommand) CommandName() string { return "RegisterUser" }

// UpdateProfileCommand - update profilya
type UpdateProfileCommand struct {
	UserID      uuid.UUID
	DisplayName *string // optsionalno
	Email       *string // optsionalno
}

func (c UpdateProfileCommand) CommandName() string { return "UpdateProfile" }

// PromoteToAdminCommand - povyshenie before admin
type PromoteToAdminCommand struct {
	UserID     uuid.UUID
	PromotedBy uuid.UUID // dolzhen byt system admin
}

func (c PromoteToAdminCommand) CommandName() string { return "PromoteToAdmin" }
