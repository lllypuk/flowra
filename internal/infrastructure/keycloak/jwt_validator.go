package keycloak

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JWT validation errors.
var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidClaims   = errors.New("invalid claims")
	ErrMissingSubject  = errors.New("missing subject claim")
	ErrTokenExpired    = errors.New("token expired")
	ErrInvalidIssuer   = errors.New("invalid issuer")
	ErrInvalidAudience = errors.New("invalid audience")
	ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")
)

// TokenClaims represents validated JWT claims from Keycloak.
type TokenClaims struct {
	UserID        string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Username      string   `json:"preferred_username"`
	Name          string   `json:"name"`
	GivenName     string   `json:"given_name"`
	FamilyName    string   `json:"family_name"`
	RealmRoles    []string // extracted from realm_access.roles
	Groups        []string `json:"groups"`
	SessionState  string   `json:"session_state"`
	IssuedAt      time.Time
	ExpiresAt     time.Time
}

// JWTValidator validates Keycloak JWT tokens.
type JWTValidator interface {
	// Validate validates token and returns claims.
	Validate(ctx context.Context, tokenString string) (*TokenClaims, error)

	// Close stops background JWKS refresh.
	Close() error
}

// JWTValidatorConfig contains configuration for JWTValidator.
type JWTValidatorConfig struct {
	KeycloakURL     string
	Realm           string
	ClientID        string        // Expected audience
	Leeway          time.Duration // Clock skew tolerance
	RefreshInterval time.Duration // JWKS refresh interval
	Logger          *slog.Logger
}

// Default configuration values.
const (
	DefaultLeeway          = 30 * time.Second
	DefaultRefreshInterval = 1 * time.Hour
)

// jwtValidator implements JWTValidator using JWKS for offline validation.
type jwtValidator struct {
	jwks      keyfunc.Keyfunc
	config    JWTValidatorConfig
	issuerURL string
	logger    *slog.Logger
	cancel    context.CancelFunc
}

// NewJWTValidator creates a new JWT validator with JWKS caching.
func NewJWTValidator(config JWTValidatorConfig) (JWTValidator, error) {
	if config.KeycloakURL == "" {
		return nil, fmt.Errorf("%w: KeycloakURL is required", ErrJWKSFetchFailed)
	}
	if config.Realm == "" {
		return nil, fmt.Errorf("%w: Realm is required", ErrJWKSFetchFailed)
	}

	// Apply defaults
	if config.Leeway == 0 {
		config.Leeway = DefaultLeeway
	}
	if config.RefreshInterval == 0 {
		config.RefreshInterval = DefaultRefreshInterval
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	issuerURL := fmt.Sprintf("%s/realms/%s", config.KeycloakURL, config.Realm)
	jwksURL := fmt.Sprintf("%s/protocol/openid-connect/certs", issuerURL)

	logger.Info("initializing JWT validator",
		slog.String("jwks_url", jwksURL),
		slog.Duration("refresh_interval", config.RefreshInterval),
	)

	// Create a context that will be used to control the refresh goroutine
	ctx, cancel := context.WithCancel(context.Background())

	// Configure storage with HTTP client and refresh options
	storageOpts := jwkset.HTTPClientStorageOptions{
		Ctx:             ctx,
		RefreshInterval: config.RefreshInterval,
		RefreshErrorHandler: func(_ context.Context, err error) {
			logger.Error("failed to refresh JWKS", slog.Any("error", err))
		},
	}

	storage, err := jwkset.NewStorageFromHTTP(jwksURL, storageOpts)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %w", ErrJWKSFetchFailed, err)
	}

	// Create keyfunc with the storage
	jwks, err := keyfunc.New(keyfunc.Options{
		Ctx:     ctx,
		Storage: storage,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %w", ErrJWKSFetchFailed, err)
	}

	return &jwtValidator{
		jwks:      jwks,
		config:    config,
		issuerURL: issuerURL,
		logger:    logger,
		cancel:    cancel,
	}, nil
}

// Validate validates token and returns claims.
func (v *jwtValidator) Validate(_ context.Context, tokenString string) (*TokenClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Build parser options
	parserOpts := []jwt.ParserOption{
		jwt.WithLeeway(v.config.Leeway),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(v.issuerURL),
	}

	// Add audience validation if ClientID is configured
	if v.config.ClientID != "" {
		parserOpts = append(parserOpts, jwt.WithAudience(v.config.ClientID))
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, v.jwks.Keyfunc, parserOpts...)
	if err != nil {
		// Wrap specific errors for better error handling
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %w", ErrTokenExpired, err)
		}
		if errors.Is(err, jwt.ErrTokenMalformed) || errors.Is(err, jwt.ErrTokenUnverifiable) ||
			errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
		}
		if errors.Is(err, jwt.ErrTokenInvalidIssuer) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidIssuer, err)
		}
		if errors.Is(err, jwt.ErrTokenInvalidAudience) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidAudience, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return v.extractClaims(claims)
}

// extractClaims extracts TokenClaims from raw JWT claims.
func (v *jwtValidator) extractClaims(claims jwt.MapClaims) (*TokenClaims, error) {
	tc := &TokenClaims{}

	// Required claims
	tc.UserID, _ = claims["sub"].(string)
	if tc.UserID == "" {
		return nil, ErrMissingSubject
	}

	// Optional string claims
	tc.Email, _ = claims["email"].(string)
	tc.EmailVerified, _ = claims["email_verified"].(bool)
	tc.Username, _ = claims["preferred_username"].(string)
	tc.Name, _ = claims["name"].(string)
	tc.GivenName, _ = claims["given_name"].(string)
	tc.FamilyName, _ = claims["family_name"].(string)
	tc.SessionState, _ = claims["session_state"].(string)

	// Extract realm roles from realm_access.roles
	if realmAccess, realmOK := claims["realm_access"].(map[string]any); realmOK {
		if roles, rolesOK := realmAccess["roles"].([]any); rolesOK {
			tc.RealmRoles = make([]string, 0, len(roles))
			for _, role := range roles {
				if r, roleOK := role.(string); roleOK {
					tc.RealmRoles = append(tc.RealmRoles, r)
				}
			}
		}
	}

	// Extract groups
	if groups, groupsOK := claims["groups"].([]any); groupsOK {
		tc.Groups = make([]string, 0, len(groups))
		for _, group := range groups {
			if g, groupOK := group.(string); groupOK {
				tc.Groups = append(tc.Groups, g)
			}
		}
	}

	// Time claims
	if iat, ok := claims["iat"].(float64); ok {
		tc.IssuedAt = time.Unix(int64(iat), 0)
	}
	if exp, ok := claims["exp"].(float64); ok {
		tc.ExpiresAt = time.Unix(int64(exp), 0)
	}

	return tc, nil
}

// Close stops background JWKS refresh.
func (v *jwtValidator) Close() error {
	v.logger.Info("closing JWT validator")
	if v.cancel != nil {
		v.cancel()
	}
	return nil
}
