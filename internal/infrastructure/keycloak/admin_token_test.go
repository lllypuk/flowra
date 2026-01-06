package keycloak_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdminTokenManager(t *testing.T) {
	t.Run("creates manager with default values", func(t *testing.T) {
		config := keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		}

		manager := keycloak.NewAdminTokenManager(config)

		require.NotNil(t, manager)
	})

	t.Run("uses custom token buffer", func(t *testing.T) {
		config := keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
			TokenBuffer: 60 * time.Second,
		}

		manager := keycloak.NewAdminTokenManager(config)

		require.NotNil(t, manager)
	})

	t.Run("uses custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 10 * time.Second}
		config := keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:8090",
			Realm:       "master",
			ClientID:    "admin-cli",
			HTTPClient:  customClient,
		}

		manager := keycloak.NewAdminTokenManager(config)

		require.NotNil(t, manager)
	})
}

func TestAdminTokenManager_GetToken_PasswordGrant(t *testing.T) {
	t.Run("fetches token using password grant", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/realms/master/protocol/openid-connect/token", r.URL.Path)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			err := r.ParseForm()
			assert.NoError(t, err)

			assert.Equal(t, "password", r.FormValue("grant_type"))
			assert.Equal(t, "admin-cli", r.FormValue("client_id"))
			assert.Equal(t, "admin", r.FormValue("username"))
			assert.Equal(t, "admin123", r.FormValue("password"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-admin-token",
				"expires_in":   300,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		token, err := manager.GetToken(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "test-admin-token", token)
	})
}

func TestAdminTokenManager_GetToken_ClientCredentialsGrant(t *testing.T) {
	t.Run("fetches token using client credentials grant", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			assert.NoError(t, err)

			assert.Equal(t, "client_credentials", r.FormValue("grant_type"))
			assert.Equal(t, "admin-cli", r.FormValue("client_id"))
			assert.Equal(t, "super-secret", r.FormValue("client_secret"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "client-credentials-token",
				"expires_in":   300,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL:  server.URL,
			Realm:        "master",
			ClientID:     "admin-cli",
			ClientSecret: "super-secret",
		})

		token, err := manager.GetToken(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "client-credentials-token", token)
	})
}

func TestAdminTokenManager_GetToken_Caching(t *testing.T) {
	t.Run("caches token and returns cached value", func(t *testing.T) {
		var callCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "cached-token",
				"expires_in":   300,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		// First call - should hit the server
		token1, err := manager.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token1)

		// Second call - should return cached token
		token2, err := manager.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token2)

		// Third call - should still return cached token
		token3, err := manager.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token3)

		// Server should only have been called once
		assert.Equal(t, int32(1), callCount.Load())
	})
}

func TestAdminTokenManager_GetToken_RefreshBeforeExpiry(t *testing.T) {
	t.Run("refreshes token when approaching expiry", func(t *testing.T) {
		var callCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			count := callCount.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("token-%d", count),
				"expires_in":   1, // Very short expiry
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
			TokenBuffer: 2 * time.Second, // Buffer is longer than token lifetime
		})

		// First call
		_, err := manager.GetToken(context.Background())
		require.NoError(t, err)

		// Second call should refresh because buffer exceeds remaining time
		_, err = manager.GetToken(context.Background())
		require.NoError(t, err)

		// Should have made 2 requests
		assert.Equal(t, int32(2), callCount.Load())
	})
}

func TestAdminTokenManager_GetToken_ErrorHandling(t *testing.T) {
	t.Run("returns error on HTTP failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "invalid_grant", "error_description": "Invalid credentials"}`))
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "wrong-password",
		})

		_, err := manager.GetToken(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("returns error on network failure", func(t *testing.T) {
		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: "http://localhost:59999", // Non-existent server
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		_, err := manager.GetToken(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "admin token request failed")
	})

	t.Run("returns error on invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not valid json`))
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		_, err := manager.GetToken(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode")
	})
}

func TestAdminTokenManager_GetToken_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Simulate slow server
			time.Sleep(100 * time.Millisecond)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "token",
				"expires_in":   300,
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := manager.GetToken(ctx)

		require.Error(t, err)
	})
}

func TestAdminTokenManager_GetToken_Concurrency(t *testing.T) {
	t.Run("handles concurrent requests safely", func(t *testing.T) {
		var callCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount.Add(1)
			// Small delay to increase chance of concurrent access
			time.Sleep(10 * time.Millisecond)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "concurrent-token",
				"expires_in":   300,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		tokens := make([]string, numGoroutines)
		errs := make([]error, numGoroutines)

		for i := range numGoroutines {
			go func(idx int) {
				defer wg.Done()
				tokens[idx], errs[idx] = manager.GetToken(context.Background())
			}(i)
		}

		wg.Wait()

		// All requests should succeed
		for i := range numGoroutines {
			require.NoError(t, errs[i], "goroutine %d failed", i)
			assert.Equal(t, "concurrent-token", tokens[i], "goroutine %d got wrong token", i)
		}

		// Due to double-check locking, only a few requests should hit the server
		// (ideally 1, but race conditions might cause a few more)
		assert.LessOrEqual(t, callCount.Load(), int32(5),
			"expected minimal server calls due to caching, got %d", callCount.Load())
	})
}

func TestAdminTokenManager_InvalidateToken(t *testing.T) {
	t.Run("invalidates cached token", func(t *testing.T) {
		var callCount atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			count := callCount.Add(1)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("token-v%d", count),
				"expires_in":   300,
				"token_type":   "Bearer",
			})
		}))
		defer server.Close()

		manager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: server.URL,
			Realm:       "master",
			ClientID:    "admin-cli",
			Username:    "admin",
			Password:    "admin123",
		})

		// First call
		token1, err := manager.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "token-v1", token1)

		// Invalidate
		manager.InvalidateToken()

		// Should fetch new token
		token2, err := manager.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "token-v2", token2)

		assert.Equal(t, int32(2), callCount.Load())
	})
}
