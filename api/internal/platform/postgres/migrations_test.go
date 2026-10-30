package postgres_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMigrations_EnforceUUIDv7AndNoLegacyTextIDs(t *testing.T) {
	migrationsDir := filepath.Join("..", "..", "..", "migrations")
	matches, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(matches)

	expectedFiles := []string{
		"001_auth_and_users.sql",
		"002_portfolio_hierarchy.sql",
		"003_leases_and_tenants.sql",
		"004_payments_and_cash_journal.sql",
		"005_templates_inspections_incidents.sql",
		"006_accounting_and_annual_accounts.sql",
	}
	if len(matches) != len(expectedFiles) {
		t.Fatalf("expected %d migration files (%v), got %d (%v)", len(expectedFiles), expectedFiles, len(matches), matches)
	}
	for i, want := range expectedFiles {
		if got := filepath.Base(matches[i]); got != want {
			t.Fatalf("migration[%d] = %q, want %q", i, got, want)
		}
	}

	var combined strings.Builder
	for _, p := range matches {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		combined.Write(raw)
		combined.WriteByte('\n')
	}
	allSQL := combined.String()

	if strings.Contains(allSQL, "pcg_accounts") {
		t.Fatal("dead table pcg_accounts must not appear in greenfield migrations")
	}
	if strings.Contains(allSQL, "id TEXT PRIMARY KEY") {
		t.Fatal("legacy 'id TEXT PRIMARY KEY' must not appear in greenfield migrations")
	}
	pkCount := strings.Count(allSQL, "id UUID PRIMARY KEY DEFAULT uuidv7()")
	if pkCount != 24 {
		t.Fatalf("expected 24 tables with 'id UUID PRIMARY KEY DEFAULT uuidv7()', got %d", pkCount)
	}

	forbiddenSnippets := []string{
		"legal_entity_id TEXT",
		"lease_id TEXT",
		"template_id TEXT",
		"property_id TEXT",
		"lot_id TEXT",
		"tenant_id TEXT",
		"user_id TEXT",
		"family_id TEXT",
		"payment_id TEXT",
		"rent_call_id TEXT",
		"inspection_id TEXT",
		"entry_inspection_id TEXT",
		"journal_entry_id TEXT",
		"approved_by TEXT",
		"replaced_by_id TEXT",
	}
	for _, forbidden := range forbiddenSnippets {
		if strings.Contains(allSQL, forbidden) {
			t.Errorf("forbidden legacy TEXT key column %q still present in migrations", forbidden)
		}
	}

	requiredSnippets := []string{
		"legal_entity_id UUID REFERENCES legal_entities(id) ON DELETE SET NULL",
		"lease_id UUID REFERENCES leases(id) ON DELETE SET NULL",
		"template_id UUID REFERENCES document_templates(id) ON DELETE SET NULL",
		"property_id UUID REFERENCES properties(id) ON DELETE SET NULL",
		"lot_id UUID REFERENCES lots(id) ON DELETE SET NULL",
		"lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE",
		"line_id TEXT NOT NULL DEFAULT ''",
		"line_id TEXT NOT NULL",
		"disabled_line_ids TEXT[] NOT NULL DEFAULT '{}'",
	}
	for _, req := range requiredSnippets {
		if !strings.Contains(allSQL, req) {
			t.Errorf("expected SQL snippet %q in migrations", req)
		}
	}
}
