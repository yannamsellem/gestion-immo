package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// DocumentTemplateRepo implements domain.DocumentTemplateRepository using postgres.DBTX.
type DocumentTemplateRepo struct {
	db postgres.DBTX
}

// NewDocumentTemplateRepo constructs a new PostgreSQL repository for DocumentTemplate aggregates.
func NewDocumentTemplateRepo(db postgres.DBTX) *DocumentTemplateRepo {
	return &DocumentTemplateRepo{db: db}
}

// SaveTemplate inserts or updates a DocumentTemplate in PostgreSQL.
func (r *DocumentTemplateRepo) SaveTemplate(ctx context.Context, tpl domain.DocumentTemplate) error {
	const query = `
		INSERT INTO document_templates (
			id, name, template_type, format, r2_key, html_body, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			template_type = EXCLUDED.template_type,
			format = EXCLUDED.format,
			r2_key = EXCLUDED.r2_key,
			html_body = EXCLUDED.html_body,
			updated_at = EXCLUDED.updated_at
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		tpl.ID,
		tpl.Name,
		string(tpl.TemplateType),
		string(tpl.Format),
		tpl.R2Key,
		tpl.HTMLBody,
		tpl.CreatedAt,
		tpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("document/postgres/document_template_repo: save template %s: %w", tpl.ID, err)
	}
	return nil
}

// FindTemplateByID retrieves a DocumentTemplate by ID.
func (r *DocumentTemplateRepo) FindTemplateByID(ctx context.Context, id string) (*domain.DocumentTemplate, error) {
	const query = `
		SELECT id, name, template_type, format, r2_key, html_body, created_at, updated_at
		FROM document_templates
		WHERE id = $1
	`
	var tpl domain.DocumentTemplate
	var tplType, format string
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&tpl.ID,
		&tpl.Name,
		&tplType,
		&format,
		&tpl.R2Key,
		&tpl.HTMLBody,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("document/postgres/document_template_repo: find template by id: %w", err)
	}
	tpl.TemplateType = domain.TemplateType(tplType)
	tpl.Format = domain.TemplateFormat(format)
	return &tpl, nil
}

// ListTemplates returns all DocumentTemplates ordered by creation date.
func (r *DocumentTemplateRepo) ListTemplates(ctx context.Context) ([]domain.DocumentTemplate, error) {
	const query = `
		SELECT id, name, template_type, format, r2_key, html_body, created_at, updated_at
		FROM document_templates
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("document/postgres/document_template_repo: list templates: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentTemplate
	for rows.Next() {
		var tpl domain.DocumentTemplate
		var tplType, format string
		if err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tplType,
			&format,
			&tpl.R2Key,
			&tpl.HTMLBody,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("document/postgres/document_template_repo: scan template: %w", err)
		}
		tpl.TemplateType = domain.TemplateType(tplType)
		tpl.Format = domain.TemplateFormat(format)
		out = append(out, tpl)
	}
	return out, rows.Err()
}
