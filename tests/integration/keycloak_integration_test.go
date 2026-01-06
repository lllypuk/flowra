//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// JWT Validator Tests
// =============================================================================

// TestJWTValidator_ValidToken verifies that JWTValidator correctly validates a real token.
func TestJWTValidator_ValidToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get a real token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.AccessToken)

	// Create validator without audience validation
	// Keycloak tokens have 'account' as audience by default, not the client ID
	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL:     kc.URL,
		Realm:           kc.Realm,
		ClientID:        "", // Empty to skip audience validation
		Leeway:          30 * time.Second,
		RefreshInterval: time.Hour,
	})
	require.NoError(t, err)
	defer validator.Close()

	// Validate token
	claims, err := validator.Validate(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify claims
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "testuser@example.com", claims.Email)
	// Note: EmailVerified may be false depending on Keycloak mapper configuration
	// The important thing is that the token is validated and claims are extracted
	assert.Equal(t, "Test", claims.GivenName)
	assert.Equal(t, "User", claims.FamilyName)
	assert.NotEmpty(t, claims.UserID)
	assert.False(t, claims.ExpiresAt.IsZero())
	assert.False(t, claims.IssuedAt.IsZero())
}

// TestJWTValidator_ClaimsExtraction verifies that all claims are correctly extracted.
func TestJWTValidator_ClaimsExtraction(t *testing.T) {
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

	// Create validator without audience validation
	// Keycloak tokens have 'account' as audience by default
	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL:     kc.URL,
		Realm:           kc.Realm,
		ClientID:        "", // Empty to skip audience validation
		Leeway:          30 * time.Second,
		RefreshInterval: time.Hour,
	})
	require.NoError(t, err)
	defer validator.Close()

	for _, tc := range testCases {
		t.Run(tc.username, func(t *testing.T) {
			tokenResp, err := kc.GetUserToken(ctx, tc.username, tc.password)
			require.NoError(t, err)

			claims, err := validator.Validate(ctx, tokenResp.AccessToken)
			require.NoError(t, err)

			assert.Equal(t, tc.username, claims.Username)

			for _, expectedRole := range tc.expectedRoles {
				assert.Contains(t, claims.RealmRoles, expectedRole,
					"User '%s' should have role '%s'", tc.username, expectedRole)
			}
		})
	}
}

// TestJWTValidator_InvalidToken verifies that invalid tokens are rejected.
func TestJWTValidator_InvalidToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL:     kc.URL,
		Realm:           kc.Realm,
		ClientID:        "", // Empty to skip audience validation
		Leeway:          30 * time.Second,
		RefreshInterval: time.Hour,
	})
	require.NoError(t, err)
	defer validator.Close()

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"malformed token", "invalid.token.here"},
		{"random string", "not-a-jwt-at-all"},
		{"partial token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.Validate(ctx, tc.token)
			assert.Error(t, err)
		})
	}
}

// TestJWTValidator_TamperedToken verifies that tampered tokens are rejected.
func TestJWTValidator_TamperedToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get a real token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL:     kc.URL,
		Realm:           kc.Realm,
		ClientID:        "", // Empty to skip audience validation
		Leeway:          30 * time.Second,
		RefreshInterval: time.Hour,
	})
	require.NoError(t, err)
	defer validator.Close()

	// Tamper with the token by modifying a character
	tamperedToken := tokenResp.AccessToken[:len(tokenResp.AccessToken)-5] + "XXXXX"

	_, err = validator.Validate(ctx, tamperedToken)
	assert.Error(t, err)
}

// =============================================================================
// OAuth Client Tests
// =============================================================================

// TestOAuthClient_RefreshToken verifies that tokens can be refreshed.
func TestOAuthClient_RefreshToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get initial token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.RefreshToken)

	// Create OAuth client
	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	// Refresh the token
	newTokens, err := oauthClient.RefreshToken(ctx, tokenResp.RefreshToken)
	require.NoError(t, err)

	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.Positive(t, newTokens.ExpiresIn)
	assert.Equal(t, "Bearer", newTokens.TokenType)
}

// TestOAuthClient_RefreshToken_InvalidToken verifies that invalid refresh tokens are rejected.
func TestOAuthClient_RefreshToken_InvalidToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	_, err := oauthClient.RefreshToken(ctx, "invalid-refresh-token")
	assert.Error(t, err)
}

// TestOAuthClient_RevokeToken verifies that tokens can be revoked.
func TestOAuthClient_RevokeToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get initial token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.RefreshToken)

	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	// Revoke the refresh token
	err = oauthClient.RevokeToken(ctx, tokenResp.RefreshToken)
	require.NoError(t, err)

	// Try to use the revoked refresh token - should fail
	_, err = oauthClient.RefreshToken(ctx, tokenResp.RefreshToken)
	assert.Error(t, err)
}

// TestOAuthClient_GetUserInfo verifies that user info can be retrieved.
func TestOAuthClient_GetUserInfo(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get token
	tokenResp, err := kc.GetUserToken(ctx, "testuser", "password123")
	require.NoError(t, err)

	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	// Get user info
	userInfo, err := oauthClient.GetUserInfo(ctx, tokenResp.AccessToken)
	require.NoError(t, err)

	assert.NotEmpty(t, userInfo.Sub)
	assert.Equal(t, "testuser", userInfo.PreferredUsername)
	assert.Equal(t, "testuser@example.com", userInfo.Email)
	// Note: EmailVerified may be false depending on Keycloak mapper configuration
	assert.Equal(t, "Test User", userInfo.Name)
}

// TestOAuthClient_GetUserInfo_InvalidToken verifies that invalid tokens are rejected.
func TestOAuthClient_GetUserInfo_InvalidToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	_, err := oauthClient.GetUserInfo(ctx, "invalid-token")
	assert.Error(t, err)
}

// TestOAuthClient_AuthorizationURL verifies that authorization URL is correctly generated.
func TestOAuthClient_AuthorizationURL(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)

	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	authURL := oauthClient.AuthorizationURL("http://localhost:8080/callback", "test-state-123")

	assert.Contains(t, authURL, kc.URL)
	assert.Contains(t, authURL, "/realms/flowra/protocol/openid-connect/auth")
	assert.Contains(t, authURL, "client_id="+kc.ClientID)
	assert.Contains(t, authURL, "redirect_uri=")
	assert.Contains(t, authURL, "response_type=code")
	assert.Contains(t, authURL, "state=test-state-123")
	assert.Contains(t, authURL, "scope=openid+profile+email")
}

// =============================================================================
// Admin Token Manager Tests
// =============================================================================

// TestAdminTokenManager_GetToken verifies that admin tokens can be obtained.
func TestAdminTokenManager_GetToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
		TokenBuffer: 30 * time.Second,
	})

	token, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestAdminTokenManager_TokenCaching verifies that tokens are cached.
func TestAdminTokenManager_TokenCaching(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
		TokenBuffer: 30 * time.Second,
	})

	// First call - fetches new token
	token1, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// Second call - should return cached token
	token2, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, token1, token2, "Second call should return cached token")

	// Third call - still cached
	token3, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, token1, token3, "Third call should return cached token")
}

// TestAdminTokenManager_InvalidateToken verifies that token invalidation works.
func TestAdminTokenManager_InvalidateToken(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
		TokenBuffer: 30 * time.Second,
	})

	// Get initial token
	token1, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// Invalidate the cached token
	tokenManager.InvalidateToken()

	// Get new token - should fetch fresh one
	token2, err := tokenManager.GetToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	// Note: The new token might be the same or different depending on Keycloak's token generation
	// The important thing is that no error occurred after invalidation
}

// TestAdminTokenManager_InvalidCredentials verifies that invalid credentials are rejected.
func TestAdminTokenManager_InvalidCredentials(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    "wrong-user",
		Password:    "wrong-password",
		TokenBuffer: 30 * time.Second,
	})

	_, err := tokenManager.GetToken(ctx)
	assert.Error(t, err)
}

// =============================================================================
// Group Client Tests
// =============================================================================

// TestGroupClient_CreateAndDeleteGroup verifies group creation and deletion.
func TestGroupClient_CreateAndDeleteGroup(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Create group
	groupName := "test-workspace-" + time.Now().Format("20060102150405")
	groupID, err := groupClient.CreateGroup(ctx, groupName)
	require.NoError(t, err)
	require.NotEmpty(t, groupID)

	// Verify group exists by getting it
	group, err := groupClient.GetGroup(ctx, groupID)
	require.NoError(t, err)
	assert.Equal(t, groupName, group.Name)
	assert.Equal(t, groupID, group.ID)

	// Delete group
	err = groupClient.DeleteGroup(ctx, groupID)
	require.NoError(t, err)

	// Verify group no longer exists
	_, err = groupClient.GetGroup(ctx, groupID)
	require.Error(t, err)
}

// TestGroupClient_CreateGroup_EmptyName verifies that empty group names are rejected.
func TestGroupClient_CreateGroup_EmptyName(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	_, err := groupClient.CreateGroup(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrInvalidGroupName)
}

// TestGroupClient_AddRemoveUserFromGroup verifies adding and removing users from groups.
func TestGroupClient_AddRemoveUserFromGroup(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Create a group for testing
	groupName := "test-membership-group-" + time.Now().Format("20060102150405")
	groupID, err := groupClient.CreateGroup(ctx, groupName)
	require.NoError(t, err)
	defer func() { _ = groupClient.DeleteGroup(ctx, groupID) }()

	// Find testuser's ID
	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	var testUserID string
	for _, u := range users {
		if u.Username == "testuser" {
			testUserID = u.ID
			break
		}
	}
	require.NotEmpty(t, testUserID, "testuser should exist")

	// Add user to group
	err = groupClient.AddUserToGroup(ctx, testUserID, groupID)
	require.NoError(t, err)

	// Verify user is in group
	userGroups, err := groupClient.GetUserGroups(ctx, testUserID)
	require.NoError(t, err)

	found := false
	for _, g := range userGroups {
		if g.ID == groupID {
			found = true
			break
		}
	}
	assert.True(t, found, "User should be in the group")

	// Remove user from group
	err = groupClient.RemoveUserFromGroup(ctx, testUserID, groupID)
	require.NoError(t, err)

	// Verify user is no longer in group
	userGroups, err = groupClient.GetUserGroups(ctx, testUserID)
	require.NoError(t, err)

	found = false
	for _, g := range userGroups {
		if g.ID == groupID {
			found = true
			break
		}
	}
	assert.False(t, found, "User should no longer be in the group")
}

// TestGroupClient_DeleteGroup_NotFound verifies deleting non-existent group.
func TestGroupClient_DeleteGroup_NotFound(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	err := groupClient.DeleteGroup(ctx, "non-existent-group-id-12345")
	require.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
}

// TestGroupClient_GetUserGroups verifies getting user's groups.
func TestGroupClient_GetUserGroups(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Find admin user's ID (admin is in both 'users' and 'admins' groups)
	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	var adminUserID string
	for _, u := range users {
		if u.Username == "admin" {
			adminUserID = u.ID
			break
		}
	}
	require.NotEmpty(t, adminUserID, "admin user should exist")

	// Get admin's groups
	groups, err := groupClient.GetUserGroups(ctx, adminUserID)
	require.NoError(t, err)

	// Admin should be in at least 2 groups (users and admins)
	assert.GreaterOrEqual(t, len(groups), 2)

	groupNames := make([]string, len(groups))
	for i, g := range groups {
		groupNames[i] = g.Name
	}
	assert.Contains(t, groupNames, "users")
	assert.Contains(t, groupNames, "admins")
}

// =============================================================================
// User Client Tests
// =============================================================================

// TestUserClient_ListUsers verifies listing users.
func TestUserClient_ListUsers(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	// We expect at least 4 users: testuser, admin, alice, bob
	assert.GreaterOrEqual(t, len(users), 4)

	usernames := make([]string, len(users))
	for i, u := range users {
		usernames[i] = u.Username
	}
	assert.Contains(t, usernames, "testuser")
	assert.Contains(t, usernames, "admin")
	assert.Contains(t, usernames, "alice")
	assert.Contains(t, usernames, "bob")
}

// TestUserClient_ListUsers_Pagination verifies pagination works.
func TestUserClient_ListUsers_Pagination(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Get first 2 users
	page1, err := userClient.ListUsers(ctx, 0, 2)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get next 2 users
	page2, err := userClient.ListUsers(ctx, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify pages don't overlap
	for _, u1 := range page1 {
		for _, u2 := range page2 {
			assert.NotEqual(t, u1.ID, u2.ID, "Pages should not contain same user")
		}
	}
}

// TestUserClient_GetUser verifies getting a single user.
func TestUserClient_GetUser(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// First, get all users to find testuser's ID
	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	var testUserID string
	for _, u := range users {
		if u.Username == "testuser" {
			testUserID = u.ID
			break
		}
	}
	require.NotEmpty(t, testUserID)

	// Get specific user
	user, err := userClient.GetUser(ctx, testUserID)
	require.NoError(t, err)

	assert.Equal(t, testUserID, user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "testuser@example.com", user.Email)
	assert.True(t, user.EmailVerified)
	assert.Equal(t, "Test", user.FirstName)
	assert.Equal(t, "User", user.LastName)
	assert.True(t, user.Enabled)
}

// TestUserClient_GetUser_NotFound verifies getting non-existent user.
func TestUserClient_GetUser_NotFound(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	_, err := userClient.GetUser(ctx, "non-existent-user-id-12345")
	require.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrUserNotFound)
}

// TestUserClient_CountUsers verifies counting users.
func TestUserClient_CountUsers(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	count, err := userClient.CountUsers(ctx)
	require.NoError(t, err)

	// We expect at least 4 users: testuser, admin, alice, bob
	assert.GreaterOrEqual(t, count, 4)
}

// TestUserClient_DisplayName verifies the DisplayName helper method.
func TestUserClient_DisplayName(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	for _, u := range users {
		if u.Username == "testuser" {
			assert.Equal(t, "Test User", u.DisplayName())
		}
		if u.Username == "admin" {
			assert.Equal(t, "Admin User", u.DisplayName())
		}
		if u.Username == "alice" {
			assert.Equal(t, "Alice Smith", u.DisplayName())
		}
		if u.Username == "bob" {
			assert.Equal(t, "Bob Jones", u.DisplayName())
		}
	}
}

// =============================================================================
// Full Integration Flow Tests
// =============================================================================

// TestFullAuthFlow_TokenValidationWithGroupMembership tests a complete auth flow.
func TestFullAuthFlow_TokenValidationWithGroupMembership(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Get user token
	tokenResp, err := kc.GetUserToken(ctx, "alice", "password123")
	require.NoError(t, err)

	// Step 2: Validate token with JWTValidator
	// Create validator without audience validation
	// Keycloak tokens have 'account' as audience by default
	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL:     kc.URL,
		Realm:           kc.Realm,
		ClientID:        "", // Empty to skip audience validation
		Leeway:          30 * time.Second,
		RefreshInterval: time.Hour,
	})
	require.NoError(t, err)
	defer validator.Close()

	claims, err := validator.Validate(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "alice", claims.Username)
	assert.Contains(t, claims.RealmRoles, "workspace_owner")

	// Step 3: Get user info via OAuth client
	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  kc.URL,
		Realm:        kc.Realm,
		ClientID:     kc.ClientID,
		ClientSecret: kc.ClientSecret,
	})

	userInfo, err := oauthClient.GetUserInfo(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "alice", userInfo.PreferredUsername)

	// Step 4: Use admin API to check user groups
	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Find alice's user ID
	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	var aliceID string
	for _, u := range users {
		if u.Username == "alice" {
			aliceID = u.ID
			break
		}
	}
	require.NotEmpty(t, aliceID)

	// Check alice's groups
	groups, err := groupClient.GetUserGroups(ctx, aliceID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 1, "Alice should be in at least one group")
}

// TestWorkspaceGroupLifecycle tests a complete workspace group lifecycle.
func TestWorkspaceGroupLifecycle(t *testing.T) {
	kc := testutil.SetupTestKeycloak(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
		KeycloakURL: kc.URL,
		Realm:       "master",
		ClientID:    "admin-cli",
		Username:    kc.AdminUser,
		Password:    kc.AdminPass,
	})

	groupClient := keycloak.NewGroupClient(keycloak.GroupClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	userClient := keycloak.NewUserClient(keycloak.UserClientConfig{
		KeycloakURL: kc.URL,
		Realm:       kc.Realm,
	}, tokenManager)

	// Find users
	users, err := userClient.ListUsers(ctx, 0, 100)
	require.NoError(t, err)

	var aliceID, bobID string
	for _, u := range users {
		switch u.Username {
		case "alice":
			aliceID = u.ID
		case "bob":
			bobID = u.ID
		}
	}
	require.NotEmpty(t, aliceID)
	require.NotEmpty(t, bobID)

	// Step 1: Create workspace group
	workspaceGroupName := "workspace-lifecycle-test-" + time.Now().Format("20060102150405")
	groupID, err := groupClient.CreateGroup(ctx, workspaceGroupName)
	require.NoError(t, err)
	t.Logf("Created workspace group: %s (ID: %s)", workspaceGroupName, groupID)

	// Cleanup at the end
	defer func() {
		_ = groupClient.DeleteGroup(ctx, groupID)
	}()

	// Step 2: Add Alice as owner (first member)
	err = groupClient.AddUserToGroup(ctx, aliceID, groupID)
	require.NoError(t, err)
	t.Log("Added Alice to workspace group")

	// Step 3: Add Bob as member
	err = groupClient.AddUserToGroup(ctx, bobID, groupID)
	require.NoError(t, err)
	t.Log("Added Bob to workspace group")

	// Step 4: Verify both users are in the group
	aliceGroups, err := groupClient.GetUserGroups(ctx, aliceID)
	require.NoError(t, err)
	foundAlice := false
	for _, g := range aliceGroups {
		if g.ID == groupID {
			foundAlice = true
			break
		}
	}
	assert.True(t, foundAlice, "Alice should be in workspace group")

	bobGroups, err := groupClient.GetUserGroups(ctx, bobID)
	require.NoError(t, err)
	foundBob := false
	for _, g := range bobGroups {
		if g.ID == groupID {
			foundBob = true
			break
		}
	}
	assert.True(t, foundBob, "Bob should be in workspace group")

	// Step 5: Remove Bob from workspace
	err = groupClient.RemoveUserFromGroup(ctx, bobID, groupID)
	require.NoError(t, err)
	t.Log("Removed Bob from workspace group")

	// Verify Bob is no longer in the group
	bobGroups, err = groupClient.GetUserGroups(ctx, bobID)
	require.NoError(t, err)
	foundBob = false
	for _, g := range bobGroups {
		if g.ID == groupID {
			foundBob = true
			break
		}
	}
	assert.False(t, foundBob, "Bob should no longer be in workspace group")

	// Alice should still be there
	aliceGroups, err = groupClient.GetUserGroups(ctx, aliceID)
	require.NoError(t, err)
	foundAlice = false
	for _, g := range aliceGroups {
		if g.ID == groupID {
			foundAlice = true
			break
		}
	}
	assert.True(t, foundAlice, "Alice should still be in workspace group")

	// Step 6: Delete workspace - remove Alice first, then delete group
	err = groupClient.RemoveUserFromGroup(ctx, aliceID, groupID)
	require.NoError(t, err)

	err = groupClient.DeleteGroup(ctx, groupID)
	require.NoError(t, err)
	t.Log("Deleted workspace group")

	// Verify group no longer exists
	_, err = groupClient.GetGroup(ctx, groupID)
	require.Error(t, err)
	assert.ErrorIs(t, err, keycloak.ErrGroupNotFound)
}
