package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Group client errors.
var (
	ErrGroupNotFound    = errors.New("group not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrGroupExists      = errors.New("group already exists")
	ErrInvalidGroupName = errors.New("invalid group name")
)

// GroupClientConfig contains configuration for GroupClient.
type GroupClientConfig struct {
	// KeycloakURL is the base URL of Keycloak server.
	KeycloakURL string

	// Realm is the realm where groups are managed.
	Realm string

	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// GroupClient handles group management operations with Keycloak Admin API.
type GroupClient struct {
	config       GroupClientConfig
	tokenManager *AdminTokenManager
	httpClient   *http.Client
}

const defaultGroupHTTPTimeout = 30 * time.Second

// NewGroupClient creates a new Keycloak group management client.
func NewGroupClient(config GroupClientConfig, tokenManager *AdminTokenManager) *GroupClient {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultGroupHTTPTimeout,
		}
	}

	return &GroupClient{
		config: GroupClientConfig{
			KeycloakURL: strings.TrimSuffix(config.KeycloakURL, "/"),
			Realm:       config.Realm,
		},
		tokenManager: tokenManager,
		httpClient:   httpClient,
	}
}

// CreateGroup creates a new group in Keycloak and returns its ID.
func (c *GroupClient) CreateGroup(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", ErrInvalidGroupName
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get admin token: %w", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/groups", c.config.KeycloakURL, c.config.Realm)

	body := map[string]string{"name": name}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create group request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		// Extract group ID from Location header
		// Location: http://localhost:8090/admin/realms/flowra/groups/abc-123-...
		location := resp.Header.Get("Location")
		if location == "" {
			return "", errors.New("missing Location header in response")
		}
		parts := strings.Split(location, "/")
		if len(parts) == 0 {
			return "", fmt.Errorf("invalid Location header: %s", location)
		}
		groupID := parts[len(parts)-1]
		return groupID, nil

	case http.StatusConflict:
		return "", ErrGroupExists

	default:
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create group failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

// DeleteGroup deletes a group from Keycloak.
func (c *GroupClient) DeleteGroup(ctx context.Context, groupID string) error {
	if groupID == "" {
		return ErrGroupNotFound
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/groups/%s",
		c.config.KeycloakURL, c.config.Realm, groupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete group request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrGroupNotFound
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete group failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

// AddUserToGroup adds a user to a group in Keycloak.
func (c *GroupClient) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	return c.modifyUserGroupMembership(ctx, userID, groupID, http.MethodPut, "add user to group")
}

// RemoveUserFromGroup removes a user from a group in Keycloak.
func (c *GroupClient) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	return c.modifyUserGroupMembership(ctx, userID, groupID, http.MethodDelete, "remove user from group")
}

// modifyUserGroupMembership is a helper that handles both add and remove user from group operations.
func (c *GroupClient) modifyUserGroupMembership(
	ctx context.Context,
	userID, groupID string,
	method string,
	operation string,
) error {
	if userID == "" {
		return ErrUserNotFound
	}
	if groupID == "" {
		return ErrGroupNotFound
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/users/%s/groups/%s",
		c.config.KeycloakURL, c.config.Realm, userID, groupID)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", operation, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		// Could be user or group not found
		respBody, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(respBody), "User") {
			return ErrUserNotFound
		}
		return ErrGroupNotFound
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s failed with status %d: %s", operation, resp.StatusCode, string(respBody))
	}
}

// GetGroup retrieves a group by ID from Keycloak.
func (c *GroupClient) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	if groupID == "" {
		return nil, ErrGroupNotFound
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/groups/%s",
		c.config.KeycloakURL, c.config.Realm, groupID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get group request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var group Group
		decodeErr := json.NewDecoder(resp.Body).Decode(&group)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode group response: %w", decodeErr)
		}
		return &group, nil
	case http.StatusNotFound:
		return nil, ErrGroupNotFound
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get group failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

// GetUserGroups retrieves all groups that a user belongs to.
func (c *GroupClient) GetUserGroups(ctx context.Context, userID string) ([]Group, error) {
	if userID == "" {
		return nil, ErrUserNotFound
	}

	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	url := fmt.Sprintf("%s/admin/realms/%s/users/%s/groups",
		c.config.KeycloakURL, c.config.Realm, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user groups request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var groups []Group
		decodeErr := json.NewDecoder(resp.Body).Decode(&groups)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode groups response: %w", decodeErr)
		}
		return groups, nil
	case http.StatusNotFound:
		return nil, ErrUserNotFound
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user groups failed with status %d: %s", resp.StatusCode, string(respBody))
	}
}

// Group represents a Keycloak group.
type Group struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Path       string            `json:"path"`
	SubGroups  []Group           `json:"subGroups,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}
