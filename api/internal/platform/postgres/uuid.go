package postgres

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// NullableUUID trims s and returns nil when empty so pgx binds SQL NULL on nullable UUID columns.
func NullableUUID(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// DerefString safely unwraps a nullable scanned string pointer into a plain string ("" when nil).
func DerefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// IsForeignKeyViolation reports whether err wraps a PostgreSQL SQLSTATE 23503 foreign_key_violation.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
