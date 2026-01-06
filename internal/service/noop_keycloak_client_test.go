package service_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoOpKeycloakClient(t *testing.T) {
	client := service.NewNoOpKeycloakClient()
	assert.NotNil(t, client)
}

func TestNoOpKeycloakClient_CreateGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
	}{
		{
			name:      "creates group with valid name",
			groupName: "test-workspace",
		},
		{
			name:      "creates group with empty name",
			groupName: "",
		},
		{
			name:      "creates group with special characters",
			groupName: "my-workspace-123!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := service.NewNoOpKeycloakClient()
			ctx := context.Background()

			groupID, err := client.CreateGroup(ctx, tt.groupName)

			require.NoError(t, err)
			assert.NotEmpty(t, groupID)
			// Verify it's a valid UUID format (36 characters with dashes)
			assert.Len(t, groupID, 36)
		})
	}
}

func TestNoOpKeycloakClient_CreateGroup_ReturnsUniqueIDs(t *testing.T) {
	client := service.NewNoOpKeycloakClient()
	ctx := context.Background()

	id1, err1 := client.CreateGroup(ctx, "group1")
	id2, err2 := client.CreateGroup(ctx, "group2")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, id1, id2, "each call should return a unique ID")
}

func TestNoOpKeycloakClient_DeleteGroup(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
	}{
		{
			name:    "deletes group with valid ID",
			groupID: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "deletes group with empty ID",
			groupID: "",
		},
		{
			name:    "deletes group with any string",
			groupID: "any-string-works",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := service.NewNoOpKeycloakClient()
			ctx := context.Background()

			err := client.DeleteGroup(ctx, tt.groupID)

			assert.NoError(t, err)
		})
	}
}

func TestNoOpKeycloakClient_AddUserToGroup(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		groupID string
	}{
		{
			name:    "adds user to group with valid IDs",
			userID:  "user-123",
			groupID: "group-456",
		},
		{
			name:    "adds user to group with empty user ID",
			userID:  "",
			groupID: "group-456",
		},
		{
			name:    "adds user to group with empty group ID",
			userID:  "user-123",
			groupID: "",
		},
		{
			name:    "adds user to group with both empty",
			userID:  "",
			groupID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := service.NewNoOpKeycloakClient()
			ctx := context.Background()

			err := client.AddUserToGroup(ctx, tt.userID, tt.groupID)

			assert.NoError(t, err)
		})
	}
}

func TestNoOpKeycloakClient_RemoveUserFromGroup(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		groupID string
	}{
		{
			name:    "removes user from group with valid IDs",
			userID:  "user-123",
			groupID: "group-456",
		},
		{
			name:    "removes user from group with empty user ID",
			userID:  "",
			groupID: "group-456",
		},
		{
			name:    "removes user from group with empty group ID",
			userID:  "user-123",
			groupID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := service.NewNoOpKeycloakClient()
			ctx := context.Background()

			err := client.RemoveUserFromGroup(ctx, tt.userID, tt.groupID)

			assert.NoError(t, err)
		})
	}
}

func TestNoOpKeycloakClient_FullWorkflow(t *testing.T) {
	// Test a complete workflow: create group -> add user -> remove user -> delete group
	client := service.NewNoOpKeycloakClient()
	ctx := context.Background()

	// Create group
	groupID, err := client.CreateGroup(ctx, "my-workspace")
	require.NoError(t, err)
	require.NotEmpty(t, groupID)

	// Add user to group
	userID := "user-abc-123"
	err = client.AddUserToGroup(ctx, userID, groupID)
	require.NoError(t, err)

	// Remove user from group
	err = client.RemoveUserFromGroup(ctx, userID, groupID)
	require.NoError(t, err)

	// Delete group
	err = client.DeleteGroup(ctx, groupID)
	require.NoError(t, err)
}

func TestNoOpKeycloakClient_CanceledContext(t *testing.T) {
	// NoOp client should work even with canceled context
	client := service.NewNoOpKeycloakClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// All operations should still succeed (they don't actually do anything)
	groupID, err := client.CreateGroup(ctx, "test")
	require.NoError(t, err)
	assert.NotEmpty(t, groupID)

	err = client.DeleteGroup(ctx, groupID)
	require.NoError(t, err)

	err = client.AddUserToGroup(ctx, "user", groupID)
	require.NoError(t, err)

	err = client.RemoveUserFromGroup(ctx, "user", groupID)
	require.NoError(t, err)
}
