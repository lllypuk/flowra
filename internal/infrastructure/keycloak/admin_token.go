package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AdminTokenConfig contains configuration for AdminTokenManager.
type AdminTokenConfig struct {
	// KeycloakURL is the base URL of Keycloak server (e.g., http://localhost:8090).
	KeycloakURL string

	// Realm is the realm to authenticate against (usually "master" for admin operations).
	Realm string

	// ClientID is the OAuth2 client ID (usually "admin-cli").
	ClientID string

	// ClientSecret is used for client_credentials grant. If empty, password grant is used.
	ClientSecret string

	// Username is the admin username (used with password grant).
	Username string

	// Password is the admin password (used with password grant).
	Password string

	// TokenBuffer is the time before token expiry to trigger refresh (default 30s).
	TokenBuffer time.Duration

	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// AdminTokenManager manages admin API tokens with automatic caching and refresh.
type AdminTokenManager struct {
	config     AdminTokenConfig
	httpClient *http.Client

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

const (
	defaultTokenBuffer      = 30 * time.Second
	defaultAdminHTTPTimeout = 30 * time.Second
)

// NewAdminTokenManager creates a new AdminTokenManager.
func NewAdminTokenManager(config AdminTokenConfig) *AdminTokenManager {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultAdminHTTPTimeout,
		}
	}

	tokenBuffer := config.TokenBuffer
	if tokenBuffer == 0 {
		tokenBuffer = defaultTokenBuffer
	}

	return &AdminTokenManager{
		config: AdminTokenConfig{
			KeycloakURL:  strings.TrimSuffix(config.KeycloakURL, "/"),
			Realm:        config.Realm,
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Username:     config.Username,
			Password:     config.Password,
			TokenBuffer:  tokenBuffer,
		},
		httpClient: httpClient,
	}
}

// GetToken returns a valid admin token, refreshing if needed.
func (m *AdminTokenManager) GetToken(ctx context.Context) (string, error) {
	// Fast path: check if we have a valid cached token
	m.mu.RLock()
	if m.token != "" && time.Now().Add(m.config.TokenBuffer).Before(m.expiresAt) {
		token := m.token
		m.mu.RUnlock()
		return token, nil
	}
	m.mu.RUnlock()

	// Slow path: need to refresh
	return m.refreshToken(ctx)
}

// refreshToken fetches a new token from Keycloak.
func (m *AdminTokenManager) refreshToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if m.token != "" && time.Now().Add(m.config.TokenBuffer).Before(m.expiresAt) {
		return m.token, nil
	}

	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		m.config.KeycloakURL, m.config.Realm)

	data := m.buildTokenRequestData()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("admin token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("admin token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp adminTokenResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp)
	if decodeErr != nil {
		return "", fmt.Errorf("failed to decode token response: %w", decodeErr)
	}

	// Update cached token
	m.token = tokenResp.AccessToken
	m.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return m.token, nil
}

// buildTokenRequestData builds the form data for token request.
func (m *AdminTokenManager) buildTokenRequestData() url.Values {
	data := url.Values{}
	data.Set("client_id", m.config.ClientID)

	if m.config.ClientSecret != "" {
		// Client credentials grant
		data.Set("grant_type", "client_credentials")
		data.Set("client_secret", m.config.ClientSecret)
	} else {
		// Password grant
		data.Set("grant_type", "password")
		data.Set("username", m.config.Username)
		data.Set("password", m.config.Password)
	}

	return data
}

// InvalidateToken clears the cached token, forcing a refresh on next GetToken call.
func (m *AdminTokenManager) InvalidateToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = ""
	m.expiresAt = time.Time{}
}

// adminTokenResponse represents the token response from Keycloak.
type adminTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope,omitempty"`
}
