package middleware

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

// KeycloakValidatorAdapter adapts keycloak.JWTValidator to middleware.TokenValidator interface.
// It converts Keycloak token claims to middleware TokenClaims format.
type KeycloakValidatorAdapter struct {
	validator  keycloak.JWTValidator
	adminRoles []string // roles that mark system admin (default: ["admin"])
}

// AdapterOption configures KeycloakValidatorAdapter.
type AdapterOption func(*KeycloakValidatorAdapter)

// WithAdminRoles sets the roles that identify system administrators.
func WithAdminRoles(roles ...string) AdapterOption {
	return func(a *KeycloakValidatorAdapter) {
		a.adminRoles = roles
	}
}

// NewKeycloakValidatorAdapter creates a new adapter that bridges keycloak.JWTValidator
// to the middleware.TokenValidator interface.
//
// Usage:
//
//	jwtValidator, _ := keycloak.NewJWTValidator(config)
//	adapter := middleware.NewKeycloakValidatorAdapter(jwtValidator)
//	authConfig := middleware.AuthConfig{
//	    TokenValidator: adapter,
//	}
func NewKeycloakValidatorAdapter(validator keycloak.JWTValidator, opts ...AdapterOption) *KeycloakValidatorAdapter {
	if validator == nil {
		panic("keycloak validator is required")
	}

	adapter := &KeycloakValidatorAdapter{
		validator:  validator,
		adminRoles: []string{"admin", "system-admin"},
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
}

// ValidateToken validates a JWT token and returns middleware.TokenClaims.
// It implements the middleware.TokenValidator interface.
func (a *KeycloakValidatorAdapter) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	keycloakClaims, err := a.validator.Validate(ctx, token)
	if err != nil {
		return nil, a.mapError(err)
	}

	return a.convertClaims(keycloakClaims), nil
}

// convertClaims converts keycloak.TokenClaims to middleware.TokenClaims.
func (a *KeycloakValidatorAdapter) convertClaims(kc *keycloak.TokenClaims) *TokenClaims {
	claims := &TokenClaims{
		// Keycloak UserID (sub claim) becomes ExternalUserID in middleware
		ExternalUserID: kc.UserID,
		Username:       kc.Username,
		Email:          kc.Email,
		Roles:          kc.RealmRoles,
		Groups:         kc.Groups,
		ExpiresAt:      kc.ExpiresAt,
		IsSystemAdmin:  a.isSystemAdmin(kc.RealmRoles),
	}

	return claims
}

// isSystemAdmin checks if any of the user's roles match admin roles.
func (a *KeycloakValidatorAdapter) isSystemAdmin(roles []string) bool {
	roleSet := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}

	for _, adminRole := range a.adminRoles {
		if _, ok := roleSet[adminRole]; ok {
			return true
		}
	}

	return false
}

// mapError maps keycloak errors to middleware errors.
func (a *KeycloakValidatorAdapter) mapError(err error) error {
	switch {
	case errors.Is(err, keycloak.ErrInvalidToken):
		return ErrInvalidToken
	case errors.Is(err, keycloak.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, keycloak.ErrInvalidClaims):
		return ErrInvalidToken
	case errors.Is(err, keycloak.ErrMissingSubject):
		return ErrInvalidToken
	case errors.Is(err, keycloak.ErrInvalidIssuer):
		return ErrInvalidToken
	case errors.Is(err, keycloak.ErrInvalidAudience):
		return ErrInvalidToken
	default:
		// Wrap unknown errors as invalid token
		return errors.Join(ErrInvalidToken, err)
	}
}

// Close closes the underlying keycloak validator.
func (a *KeycloakValidatorAdapter) Close() error {
	return a.validator.Close()
}
