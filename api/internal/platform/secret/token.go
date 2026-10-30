package secret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateToken generates an unprefixed 256-bit (32 random bytes) cryptographic token
// encoded as a 64-character lowercase hexadecimal string.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("secret: failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GeneratePrefixedToken generates a 256-bit cryptographic token formatted as "<prefix>_<64-hex-chars>".
func GeneratePrefixedToken(prefix string) (string, error) {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "", fmt.Errorf("secret: token prefix must not be empty")
	}
	raw, err := GenerateToken()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", trimmed, raw), nil
}
