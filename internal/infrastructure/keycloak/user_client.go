package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UserClientConfig contains configuration for UserClient.
type UserClientConfig struct {
	// KeycloakURL is the base URL of Keycloak server.
	KeycloakURL string

	// Realm is the realm where users are managed.
	Realm string

	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// UserClient handles user management operations with Keycloak Admin API.
type UserClient struct {
	config       UserClientConfig
	tokenManager *AdminTokenManager
	httpClient   *http.Client
}

const defaultUserHTTPTimeout = 60 * time.Second

// User represents a user from Keycloak Admin API.
type User struct {
	ID               string              `json:"id"`
	Username         string              `json:"username"`
	Email            string              `json:"email"`
	EmailVerified    bool                `json:"emailVerified"`
	FirstName        string              `json:"firstName"`
	LastName         string              `json:"lastName"`
	Enabled          bool                `json:"enabled"`
	CreatedTimestamp int64               `json:"createdTimestamp"`
	Attributes       map[string][]string `json:"attributes,omitempty"`
}

// NewUserClient creates a new Keycloak user management client.
func NewUserClient(config UserClientConfig, tokenManager *AdminTokenManager) *UserClient {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultUserHTTPTimeout,
		}
	}

	return &UserClient{
		config: UserClientConfig{
			KeycloakURL: strings.TrimSuffix(config.KeycloakURL, "/"),
			Realm:       config.Realm,
		},
		tokenManager: tokenManager,
		httpClient:   httpClient,
	}
}

// ListUsers returns users from Keycloak with pagination.
// first: starting index (0-based)
// limit: maximum number of users to return
func (c *UserClient) ListUsers(ctx context.Context, first, limit int) ([]User, error) {
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	reqURL := fmt.Sprintf("%s/admin/realms/%s/users?first=%d&max=%d",
		c.config.KeycloakURL, c.config.Realm, first, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list users request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list users failed with status %d: %s", resp.StatusCode, string(body))
	}

	var users []User
	decodeErr := json.NewDecoder(resp.Body).Decode(&users)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode users response: %w", decodeErr)
	}

	return users, nil
}

// GetUser returns a single user by ID from Keycloak.
//
//nolint:dupl // Similar structure to GroupClient.GetGroup but different types and endpoints
func (c *UserClient) GetUser(ctx context.Context, userID string) (*User, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	reqURL := fmt.Sprintf("%s/admin/realms/%s/users/%s",
		c.config.KeycloakURL, c.config.Realm, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var user User
		decodeErr := json.NewDecoder(resp.Body).Decode(&user)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode user response: %w", decodeErr)
		}
		return &user, nil
	case http.StatusNotFound:
		return nil, ErrUserNotFound
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user failed with status %d: %s", resp.StatusCode, string(body))
	}
}

// CountUsers returns the total number of users in the realm.
func (c *UserClient) CountUsers(ctx context.Context) (int, error) {
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get admin token: %w", err)
	}

	reqURL := fmt.Sprintf("%s/admin/realms/%s/users/count",
		c.config.KeycloakURL, c.config.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("count users request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("count users failed with status %d: %s", resp.StatusCode, string(body))
	}

	var count int
	decodeErr := json.NewDecoder(resp.Body).Decode(&count)
	if decodeErr != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", decodeErr)
	}

	return count, nil
}

// DisplayName returns the full display name for a Keycloak user.
// It concatenates FirstName and LastName, trimming any extra spaces.
func (u *User) DisplayName() string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return u.Username
	}
	return name
}
