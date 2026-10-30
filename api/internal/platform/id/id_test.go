package id_test

import (
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/id"
)

func TestIsValidUUID_StrictCanonical36Char(t *testing.T) {
	const canonical = "0192dce0-7c4a-7000-8000-000000000001"
	noHyphens := strings.ReplaceAll(canonical, "-", "")
	cases := []struct {
		in   string
		want bool
	}{
		{canonical, true},
		{strings.ToUpper(canonical), true},
		{"", false},
		{"ent_0123456789abcdef0123456789abcdef", false},
		{"urn:uuid:" + canonical, false},
		{"{" + canonical + "}", false},
		{noHyphens, false},
		{" " + canonical, false},
	}
	for _, tc := range cases {
		if got := id.IsValidUUID(tc.in); got != tc.want {
			t.Errorf("IsValidUUID(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
