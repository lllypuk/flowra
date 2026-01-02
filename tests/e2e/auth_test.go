//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_Login_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create a test user that can login
	testUser := suite.CreateTestUser("loginuser")

	// Create HTTP client (no auth for login)
	client := suite.NewHTTPClient("")

	// Login with valid code
	resp := client.Post("/auth/login", map[string]string{
		"code":         "code-loginuser",
		"redirect_uri": "http://localhost/callback",
	})

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			User         struct {
				ID       string `json:"id"`
				Email    string `json:"email"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			User         struct {
				ID       string `json:"id"`
				Email    string `json:"email"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.AccessToken)
	assert.NotEmpty(t, result.Data.RefreshToken)
	assert.Equal(t, 3600, result.Data.ExpiresIn)
	assert.Equal(t, testUser.Email, result.Data.User.Email)
	assert.Equal(t, testUser.Username, result.Data.User.Username)
}

func TestAuth_Login_InvalidCode(t *testing.T) {
	suite := NewE2ETestSuite(t)

	client := suite.NewHTTPClient("")

	// Login with invalid code
	resp := client.Post("/auth/login", map[string]string{
		"code":         "invalid-code",
		"redirect_uri": "http://localhost/callback",
	})

	AssertStatus(t, resp, http.StatusUnauthorized)

	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}](t, resp)

	assert.False(t, result.Success)
	assert.Equal(t, "INVALID_CREDENTIALS", result.Error.Code)
}

func TestAuth_Login_MissingCode(t *testing.T) {
	suite := NewE2ETestSuite(t)

	client := suite.NewHTTPClient("")

	// Login without code
	resp := client.Post("/auth/login", map[string]string{
		"redirect_uri": "http://localhost/callback",
	})

	AssertStatus(t, resp, http.StatusBadRequest)

	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}](t, resp)

	assert.False(t, result.Success)
	assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
}

func TestAuth_Login_MissingRedirectURI(t *testing.T) {
	suite := NewE2ETestSuite(t)

	client := suite.NewHTTPClient("")

	// Login without redirect_uri
	resp := client.Post("/auth/login", map[string]string{
		"code": "some-code",
	})

	AssertStatus(t, resp, http.StatusBadRequest)

	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}](t, resp)

	assert.False(t, result.Success)
	assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
}

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

func TestAuth_Logout_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create and authenticate user
	testUser := suite.CreateTestUser("logoutuser")
	client := suite.NewHTTPClient(testUser.Token)

	// Logout
	resp := client.Post("/auth/logout", nil)

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.Contains(t, result.Data.Message, "Logged out")
}

func TestAuth_Logout_Unauthorized(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Logout without token
	client := suite.NewHTTPClient("")

	resp := client.Post("/auth/logout", nil)

	AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_Refresh_Success(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user
	testUser := suite.CreateTestUser("refreshuser")
	client := suite.NewHTTPClient(testUser.Token)

	// Refresh token
	resp := client.Post("/auth/refresh", map[string]string{
		"refresh_token": "mock-refresh-token-" + testUser.ID.String(),
	})

	AssertStatus(t, resp, http.StatusOK)

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}](t, resp)

	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.AccessToken)
	assert.NotEmpty(t, result.Data.RefreshToken)
	assert.Equal(t, 3600, result.Data.ExpiresIn)
}

func TestAuth_Refresh_MissingToken(t *testing.T) {
	suite := NewE2ETestSuite(t)

	// Create user
	testUser := suite.CreateTestUser("refreshuser2")
	client := suite.NewHTTPClient(testUser.Token)

	// Refresh without token
	resp := client.Post("/auth/refresh", map[string]string{})

	AssertStatus(t, resp, http.StatusBadRequest)

	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	result = ParseResponse[struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}](t, resp)

	assert.False(t, result.Success)
	assert.Equal(t, "VALIDATION_ERROR", result.Error.Code)
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
