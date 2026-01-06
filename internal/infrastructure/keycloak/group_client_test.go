package keycloak_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestGroupClient creates a GroupClient with a mock token manager and test server URL.
func createTestGroupClient(t *testing.T, serverURL string) *keycloak.GroupClient {
	t.Helper()

	// Create a token server that always returns a valid token
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-admin-token",
			"expires_in":   300,
			"token_type":   "Bearer",
		})
	}))
	t.Cleanup(tokenServer.Close)

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: tokenServer.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    "admin",
		Password:    "admin123",
	})

	return keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: serverURL,
		Realm:       "flowra",
	}, tokenManager)
}

func TestNewGroupClient(t *testing.T) {
	t.Run("creates client with default values", func(t *testing.T) {
		tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
		})

		client := keycloak.NewGroupClient(keycloak.GroupClientConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "flowra",
		}, tokenManager)

		require.NotNil(t, client)
	})

	t.Run("uses custom HTTP client", func(t *testing.T) {
		tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
		})

		customClient := &http.Client{Timeout: 10 * time.Second}
		client := keycloak.NewGroupClient(keycloak.GroupClientConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "flowra",
			HTTPClient:  customClient,
		}, tokenManager)

		require.NotNil(t, client)
	})
}

func TestGroupClient_CreateGroup(t *testing.T) {
	t.Run("creates group successfully", func(t *testing.T) {
		groupID := "abc-123-def-456"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/groups", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, "test-workspace", body["name"])

			w.Header().Set("Location", "http://localhost:8090/admin/realms/flowra/groups/"+groupID)
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		id, err := client.CreateGroup(context.Background(), "test-workspace")

		require.NoError(t, err)
		assert.Equal(t, groupID, id)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.CreateGroup(context.Background(), "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrInvalidGroupName)
	})

	t.Run("returns error on conflict", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error": "Group already exists"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.CreateGroup(context.Background(), "existing-group")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupExists)
	})

	t.Run("returns error on server failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.CreateGroup(context.Background(), "test-group")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("returns error on missing Location header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.CreateGroup(context.Background(), "test-group")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Location header")
	})
}

func TestGroupClient_DeleteGroup(t *testing.T) {
	t.Run("deletes group successfully", func(t *testing.T) {
		groupID := "abc-123-def-456"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/groups/"+groupID, r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.DeleteGroup(context.Background(), groupID)

		require.NoError(t, err)
	})

	t.Run("returns error for empty groupID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.DeleteGroup(context.Background(), "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("returns error when group not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Group not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.DeleteGroup(context.Background(), "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("handles OK status as success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.DeleteGroup(context.Background(), "test-group")

		require.NoError(t, err)
	})
}

func TestGroupClient_AddUserToGroup(t *testing.T) {
	t.Run("adds user to group successfully", func(t *testing.T) {
		userID := "user-123"
		groupID := "group-456"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/admin/realms/flowra/users/" + userID + "/groups/" + groupID
			assert.Equal(t, expectedPath, r.URL.Path)
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), userID, groupID)

		require.NoError(t, err)
	})

	t.Run("returns error for empty userID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), "", "group-123")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns error for empty groupID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), "user-123", "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("returns user not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "User not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), "non-existent", "group-123")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns group not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Group not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), "user-123", "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("handles OK status as success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.AddUserToGroup(context.Background(), "user-123", "group-456")

		require.NoError(t, err)
	})
}

func TestGroupClient_RemoveUserFromGroup(t *testing.T) {
	t.Run("removes user from group successfully", func(t *testing.T) {
		userID := "user-123"
		groupID := "group-456"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/admin/realms/flowra/users/" + userID + "/groups/" + groupID
			assert.Equal(t, expectedPath, r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.RemoveUserFromGroup(context.Background(), userID, groupID)

		require.NoError(t, err)
	})

	t.Run("returns error for empty userID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.RemoveUserFromGroup(context.Background(), "", "group-123")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns error for empty groupID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.RemoveUserFromGroup(context.Background(), "user-123", "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("returns user not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "User not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.RemoveUserFromGroup(context.Background(), "non-existent", "group-123")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns group not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Group not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		err := client.RemoveUserFromGroup(context.Background(), "user-123", "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})
}

func TestGroupClient_GetGroup(t *testing.T) {
	t.Run("gets group successfully", func(t *testing.T) {
		groupID := "abc-123-def-456"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/groups/"+groupID, r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(keycloak.Group{
				ID:   groupID,
				Name: "test-workspace",
				Path: "/test-workspace",
			})
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		group, err := client.GetGroup(context.Background(), groupID)

		require.NoError(t, err)
		assert.Equal(t, groupID, group.ID)
		assert.Equal(t, "test-workspace", group.Name)
		assert.Equal(t, "/test-workspace", group.Path)
	})

	t.Run("returns error for empty groupID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.GetGroup(context.Background(), "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("returns error when group not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Group not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.GetGroup(context.Background(), "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not valid json`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.GetGroup(context.Background(), "test-group")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
	})
}

func TestGroupClient_GetUserGroups(t *testing.T) {
	t.Run("gets user groups successfully", func(t *testing.T) {
		userID := "user-123"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/users/"+userID+"/groups", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]keycloak.Group{
				{ID: "group-1", Name: "workspace-1", Path: "/workspace-1"},
				{ID: "group-2", Name: "workspace-2", Path: "/workspace-2"},
			})
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		groups, err := client.GetUserGroups(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, groups, 2)
		assert.Equal(t, "group-1", groups[0].ID)
		assert.Equal(t, "workspace-1", groups[0].Name)
		assert.Equal(t, "group-2", groups[1].ID)
		assert.Equal(t, "workspace-2", groups[1].Name)
	})

	t.Run("returns empty slice for user with no groups", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]keycloak.Group{})
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		groups, err := client.GetUserGroups(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Empty(t, groups)
	})

	t.Run("returns error for empty userID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.GetUserGroups(context.Background(), "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "User not found"}`))
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		_, err := client.GetUserGroups(context.Background(), "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})
}

func TestGroupClient_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Simulate slow server
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := client.CreateGroup(ctx, "test-group")

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "deadline"))
	})
}

func TestGroupClient_FullWorkflow(t *testing.T) {
	t.Run("create, get, add user, remove user, delete workflow", func(t *testing.T) {
		groupID := "workflow-group-123"
		userID := "workflow-user-456"

		requestLog := make([]string, 0)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestLog = append(requestLog, r.Method+" "+r.URL.Path)

			switch {
			case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/groups"):
				// Create group
				w.Header().Set("Location", "http://localhost:8090/admin/realms/flowra/groups/"+groupID)
				w.WriteHeader(http.StatusCreated)

			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/groups/"+groupID):
				// Get group
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(keycloak.Group{
					ID:   groupID,
					Name: "test-workspace",
					Path: "/test-workspace",
				})

			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/users/"):
				// Add user to group
				w.WriteHeader(http.StatusNoContent)

			case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/users/"):
				// Remove user from group
				w.WriteHeader(http.StatusNoContent)

			case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/groups/"+groupID):
				// Delete group
				w.WriteHeader(http.StatusNoContent)

			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := createTestGroupClient(t, server.URL)
		ctx := context.Background()

		// 1. Create group
		id, err := client.CreateGroup(ctx, "test-workspace")
		require.NoError(t, err)
		assert.Equal(t, groupID, id)

		// 2. Get group
		group, err := client.GetGroup(ctx, groupID)
		require.NoError(t, err)
		assert.Equal(t, "test-workspace", group.Name)

		// 3. Add user to group
		err = client.AddUserToGroup(ctx, userID, groupID)
		require.NoError(t, err)

		// 4. Remove user from group
		err = client.RemoveUserFromGroup(ctx, userID, groupID)
		require.NoError(t, err)

		// 5. Delete group
		err = client.DeleteGroup(ctx, groupID)
		require.NoError(t, err)

		// Verify all requests were made in order
		assert.Len(t, requestLog, 5)
		assert.Contains(t, requestLog[0], "POST")
		assert.Contains(t, requestLog[1], "GET")
		assert.Contains(t, requestLog[2], "PUT")
		assert.Contains(t, requestLog[3], "DELETE")
		assert.Contains(t, requestLog[3], "/users/")
		assert.Contains(t, requestLog[4], "DELETE")
		assert.Contains(t, requestLog[4], "/groups/")
	})
}
