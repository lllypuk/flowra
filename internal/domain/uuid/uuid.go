package uuid

import (
	"github.com/google/uuid"
)

// UUID type alias для UUID
type UUID string

// MustParseUUID парсит строку в UUID или паникует
func MustParseUUID(s string) UUID {
	id, err := ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// NewUUID создает новый UUID
func NewUUID() UUID {
	return UUID(uuid.New().String())
}

// ParseUUID парсит строку в UUID
func ParseUUID(s string) (UUID, error) {
	_, err := uuid.Parse(s)
	if err != nil {
		return "", err
	}
	return UUID(s), nil
}

// String возвращает строковое представление
func (u UUID) String() string {
	return string(u)
}

// IsZero проверяет, является ли UUID нулевым
func (u UUID) IsZero() bool {
	return u == ""
}

// FromGoogleUUID конвертирует google/uuid в domain UUID
func FromGoogleUUID(id uuid.UUID) UUID {
	return UUID(id.String())
}

// ToGoogleUUID конвертирует domain UUID в google/uuid
func (u UUID) ToGoogleUUID() (uuid.UUID, error) {
	return uuid.Parse(string(u))
}
