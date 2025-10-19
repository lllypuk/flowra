package workspace

import "context"

// KeycloakClient определяет интерфейс для работы с Keycloak
// Следуем принципу: интерфейс объявляется на стороне потребителя
type KeycloakClient interface {
	// CreateGroup создает группу в Keycloak
	CreateGroup(ctx context.Context, name string) (groupID string, err error)

	// DeleteGroup удаляет группу в Keycloak
	DeleteGroup(ctx context.Context, groupID string) error

	// AddUserToGroup добавляет пользователя в группу Keycloak
	AddUserToGroup(ctx context.Context, userID, groupID string) error

	// RemoveUserFromGroup удаляет пользователя из группы Keycloak
	RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
}
