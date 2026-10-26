package id_test

import (
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/id"
)

func TestGenerate_PrefixFormattingAndUniqueness(t *testing.T) {
	const count = 1000
	seen := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		got, err := id.Generate("ent")
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if !strings.HasPrefix(got, "ent_") {
			t.Fatalf("expected prefix 'ent_', got %q", got)
		}
		// "ent_" (4 chars) + 32 hex chars (16 random bytes) = 36 chars
		if len(got) != 4+32 {
			t.Fatalf("expected length %d, got %d (%q)", 4+32, len(got), got)
		}
		if _, exists := seen[got]; exists {
			t.Fatalf("duplicate ID generated on iteration %d: %q", i, got)
		}
		seen[got] = struct{}{}
	}
}
