package workspace

import "context"

// KeycloakClient defines interface for work s Keycloak
// sleduem printsipu: interface obyavlyaetsya on storone potrebitelya
type KeycloakClient interface {
	// CreateGroup creates groups in Keycloak
	CreateGroup(ctx context.Context, name string) (groupID string, err error)

	// DeleteGroup udalyaet groups in Keycloak
	DeleteGroup(ctx context.Context, groupID string) error

	// AddUserToGroup adds user in groups Keycloak
	AddUserToGroup(ctx context.Context, userID, groupID string) error

	// RemoveUserFromGroup udalyaet user from groups Keycloak
	RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
}
