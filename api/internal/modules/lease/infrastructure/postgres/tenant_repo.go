package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// TenantRepo implements domain.TenantRepository against PostgreSQL.
type TenantRepo struct {
	db postgres.DBTX
}

// NewTenantRepo constructs a PostgreSQL repository dedicated to Tenant.
func NewTenantRepo(db postgres.DBTX) *TenantRepo {
	return &TenantRepo{db: db}
}

// CreateTenant inserts a new tenant row into PostgreSQL.
func (r *TenantRepo) CreateTenant(ctx context.Context, tenant domain.Tenant) error {
	const query = `
		INSERT INTO tenants (
			id, full_name, email, phone, address, archived, archived_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		tenant.ID,
		tenant.FullName,
		tenant.Email,
		tenant.Phone,
		tenant.Address,
		tenant.Archived,
		tenant.ArchivedAt,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("lease/postgres/tenant_repo: create tenant: %w", err)
	}
	return nil
}

// FindTenantByID retrieves a tenant by primary key.
func (r *TenantRepo) FindTenantByID(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	if !id.IsValidUUID(tenantID) {
		return nil, domain.ErrTenantNotFound
	}
	const query = `
		SELECT id, full_name, email, phone, address, archived, archived_at, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query, tenantID)
	return scanTenant(row)
}

// ArchiveTenant marks a tenant as archived with the given timestamp.
func (r *TenantRepo) ArchiveTenant(ctx context.Context, tenantID string, archivedAt time.Time) error {
	if !id.IsValidUUID(tenantID) {
		return domain.ErrTenantNotFound
	}
	const query = `
		UPDATE tenants
		SET archived = TRUE, archived_at = $2, updated_at = $2
		WHERE id = $1
	`
	tag, err := postgres.Conn(ctx, r.db).Exec(ctx, query, tenantID, archivedAt)
	if err != nil {
		return fmt.Errorf("lease/postgres/tenant_repo: archive tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTenantNotFound
	}
	return nil
}

// ListTenants lists present tenants, or all tenants if includeArchived is true.
func (r *TenantRepo) ListTenants(ctx context.Context, includeArchived bool) ([]domain.Tenant, error) {
	query := `
		SELECT id, full_name, email, phone, address, archived, archived_at, created_at, updated_at
		FROM tenants
	`
	if !includeArchived {
		query += ` WHERE archived = FALSE`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("lease/postgres/tenant_repo: list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lease/postgres/tenant_repo: iterate tenants: %w", err)
	}
	return tenants, nil
}

func scanTenant(row pgx.Row) (*domain.Tenant, error) {
	var t domain.Tenant
	err := row.Scan(
		&t.ID,
		&t.FullName,
		&t.Email,
		&t.Phone,
		&t.Address,
		&t.Archived,
		&t.ArchivedAt,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("lease/postgres/tenant_repo: scan tenant: %w", err)
	}
	return &t, nil
}
