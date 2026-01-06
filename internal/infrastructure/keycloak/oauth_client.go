package keycloak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth client errors.
var (
	ErrTokenExchangeFailed = errors.New("failed to exchange authorization code")
	ErrTokenRefreshFailed  = errors.New("failed to refresh token")
	ErrTokenRevokeFailed   = errors.New("failed to revoke token")
	ErrUserInfoFailed      = errors.New("failed to get user info")
	ErrInvalidResponse     = errors.New("invalid response from Keycloak")
)

// TokenResponse represents the OAuth2 token response from Keycloak.
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

// UserInfo represents the user information from Keycloak userinfo endpoint.
type UserInfo struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
}

// OAuthClientConfig contains configuration for OAuthClient.
type OAuthClientConfig struct {
	KeycloakURL  string
	Realm        string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	Logger       *slog.Logger
}

// OAuthClient handles OAuth2 operations with Keycloak.
type OAuthClient struct {
	keycloakURL  string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	logger       *slog.Logger
}

const (
	defaultHTTPTimeout = 30 * time.Second
)

// NewOAuthClient creates a new Keycloak OAuth client.
func NewOAuthClient(cfg OAuthClientConfig) *OAuthClient {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &OAuthClient{
		keycloakURL:  strings.TrimSuffix(cfg.KeycloakURL, "/"),
		realm:        cfg.Realm,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		httpClient:   httpClient,
		logger:       logger,
	}
}

// tokenEndpoint returns the Keycloak token endpoint URL.
func (c *OAuthClient) tokenEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.keycloakURL, c.realm)
}

// userInfoEndpoint returns the Keycloak userinfo endpoint URL.
func (c *OAuthClient) userInfoEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", c.keycloakURL, c.realm)
}

// revokeEndpoint returns the Keycloak token revocation endpoint URL.
func (c *OAuthClient) revokeEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/revoke", c.keycloakURL, c.realm)
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
func (c *OAuthClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenExchangeFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenExchangeFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.ErrorContext(ctx, "token exchange failed",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("%w: status %d", ErrTokenExchangeFailed, resp.StatusCode)
	}

	var tokenResp TokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp); decodeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, decodeErr)
	}

	return &tokenResp, nil
}

// RefreshToken refreshes an access token using a refresh token.
func (c *OAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.ErrorContext(ctx, "token refresh failed",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("%w: status %d", ErrTokenRefreshFailed, resp.StatusCode)
	}

	var tokenResp TokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp); decodeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, decodeErr)
	}

	return &tokenResp, nil
}

// RevokeToken revokes a refresh token in Keycloak.
func (c *OAuthClient) RevokeToken(ctx context.Context, refreshToken string) error {
	data := url.Values{}
	data.Set("token", refreshToken)
	data.Set("token_type_hint", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.revokeEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTokenRevokeFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTokenRevokeFailed, err)
	}
	defer resp.Body.Close()

	// Keycloak returns 200 OK on successful revocation
	// It also returns 200 if the token is already revoked or invalid (idempotent)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.ErrorContext(ctx, "token revocation failed",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return fmt.Errorf("%w: status %d", ErrTokenRevokeFailed, resp.StatusCode)
	}

	return nil
}

// GetUserInfo retrieves user information from Keycloak using an access token.
func (c *OAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoEndpoint(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfoFailed, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserInfoFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.ErrorContext(ctx, "get user info failed",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("%w: status %d", ErrUserInfoFailed, resp.StatusCode)
	}

	var userInfo UserInfo
	if decodeErr := json.NewDecoder(resp.Body).Decode(&userInfo); decodeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, decodeErr)
	}

	return &userInfo, nil
}

// AuthorizationURL generates the Keycloak authorization URL for OAuth2 login.
func (c *OAuthClient) AuthorizationURL(redirectURI, state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", state)

	return fmt.Sprintf(
		"%s/realms/%s/protocol/openid-connect/auth?%s",
		c.keycloakURL,
		c.realm,
		params.Encode(),
	)
}
