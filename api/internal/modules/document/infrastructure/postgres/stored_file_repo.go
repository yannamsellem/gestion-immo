package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// StoredFileRepo implements domain.StoredFileRepository using postgres.DBTX.
type StoredFileRepo struct {
	db postgres.DBTX
}

// NewStoredFileRepo constructs a new PostgreSQL repository for StoredFile aggregates.
func NewStoredFileRepo(db postgres.DBTX) *StoredFileRepo {
	return &StoredFileRepo{db: db}
}

// SaveStoredFile inserts a StoredFile record in PostgreSQL.
func (r *StoredFileRepo) SaveStoredFile(ctx context.Context, file domain.StoredFile) error {
	const query = `
		INSERT INTO stored_files (
			id, legal_entity_id, lease_id, category, file_name, content_type, r2_key, size_bytes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		file.ID,
		file.LegalEntityID,
		file.LeaseID,
		file.Category,
		file.FileName,
		file.ContentType,
		file.R2Key,
		file.SizeBytes,
		file.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("document/postgres/stored_file_repo: save stored file %s: %w", file.ID, err)
	}
	return nil
}

// FindStoredFileByID retrieves a StoredFile by ID.
func (r *StoredFileRepo) FindStoredFileByID(ctx context.Context, id string) (*domain.StoredFile, error) {
	const query = `
		SELECT id, legal_entity_id, lease_id, category, file_name, content_type, r2_key, size_bytes, created_at
		FROM stored_files
		WHERE id = $1
	`
	var f domain.StoredFile
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&f.ID,
		&f.LegalEntityID,
		&f.LeaseID,
		&f.Category,
		&f.FileName,
		&f.ContentType,
		&f.R2Key,
		&f.SizeBytes,
		&f.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStoredFileNotFound
		}
		return nil, fmt.Errorf("document/postgres/stored_file_repo: find stored file by id: %w", err)
	}
	return &f, nil
}

// ListStoredFilesByLease returns all StoredFiles attached to a lease.
func (r *StoredFileRepo) ListStoredFilesByLease(ctx context.Context, leaseID string) ([]domain.StoredFile, error) {
	const query = `
		SELECT id, legal_entity_id, lease_id, category, file_name, content_type, r2_key, size_bytes, created_at
		FROM stored_files
		WHERE lease_id = $1
		ORDER BY created_at ASC
	`
	return r.queryStoredFiles(ctx, query, leaseID)
}

// ListStoredFilesByEntity returns all StoredFiles attached to a legal entity.
func (r *StoredFileRepo) ListStoredFilesByEntity(ctx context.Context, entityID string) ([]domain.StoredFile, error) {
	const query = `
		SELECT id, legal_entity_id, lease_id, category, file_name, content_type, r2_key, size_bytes, created_at
		FROM stored_files
		WHERE legal_entity_id = $1
		ORDER BY created_at ASC
	`
	return r.queryStoredFiles(ctx, query, entityID)
}

func (r *StoredFileRepo) queryStoredFiles(ctx context.Context, query string, arg string) ([]domain.StoredFile, error) {
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("document/postgres/stored_file_repo: query stored files: %w", err)
	}
	defer rows.Close()

	var out []domain.StoredFile
	for rows.Next() {
		var f domain.StoredFile
		if err := rows.Scan(
			&f.ID,
			&f.LegalEntityID,
			&f.LeaseID,
			&f.Category,
			&f.FileName,
			&f.ContentType,
			&f.R2Key,
			&f.SizeBytes,
			&f.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("document/postgres/stored_file_repo: scan stored file: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
