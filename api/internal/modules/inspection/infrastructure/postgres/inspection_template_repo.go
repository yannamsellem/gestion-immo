package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// InspectionTemplateRepo implements domain.InspectionTemplateRepository using postgres.DBTX.
type InspectionTemplateRepo struct {
	db postgres.DBTX
}

// NewInspectionTemplateRepo constructs a new PostgreSQL repository for InspectionTemplate aggregates.
func NewInspectionTemplateRepo(db postgres.DBTX) *InspectionTemplateRepo {
	return &InspectionTemplateRepo{db: db}
}

// SaveTemplate inserts or updates an InspectionTemplate in PostgreSQL.
func (r *InspectionTemplateRepo) SaveTemplate(ctx context.Context, tpl domain.InspectionTemplate) error {
	const query = `
		INSERT INTO inspection_templates (
			id, name, rooms, default_elements, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			rooms = EXCLUDED.rooms,
			default_elements = EXCLUDED.default_elements,
			updated_at = EXCLUDED.updated_at
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		tpl.ID,
		tpl.Name,
		tpl.Rooms,
		tpl.DefaultElements,
		tpl.CreatedAt,
		tpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inspection/postgres/inspection_template_repo: save inspection template %s: %w", tpl.ID, err)
	}
	return nil
}

// FindTemplateByID retrieves an InspectionTemplate by ID.
func (r *InspectionTemplateRepo) FindTemplateByID(ctx context.Context, id string) (*domain.InspectionTemplate, error) {
	const query = `
		SELECT id, name, rooms, default_elements, created_at, updated_at
		FROM inspection_templates
		WHERE id = $1
	`
	var tpl domain.InspectionTemplate
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Rooms,
		&tpl.DefaultElements,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("inspection/postgres/inspection_template_repo: find inspection template by id: %w", err)
	}
	return &tpl, nil
}

// ListTemplates returns all InspectionTemplates ordered by creation date.
func (r *InspectionTemplateRepo) ListTemplates(ctx context.Context) ([]domain.InspectionTemplate, error) {
	const query = `
		SELECT id, name, rooms, default_elements, created_at, updated_at
		FROM inspection_templates
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("inspection/postgres/inspection_template_repo: list inspection templates: %w", err)
	}
	defer rows.Close()

	var out []domain.InspectionTemplate
	for rows.Next() {
		var tpl domain.InspectionTemplate
		if err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tpl.Rooms,
			&tpl.DefaultElements,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("inspection/postgres/inspection_template_repo: scan inspection template: %w", err)
		}
		out = append(out, tpl)
	}
	return out, rows.Err()
}
