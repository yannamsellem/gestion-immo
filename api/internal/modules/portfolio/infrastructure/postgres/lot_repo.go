package postgres

import (
	"context"
	"fmt"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// LotRepo implements domain.LotRepository against PostgreSQL.
type LotRepo struct {
	db postgres.DBTX
}

// NewLotRepo constructs a PostgreSQL repository dedicated to Lot.
func NewLotRepo(db postgres.DBTX) *LotRepo {
	return &LotRepo{db: db}
}

// CreateLot inserts a new rentable lot into PostgreSQL.
func (r *LotRepo) CreateLot(ctx context.Context, lot domain.Lot) error {
	const query = `
		INSERT INTO lots (
			id, property_id, label, lot_type, surface_m2, ownership_share, floor, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		lot.ID,
		lot.PropertyID,
		lot.Label,
		lot.LotType,
		lot.SurfaceM2,
		lot.OwnershipShare,
		lot.Floor,
		lot.CreatedAt,
		lot.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("portfolio/postgres/lot_repo: create lot: %w", err)
	}
	return nil
}

// ListLotsByProperty lists all lots belonging to a property.
func (r *LotRepo) ListLotsByProperty(ctx context.Context, propertyID string) ([]domain.Lot, error) {
	const query = `
		SELECT id, property_id, label, lot_type, surface_m2, ownership_share, floor, created_at, updated_at
		FROM lots
		WHERE property_id = $1
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, propertyID)
	if err != nil {
		return nil, fmt.Errorf("portfolio/postgres/lot_repo: list lots: %w", err)
	}
	defer rows.Close()

	var lots []domain.Lot
	for rows.Next() {
		var l domain.Lot
		if err := rows.Scan(
			&l.ID,
			&l.PropertyID,
			&l.Label,
			&l.LotType,
			&l.SurfaceM2,
			&l.OwnershipShare,
			&l.Floor,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("portfolio/postgres/lot_repo: scan lot: %w", err)
		}
		lots = append(lots, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio/postgres/lot_repo: iterate lots: %w", err)
	}
	return lots, nil
}
