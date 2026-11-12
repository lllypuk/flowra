package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
)

// OwnerExternalID is the external identity every owner token resolves to; it survives token rotation.
const OwnerExternalID = "owner"

// ErrNoOwnerTokens means an owner validator was requested without any token to accept.
var ErrNoOwnerTokens = errors.New("no owner tokens configured")

// OwnerToken is a named bearer secret known only by its hex SHA-256 digest.
type OwnerToken struct {
	Name   string
	SHA256 string
}

// OwnerIdentity describes the single user behind all owner tokens.
type OwnerIdentity struct {
	Username string
	Email    string
}

// OwnerTokenValidator accepts configured owner tokens and nothing else.
type OwnerTokenValidator struct {
	tokens []ownerDigest
	owner  OwnerIdentity
	logger *slog.Logger
}

type ownerDigest struct {
	name   string
	digest [sha256.Size]byte
}

// NewOwnerTokenValidator builds a validator; it refuses an empty or malformed token list.
func NewOwnerTokenValidator(
	tokens []OwnerToken,
	owner OwnerIdentity,
	logger *slog.Logger,
) (*OwnerTokenValidator, error) {
	if len(tokens) == 0 {
		return nil, ErrNoOwnerTokens
	}
	if logger == nil {
		logger = slog.Default()
	}

	digests := make([]ownerDigest, 0, len(tokens))
	for _, token := range tokens {
		raw, err := hex.DecodeString(token.SHA256)
		if err != nil || len(raw) != sha256.Size {
			return nil, fmt.Errorf("owner token %q: digest is not a hex SHA-256", token.Name)
		}
		digest := ownerDigest{name: token.Name}
		copy(digest.digest[:], raw)
		digests = append(digests, digest)
	}

	return &OwnerTokenValidator{tokens: digests, owner: owner, logger: logger}, nil
}

// ValidateToken compares the token digest with every configured digest in constant time.
func (v *OwnerTokenValidator) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	presented := sha256.Sum256([]byte(token))
	matched := ""
	for _, known := range v.tokens {
		if subtle.ConstantTimeCompare(presented[:], known.digest[:]) == 1 {
			matched = known.name
		}
	}
	if matched == "" {
		return nil, ErrInvalidToken
	}

	v.logger.DebugContext(ctx, "owner token accepted", slog.String("token_name", matched))

	return &TokenClaims{
		ExternalUserID: OwnerExternalID,
		Username:       v.owner.Username,
		Email:          v.owner.Email,
		Roles:          []string{"user"},
	}, nil
}
