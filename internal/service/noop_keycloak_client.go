package service

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// NoOpKeycloakClient implements workspace.KeycloakClient interface
// and does nothing - for use when Keycloak integration is not required.
type NoOpKeycloakClient struct{}

// NewNoOpKeycloakClient creates a new NoOpKeycloakClient.
func NewNoOpKeycloakClient() *NoOpKeycloakClient {
	return &NoOpKeycloakClient{}
}

// CreateGroup returns a generated group ID without actually creating anything.
func (c *NoOpKeycloakClient) CreateGroup(_ context.Context, _ string) (string, error) {
	// Return a generated UUID as the group ID
	return uuid.NewUUID().String(), nil
}

// DeleteGroup does nothing and returns nil.
func (c *NoOpKeycloakClient) DeleteGroup(_ context.Context, _ string) error {
	return nil
}

// AddUserToGroup does nothing and returns nil.
func (c *NoOpKeycloakClient) AddUserToGroup(_ context.Context, _, _ string) error {
	return nil
}

// RemoveUserFromGroup does nothing and returns nil.
func (c *NoOpKeycloakClient) RemoveUserFromGroup(_ context.Context, _, _ string) error {
	return nil
}
