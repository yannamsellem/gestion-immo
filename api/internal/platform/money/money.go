package money

import (
	"fmt"
	"strconv"
	"strings"
)

// AmountCents represents a monetary amount in exact integer euro cents (1 EUR = 100 cents).
type AmountCents int64

// ComputeVATCents computes the exact VAT in cents from a net (HT) amount and a VAT rate
// expressed in basis points (e.g., 2000 = 20.00%) using half-up integer rounding:
// (ht * rateBasisPoints + 5000) / 10000.
func ComputeVATCents(ht AmountCents, rateBasisPoints int64) AmountCents {
	if ht == 0 || rateBasisPoints <= 0 {
		return 0
	}
	if ht < 0 {
		return -AmountCents((int64(-ht)*rateBasisPoints + 5000) / 10000)
	}
	return AmountCents((int64(ht)*rateBasisPoints + 5000) / 10000)
}

// Add returns the sum of two AmountCents values.
func (a AmountCents) Add(other AmountCents) AmountCents {
	return a + other
}

// Sub returns the difference between two AmountCents values.
func (a AmountCents) Sub(other AmountCents) AmountCents {
	return a - other
}

// Abs returns the absolute value of AmountCents.
func (a AmountCents) Abs() AmountCents {
	if a < 0 {
		return -a
	}
	return a
}

// IsNegative reports whether the amount is strictly less than zero.
func (a AmountCents) IsNegative() bool {
	return a < 0
}

// FormatEUR formats the amount in French Euro notation (e.g., "1 600,00 €", "-18 310,00 €").
func (a AmountCents) FormatEUR() string {
	sign := ""
	val := int64(a)
	if val < 0 {
		sign = "-"
		val = -val
	}
	euros := val / 100
	cents := val % 100

	euroStr := strconv.FormatInt(euros, 10)
	var grouped strings.Builder
	n := len(euroStr)
	for i, ch := range euroStr {
		if i > 0 && (n-i)%3 == 0 {
			grouped.WriteByte(' ')
		}
		grouped.WriteRune(ch)
	}

	return fmt.Sprintf("%s%s,%02d €", sign, grouped.String(), cents)
}
