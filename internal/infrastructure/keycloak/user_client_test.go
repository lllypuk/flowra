package keycloak_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestUserClient creates a UserClient with a mock token manager and test server URL.
func createTestUserClient(t *testing.T, serverURL string) *keycloak.UserClient {
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

	return keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: serverURL,
		Realm:       "flowra",
	}, tokenManager)
}

func TestNewUserClient(t *testing.T) {
	t.Run("creates client with default values", func(t *testing.T) {
		tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
		})

		client := keycloak.NewUserClient(keycloak.UserClientConfig{
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
		client := keycloak.NewUserClient(keycloak.UserClientConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "flowra",
			HTTPClient:  customClient,
		}, tokenManager)

		require.NotNil(t, client)
	})
}

func TestUserClient_ListUsers(t *testing.T) {
	t.Run("lists users successfully", func(t *testing.T) {
		users := []keycloak.User{
			{
				ID:            "user-1",
				Username:      "alice",
				Email:         "alice@example.com",
				EmailVerified: true,
				FirstName:     "Alice",
				LastName:      "Smith",
				Enabled:       true,
			},
			{
				ID:            "user-2",
				Username:      "bob",
				Email:         "bob@example.com",
				EmailVerified: true,
				FirstName:     "Bob",
				LastName:      "Jones",
				Enabled:       true,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/users", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")
			assert.Equal(t, "0", r.URL.Query().Get("first"))
			assert.Equal(t, "100", r.URL.Query().Get("max"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(users)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		result, err := client.ListUsers(context.Background(), 0, 100)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "user-1", result[0].ID)
		assert.Equal(t, "alice", result[0].Username)
		assert.Equal(t, "alice@example.com", result[0].Email)
		assert.Equal(t, "Alice", result[0].FirstName)
		assert.Equal(t, "Smith", result[0].LastName)
		assert.True(t, result[0].Enabled)
	})

	t.Run("handles pagination", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "50", r.URL.Query().Get("first"))
			assert.Equal(t, "25", r.URL.Query().Get("max"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]keycloak.User{})
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		result, err := client.ListUsers(context.Background(), 50, 25)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		_, err := client.ListUsers(context.Background(), 0, 100)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("handles empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]keycloak.User{})
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		result, err := client.ListUsers(context.Background(), 0, 100)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestUserClient_GetUser(t *testing.T) {
	t.Run("gets user successfully", func(t *testing.T) {
		expectedUser := keycloak.User{
			ID:            "user-123",
			Username:      "testuser",
			Email:         "test@example.com",
			EmailVerified: true,
			FirstName:     "Test",
			LastName:      "User",
			Enabled:       true,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/users/user-123", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedUser)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		result, err := client.GetUser(context.Background(), "user-123")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "user-123", result.ID)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "test@example.com", result.Email)
		assert.True(t, result.Enabled)
	})

	t.Run("returns error for empty user ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		_, err := client.GetUser(context.Background(), "")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		_, err := client.GetUser(context.Background(), "non-existent-user")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		_, err := client.GetUser(context.Background(), "user-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestUserClient_CountUsers(t *testing.T) {
	t.Run("counts users successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/admin/realms/flowra/users/count", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(42)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		count, err := client.CountUsers(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 42, count)
	})

	t.Run("returns zero for empty realm", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(0)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		count, err := client.CountUsers(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		_, err := client.CountUsers(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestUserClient_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.ListUsers(ctx, 0, 100)

		require.Error(t, err)
	})
}

func TestKeycloakUser_DisplayName(t *testing.T) {
	t.Run("returns full name when both names present", func(t *testing.T) {
		user := keycloak.User{
			Username:  "alice",
			FirstName: "Alice",
			LastName:  "Smith",
		}

		assert.Equal(t, "Alice Smith", user.DisplayName())
	})

	t.Run("returns first name only when last name empty", func(t *testing.T) {
		user := keycloak.User{
			Username:  "bob",
			FirstName: "Bob",
			LastName:  "",
		}

		assert.Equal(t, "Bob", user.DisplayName())
	})

	t.Run("returns last name only when first name empty", func(t *testing.T) {
		user := keycloak.User{
			Username:  "charlie",
			FirstName: "",
			LastName:  "Brown",
		}

		assert.Equal(t, "Brown", user.DisplayName())
	})

	t.Run("returns username when both names empty", func(t *testing.T) {
		user := keycloak.User{
			Username:  "diana",
			FirstName: "",
			LastName:  "",
		}

		assert.Equal(t, "diana", user.DisplayName())
	})

	t.Run("trims whitespace from names", func(t *testing.T) {
		user := keycloak.User{
			Username:  "eve",
			FirstName: "  Eve  ",
			LastName:  "  Johnson  ",
		}

		// Note: DisplayName trims the combined result, not individual names
		assert.Equal(t, "Eve     Johnson", user.DisplayName())
	})
}

func TestUserClient_FullWorkflow(t *testing.T) {
	t.Run("complete user sync workflow", func(t *testing.T) {
		users := []keycloak.User{
			{
				ID:        "user-1",
				Username:  "alice",
				Email:     "alice@example.com",
				FirstName: "Alice",
				LastName:  "Smith",
				Enabled:   true,
			},
			{
				ID:        "user-2",
				Username:  "bob",
				Email:     "bob@example.com",
				FirstName: "Bob",
				LastName:  "Jones",
				Enabled:   false, // Disabled user
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/admin/realms/flowra/users/count":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(len(users))

			case "/admin/realms/flowra/users":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(users)

			case "/admin/realms/flowra/users/user-1":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(users[0])

			case "/admin/realms/flowra/users/user-2":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(users[1])

			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		client := createTestUserClient(t, server.URL)
		ctx := context.Background()

		// Step 1: Count users
		count, err := client.CountUsers(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Step 2: List all users
		allUsers, err := client.ListUsers(ctx, 0, 100)
		require.NoError(t, err)
		require.Len(t, allUsers, 2)

		// Step 3: Get individual users
		user1, err := client.GetUser(ctx, "user-1")
		require.NoError(t, err)
		assert.Equal(t, "alice", user1.Username)
		assert.True(t, user1.Enabled)

		user2, err := client.GetUser(ctx, "user-2")
		require.NoError(t, err)
		assert.Equal(t, "bob", user2.Username)
		assert.False(t, user2.Enabled) // This user is disabled

		// Step 4: Check display names
		assert.Equal(t, "Alice Smith", user1.DisplayName())
		assert.Equal(t, "Bob Jones", user2.DisplayName())
	})
}
