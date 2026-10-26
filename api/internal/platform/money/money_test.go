// internal/platform/money/money_test.go
package money_test

import (
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

func TestComputeVATCents_ExactIntegerArithmeticAndRounding(t *testing.T) {
	tests := []struct {
		name            string
		ht              money.AmountCents
		rateBasisPoints int64
		wantVAT         money.AmountCents
	}{
		{
			name:            "commercial rent 1912.67 EUR at 20% VAT rounds to 382.53 EUR",
			ht:              money.AmountCents(1912_67),
			rateBasisPoints: 2000,
			wantVAT:         money.AmountCents(382_53),
		},
		{
			name:            "half centime rounds up at 5000 boundary (10.025 -> 10.03)",
			ht:              money.AmountCents(100_25),
			rateBasisPoints: 1000,
			wantVAT:         money.AmountCents(10_03),
		},
		{
			name:            "zero rate returns zero VAT",
			ht:              money.AmountCents(1500_00),
			rateBasisPoints: 0,
			wantVAT:         0,
		},
		{
			name:            "zero HT returns zero VAT",
			ht:              0,
			rateBasisPoints: 2000,
			wantVAT:         0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.ComputeVATCents(tc.ht, tc.rateBasisPoints)
			if got != tc.wantVAT {
				t.Fatalf("ComputeVATCents(%d, %d) = %d, want %d", tc.ht, tc.rateBasisPoints, got, tc.wantVAT)
			}
		})
	}
}

func TestAmountCents_FormatEUR(t *testing.T) {
	if got := money.AmountCents(1600_00).FormatEUR(); got != "1 600,00 €" {
		t.Fatalf("FormatEUR(1600_00) = %q, want %q", got, "1 600,00 €")
	}
	if got := money.AmountCents(-18310_00).FormatEUR(); got != "-18 310,00 €" {
		t.Fatalf("FormatEUR(-18310_00) = %q, want %q", got, "-18 310,00 €")
	}
	if got := money.AmountCents(382_53).FormatEUR(); got != "382,53 €" {
		t.Fatalf("FormatEUR(382_53) = %q, want %q", got, "382,53 €")
	}
}
