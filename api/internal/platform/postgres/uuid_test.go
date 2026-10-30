package postgres_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

func TestNullableUUIDAndDerefString(t *testing.T) {
	if got := postgres.NullableUUID(""); got != nil {
		t.Fatalf("expected nil for empty string, got %v", *got)
	}
	if got := postgres.NullableUUID("   "); got != nil {
		t.Fatalf("expected nil for whitespace string, got %v", *got)
	}
	raw := "  0192dce0-0000-7000-8000-000000000001  "
	ptr := postgres.NullableUUID(raw)
	if ptr == nil || *ptr != "0192dce0-0000-7000-8000-000000000001" {
		t.Fatalf("expected trimmed UUID pointer, got %v", ptr)
	}
	if got := postgres.DerefString(nil); got != "" {
		t.Fatalf("expected empty string from nil pointer, got %q", got)
	}
	if got := postgres.DerefString(ptr); got != "0192dce0-0000-7000-8000-000000000001" {
		t.Fatalf("expected dereferenced string, got %q", got)
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	fkErr := fmt.Errorf("insert failed: %w", &pgconn.PgError{Code: "23503"})
	if !postgres.IsForeignKeyViolation(fkErr) {
		t.Fatal("expected SQLSTATE 23503 to be detected as foreign key violation")
	}
	uniqueErr := fmt.Errorf("insert failed: %w", &pgconn.PgError{Code: "23505"})
	if postgres.IsForeignKeyViolation(uniqueErr) {
		t.Fatal("did not expect SQLSTATE 23505 to be detected as foreign key violation")
	}
	if postgres.IsForeignKeyViolation(errors.New("plain error")) || postgres.IsForeignKeyViolation(nil) {
		t.Fatal("did not expect non-PgError or nil to be detected as foreign key violation")
	}
}
