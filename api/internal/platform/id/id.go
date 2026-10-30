package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// NewV7 generates a canonical lowercase 36-character RFC 9562 UUIDv7 string
// with sub-millisecond monotonic ordering via github.com/google/uuid.
func NewV7() (string, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("id: failed to generate UUIDv7: %w", err)
	}
	return u.String(), nil
}

// MustNewV7 generates a canonical lowercase 36-character RFC 9562 UUIDv7 string
// and panics only if the operating system CSPRNG fails.
func MustNewV7() string {
	s, err := NewV7()
	if err != nil {
		panic(err)
	}
	return s
}

// IsValidUUID reports whether s is a canonical 36-character 8-4-4-4-12 UUID string
// (rejecting non-canonical urn:uuid:..., {...}, or 32-character hyphenless forms).
func IsValidUUID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	return uuid.Validate(s) == nil
}

// Deprecated: Generate is retained temporarily during the UUIDv7 module migration and removed in Task 5.
func Generate(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("id: failed to generate random id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes)), nil
}
