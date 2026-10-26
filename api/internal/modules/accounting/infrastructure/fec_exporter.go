package infrastructure

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// NormativeFECColumns lists the 18 mandatory DGFiP FEC columns (Article A.47 A-1 du Livre des Procédures Fiscales).
var NormativeFECColumns = []string{
	"JournalCode",
	"JournalLib",
	"EcritureNum",
	"EcritureDate",
	"CompteNum",
	"CompteLib",
	"CompAuxNum",
	"CompAuxLib",
	"PieceRef",
	"PieceDate",
	"EcritureLib",
	"Debit",
	"Credit",
	"EcritureLet",
	"DateLet",
	"ValidDate",
	"Montantdevise",
	"Idevise",
}

// FECExporter serializes accounting entries into a normative 18-column DGFiP FEC stream.
type FECExporter struct {
	delimiter string
}

// NewFECExporter constructs a new FECExporter using TAB ("\t") as the normative DGFiP column separator.
func NewFECExporter() *FECExporter {
	return &FECExporter{delimiter: "\t"}
}

// WithDelimiter returns a copy of FECExporter configured with the given column delimiter ("\t" or "|").
func (e *FECExporter) WithDelimiter(delim string) *FECExporter {
	if delim != "\t" && delim != "|" {
		delim = "\t"
	}
	return &FECExporter{delimiter: delim}
}

// ExportFEC serializes entries into the 18-column DGFiP FEC format (UTF-8, CRLF/LF lines, French decimal comma).
func (e *FECExporter) ExportFEC(entries []domain.FECEntry) ([]byte, error) {
	delim := "\t"
	if e != nil && (e.delimiter == "\t" || e.delimiter == "|") {
		delim = e.delimiter
	}

	var buf bytes.Buffer
	buf.WriteString(strings.Join(NormativeFECColumns, delim))
	buf.WriteByte('\n')

	for _, row := range entries {
		entryDate := formatFECDate(row.EntryDate)
		documentDate := formatFECDate(row.DocumentDate)
		if documentDate == "" {
			documentDate = entryDate
		}
		validationDate := formatFECDate(row.ValidationDate)
		if validationDate == "" {
			validationDate = entryDate
		}

		currencyCode := strings.TrimSpace(row.CurrencyCode)
		if currencyCode == "" {
			currencyCode = "EUR"
		}

		foreignCurrencyAmount := ""
		if row.ForeignCurrencyAmountCents != 0 {
			foreignCurrencyAmount = formatFECAmount(row.ForeignCurrencyAmountCents)
		}

		cols := []string{
			sanitizeFECField(row.JournalCode),
			sanitizeFECField(row.JournalLabel),
			sanitizeFECField(row.EntryNumber),
			entryDate,
			sanitizeFECField(row.AccountCode),
			sanitizeFECField(row.AccountLabel),
			sanitizeFECField(row.SubAccountCode),
			sanitizeFECField(row.SubAccountLabel),
			sanitizeFECField(row.DocumentRef),
			documentDate,
			sanitizeFECField(row.EntryLabel),
			formatFECAmount(row.DebitCents),
			formatFECAmount(row.CreditCents),
			sanitizeFECField(row.ReconciliationCode),
			sanitizeFECField(row.ReconciliationDate),
			validationDate,
			foreignCurrencyAmount,
			sanitizeFECField(currencyCode),
		}

		buf.WriteString(strings.Join(cols, delim))
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func formatFECDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("20060102")
}

// formatFECAmount formats an exact integer cent amount into DGFiP FEC notation (e.g. "133607,00").
func formatFECAmount(c money.AmountCents) string {
	val := int64(c)
	sign := ""
	if val < 0 {
		sign = "-"
		val = -val
	}
	euros := val / 100
	cents := val % 100
	return fmt.Sprintf("%s%d,%02d", sign, euros, cents)
}

func sanitizeFECField(s string) string {
	cleaned := strings.ReplaceAll(s, "\t", " ")
	cleaned = strings.ReplaceAll(cleaned, "|", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	return strings.TrimSpace(cleaned)
}
