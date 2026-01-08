package uuid

import (
	"crypto/sha256"

	"github.com/google/uuid"
)

// UUID type alias for UUID
type UUID string

// MustParseUUID парсит строку in UUID or паникует
func MustParseUUID(s string) UUID {
	id, err := ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// NewUUID creates New UUID
func NewUUID() UUID {
	return UUID(uuid.New().String())
}

// ParseUUID парсит строку in UUID
func ParseUUID(s string) (UUID, error) {
	_, err := uuid.Parse(s)
	if err != nil {
		return "", err
	}
	return UUID(s), nil
}

// String returns строковое view
func (u UUID) String() string {
	return string(u)
}

// IsZero checks, is ли UUID нулевым
func (u UUID) IsZero() bool {
	return u == ""
}

// FromGoogleUUID конвертирует google/uuid in domain UUID
func FromGoogleUUID(id uuid.UUID) UUID {
	return UUID(id.String())
}

// ToGoogleUUID конвертирует domain UUID in google/uuid
func (u UUID) ToGoogleUUID() (uuid.UUID, error) {
	return uuid.Parse(string(u))
}

// UUID version and variant constants for RFC 4122.
const (
	uuidVersion4    = 0x40 // Version 4 (random)
	uuidVariantRFC  = 0x80 // RFC 4122 variant
	uuidVersionMask = 0x0f
	uuidVariantMask = 0x3f
)

// DeterministicUUID generates a deterministic UUID from a string seed.
// The same seed will always produce the same UUID.
// Useful for development/testing to ensure consistent IDs.
func DeterministicUUID(seed string) UUID {
	hash := sha256.Sum256([]byte(seed))
	// Use first 16 bytes of hash to create UUID
	var uuidBytes [16]byte
	copy(uuidBytes[:], hash[:16])
	// Set version 4 and variant bits per RFC 4122
	uuidBytes[6] = (uuidBytes[6] & uuidVersionMask) | uuidVersion4
	uuidBytes[8] = (uuidBytes[8] & uuidVariantMask) | uuidVariantRFC
	id, _ := uuid.FromBytes(uuidBytes[:])
	return UUID(id.String())
}
