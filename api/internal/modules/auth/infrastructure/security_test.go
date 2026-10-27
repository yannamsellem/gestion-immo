package infrastructure_test

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestAuthSecurity_NeverLogsOrStoresPlaintextSecrets(t *testing.T) {
	hasher := infrastructure.NewArgon2Hasher()
	rawPass := secret.New("MySuperSecretPassword123!")
	passHash, err := hasher.HashPassword(rawPass)
	if err != nil || passHash == rawPass.Value() || strings.Contains(passHash, rawPass.Value()) {
		t.Fatalf("password hash invalid or plaintext: err=%v", err)
	}
	rawRefresh := secret.New("rt_live_9876543210abcdef")
	if hasher.HashToken(rawRefresh) == rawRefresh.Value() {
		t.Fatal("refresh token stored in plaintext")
	}

	authority, err := domain.GenerateTokenAuthority("kid_sec_test")
	if err != nil {
		t.Fatalf("failed to generate TokenAuthority: %v", err)
	}
	privPEM, _, err := authority.EncodePEM()
	if err != nil {
		t.Fatal(err)
	}

	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	jwtTok, err := jwtSvc.GenerateAccessToken("usr_1", "yann@investissimo.fr", domain.RoleSuperAdmin)
	if err != nil {
		t.Fatal(err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	logger.Info("auth event", "password", rawPass, "jwt", jwtTok, "refresh_token", rawRefresh, "authority", *authority)
	for _, s := range []string{rawPass.Value(), jwtTok.Value(), rawRefresh.Value(), string(privPEM)} {
		if strings.Contains(logBuf.String(), s) {
			t.Fatalf("plaintext secret leaked in log output: %s", logBuf.String())
		}
	}
}

func TestJwtService_TokenAuthorityVolumeKeysAndKIDValidation(t *testing.T) {
	keysDir := t.TempDir()

	// Simulate manual upload of private.pem, public.pem, and kid to the mounted server volume
	seedAuth, err := domain.GenerateTokenAuthority("kid_volume_2026")
	if err != nil {
		t.Fatalf("GenerateTokenAuthority failed: %v", err)
	}
	privPEM, pubPEM, err := seedAuth.EncodePEM()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "private.pem"), privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "public.pem"), pubPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "kid"), []byte("kid_volume_2026\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load TokenAuthority from the mounted read-only volume directory
	loadedAuth, err := domain.LoadTokenAuthorityFromDir(keysDir, "")
	if err != nil {
		t.Fatalf("LoadTokenAuthorityFromDir failed: %v", err)
	}
	if loadedAuth.KID() != "kid_volume_2026" {
		t.Fatalf("expected KID 'kid_volume_2026', got %q", loadedAuth.KID())
	}

	jwtSvc, err := infrastructure.NewJwtService(loadedAuth, "gestion-immo", 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tok, err := jwtSvc.GenerateAccessToken("usr_42", "yann@investissimo.fr", domain.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := jwtSvc.ValidateAccessToken(tok.Value())
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}
	if claims.UserID != "usr_42" || claims.Email != "yann@investissimo.fr" || claims.Role != domain.RoleSuperAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Tampered signature must fail
	if _, err := jwtSvc.ValidateAccessToken(tok.Value() + "tamper"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on tampered token, got %v", err)
	}

	// Token signed by a different TokenAuthority (mismatched kid / public key) must fail
	otherAuth, _ := domain.GenerateTokenAuthority("kid_other")
	otherJwtSvc, _ := infrastructure.NewJwtService(otherAuth, "gestion-immo", 15*time.Minute)
	otherTok, _ := otherJwtSvc.GenerateAccessToken("usr_42", "yann@investissimo.fr", domain.RoleSuperAdmin)
	if _, err := jwtSvc.ValidateAccessToken(otherTok.Value()); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on token with mismatched kid/authority, got %v", err)
	}
	// Also verify Smallstep generate-keys layout (only private.pem + jwks.json without public.pem)
	stepDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stepDir, "private.pem"), privPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	jwksJSON := `{"keys":[{"use":"sig","kty":"OKP","kid":"kid_step_jwks_2026","crv":"Ed25519","alg":"EdDSA"}]}`
	if err := os.WriteFile(filepath.Join(stepDir, "jwks.json"), []byte(jwksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	stepAuth, err := domain.LoadTokenAuthorityFromDir(stepDir, "")
	if err != nil {
		t.Fatalf("LoadTokenAuthorityFromDir (Smallstep layout) failed: %v", err)
	}
	if stepAuth.KID() != "kid_step_jwks_2026" {
		t.Fatalf("expected KID 'kid_step_jwks_2026' from jwks.json, got %q", stepAuth.KID())
	}
}

func TestPasswordHasher_EnforcesMaxPasswordBytes(t *testing.T) {
	hasher := infrastructure.NewArgon2Hasher()
	validHash, err := hasher.HashPassword(secret.New(strings.Repeat("x", infrastructure.MaxPasswordBytes)))
	if err != nil {
		t.Fatalf("expected %d-byte password to succeed, got %v", infrastructure.MaxPasswordBytes, err)
	}

	tooLong := secret.New(strings.Repeat("x", infrastructure.MaxPasswordBytes+1))
	if _, err := hasher.HashPassword(tooLong); err == nil {
		t.Fatal("expected HashPassword to reject password exceeding MaxPasswordBytes")
	}
	if ok, err := hasher.VerifyPassword(tooLong, validHash); err == nil || ok {
		t.Fatalf("expected VerifyPassword to reject password exceeding MaxPasswordBytes, got ok=%v err=%v", ok, err)
	}
}
