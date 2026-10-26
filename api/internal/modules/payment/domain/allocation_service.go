package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

var (
	isoMonthPattern   = regexp.MustCompile(`\b(20\d{2})-(0[1-9]|1[0-2])\b`)
	slashMonthPattern = regexp.MustCompile(`\b(0?[1-9]|1[0-2])/(20\d{2})\b`)
	yearOnlyPattern   = regexp.MustCompile(`\b(20\d{2})\b`)
	wordSplitPattern  = regexp.MustCompile(`[^a-z0-9]+`)
)

var frenchMonthNumbers = map[string]int{
	"janvier":   1,
	"fevrier":   2,
	"mars":      3,
	"avril":     4,
	"mai":       5,
	"juin":      6,
	"juillet":   7,
	"aout":      8,
	"septembre": 9,
	"octobre":   10,
	"novembre":  11,
	"decembre":  12,
}

// AllocationService implements explicit-month detection and FIFO fallback rent payment allocation.
type AllocationService struct{}

// NewAllocationService constructs a stateless AllocationService.
func NewAllocationService() *AllocationService {
	return &AllocationService{}
}

// DetectTargetMonth inspects a payment transfer label for an explicit target period ("YYYY-MM").
// It supports ISO "YYYY-MM", French "MM/YYYY", and French month names ("janvier".."décembre",
// case-insensitive and accent-insensitive). When a month name is present without an explicit
// 4-digit year, fallbackYear is used.
func (s *AllocationService) DetectTargetMonth(label string, fallbackYear int) (string, bool) {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return "", false
	}

	if m := isoMonthPattern.FindStringSubmatch(trimmed); len(m) == 3 {
		return m[1] + "-" + m[2], true
	}

	if m := slashMonthPattern.FindStringSubmatch(trimmed); len(m) == 3 {
		monthNum, err := strconv.Atoi(m[1])
		if err == nil && monthNum >= 1 && monthNum <= 12 {
			return fmt.Sprintf("%s-%02d", m[2], monthNum), true
		}
	}

	normalized := normalizeFrenchLabel(trimmed)
	tokens := wordSplitPattern.Split(normalized, -1)

	detectedMonth := 0
	for _, tok := range tokens {
		if m, ok := frenchMonthNumbers[tok]; ok {
			detectedMonth = m
			break
		}
	}
	if detectedMonth == 0 {
		return "", false
	}

	year := fallbackYear
	if ym := yearOnlyPattern.FindStringSubmatch(normalized); len(ym) == 2 {
		if parsedYear, err := strconv.Atoi(ym[1]); err == nil {
			year = parsedYear
		}
	}

	return fmt.Sprintf("%04d-%02d", year, detectedMonth), true
}

// AllocatePayment allocates amountCents across unpaidCalls.
// Rule:
//  1. If the payment label explicitly designates an unpaid period month (e.g., "Virement loyer mars"),
//     that target month is allocated first.
//  2. Any remaining payment amount (or the full amount if no explicit month matched) is allocated
//     in chronological FIFO order (oldest PeriodMonth first).
//
// The returned slice of RentCall preserves the input slice ordering without mutating the caller's slice.
func (s *AllocationService) AllocatePayment(
	unpaidCalls []leasedomain.RentCall,
	amountCents money.AmountCents,
	label string,
	paymentDate time.Time,
) ([]RentAllocation, []leasedomain.RentCall) {
	updated := make([]leasedomain.RentCall, len(unpaidCalls))
	copy(updated, unpaidCalls)

	if amountCents <= 0 || len(updated) == 0 {
		return nil, updated
	}

	order := make([]int, len(updated))
	for i := range updated {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a := updated[order[i]]
		b := updated[order[j]]
		if a.PeriodMonth != b.PeriodMonth {
			return a.PeriodMonth < b.PeriodMonth
		}
		return a.DueDate.Before(b.DueDate)
	})

	remaining := amountCents
	var allocs []RentAllocation

	applyToIndex := func(idx int) {
		if remaining <= 0 || updated[idx].RemainingDueCents <= 0 {
			return
		}
		applied := remaining
		if applied > updated[idx].RemainingDueCents {
			applied = updated[idx].RemainingDueCents
		}
		updated[idx].RemainingDueCents = updated[idx].RemainingDueCents.Sub(applied)
		remaining = remaining.Sub(applied)

		if updated[idx].RemainingDueCents == 0 {
			updated[idx].Status = leasedomain.StatusPaid
		} else {
			updated[idx].Status = leasedomain.StatusPartial
		}
		updated[idx].UpdatedAt = paymentDate

		allocs = append(allocs, RentAllocation{
			RentCallID:   updated[idx].ID,
			PeriodMonth:  updated[idx].PeriodMonth,
			AppliedCents: applied,
			CreatedAt:    paymentDate,
		})
	}

	fallbackYear := paymentDate.Year()
	if fallbackYear <= 0 {
		fallbackYear = time.Now().UTC().Year()
	}

	if targetMonth, ok := s.DetectTargetMonth(label, fallbackYear); ok {
		for _, idx := range order {
			if updated[idx].PeriodMonth == targetMonth && updated[idx].RemainingDueCents > 0 {
				applyToIndex(idx)
			}
		}
	}

	for _, idx := range order {
		if remaining <= 0 {
			break
		}
		if updated[idx].RemainingDueCents > 0 {
			applyToIndex(idx)
		}
	}

	return allocs, updated
}

func normalizeFrenchLabel(s string) string {
	lower := strings.ToLower(s)
	replacer := strings.NewReplacer(
		"é", "e",
		"è", "e",
		"ê", "e",
		"ë", "e",
		"à", "a",
		"â", "a",
		"ä", "a",
		"û", "u",
		"ù", "u",
		"ü", "u",
		"î", "i",
		"ï", "i",
		"ô", "o",
		"ö", "o",
		"ç", "c",
	)
	return replacer.Replace(lower)
}
