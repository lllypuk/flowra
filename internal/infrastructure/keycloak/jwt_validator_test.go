package keycloak_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

// testKeyID is the key ID used in tests.
const testKeyID = "test-key-id"

// testKeys holds the RSA key pair for testing.
type testKeys struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// generateTestKeys creates a new RSA key pair for testing.
func generateTestKeys(t *testing.T) *testKeys {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return &testKeys{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}
}

// jwksResponse creates a JWKS response JSON for the test public key.
func jwksResponse(t *testing.T, keys *testKeys) []byte {
	t.Helper()
	n := base64.RawURLEncoding.EncodeToString(keys.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(keys.publicKey.E)).Bytes())

	response := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"kid": testKeyID,
				"n":   n,
				"e":   e,
			},
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)
	return data
}

// setupMockKeycloak creates a mock Keycloak server with JWKS endpoint.
func setupMockKeycloak(t *testing.T, keys *testKeys) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/realms/test-realm/protocol/openid-connect/certs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jwksResponse(t, keys))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// createTestToken creates a signed JWT token for testing.
func createTestToken(t *testing.T, keys *testKeys, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID

	tokenString, err := token.SignedString(keys.privateKey)
	require.NoError(t, err)
	return tokenString
}

// standardClaims returns standard valid claims for testing.
func standardClaims(issuerURL string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":                issuerURL,
		"sub":                "user-123",
		"aud":                "test-client",
		"exp":                now.Add(time.Hour).Unix(),
		"iat":                now.Unix(),
		"email":              "test@example.com",
		"email_verified":     true,
		"preferred_username": "testuser",
		"name":               "Test User",
		"given_name":         "Test",
		"family_name":        "User",
		"session_state":      "session-abc",
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"user", "admin"},
		},
		"groups": []interface{}{"/team-a", "/team-b"},
	}
}

func TestNewJWTValidator(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)

	t.Run("success", func(t *testing.T) {
		validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
			KeycloakURL: server.URL,
			Realm:       "test-realm",
			ClientID:    "test-client",
		})
		require.NoError(t, err)
		require.NotNil(t, validator)
		require.NoError(t, validator.Close())
	})

	t.Run("missing keycloak url", func(t *testing.T) {
		validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
			Realm:    "test-realm",
			ClientID: "test-client",
		})
		require.Error(t, err)
		require.Nil(t, validator)
		assert.ErrorIs(t, err, keycloak.ErrJWKSFetchFailed)
	})

	t.Run("missing realm", func(t *testing.T) {
		validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
			KeycloakURL: server.URL,
			ClientID:    "test-client",
		})
		require.Error(t, err)
		require.Nil(t, validator)
		assert.ErrorIs(t, err, keycloak.ErrJWKSFetchFailed)
	})

	t.Run("invalid jwks url", func(t *testing.T) {
		validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
			KeycloakURL: "http://invalid-host-that-does-not-exist:9999",
			Realm:       "test-realm",
			ClientID:    "test-client",
		})
		require.Error(t, err)
		require.Nil(t, validator)
		assert.ErrorIs(t, err, keycloak.ErrJWKSFetchFailed)
	})
}

func TestJWTValidator_Validate(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)
	issuerURL := server.URL + "/realms/test-realm"

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		ClientID:    "test-client",
		Leeway:      30 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = validator.Close() })

	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		tokenString := createTestToken(t, keys, claims)

		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)

		assert.Equal(t, "user-123", result.UserID)
		assert.Equal(t, "test@example.com", result.Email)
		assert.True(t, result.EmailVerified)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "Test", result.GivenName)
		assert.Equal(t, "User", result.FamilyName)
		assert.Equal(t, "session-abc", result.SessionState)
		assert.ElementsMatch(t, []string{"user", "admin"}, result.RealmRoles)
		assert.ElementsMatch(t, []string{"/team-a", "/team-b"}, result.Groups)
		assert.False(t, result.IssuedAt.IsZero())
		assert.False(t, result.ExpiresAt.IsZero())
	})

	t.Run("empty token", func(t *testing.T) {
		result, validateErr := validator.Validate(ctx, "")
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrInvalidToken)
	})

	t.Run("malformed token", func(t *testing.T) {
		result, validateErr := validator.Validate(ctx, "not-a-valid-jwt")
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrInvalidToken)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["exp"] = time.Now().Add(-time.Hour).Unix() // Expired 1 hour ago

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrTokenExpired)
	})

	t.Run("wrong issuer", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["iss"] = "https://wrong-issuer.com/realms/other"

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrInvalidIssuer)
	})

	t.Run("wrong audience", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["aud"] = "wrong-client"

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrInvalidAudience)
	})

	t.Run("missing subject", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		delete(claims, "sub")

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrMissingSubject)
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create a token with different keys
		otherKeys := generateTestKeys(t)
		claims := standardClaims(issuerURL)
		tokenString := createTestToken(t, otherKeys, claims)

		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrInvalidToken)
	})

	t.Run("token without exp claim", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		delete(claims, "exp")

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
	})

	t.Run("minimal valid claims", func(t *testing.T) {
		now := time.Now()
		claims := jwt.MapClaims{
			"iss": issuerURL,
			"sub": "minimal-user",
			"aud": "test-client",
			"exp": now.Add(time.Hour).Unix(),
			"iat": now.Unix(),
		}

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)

		assert.Equal(t, "minimal-user", result.UserID)
		assert.Empty(t, result.Email)
		assert.Empty(t, result.Username)
		assert.Nil(t, result.RealmRoles)
		assert.Nil(t, result.Groups)
	})
}

func TestJWTValidator_ValidateWithoutAudience(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)
	issuerURL := server.URL + "/realms/test-realm"

	// Create validator without ClientID - should skip audience validation
	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		// No ClientID
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = validator.Close() })

	ctx := context.Background()

	t.Run("accepts any audience when ClientID not configured", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["aud"] = "any-client-should-work"

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
		assert.Equal(t, "user-123", result.UserID)
	})
}

func TestJWTValidator_Leeway(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)
	issuerURL := server.URL + "/realms/test-realm"

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		ClientID:    "test-client",
		Leeway:      1 * time.Minute, // 1 minute leeway
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = validator.Close() })

	ctx := context.Background()

	t.Run("accepts recently expired token within leeway", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["exp"] = time.Now().Add(-30 * time.Second).Unix() // Expired 30 seconds ago

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
	})

	t.Run("rejects token expired beyond leeway", func(t *testing.T) {
		claims := standardClaims(issuerURL)
		claims["exp"] = time.Now().Add(-2 * time.Minute).Unix() // Expired 2 minutes ago

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.Error(t, validateErr)
		require.Nil(t, result)
		assert.ErrorIs(t, validateErr, keycloak.ErrTokenExpired)
	})
}

func TestJWTValidator_ExtractClaims(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)
	issuerURL := server.URL + "/realms/test-realm"

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		ClientID:    "test-client",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = validator.Close() })

	ctx := context.Background()

	t.Run("handles empty realm_access", func(t *testing.T) {
		now := time.Now()
		claims := jwt.MapClaims{
			"iss":          issuerURL,
			"sub":          "user-123",
			"aud":          "test-client",
			"exp":          now.Add(time.Hour).Unix(),
			"iat":          now.Unix(),
			"realm_access": map[string]interface{}{},
		}

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
		assert.Nil(t, result.RealmRoles)
	})

	t.Run("handles realm_access without roles", func(t *testing.T) {
		now := time.Now()
		claims := jwt.MapClaims{
			"iss": issuerURL,
			"sub": "user-123",
			"aud": "test-client",
			"exp": now.Add(time.Hour).Unix(),
			"iat": now.Unix(),
			"realm_access": map[string]interface{}{
				"other_field": "value",
			},
		}

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
		assert.Nil(t, result.RealmRoles)
	})

	t.Run("handles mixed type roles array", func(t *testing.T) {
		now := time.Now()
		claims := jwt.MapClaims{
			"iss": issuerURL,
			"sub": "user-123",
			"aud": "test-client",
			"exp": now.Add(time.Hour).Unix(),
			"iat": now.Unix(),
			"realm_access": map[string]interface{}{
				"roles": []interface{}{"valid-role", 123, "another-role"},
			},
		}

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
		// Should only include string roles
		assert.ElementsMatch(t, []string{"valid-role", "another-role"}, result.RealmRoles)
	})

	t.Run("handles mixed type groups array", func(t *testing.T) {
		now := time.Now()
		claims := jwt.MapClaims{
			"iss":    issuerURL,
			"sub":    "user-123",
			"aud":    "test-client",
			"exp":    now.Add(time.Hour).Unix(),
			"iat":    now.Unix(),
			"groups": []interface{}{"/group-a", 456, "/group-b"},
		}

		tokenString := createTestToken(t, keys, claims)
		result, validateErr := validator.Validate(ctx, tokenString)
		require.NoError(t, validateErr)
		require.NotNil(t, result)
		// Should only include string groups
		assert.ElementsMatch(t, []string{"/group-a", "/group-b"}, result.Groups)
	})
}

func TestJWTValidator_Close(t *testing.T) {
	keys := generateTestKeys(t)
	server := setupMockKeycloak(t, keys)

	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		ClientID:    "test-client",
	})
	require.NoError(t, err)

	// Close should not error
	closeErr := validator.Close()
	require.NoError(t, closeErr)

	// Multiple closes should be safe
	closeErr = validator.Close()
	require.NoError(t, closeErr)
}

func BenchmarkJWTValidator_Validate(b *testing.B) {
	// Generate keys
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	publicKey := &privateKey.PublicKey

	// Create JWKS response
	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes())
	jwksData, _ := json.Marshal(map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"kid": testKeyID,
				"n":   n,
				"e":   e,
			},
		},
	})

	// Setup mock server
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/test-realm/protocol/openid-connect/certs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksData)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	issuerURL := server.URL + "/realms/test-realm"

	// Create validator
	validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
		KeycloakURL: server.URL,
		Realm:       "test-realm",
		ClientID:    "test-client",
	})
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = validator.Close() }()

	// Create token
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                issuerURL,
		"sub":                "user-123",
		"aud":                "test-client",
		"exp":                now.Add(time.Hour).Unix(),
		"iat":                now.Unix(),
		"email":              "test@example.com",
		"email_verified":     true,
		"preferred_username": "testuser",
		"name":               "Test User",
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"user", "admin"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		_, validateErr := validator.Validate(ctx, tokenString)
		if validateErr != nil {
			b.Fatal(validateErr)
		}
	}
}
