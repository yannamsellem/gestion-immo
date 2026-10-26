package domain

import "github.com/yannamsellem/gestion-immo/internal/platform/secret"

// MaxPasswordBytes is the maximum allowed plaintext password length in bytes.
const MaxPasswordBytes = 128

// AccessTokenManager defines the contract for issuing and validating JWT access tokens.
type AccessTokenManager interface {
	GenerateAccessToken(userID string, email string, role UserRole) (secret.String, error)
	ValidateAccessToken(rawToken string) (*Claims, error)
}

// SecretHasher defines the contract for hashing and verifying passwords and opaque tokens.
type SecretHasher interface {
	HashPassword(plain secret.String) (string, error)
	VerifyPassword(plain secret.String, encodedHash string) (bool, error)
	HashToken(rawToken secret.String) string
}
