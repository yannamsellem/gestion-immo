package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Generate returns a cryptographically random identifier formatted as "<prefix>_<32-hex-chars>"
// using 16 random bytes from crypto/rand.
func Generate(prefix string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("id: failed to generate random id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf[:])), nil
}
