package keycloak_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

func TestNewOAuthClient(t *testing.T) {
	t.Run("creates client with default http client", func(t *testing.T) {
		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  "http://localhost:8080",
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		assert.NotNil(t, client)
	})

	t.Run("creates client with custom http client", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 10 * time.Second,
		}

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  "http://localhost:8080",
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			HTTPClient:   customClient,
		})

		assert.NotNil(t, client)
	})
}

func TestOAuthClient_ExchangeCode(t *testing.T) {
	t.Run("successfully exchanges code for tokens", func(t *testing.T) {
		expectedResponse := keycloak.TokenResponse{
			AccessToken:      "test-access-token",
			RefreshToken:     "test-refresh-token",
			ExpiresIn:        3600,
			RefreshExpiresIn: 7200,
			TokenType:        "Bearer",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/realms/test-realm/protocol/openid-connect/token")
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
			assert.Equal(t, "test-code", r.Form.Get("code"))
			assert.Equal(t, "http://localhost/callback", r.Form.Get("redirect_uri"))
			assert.Equal(t, "test-client", r.Form.Get("client_id"))
			assert.Equal(t, "test-secret", r.Form.Get("client_secret"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.AccessToken, result.AccessToken)
		assert.Equal(t, expectedResponse.RefreshToken, result.RefreshToken)
		assert.Equal(t, expectedResponse.ExpiresIn, result.ExpiresIn)
		assert.Equal(t, expectedResponse.RefreshExpiresIn, result.RefreshExpiresIn)
	})

	t.Run("returns error on invalid code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid_grant", "error_description": "Code not valid"}`))
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.ExchangeCode(context.Background(), "invalid-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, keycloak.ErrTokenExchangeFailed)
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on network failure", func(t *testing.T) {
		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  "http://localhost:99999", // Invalid port
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, keycloak.ErrInvalidResponse)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.ExchangeCode(ctx, "test-code", "http://localhost/callback")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOAuthClient_RefreshToken(t *testing.T) {
	t.Run("successfully refreshes token", func(t *testing.T) {
		expectedResponse := keycloak.TokenResponse{
			AccessToken:      "new-access-token",
			RefreshToken:     "new-refresh-token",
			ExpiresIn:        3600,
			RefreshExpiresIn: 7200,
			TokenType:        "Bearer",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/realms/test-realm/protocol/openid-connect/token")

			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			assert.Equal(t, "old-refresh-token", r.Form.Get("refresh_token"))
			assert.Equal(t, "test-client", r.Form.Get("client_id"))
			assert.Equal(t, "test-secret", r.Form.Get("client_secret"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.RefreshToken(context.Background(), "old-refresh-token")

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.AccessToken, result.AccessToken)
		assert.Equal(t, expectedResponse.RefreshToken, result.RefreshToken)
	})

	t.Run("returns error on expired refresh token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid_grant", "error_description": "Token is not active"}`))
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.RefreshToken(context.Background(), "expired-token")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, keycloak.ErrTokenRefreshFailed)
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.RefreshToken(context.Background(), "test-token")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOAuthClient_RevokeToken(t *testing.T) {
	t.Run("successfully revokes token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/realms/test-realm/protocol/openid-connect/revoke")

			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			assert.Equal(t, "test-refresh-token", r.Form.Get("token"))
			assert.Equal(t, "refresh_token", r.Form.Get("token_type_hint"))
			assert.Equal(t, "test-client", r.Form.Get("client_id"))
			assert.Equal(t, "test-secret", r.Form.Get("client_secret"))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		err := client.RevokeToken(context.Background(), "test-refresh-token")

		require.NoError(t, err)
	})

	t.Run("succeeds for already revoked token (idempotent)", func(t *testing.T) {
		// Keycloak returns 200 even if token is already revoked
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		err := client.RevokeToken(context.Background(), "already-revoked-token")

		require.NoError(t, err)
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		err := client.RevokeToken(context.Background(), "test-token")

		require.Error(t, err)
		assert.ErrorIs(t, err, keycloak.ErrTokenRevokeFailed)
	})
}

func TestOAuthClient_GetUserInfo(t *testing.T) {
	t.Run("successfully gets user info", func(t *testing.T) {
		expectedUserInfo := keycloak.UserInfo{
			Sub:               "user-123",
			PreferredUsername: "testuser",
			Email:             "test@example.com",
			EmailVerified:     true,
			Name:              "Test User",
			GivenName:         "Test",
			FamilyName:        "User",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/realms/test-realm/protocol/openid-connect/userinfo")
			assert.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedUserInfo)
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.GetUserInfo(context.Background(), "test-access-token")

		require.NoError(t, err)
		assert.Equal(t, expectedUserInfo.Sub, result.Sub)
		assert.Equal(t, expectedUserInfo.PreferredUsername, result.PreferredUsername)
		assert.Equal(t, expectedUserInfo.Email, result.Email)
		assert.Equal(t, expectedUserInfo.Name, result.Name)
	})

	t.Run("returns error on invalid access token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid_token"}`))
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.GetUserInfo(context.Background(), "invalid-token")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, keycloak.ErrUserInfoFailed)
	})

	t.Run("returns error on invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  server.URL,
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		result, err := client.GetUserInfo(context.Background(), "test-token")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, keycloak.ErrInvalidResponse)
	})
}

func TestOAuthClient_AuthorizationURL(t *testing.T) {
	t.Run("generates correct authorization URL", func(t *testing.T) {
		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  "http://localhost:8080",
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		url := client.AuthorizationURL("http://localhost/callback", "test-state")

		assert.Contains(t, url, "http://localhost:8080/realms/test-realm/protocol/openid-connect/auth")
		assert.Contains(t, url, "client_id=test-client")
		assert.Contains(t, url, "redirect_uri=")
		assert.Contains(t, url, "response_type=code")
		assert.Contains(t, url, "scope=openid+profile+email")
		assert.Contains(t, url, "state=test-state")
	})

	t.Run("handles trailing slash in keycloak URL", func(t *testing.T) {
		client := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
			KeycloakURL:  "http://localhost:8080/",
			Realm:        "test-realm",
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		})

		url := client.AuthorizationURL("http://localhost/callback", "test-state")

		// Should not have double slashes
		assert.NotContains(t, url, "http://localhost:8080//realms")
		assert.Contains(t, url, "http://localhost:8080/realms")
	})
}
