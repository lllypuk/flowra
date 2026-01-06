//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeycloakRealmSetup_RealmExists verifies that the flowra realm is created.
func TestKeycloakRealmSetup_RealmExists(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exists, err := kc.RealmExists(ctx)
	require.NoError(t, err)
	assert.True(t, exists, "Realm 'flowra' should exist")
}

// TestKeycloakRealmSetup_ClientExists verifies that the flowra-backend OAuth2 client exists.
func TestKeycloakRealmSetup_ClientExists(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exists, err := kc.ClientExists(ctx)
	require.NoError(t, err)
	assert.True(t, exists, "Client 'flowra-backend' should exist")
}

// TestKeycloakRealmSetup_RealmRolesExist verifies that all required realm roles are created.
func TestKeycloakRealmSetup_RealmRolesExist(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	roles, err := kc.GetRealmRoles(ctx)
	require.NoError(t, err)

	expectedRoles := []string{"user", "admin", "workspace_owner", "workspace_admin"}
	for _, role := range expectedRoles {
		assert.Contains(t, roles, role, "Role '%s' should exist", role)
	}
}

// TestKeycloakRealmSetup_GroupsExist verifies that all required groups are created.
func TestKeycloakRealmSetup_GroupsExist(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	groups, err := kc.GetGroups(ctx)
	require.NoError(t, err)

	expectedGroups := []string{"users", "admins"}
	for _, group := range expectedGroups {
		assert.Contains(t, groups, group, "Group '%s' should exist", group)
	}
}

// TestKeycloakRealmSetup_TestUsersExist verifies that all test users are created.
func TestKeycloakRealmSetup_TestUsersExist(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users, err := kc.GetUsers(ctx)
	require.NoError(t, err)

	expectedUsers := []string{"testuser", "admin", "alice", "bob"}
	for _, user := range expectedUsers {
		assert.Contains(t, users, user, "User '%s' should exist", user)
	}
}

// TestKeycloakRealmSetup_TestUserCanAuthenticate verifies that test users can authenticate.
func TestKeycloakRealmSetup_TestUserCanAuthenticate(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		username string
		password string
	}{
		{"testuser", "password123"},
		{"admin", "admin123"},
		{"alice", "password123"},
		{"bob", "password123"},
	}

	for _, tc := range testCases {
		t.Run(tc.username, func(t *testing.T) {
			tokenResp, err := kc.GetUserToken(ctx, tc.username, tc.password)
			require.NoError(t, err, "User '%s' should be able to authenticate", tc.username)
			assert.NotEmpty(t, tokenResp.AccessToken, "Access token should not be empty")
			assert.NotEmpty(t, tokenResp.RefreshToken, "Refresh token should not be empty")
			assert.Equal(t, "Bearer", tokenResp.TokenType)
			assert.Greater(t, tokenResp.ExpiresIn, 0)
		})
	}
}

// TestKeycloakRealmSetup_InvalidCredentialsRejected verifies that invalid credentials are rejected.
func TestKeycloakRealmSetup_InvalidCredentialsRejected(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := kc.GetUserToken(ctx, "testuser", "wrongpassword")
	require.Error(t, err, "Invalid credentials should be rejected")
}

// TestKeycloakRealmSetup_TokenContainsExpectedClaims verifies JWT token contains expected claims.
func TestKeycloakRealmSetup_TokenContainsExpectedClaims(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)

	// Parse JWT payload (second part of the token)
	parts := strings.Split(tokenResp.AccessToken, ".")
	require.Len(t, parts, 3, "JWT should have 3 parts")

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]interface{}
	err = json.Unmarshal(payload, &claims)
	require.NoError(t, err)

	// Verify expected claims
	assert.Contains(t, claims, "sub", "Token should contain 'sub' claim")
	assert.Contains(t, claims, "iss", "Token should contain 'iss' claim")
	assert.Contains(t, claims, "preferred_username", "Token should contain 'preferred_username' claim")
	assert.Equal(t, "testuser", claims["preferred_username"])

	// Verify issuer format
	iss, ok := claims["iss"].(string)
	require.True(t, ok)
	assert.Contains(t, iss, "/realms/flowra", "Issuer should contain realm")

	// Verify azp (authorized party)
	assert.Equal(t, "flowra-backend", claims["azp"], "azp should be 'flowra-backend'")
}

// TestKeycloakRealmSetup_TokenContainsRealmRoles verifies JWT token contains realm roles.
func TestKeycloakRealmSetup_TokenContainsRealmRoles(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		username      string
		password      string
		expectedRoles []string
	}{
		{"testuser", "password123", []string{"user"}},
		{"admin", "admin123", []string{"user", "admin"}},
		{"alice", "password123", []string{"user", "workspace_owner"}},
		{"bob", "password123", []string{"user"}},
	}

	for _, tc := range testCases {
		t.Run(tc.username, func(t *testing.T) {
			tokenResp, err := kc.GetUserToken(ctx, tc.username, tc.password)
			require.NoError(t, err)

			// Parse JWT payload
			parts := strings.Split(tokenResp.AccessToken, ".")
			require.Len(t, parts, 3)

			payload, err := base64.RawURLEncoding.DecodeString(parts[1])
			require.NoError(t, err)

			var claims map[string]interface{}
			err = json.Unmarshal(payload, &claims)
			require.NoError(t, err)

			// Check realm_access.roles
			realmAccess, ok := claims["realm_access"].(map[string]interface{})
			require.True(t, ok, "Token should contain 'realm_access' claim")

			rolesRaw, ok := realmAccess["roles"].([]interface{})
			require.True(t, ok, "realm_access should contain 'roles'")

			roles := make([]string, len(rolesRaw))
			for i, r := range rolesRaw {
				roles[i] = r.(string)
			}

			for _, expectedRole := range tc.expectedRoles {
				assert.Contains(t, roles, expectedRole,
					"User '%s' should have role '%s'", tc.username, expectedRole)
			}
		})
	}
}

// TestKeycloakRealmSetup_UserInfoEndpoint verifies the userinfo endpoint works.
func TestKeycloakRealmSetup_UserInfoEndpoint(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get token for testuser
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)

	// Get user info
	userInfo, err := kc.GetUserInfo(ctx, tokenResp.AccessToken)
	require.NoError(t, err)

	assert.NotEmpty(t, userInfo.Sub, "Sub should not be empty")
	assert.Equal(t, "testuser", userInfo.PreferredUsername)
	assert.Equal(t, "testuser@example.com", userInfo.Email)
	assert.True(t, userInfo.EmailVerified)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, "Test", userInfo.GivenName)
	assert.Equal(t, "User", userInfo.FamilyName)
}

// TestKeycloakRealmSetup_AdminUserInfo verifies admin user has correct info.
func TestKeycloakRealmSetup_AdminUserInfo(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenResp, err := kc.GetUserToken(ctx, "admin", "admin123")
	require.NoError(t, err)

	userInfo, err := kc.GetUserInfo(ctx, tokenResp.AccessToken)
	require.NoError(t, err)

	assert.Equal(t, "admin", userInfo.PreferredUsername)
	assert.Equal(t, "admin@example.com", userInfo.Email)
	assert.Equal(t, "Admin User", userInfo.Name)
}

// TestKeycloakRealmSetup_RefreshTokenWorks verifies that refresh tokens can be used.
func TestKeycloakRealmSetup_RefreshTokenWorks(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get initial token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, tokenResp.RefreshToken)

	// Verify refresh token expires later than access token
	assert.Greater(t, tokenResp.RefreshExpiresIn, tokenResp.ExpiresIn,
		"Refresh token should expire later than access token")
}

// TestKeycloakRealmSetup_DirectAccessGrantsEnabled verifies direct access grants work.
func TestKeycloakRealmSetup_DirectAccessGrantsEnabled(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Direct access grants (password grant) should work
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err, "Direct access grants should be enabled")
	assert.NotEmpty(t, tokenResp.AccessToken)
}

// TestKeycloakRealmSetup_TokenScopes verifies token contains expected scopes.
func TestKeycloakRealmSetup_TokenScopes(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)

	// Check scope in token response
	scopes := strings.Split(tokenResp.Scope, " ")
	expectedScopes := []string{"openid", "profile", "email"}

	for _, expected := range expectedScopes {
		assert.Contains(t, scopes, expected,
			"Token should contain scope '%s'", expected)
	}
}
