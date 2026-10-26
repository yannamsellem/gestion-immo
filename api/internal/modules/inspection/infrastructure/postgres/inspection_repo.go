package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// InspectionRepo implements domain.InspectionRepository using postgres.DBTX.
type InspectionRepo struct {
	pool *pgxpool.Pool
	db   postgres.DBTX
}

// NewInspectionRepo constructs a new PostgreSQL repository for Inspection aggregates.
func NewInspectionRepo(pool *pgxpool.Pool, db postgres.DBTX) *InspectionRepo {
	if db == nil {
		db = pool
	}
	return &InspectionRepo{pool: pool, db: db}
}

// SaveInspection inserts or updates an Inspection record in PostgreSQL.
func (r *InspectionRepo) SaveInspection(ctx context.Context, insp domain.Inspection) error {
	const query = `
		INSERT INTO inspections (
			id, lease_id, template_id, inspection_type, entry_inspection_id,
			report_pdf_r2_key, inspection_date, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9
		)
		ON CONFLICT (id) DO UPDATE SET
			entry_inspection_id = EXCLUDED.entry_inspection_id,
			report_pdf_r2_key = EXCLUDED.report_pdf_r2_key,
			inspection_date = EXCLUDED.inspection_date,
			updated_at = EXCLUDED.updated_at
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		insp.ID,
		insp.LeaseID,
		insp.TemplateID,
		string(insp.Type),
		insp.EntryInspectionID,
		insp.ReportPDFR2Key,
		insp.InspectionDate,
		insp.CreatedAt,
		insp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inspection/postgres/inspection_repo: save inspection %s: %w", insp.ID, err)
	}
	return nil
}

// FindInspectionByID retrieves an Inspection and its items by ID.
func (r *InspectionRepo) FindInspectionByID(ctx context.Context, id string) (*domain.Inspection, error) {
	const query = `
		SELECT id, lease_id, template_id, inspection_type, entry_inspection_id,
		       report_pdf_r2_key, inspection_date, created_at, updated_at
		FROM inspections
		WHERE id = $1
	`
	var insp domain.Inspection
	var inspType string
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&insp.ID,
		&insp.LeaseID,
		&insp.TemplateID,
		&inspType,
		&insp.EntryInspectionID,
		&insp.ReportPDFR2Key,
		&insp.InspectionDate,
		&insp.CreatedAt,
		&insp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInspectionNotFound
		}
		return nil, fmt.Errorf("inspection/postgres/inspection_repo: find inspection by id: %w", err)
	}
	insp.Type = domain.InspectionType(inspType)

	items, err := r.loadInspectionItems(ctx, insp.ID)
	if err != nil {
		return nil, err
	}
	insp.Items = items
	return &insp, nil
}

// ListInspectionsByLease returns all Inspections and their items for a lease.
func (r *InspectionRepo) ListInspectionsByLease(ctx context.Context, leaseID string) ([]domain.Inspection, error) {
	const query = `
		SELECT id, lease_id, template_id, inspection_type, entry_inspection_id,
		       report_pdf_r2_key, inspection_date, created_at, updated_at
		FROM inspections
		WHERE lease_id = $1
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, leaseID)
	if err != nil {
		return nil, fmt.Errorf("inspection/postgres/inspection_repo: list inspections by lease: %w", err)
	}
	defer rows.Close()

	var out []domain.Inspection
	for rows.Next() {
		var insp domain.Inspection
		var inspType string
		if err := rows.Scan(
			&insp.ID,
			&insp.LeaseID,
			&insp.TemplateID,
			&inspType,
			&insp.EntryInspectionID,
			&insp.ReportPDFR2Key,
			&insp.InspectionDate,
			&insp.CreatedAt,
			&insp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("inspection/postgres/inspection_repo: scan inspection: %w", err)
		}
		insp.Type = domain.InspectionType(inspType)
		out = append(out, insp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		items, err := r.loadInspectionItems(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

// UpsertInspectionItems inserts or updates inspection items keyed by (inspection_id, room_name, element_name) atomically.
func (r *InspectionRepo) UpsertInspectionItems(ctx context.Context, inspectionID string, items []domain.InspectionItem) error {
	return postgres.WithinTransaction(ctx, r.pool, func(txCtx context.Context) error {
		conn := postgres.Conn(txCtx, r.db)

		const query = `
			INSERT INTO inspection_items (
				id, inspection_id, room_name, element_name, condition, observation, photo_r2_keys, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
			)
			ON CONFLICT (inspection_id, room_name, element_name) DO UPDATE SET
				condition = EXCLUDED.condition,
				observation = EXCLUDED.observation,
				photo_r2_keys = EXCLUDED.photo_r2_keys,
				updated_at = EXCLUDED.updated_at
		`
		for _, item := range items {
			photos := item.PhotoR2Keys
			if photos == nil {
				photos = []string{}
			}
			if _, err := conn.Exec(txCtx, query,
				item.ID,
				inspectionID,
				item.RoomName,
				item.ElementName,
				string(item.Condition),
				item.Observation,
				photos,
				item.CreatedAt,
				item.UpdatedAt,
			); err != nil {
				return fmt.Errorf("inspection/postgres/inspection_repo: upsert inspection item (%s/%s): %w", item.RoomName, item.ElementName, err)
			}
		}

		if _, err := conn.Exec(txCtx, `UPDATE inspections SET updated_at = NOW() WHERE id = $1`, inspectionID); err != nil {
			return fmt.Errorf("inspection/postgres/inspection_repo: touch inspection updated_at: %w", err)
		}

		return nil
	})
}

func (r *InspectionRepo) loadInspectionItems(ctx context.Context, inspectionID string) ([]domain.InspectionItem, error) {
	const query = `
		SELECT id, inspection_id, room_name, element_name, condition, observation, photo_r2_keys, created_at, updated_at
		FROM inspection_items
		WHERE inspection_id = $1
		ORDER BY created_at ASC, id ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, inspectionID)
	if err != nil {
		return nil, fmt.Errorf("inspection/postgres/inspection_repo: load inspection items: %w", err)
	}
	defer rows.Close()

	var items []domain.InspectionItem
	for rows.Next() {
		var item domain.InspectionItem
		var cond string
		if err := rows.Scan(
			&item.ID,
			&item.InspectionID,
			&item.RoomName,
			&item.ElementName,
			&cond,
			&item.Observation,
			&item.PhotoR2Keys,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("inspection/postgres/inspection_repo: scan inspection item: %w", err)
		}
		item.Condition = domain.ConditionState(cond)
		items = append(items, item)
	}
	return items, rows.Err()
}
