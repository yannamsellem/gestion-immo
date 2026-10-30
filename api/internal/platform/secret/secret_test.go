package secret_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestGenerateToken_ProducesUnprefixed256BitHex(t *testing.T) {
	tokA, err := secret.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	tokB, err := secret.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if len(tokA) != 64 || strings.Contains(tokA, "_") || tokA != strings.ToLower(tokA) {
		t.Fatalf("expected 64-char unprefixed lowercase hex token, got %q", tokA)
	}
	if decoded, err := hex.DecodeString(tokA); err != nil || len(decoded) != 32 {
		t.Fatalf("expected valid 32-byte hex decode, len=%d err=%v", len(decoded), err)
	}
	if tokA == tokB {
		t.Fatalf("expected distinct tokens, got identical %q", tokA)
	}
}

func TestGeneratePrefixedToken_FormatsPrefixAndValidatesNonEmpty(t *testing.T) {
	tok, err := secret.GeneratePrefixedToken("mcp_live")
	if err != nil {
		t.Fatalf("GeneratePrefixedToken failed: %v", err)
	}
	if !strings.HasPrefix(tok, "mcp_live_") || len(tok) != len("mcp_live_")+64 {
		t.Fatalf("expected mcp_live_<64-hex> (73 chars), got %q (len=%d)", tok, len(tok))
	}
	if _, err := secret.GeneratePrefixedToken("   "); err == nil {
		t.Fatal("expected empty prefix to return an error")
	}
}

func TestSecretString_RedactsOutput(t *testing.T) {
	s := secret.New("raw-secret-value")
	if s.String() != secret.RedactedPlaceholder {
		t.Fatalf("expected %q, got %q", secret.RedactedPlaceholder, s.String())
	}
	if s.Value() != "raw-secret-value" {
		t.Fatalf("expected raw value, got %q", s.Value())
	}
}
