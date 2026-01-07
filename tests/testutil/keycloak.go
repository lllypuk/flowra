package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Keycloak test configuration constants
const (
	keycloakCtxTimeout                = 30 * time.Second
	keycloakContainerStartupTimeout   = 180 * time.Second
	keycloakContainerTerminateTimeout = 10 * time.Second
	keycloakHealthTimeout             = 5 * time.Second
	keycloakHealthRetryDelay          = 2 * time.Second
	keycloakContainerMemoryLimit      = 512 * 1024 * 1024 // 512MB

	// Default Keycloak configuration
	keycloakAdminUser     = "admin"
	keycloakAdminPassword = "admin123"
	keycloakRealm         = "flowra"
	keycloakClientID      = "flowra-backend"
	keycloakClientSecret  = "flowra-dev-secret-change-in-production" //nolint:gosec // Development secret
)

// sharedKeycloakContainer holds the singleton Keycloak container
var (
	sharedKeycloakContainer   *SharedKeycloakContainer
	sharedKeycloakContainerMu sync.Mutex
)

// SharedKeycloakContainer represents a reusable Keycloak container for tests
type SharedKeycloakContainer struct {
	Container    testcontainers.Container
	URL          string
	AdminUser    string
	AdminPass    string
	Realm        string
	ClientID     string
	ClientSecret string
}

// KeycloakTokenResponse represents the token response from Keycloak
type KeycloakTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

// KeycloakUserInfo represents user info from Keycloak
type KeycloakUserInfo struct {
	Sub               string   `json:"sub"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name,omitempty"`
	FamilyName        string   `json:"family_name,omitempty"`
	Groups            []string `json:"groups,omitempty"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// GetSharedKeycloakContainer returns a singleton Keycloak container.
// The container is started once and reused across all tests.
func GetSharedKeycloakContainer(ctx context.Context) (*SharedKeycloakContainer, error) {
	sharedKeycloakContainerMu.Lock()
	defer sharedKeycloakContainerMu.Unlock()

	needsCreation := sharedKeycloakContainer == nil

	if !needsCreation && needsKeycloakContainerRecreation(ctx) {
		cleanupCrashedKeycloakContainer()
		needsCreation = true
	}

	if needsCreation {
		startupCtx, cancel := context.WithTimeout(context.Background(), keycloakContainerStartupTimeout)
		defer cancel()

		cont, err := startKeycloakContainer(startupCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to start Keycloak container: %w", err)
		}
		sharedKeycloakContainer = cont
	}

	return sharedKeycloakContainer, nil
}

// needsKeycloakContainerRecreation checks if the existing container needs to be recreated
func needsKeycloakContainerRecreation(ctx context.Context) bool {
	if sharedKeycloakContainer == nil || sharedKeycloakContainer.Container == nil {
		return true
	}
	running, err := isKeycloakContainerRunning(ctx, sharedKeycloakContainer.Container)
	return err != nil || !running
}

// cleanupCrashedKeycloakContainer terminates a crashed container
func cleanupCrashedKeycloakContainer() {
	if sharedKeycloakContainer == nil {
		return
	}
	if sharedKeycloakContainer.Container != nil {
		terminateCtx, cancel := context.WithTimeout(context.Background(), keycloakContainerTerminateTimeout)
		_ = sharedKeycloakContainer.Container.Terminate(terminateCtx)
		cancel()
	}
	sharedKeycloakContainer = nil
}

// isKeycloakContainerRunning checks if the container is still running
func isKeycloakContainerRunning(ctx context.Context, cont testcontainers.Container) (bool, error) {
	if cont == nil {
		return false, nil
	}
	state, err := cont.State(ctx)
	if err != nil {
		return false, err
	}
	return state.Running, nil
}

// findRealmExportFile finds the realm export file in the project
func findRealmExportFile() (string, error) {
	// Try relative paths from different locations
	possiblePaths := []string{
		"configs/keycloak/realm-export.json",
		"../configs/keycloak/realm-export.json",
		"../../configs/keycloak/realm-export.json",
		"../../../configs/keycloak/realm-export.json",
	}

	// Try to find from GOPATH or module root
	if wd, err := os.Getwd(); err == nil {
		// Walk up looking for go.mod
		dir := wd
		for range 5 {
			configPath := filepath.Join(dir, "configs", "keycloak", "realm-export.json")
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath, nil
			}
		}
	}

	return "", errors.New("realm-export.json not found")
}

// startKeycloakContainer starts a new Keycloak container with realm import
func startKeycloakContainer(ctx context.Context) (*SharedKeycloakContainer, error) {
	// Find and read realm export file
	realmExportPath, err := findRealmExportFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find realm export file: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "quay.io/keycloak/keycloak:23.0",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"KEYCLOAK_ADMIN":          keycloakAdminUser,
			"KEYCLOAK_ADMIN_PASSWORD": keycloakAdminPassword,
			"KC_DB":                   "dev-file",
			"KC_HEALTH_ENABLED":       "true",
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      realmExportPath,
				ContainerFilePath: "/opt/keycloak/data/import/realm-export.json",
				FileMode:          0o644, //nolint:mnd // Standard file permission
			},
		},
		Cmd: []string{"start-dev", "--import-realm"},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Memory = keycloakContainerMemoryLimit
			hc.MemorySwap = keycloakContainerMemoryLimit
		},
		WaitingFor: wait.ForAll(
			wait.ForHTTP("/health/ready").
				WithPort("8080/tcp").
				WithStartupTimeout(keycloakContainerStartupTimeout),
		),
	}

	cont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Keycloak container: %w", err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := cont.MappedPort(ctx, "8080")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	keycloakURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, port.Port()))

	// Wait for realm to be available
	if err := waitForRealm(ctx, keycloakURL, keycloakRealm); err != nil {
		_ = cont.Terminate(ctx)
		return nil, fmt.Errorf("realm not ready: %w", err)
	}

	return &SharedKeycloakContainer{
		Container:    cont,
		URL:          keycloakURL,
		AdminUser:    keycloakAdminUser,
		AdminPass:    keycloakAdminPassword,
		Realm:        keycloakRealm,
		ClientID:     keycloakClientID,
		ClientSecret: keycloakClientSecret,
	}, nil
}

// waitForRealm waits for the realm to be ready
func waitForRealm(ctx context.Context, keycloakURL, realm string) error {
	realmURL := fmt.Sprintf("%s/realms/%s", keycloakURL, realm)
	client := &http.Client{Timeout: keycloakHealthTimeout}

	maxRetries := 30
	for range maxRetries {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, realmURL, nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(keycloakHealthRetryDelay):
		}
	}
	return fmt.Errorf("realm %s not ready after %d attempts", realm, maxRetries)
}

// GetAdminToken obtains an admin access token
func (c *SharedKeycloakContainer) GetAdminToken(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", c.URL)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", "admin-cli")
	data.Set("username", c.AdminUser)
	data.Set("password", c.AdminPass)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get admin token: %s", string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// GetUserToken obtains an access token for a test user using direct access grants.
func (c *SharedKeycloakContainer) GetUserToken(
	ctx context.Context,
	username, password string,
) (*KeycloakTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.URL, c.Realm)

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", c.ClientID)
	data.Set("client_secret", c.ClientSecret)
	data.Set("username", username)
	data.Set("password", password)
	data.Set("scope", "openid profile email groups")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user token (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// GetUserInfo retrieves user info using an access token
func (c *SharedKeycloakContainer) GetUserInfo(ctx context.Context, accessToken string) (*KeycloakUserInfo, error) {
	userInfoURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", c.URL, c.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo KeycloakUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// RealmExists checks if the realm exists
func (c *SharedKeycloakContainer) RealmExists(ctx context.Context) (bool, error) {
	realmURL := fmt.Sprintf("%s/realms/%s", c.URL, c.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realmURL, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// ClientExists checks if the OAuth2 client exists
func (c *SharedKeycloakContainer) ClientExists(ctx context.Context) (bool, error) {
	adminToken, err := c.GetAdminToken(ctx)
	if err != nil {
		return false, err
	}

	clientsURL := fmt.Sprintf("%s/admin/realms/%s/clients?clientId=%s", c.URL, c.Realm, c.ClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientsURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var clients []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		return false, err
	}

	return len(clients) > 0, nil
}

// GetRealmRoles retrieves all realm roles
func (c *SharedKeycloakContainer) GetRealmRoles(ctx context.Context) ([]string, error) {
	adminToken, err := c.GetAdminToken(ctx)
	if err != nil {
		return nil, err
	}

	rolesURL := fmt.Sprintf("%s/admin/realms/%s/roles", c.URL, c.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rolesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get roles: status %d", resp.StatusCode)
	}

	var roles []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, err
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}
	return roleNames, nil
}

// GetGroups retrieves all groups
func (c *SharedKeycloakContainer) GetGroups(ctx context.Context) ([]string, error) {
	adminToken, err := c.GetAdminToken(ctx)
	if err != nil {
		return nil, err
	}

	groupsURL := fmt.Sprintf("%s/admin/realms/%s/groups", c.URL, c.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, groupsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get groups: status %d", resp.StatusCode)
	}

	var groups []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	groupNames := make([]string, len(groups))
	for i, g := range groups {
		groupNames[i] = g.Name
	}
	return groupNames, nil
}

// GetUsers retrieves all users in the realm
func (c *SharedKeycloakContainer) GetUsers(ctx context.Context) ([]string, error) {
	adminToken, err := c.GetAdminToken(ctx)
	if err != nil {
		return nil, err
	}

	usersURL := fmt.Sprintf("%s/admin/realms/%s/users", c.URL, c.Realm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{Timeout: keycloakHealthTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get users: status %d", resp.StatusCode)
	}

	var users []struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	usernames := make([]string, len(users))
	for i, u := range users {
		usernames[i] = u.Username
	}
	return usernames, nil
}

// SetupTestKeycloak creates a Keycloak container using the shared container.
// This is the recommended way to get a Keycloak container for tests.
func SetupTestKeycloak(t *testing.T) *SharedKeycloakContainer {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), keycloakContainerStartupTimeout)
	defer cancel()

	cont, err := GetSharedKeycloakContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to get shared Keycloak container: %v", err)
	}

	// Verify container is healthy
	if exists, err := cont.RealmExists(ctx); err != nil || !exists {
		t.Fatalf("Keycloak realm not ready: exists=%v, err=%v", exists, err)
	}

	return cont
}

// SetupTestKeycloakIsolated creates a new Keycloak container for complete isolation.
// Use this only when you need a completely clean Keycloak instance.
func SetupTestKeycloakIsolated(t *testing.T) *SharedKeycloakContainer {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), keycloakContainerStartupTimeout)
	defer cancel()

	cont, err := startKeycloakContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start Keycloak container: %v", err)
	}

	t.Cleanup(func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), keycloakContainerTerminateTimeout)
		defer terminateCancel()
		_ = cont.Container.Terminate(terminateCtx)
	})

	return cont
}

// CleanupSharedKeycloakContainer terminates the shared container.
// This is typically called from TestMain or when all tests are done.
func CleanupSharedKeycloakContainer() {
	sharedKeycloakContainerMu.Lock()
	defer sharedKeycloakContainerMu.Unlock()

	if sharedKeycloakContainer != nil {
		if sharedKeycloakContainer.Container != nil {
			ctx, cancel := context.WithTimeout(context.Background(), keycloakContainerTerminateTimeout)
			defer cancel()
			_ = sharedKeycloakContainer.Container.Terminate(ctx)
		}
		sharedKeycloakContainer = nil
	}
}
