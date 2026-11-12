//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_Me_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create and authenticate user
	testUser := suite.CreateTestUser("meuser")
	client := suite.NewHTTPClient(testUser.Token)

	// Get current user info
	resp := client.Get("/auth/me")

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Equal(t, testUser.ID.String(), result.Data.ID)
	assert.Equal(t, testUser.Email, result.Data.Email)
	assert.Equal(t, testUser.Username, result.Data.Username)
}

func TestAuth_Me_Unauthorized(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Request without token
	client := suite.NewHTTPClient("")

	resp := client.Get("/auth/me")

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_Me_InvalidToken(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Request with invalid token
	client := suite.NewHTTPClient("invalid-token")

	resp := client.Get("/auth/me")

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_HealthEndpoints(t *testing.T) {
	suite := NewE2ETestSuite(t)

	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(suite.BaseURL() + "/health")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		AssertStatus(t, resp, http.StatusOK)

		var result map[string]string
		result = ParseResponse[map[string]string](t, resp)
		assert.Equal(t, "healthy", result["status"])
	})

	t.Run("ready endpoint", func(t *testing.T) {
		resp, err := http.Get(suite.BaseURL() + "/ready")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		AssertStatus(t, resp, http.StatusOK)

		var result map[string]string
		result = ParseResponse[map[string]string](t, resp)
		assert.Equal(t, "ready", result["status"])
	})
}
