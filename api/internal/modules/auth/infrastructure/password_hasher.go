package infrastructure

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// PasswordHasher provides Argon2id password hashing and SHA-256 token hashing.
type PasswordHasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// NewArgon2Hasher constructs a PasswordHasher configured with Argon2id parameters.
func NewArgon2Hasher() *PasswordHasher {
	return &PasswordHasher{
		memory:      64 * 1024,
		iterations:  1,
		parallelism: 4,
		saltLength:  16,
		keyLength:   32,
	}
}

// HashPassword hashes a plaintext password using Argon2id and returns a PHC-formatted string.
func (h *PasswordHasher) HashPassword(plain secret.String) (string, error) {
	if plain.IsEmpty() {
		return "", errors.New("password_hasher: password must not be empty")
	}

	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password_hasher: failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(plain.Value()), salt, h.iterations, h.memory, h.parallelism, h.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.memory,
		h.iterations,
		h.parallelism,
		b64Salt,
		b64Hash,
	)
	return encoded, nil
}

// VerifyPassword verifies a plaintext password against an Argon2id PHC-encoded hash in constant time.
func (h *PasswordHasher) VerifyPassword(plain secret.String, encodedHash string) (bool, error) {
	if plain.IsEmpty() || strings.TrimSpace(encodedHash) == "" {
		return false, errors.New("password_hasher: password and hash must not be empty")
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("password_hasher: invalid argon2id hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("password_hasher: invalid argon2id version: %w", err)
	}
	if version != argon2.Version {
		return false, errors.New("password_hasher: incompatible argon2id version")
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("password_hasher: invalid argon2id parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("password_hasher: invalid salt encoding: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("password_hasher: invalid hash encoding: %w", err)
	}

	computedHash := argon2.IDKey([]byte(plain.Value()), salt, iterations, memory, parallelism, uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(computedHash, expectedHash) == 1 {
		return true, nil
	}
	return false, nil
}

// HashToken computes the deterministic SHA-256 hex digest of a raw refresh or MCP token.
// Raw tokens are never persisted in storage.
func (h *PasswordHasher) HashToken(rawToken secret.String) string {
	sum := sha256.Sum256([]byte(rawToken.Value()))
	return hex.EncodeToString(sum[:])
}
