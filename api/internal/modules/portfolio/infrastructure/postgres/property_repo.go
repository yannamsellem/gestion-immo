package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// PropertyRepo implements domain.PropertyRepository against PostgreSQL.
type PropertyRepo struct {
	db postgres.DBTX
}

// NewPropertyRepo constructs a PostgreSQL repository dedicated to Property.
func NewPropertyRepo(db postgres.DBTX) *PropertyRepo {
	return &PropertyRepo{db: db}
}

// CreateProperty inserts a new property row into PostgreSQL.
func (r *PropertyRepo) CreateProperty(ctx context.Context, prop domain.Property) error {
	const query = `
		INSERT INTO properties (
			id, legal_entity_id, label, property_type, accounting_category,
			address, acquisition_cents, work_stock_cents, total_book_value_cents,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		prop.ID,
		prop.LegalEntityID,
		prop.Label,
		prop.PropertyType,
		string(prop.AccountingCategory),
		prop.Address,
		int64(prop.AcquisitionCents),
		int64(prop.WorkStockCents),
		int64(prop.TotalBookValueCents),
		prop.CreatedAt,
		prop.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("portfolio/postgres/property_repo: create property: %w", err)
	}
	return nil
}

// FindPropertyByID retrieves a property by primary key.
func (r *PropertyRepo) FindPropertyByID(ctx context.Context, id string) (*domain.Property, error) {
	const query = `
		SELECT id, legal_entity_id, label, property_type, accounting_category,
		       address, acquisition_cents, work_stock_cents, total_book_value_cents,
		       created_at, updated_at
		FROM properties
		WHERE id = $1
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id)
	return scanProperty(row)
}

// ListPropertiesByEntity lists all properties belonging to a legal entity.
func (r *PropertyRepo) ListPropertiesByEntity(ctx context.Context, entityID string) ([]domain.Property, error) {
	const query = `
		SELECT id, legal_entity_id, label, property_type, accounting_category,
		       address, acquisition_cents, work_stock_cents, total_book_value_cents,
		       created_at, updated_at
		FROM properties
		WHERE legal_entity_id = $1
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, entityID)
	if err != nil {
		return nil, fmt.Errorf("portfolio/postgres/property_repo: list properties: %w", err)
	}
	defer rows.Close()

	var properties []domain.Property
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, err
		}
		properties = append(properties, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio/postgres/property_repo: iterate properties: %w", err)
	}
	return properties, nil
}

func scanProperty(row pgx.Row) (*domain.Property, error) {
	var p domain.Property
	var catStr string
	var acqCents, workCents, totalCents int64
	err := row.Scan(
		&p.ID,
		&p.LegalEntityID,
		&p.Label,
		&p.PropertyType,
		&catStr,
		&p.Address,
		&acqCents,
		&workCents,
		&totalCents,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPropertyNotFound
		}
		return nil, fmt.Errorf("portfolio/postgres/property_repo: scan property: %w", err)
	}
	p.AccountingCategory = domain.AccountingCategory(catStr)
	p.AcquisitionCents = money.AmountCents(acqCents)
	p.WorkStockCents = money.AmountCents(workCents)
	p.TotalBookValueCents = money.AmountCents(totalCents)
	return &p, nil
}
