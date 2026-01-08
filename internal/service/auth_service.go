package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/auth"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

// Compile-time assertion that AuthService implements httphandler.AuthService.
var _ httphandler.AuthService = (*AuthService)(nil)

// ErrUserSyncFailed AuthService errors.
var (
	ErrUserSyncFailed = errors.New("failed to sync user from Keycloak")
)

// AuthServiceUserRepository defines the interface for user data access.
// Declared on the consumer side per project guidelines.
type AuthServiceUserRepository interface {
	// FindByExternalID finds a user by their external (Keycloak) ID.
	FindByExternalID(ctx context.Context, externalID string) (*user.User, error)

	// Save saves a user (create or update).
	Save(ctx context.Context, u *user.User) error
}

// AuthServiceOAuthClient defines the interface for OAuth operations.
// Declared on the consumer side per project guidelines.
type AuthServiceOAuthClient interface {
	// ExchangeCode exchanges an authorization code for tokens.
	ExchangeCode(ctx context.Context, code, redirectURI string) (*keycloak.TokenResponse, error)

	// RefreshToken refreshes an access token using a refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*keycloak.TokenResponse, error)

	// RevokeToken revokes a refresh token.
	RevokeToken(ctx context.Context, refreshToken string) error

	// GetUserInfo retrieves user information using an access token.
	GetUserInfo(ctx context.Context, accessToken string) (*keycloak.UserInfo, error)
}

// AuthServiceTokenStore defines the interface for token storage.
// Declared on the consumer side per project guidelines.
type AuthServiceTokenStore interface {
	// StoreRefreshToken stores a refresh token with TTL.
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string, ttl time.Duration) error

	// GetRefreshToken retrieves a stored refresh token.
	GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)

	// DeleteRefreshToken removes a stored refresh token.
	DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error
}

// AuthService implements httphandler.AuthService with Keycloak integration.
type AuthService struct {
	oauthClient AuthServiceOAuthClient
	tokenStore  AuthServiceTokenStore
	userRepo    AuthServiceUserRepository
	logger      *slog.Logger
}

// AuthServiceConfig contains dependencies for AuthService.
type AuthServiceConfig struct {
	OAuthClient AuthServiceOAuthClient
	TokenStore  AuthServiceTokenStore
	UserRepo    AuthServiceUserRepository
	Logger      *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(cfg AuthServiceConfig) *AuthService {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &AuthService{
		oauthClient: cfg.OAuthClient,
		tokenStore:  cfg.TokenStore,
		userRepo:    cfg.UserRepo,
		logger:      logger,
	}
}

// Login performs OAuth2 authorization code flow.
func (s *AuthService) Login(
	ctx echo.Context,
	code, redirectURI string,
) (*httphandler.LoginResult, error) {
	reqCtx := ctx.Request().Context()

	// 1. Exchange authorization code for tokens
	tokens, err := s.oauthClient.ExchangeCode(reqCtx, code, redirectURI)
	if err != nil {
		s.logger.Error("failed to exchange authorization code",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// 2. Get user info from Keycloak
	userInfo, err := s.oauthClient.GetUserInfo(reqCtx, tokens.AccessToken)
	if err != nil {
		s.logger.Error("failed to get user info",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// 3. Find or create user in local DB
	localUser, err := s.findOrCreateUser(reqCtx, userInfo)
	if err != nil {
		s.logger.Error("failed to sync user",
			slog.String("external_id", userInfo.Sub),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%w: %w", ErrUserSyncFailed, err)
	}

	// 4. Store refresh token in Redis
	if tokens.RefreshToken != "" && tokens.RefreshExpiresIn > 0 {
		ttl := time.Duration(tokens.RefreshExpiresIn) * time.Second
		storeErr := s.tokenStore.StoreRefreshToken(
			reqCtx,
			localUser.ID(),
			tokens.RefreshToken,
			ttl,
		)
		if storeErr != nil {
			// Log but don't fail - the user can still use the access token
			s.logger.Warn("failed to store refresh token",
				slog.String("user_id", localUser.ID().String()),
				slog.String("error", storeErr.Error()),
			)
		}
	}

	s.logger.Info("user logged in successfully",
		slog.String("user_id", localUser.ID().String()),
		slog.String("username", localUser.Username()),
	)

	return &httphandler.LoginResult{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		User:         localUser,
	}, nil
}

// Logout invalidates the user's session.
func (s *AuthService) Logout(
	ctx echo.Context,
	userID uuid.UUID,
) error {
	reqCtx := ctx.Request().Context()

	// 1. Get stored refresh token
	refreshToken, err := s.tokenStore.GetRefreshToken(reqCtx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrTokenNotFound) {
			// Token already deleted or never stored - consider logout successful
			s.logger.Debug("refresh token not found during logout",
				slog.String("user_id", userID.String()),
			)
			return nil
		}
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	// 2. Revoke token in Keycloak
	if refreshToken != "" {
		if revokeErr := s.oauthClient.RevokeToken(reqCtx, refreshToken); revokeErr != nil {
			// Log but continue - we still want to delete local token
			s.logger.Warn("failed to revoke token in Keycloak",
				slog.String("user_id", userID.String()),
				slog.String("error", revokeErr.Error()),
			)
		}
	}

	// 3. Delete from Redis
	if deleteErr := s.tokenStore.DeleteRefreshToken(reqCtx, userID); deleteErr != nil {
		return fmt.Errorf("failed to delete refresh token: %w", deleteErr)
	}

	s.logger.Info("user logged out successfully",
		slog.String("user_id", userID.String()),
	)

	return nil
}

// RefreshToken refreshes the access token using a refresh token.
func (s *AuthService) RefreshToken(
	ctx echo.Context,
	refreshToken string,
) (*httphandler.RefreshResult, error) {
	reqCtx := ctx.Request().Context()

	// Refresh tokens in Keycloak
	tokens, err := s.oauthClient.RefreshToken(reqCtx, refreshToken)
	if err != nil {
		s.logger.Error("failed to refresh token",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &httphandler.RefreshResult{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// findOrCreateUser synchronizes a user from Keycloak to the local database.
func (s *AuthService) findOrCreateUser(
	ctx context.Context,
	info *keycloak.UserInfo,
) (*user.User, error) {
	// Try to find existing user by external ID (Keycloak sub)
	existingUser, err := s.userRepo.FindByExternalID(ctx, info.Sub)
	if err == nil && existingUser != nil {
		s.updateExistingUserIfNeeded(ctx, existingUser, info)
		return existingUser, nil
	}

	return s.createNewUser(ctx, info)
}

// updateExistingUserIfNeeded updates user data if it has changed in Keycloak.
func (s *AuthService) updateExistingUserIfNeeded(
	ctx context.Context,
	existingUser *user.User,
	info *keycloak.UserInfo,
) {
	var newEmail, newDisplayName *string

	if existingUser.Email() != info.Email && info.Email != "" {
		newEmail = &info.Email
	}

	if existingUser.DisplayName() != info.Name && info.Name != "" {
		newDisplayName = &info.Name
	}

	if newEmail == nil && newDisplayName == nil {
		return
	}

	if updateErr := existingUser.UpdateProfile(newDisplayName, newEmail); updateErr != nil {
		s.logger.WarnContext(ctx, "failed to update user profile",
			slog.String("user_id", existingUser.ID().String()),
			slog.String("error", updateErr.Error()),
		)
		return
	}

	if saveErr := s.userRepo.Save(ctx, existingUser); saveErr != nil {
		s.logger.WarnContext(ctx, "failed to save updated user",
			slog.String("user_id", existingUser.ID().String()),
			slog.String("error", saveErr.Error()),
		)
	}
}

// createNewUser creates a new user from Keycloak user info.
func (s *AuthService) createNewUser(
	ctx context.Context,
	info *keycloak.UserInfo,
) (*user.User, error) {
	displayName := info.Name
	if displayName == "" {
		displayName = info.PreferredUsername
	}

	newUser, err := user.NewUser(
		info.Sub,               // externalID
		info.PreferredUsername, // username
		info.Email,             // email
		displayName,            // displayName
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if saveErr := s.userRepo.Save(ctx, newUser); saveErr != nil {
		return nil, fmt.Errorf("failed to save new user: %w", saveErr)
	}

	s.logger.InfoContext(ctx, "created new user from Keycloak",
		slog.String("user_id", newUser.ID().String()),
		slog.String("external_id", info.Sub),
		slog.String("username", info.PreferredUsername),
	)

	return newUser, nil
}
