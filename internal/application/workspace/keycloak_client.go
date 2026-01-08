package workspace

import "context"

// KeycloakClient defines interface for workы с Keycloak
// Следуем принципу: interface объявляется on стороне потребителя
type KeycloakClient interface {
	// CreateGroup creates groupsу in Keycloak
	CreateGroup(ctx context.Context, name string) (groupID string, err error)

	// DeleteGroup удаляет groupsу in Keycloak
	DeleteGroup(ctx context.Context, groupID string) error

	// AddUserToGroup добавляет user in groupsу Keycloak
	AddUserToGroup(ctx context.Context, userID, groupID string) error

	// RemoveUserFromGroup удаляет user from groupsы Keycloak
	RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
}
