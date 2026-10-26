package domain

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultPrivateKeyFile = "private.pem"
	defaultPublicKeyFile  = "public.pem"
	defaultKIDFile        = "kid"
)

// TokenAuthority is the domain entity managing the asymmetric Ed25519 keypair (PrivateKey, PublicKey)
// and Key Identifier (KID) mounted read-only on the server volume for signing and verifying JWTs.
type TokenAuthority struct {
	kid        string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewTokenAuthority validates and constructs a TokenAuthority from an Ed25519 keypair and KID.
func NewTokenAuthority(kid string, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) (*TokenAuthority, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%w: invalid ed25519 private key size", ErrValidation)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: invalid ed25519 public key size", ErrValidation)
	}
	derivedPub, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok || !bytes.Equal(derivedPub, publicKey) {
		return nil, fmt.Errorf("%w: public key does not match private key", ErrValidation)
	}

	resolvedKID := strings.TrimSpace(kid)
	if resolvedKID == "" {
		resolvedKID = computeKeyID(publicKey)
	}

	privCopy := make(ed25519.PrivateKey, len(privateKey))
	copy(privCopy, privateKey)
	pubCopy := make(ed25519.PublicKey, len(publicKey))
	copy(pubCopy, publicKey)

	return &TokenAuthority{
		kid:        resolvedKID,
		privateKey: privCopy,
		publicKey:  pubCopy,
	}, nil
}

// GenerateTokenAuthority generates an in-memory Ed25519 keypair (for unit tests).
func GenerateTokenAuthority(kid string) (*TokenAuthority, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("auth: generate ed25519 keypair: %w", err)
	}
	return NewTokenAuthority(kid, priv, pub)
}

// LoadTokenAuthorityFromPEM parses PKCS#8 private key PEM and PKIX public key PEM bytes into a TokenAuthority.
func LoadTokenAuthorityFromPEM(kid string, privateKeyPEM []byte, publicKeyPEM []byte) (*TokenAuthority, error) {
	privBlock, _ := pem.Decode(privateKeyPEM)
	if privBlock == nil {
		return nil, fmt.Errorf("%w: invalid PEM block for private key", ErrValidation)
	}
	parsedPriv, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse PKCS#8 private key: %v", ErrValidation, err)
	}
	edPriv, ok := parsedPriv.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: private key is not Ed25519", ErrValidation)
	}

	pubBlock, _ := pem.Decode(publicKeyPEM)
	if pubBlock == nil {
		return nil, fmt.Errorf("%w: invalid PEM block for public key", ErrValidation)
	}
	parsedPub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: parse PKIX public key: %v", ErrValidation, err)
	}
	edPub, ok := parsedPub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: public key is not Ed25519", ErrValidation)
	}

	return NewTokenAuthority(kid, edPriv, edPub)
}

// LoadTokenAuthorityFromDir reads private.pem, public.pem, and optional kid file from a mounted read-only volume directory.
func LoadTokenAuthorityFromDir(keysDir string, kidOverride string) (*TokenAuthority, error) {
	trimmedDir := strings.TrimSpace(keysDir)
	if trimmedDir == "" {
		return nil, fmt.Errorf("%w: keys directory path is required", ErrValidation)
	}

	privBytes, err := os.ReadFile(filepath.Join(trimmedDir, defaultPrivateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("auth: read private key from %s: %w", trimmedDir, err)
	}
	pubBytes, err := os.ReadFile(filepath.Join(trimmedDir, defaultPublicKeyFile))
	if err != nil {
		return nil, fmt.Errorf("auth: read public key from %s: %w", trimmedDir, err)
	}

	kid := strings.TrimSpace(kidOverride)
	if kid == "" {
		if kidBytes, kErr := os.ReadFile(filepath.Join(trimmedDir, defaultKIDFile)); kErr == nil {
			kid = strings.TrimSpace(string(kidBytes))
		}
	}

	return LoadTokenAuthorityFromPEM(kid, privBytes, pubBytes)
}

// EncodePEM marshals the PrivateKey (PKCS#8) and PublicKey (PKIX) into PEM blocks.
func (a *TokenAuthority) EncodePEM() (privateKeyPEM []byte, publicKeyPEM []byte, err error) {
	privDER, err := x509.MarshalPKCS8PrivateKey(a.privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: marshal PKCS#8 private key: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(a.publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: marshal PKIX public key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	})
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})
	return privPEM, pubPEM, nil
}

// KID returns the Key ID associated with this TokenAuthority.
func (a *TokenAuthority) KID() string {
	return a.kid
}

// PrivateKey returns a defensive copy of the Ed25519 private key.
func (a *TokenAuthority) PrivateKey() ed25519.PrivateKey {
	out := make(ed25519.PrivateKey, len(a.privateKey))
	copy(out, a.privateKey)
	return out
}

// PublicKey returns a defensive copy of the Ed25519 public key.
func (a *TokenAuthority) PublicKey() ed25519.PublicKey {
	out := make(ed25519.PublicKey, len(a.publicKey))
	copy(out, a.publicKey)
	return out
}

// LogValue implements slog.LogValuer to guarantee the private key is never logged.
func (a TokenAuthority) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kid", a.kid),
		slog.String("private_key", "[REDACTED]"),
	)
}

func computeKeyID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return "kid_" + hex.EncodeToString(sum[:8])
}

// Claims holds the validated JWT access token claims.
type Claims struct {
	UserID    string   `json:"sub"`
	Subject   string   `json:"-"`
	Email     string   `json:"email"`
	Role      UserRole `json:"role"`
	Issuer    string   `json:"iss"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

