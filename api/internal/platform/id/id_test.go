package id_test

import (
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/id"
)

func TestIsValidUUID_StrictCanonical36Char(t *testing.T) {
	sample, err := id.NewV7()
	if err != nil {
		t.Fatalf("NewV7 failed: %v", err)
	}
	mustSample := id.MustNewV7()
	if !id.IsValidUUID(mustSample) {
		t.Fatalf("MustNewV7 produced invalid UUID %q", mustSample)
	}
	noHyphens := strings.ReplaceAll(sample, "-", "")
	cases := []struct {
		in   string
		want bool
	}{
		{sample, true},
		{strings.ToUpper(sample), true},
		{"", false},
		{"ent_0123456789abcdef0123456789abcdef", false},
		{"urn:uuid:" + sample, false},
		{"{" + sample + "}", false},
		{noHyphens, false},
		{" " + sample, false},
	}
	for _, tc := range cases {
		if got := id.IsValidUUID(tc.in); got != tc.want {
			t.Errorf("IsValidUUID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
