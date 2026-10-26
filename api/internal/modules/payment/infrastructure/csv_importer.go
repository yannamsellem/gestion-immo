package infrastructure

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// BankTransactionRow aliases domain.BankTransactionRow for convenience in infrastructure callers.
type BankTransactionRow = domain.BankTransactionRow

// CSVImporter parses bank statement exports in CSV (comma or semicolon separated) or OFX format.
type CSVImporter struct{}

// NewCSVImporter constructs a new CSVImporter.
func NewCSVImporter() *CSVImporter {
	return &CSVImporter{}
}

var (
	ofxTrnBlockPattern = regexp.MustCompile(`(?is)<STMTTRN>(.*?)</STMTTRN>`)
	ofxTagPattern      = regexp.MustCompile(`(?is)<([A-Z0-9_]+)>([^<\r\n]+)`)
)

// ParseTransactions parses bank transactions from a CSV or OFX reader.
func (imp *CSVImporter) ParseTransactions(r io.Reader) ([]domain.BankTransactionRow, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: empty reader", domain.ErrValidation)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("csv_importer: read input: %w", err)
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}

	if strings.Contains(strings.ToUpper(trimmed), "<OFX") || strings.Contains(strings.ToUpper(trimmed), "<STMTTRN>") {
		return imp.parseOFX(trimmed)
	}
	return imp.parseCSV(raw)
}

func (imp *CSVImporter) parseCSV(raw []byte) ([]domain.BankTransactionRow, error) {
	firstLine := string(raw)
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = firstLine[:idx]
	}

	reader := csv.NewReader(bytes.NewReader(raw))
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		reader.Comma = ';'
	}
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid CSV format: %v", domain.ErrValidation, err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	headerMap := make(map[string]int)
	startRow := 0
	for idx, col := range records[0] {
		norm := strings.ToLower(strings.TrimSpace(col))
		headerMap[norm] = idx
	}
	if isHeaderRow(records[0]) {
		startRow = 1
	}

	var out []domain.BankTransactionRow
	for i := startRow; i < len(records); i++ {
		rec := records[i]
		if len(rec) == 0 || (len(rec) == 1 && strings.TrimSpace(rec[0]) == "") {
			continue
		}

		row, err := parseCSVRecord(rec, headerMap, startRow == 1)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out = append(out, row)
	}
	return out, nil
}

func isHeaderRow(cols []string) bool {
	for _, c := range cols {
		n := strings.ToLower(strings.TrimSpace(c))
		switch n {
		case "lease_id", "bail_id", "amount_cents", "amount", "montant", "label", "libelle", "date", "payment_date":
			return true
		}
	}
	return false
}

func parseCSVRecord(rec []string, header map[string]int, hasHeader bool) (domain.BankTransactionRow, error) {
	getCol := func(names ...string) (string, bool) {
		if !hasHeader {
			return "", false
		}
		for _, n := range names {
			if idx, ok := header[n]; ok && idx < len(rec) {
				return strings.TrimSpace(rec[idx]), true
			}
		}
		return "", false
	}

	var (
		leaseID     string
		amountCents money.AmountCents
		label       string
		dateStr     string
		extRef      string
	)

	if hasHeader {
		leaseID, _ = getCol("lease_id", "bail_id", "lease")
		label, _ = getCol("label", "libelle", "description")
		dateStr, _ = getCol("payment_date", "date")
		extRef, _ = getCol("external_ref", "reference", "ref")

		if centsStr, ok := getCol("amount_cents", "montant_cents"); ok && centsStr != "" {
			parsed, err := strconv.ParseInt(centsStr, 10, 64)
			if err != nil {
				return domain.BankTransactionRow{}, fmt.Errorf("%w: invalid amount_cents %q", domain.ErrValidation, centsStr)
			}
			amountCents = money.AmountCents(parsed)
		} else if amtStr, ok := getCol("amount", "montant", "amount_eur"); ok && amtStr != "" {
			parsed, err := parseAmountToCents(amtStr)
			if err != nil {
				return domain.BankTransactionRow{}, err
			}
			amountCents = parsed
		}
	} else {
		// Positional fallback: lease_id, amount_cents (or EUR), label, payment_date [, external_ref]
		if len(rec) < 3 {
			return domain.BankTransactionRow{}, fmt.Errorf("%w: expected at least 3 columns (lease_id, amount_cents, label)", domain.ErrValidation)
		}
		leaseID = strings.TrimSpace(rec[0])
		parsed, err := parseAmountToCents(rec[1])
		if err != nil {
			return domain.BankTransactionRow{}, err
		}
		amountCents = parsed
		label = strings.TrimSpace(rec[2])
		if len(rec) >= 4 {
			dateStr = strings.TrimSpace(rec[3])
		}
		if len(rec) >= 5 {
			extRef = strings.TrimSpace(rec[4])
		}
	}

	paymentDate := time.Now().UTC()
	if dateStr != "" {
		parsedDate, err := parseFlexibleDate(dateStr)
		if err != nil {
			return domain.BankTransactionRow{}, err
		}
		paymentDate = parsedDate
	}

	return domain.BankTransactionRow{
		LeaseID:     leaseID,
		AmountCents: amountCents,
		Label:       label,
		PaymentDate: paymentDate,
		ExternalRef: extRef,
	}, nil
}

func (imp *CSVImporter) parseOFX(content string) ([]domain.BankTransactionRow, error) {
	blocks := ofxTrnBlockPattern.FindAllStringSubmatch(content, -1)
	var out []domain.BankTransactionRow
	for _, b := range blocks {
		tags := ofxTagPattern.FindAllStringSubmatch(b[1], -1)
		fields := make(map[string]string)
		for _, t := range tags {
			fields[strings.ToUpper(strings.TrimSpace(t[1]))] = strings.TrimSpace(t[2])
		}
		amtStr := fields["TRNAMT"]
		if amtStr == "" {
			continue
		}
		cents, err := parseDecimalEURToCents(amtStr)
		if err != nil || cents <= 0 {
			continue
		}
		label := strings.TrimSpace(fields["NAME"] + " " + fields["MEMO"])
		dateStr := fields["DTPOSTED"]
		payDate := time.Now().UTC()
		if len(dateStr) >= 8 {
			if t, err := time.Parse("20060102", dateStr[:8]); err == nil {
				payDate = t.UTC()
			}
		}
		out = append(out, domain.BankTransactionRow{
			LeaseID:     fields["CHECKNUM"],
			AmountCents: cents,
			Label:       label,
			PaymentDate: payDate,
			ExternalRef: fields["FITID"],
		})
	}
	return out, nil
}

func parseAmountToCents(raw string) (money.AmountCents, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	cleaned = strings.TrimSuffix(cleaned, "€")
	cleaned = strings.TrimSpace(cleaned)
	if strings.Contains(cleaned, ".") || strings.Contains(cleaned, ",") {
		return parseDecimalEURToCents(cleaned)
	}
	val, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid amount %q", domain.ErrValidation, raw)
	}
	return money.AmountCents(val), nil
}

// parseDecimalEURToCents parses a decimal euro string (e.g., "800.50", "800,5", "-120.00")
// into exact integer AmountCents using pure integer arithmetic without float64.
func parseDecimalEURToCents(raw string) (money.AmountCents, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	if normalized == "" {
		return 0, fmt.Errorf("%w: empty decimal amount", domain.ErrValidation)
	}

	sign := int64(1)
	if strings.HasPrefix(normalized, "-") {
		sign = -1
		normalized = strings.TrimPrefix(normalized, "-")
	} else if strings.HasPrefix(normalized, "+") {
		normalized = strings.TrimPrefix(normalized, "+")
	}

	parts := strings.Split(normalized, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("%w: invalid decimal amount %q", domain.ErrValidation, raw)
	}

	wholeStr := parts[0]
	if wholeStr == "" {
		wholeStr = "0"
	}
	wholeEuros, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid decimal amount %q", domain.ErrValidation, raw)
	}

	var fracCents int64
	if len(parts) == 2 && parts[1] != "" {
		fracStr := parts[1]
		if len(fracStr) > 2 {
			return 0, fmt.Errorf("%w: more than 2 decimal places in %q", domain.ErrValidation, raw)
		}
		if len(fracStr) == 1 {
			fracStr += "0"
		}
		fracCents, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: invalid fractional cents in %q", domain.ErrValidation, raw)
		}
	}

	return money.AmountCents(sign * (wholeEuros*100 + fracCents)), nil
}

func parseFlexibleDate(raw string) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		"02/01/2006",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: invalid date %q", domain.ErrValidation, raw)
}
